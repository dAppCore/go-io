// Package datanode provides an in-memory io.Medium backed by Borg's DataNode.
//
// DataNode is an in-memory fs.FS that serialises to tar. Wrapping it as a
// Medium lets any code that works with io.Medium transparently operate on
// an in-memory filesystem that can be snapshotted, shipped as a crash report,
// or wrapped in a TIM container for runc execution.
package datanode

import (
	"cmp"
	goio "io"
	"io/fs"
	"path"
	"slices"
	"sync"
	"time"

	core "dappco.re/go/core"
	borgdatanode "forge.lthn.ai/Snider/Borg/pkg/datanode"
)

var (
	dataNodeWalkDir = func(fsys fs.FS, root string, fn fs.WalkDirFunc) error {
		return fs.WalkDir(fsys, root, fn)
	}
	dataNodeOpen = func(dn *borgdatanode.DataNode, name string) (fs.File, error) {
		return dn.Open(name)
	}
	dataNodeReadAll = func(r goio.Reader) ([]byte, error) {
		return goio.ReadAll(r)
	}
)

// Medium is an in-memory storage backend backed by a Borg DataNode.
// All paths are relative (no leading slash). Thread-safe via RWMutex.
type Medium struct {
	dataNode    *borgdatanode.DataNode
	directories map[string]bool // explicit directory tracking
	mu          sync.RWMutex
}

// New creates a new empty DataNode Medium.
//
// Example usage:
//
//	medium := datanode.New()
//	_ = medium.Write("jobs/run.log", "started")
func New() *Medium {
	return &Medium{
		dataNode:    borgdatanode.New(),
		directories: make(map[string]bool),
	}
}

// FromTar creates a Medium from a tarball, restoring all files.
//
// Example usage:
//
//	sourceMedium := datanode.New()
//	snapshot, _ := sourceMedium.Snapshot()
//	restored, _ := datanode.FromTar(snapshot)
func FromTar(data []byte) (*Medium, error) {
	dataNode, err := borgdatanode.FromTar(data)
	if err != nil {
		return nil, core.E("datanode.FromTar", "failed to restore", err)
	}
	return &Medium{
		dataNode:    dataNode,
		directories: make(map[string]bool),
	}, nil
}

// Snapshot serialises the entire filesystem to a tarball.
// Use this for crash reports, workspace packaging, or TIM creation.
//
//	result := m.Snapshot(...)
func (m *Medium) Snapshot() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data, err := m.dataNode.ToTar()
	if err != nil {
		return nil, core.E("datanode.Snapshot", "tar failed", err)
	}
	return data, nil
}

// Restore replaces the filesystem contents from a tarball.
//
//	result := m.Restore(...)
func (m *Medium) Restore(data []byte) error {
	dataNode, err := borgdatanode.FromTar(data)
	if err != nil {
		return core.E("datanode.Restore", "tar failed", err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dataNode = dataNode
	m.directories = make(map[string]bool)
	return nil
}

// DataNode returns the underlying Borg DataNode.
// Use this to wrap the filesystem in a TIM container.
//
//	result := m.DataNode(...)
func (m *Medium) DataNode() *borgdatanode.DataNode {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.dataNode
}

// cleanPath normalises a path: strips leading slash, cleans traversal.
func cleanPath(p string) string {
	p = core.TrimPrefix(p, "/")
	p = path.Clean(p)
	if p == "." {
		return ""
	}
	return p
}

// --- io.Medium interface ---

// Read documents the Read operation.
//
//	result := m.Read(...)
func (m *Medium) Read(p string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p = cleanPath(p)
	f, err := m.dataNode.Open(p)
	if err != nil {
		return "", core.E("datanode.Read", core.Concat("not found: ", p), fs.ErrNotExist)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", core.E("datanode.Read", core.Concat("stat failed: ", p), err)
	}
	if info.IsDir() {
		return "", core.E("datanode.Read", core.Concat("is a directory: ", p), fs.ErrInvalid)
	}

	data, err := goio.ReadAll(f)
	if err != nil {
		return "", core.E("datanode.Read", core.Concat("read failed: ", p), err)
	}
	return string(data), nil
}

// Write documents the Write operation.
//
//	result := m.Write(...)
func (m *Medium) Write(p, content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	p = cleanPath(p)
	if p == "" {
		return core.E("datanode.Write", "empty path", fs.ErrInvalid)
	}
	m.dataNode.AddData(p, []byte(content))

	// ensure parent directories are tracked
	m.ensureDirsLocked(path.Dir(p))
	return nil
}

// WriteMode documents the WriteMode operation.
//
//	result := m.WriteMode(...)
func (m *Medium) WriteMode(p, content string, mode fs.FileMode) error {
	return m.Write(p, content)
}

// EnsureDir documents the EnsureDir operation.
//
//	result := m.EnsureDir(...)
func (m *Medium) EnsureDir(p string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	p = cleanPath(p)
	if p == "" {
		return nil
	}
	m.ensureDirsLocked(p)
	return nil
}

// ensureDirsLocked marks a directory and all ancestors as existing.
// Caller must hold m.mu.
func (m *Medium) ensureDirsLocked(p string) {
	for p != "" && p != "." {
		m.directories[p] = true
		p = path.Dir(p)
		if p == "." {
			break
		}
	}
}

// IsFile documents the IsFile operation.
//
//	result := m.IsFile(...)
func (m *Medium) IsFile(p string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p = cleanPath(p)
	info, err := m.dataNode.Stat(p)
	return err == nil && !info.IsDir()
}

// FileGet documents the FileGet operation.
//
//	result := m.FileGet(...)
func (m *Medium) FileGet(p string) (string, error) {
	return m.Read(p)
}

// FileSet documents the FileSet operation.
//
//	result := m.FileSet(...)
func (m *Medium) FileSet(p, content string) error {
	return m.Write(p, content)
}

// Delete documents the Delete operation.
//
//	result := m.Delete(...)
func (m *Medium) Delete(p string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	p = cleanPath(p)
	if p == "" {
		return core.E("datanode.Delete", "cannot delete root", fs.ErrPermission)
	}

	// Check if it's a file in the DataNode
	info, err := m.dataNode.Stat(p)
	if err != nil {
		// Check explicit directories
		if m.directories[p] {
			// Check if dir is empty
			hasChildren, err := m.hasPrefixLocked(p + "/")
			if err != nil {
				return core.E("datanode.Delete", core.Concat("failed to inspect directory: ", p), err)
			}
			if hasChildren {
				return core.E("datanode.Delete", core.Concat("directory not empty: ", p), fs.ErrExist)
			}
			delete(m.directories, p)
			return nil
		}
		return core.E("datanode.Delete", core.Concat("not found: ", p), fs.ErrNotExist)
	}

	if info.IsDir() {
		hasChildren, err := m.hasPrefixLocked(p + "/")
		if err != nil {
			return core.E("datanode.Delete", core.Concat("failed to inspect directory: ", p), err)
		}
		if hasChildren {
			return core.E("datanode.Delete", core.Concat("directory not empty: ", p), fs.ErrExist)
		}
		delete(m.directories, p)
		return nil
	}

	// Remove the file by creating a new DataNode without it
	if err := m.removeFileLocked(p); err != nil {
		return core.E("datanode.Delete", core.Concat("failed to delete file: ", p), err)
	}
	return nil
}

// DeleteAll documents the DeleteAll operation.
//
//	result := m.DeleteAll(...)
func (m *Medium) DeleteAll(p string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	p = cleanPath(p)
	if p == "" {
		return core.E("datanode.DeleteAll", "cannot delete root", fs.ErrPermission)
	}

	prefix := p + "/"
	found := false

	// Check if p itself is a file
	info, err := m.dataNode.Stat(p)
	if err == nil && !info.IsDir() {
		if err := m.removeFileLocked(p); err != nil {
			return core.E("datanode.DeleteAll", core.Concat("failed to delete file: ", p), err)
		}
		found = true
	}

	// Remove all files under prefix
	entries, err := m.collectAllLocked()
	if err != nil {
		return core.E("datanode.DeleteAll", core.Concat("failed to inspect tree: ", p), err)
	}
	for _, name := range entries {
		if name == p || core.HasPrefix(name, prefix) {
			if err := m.removeFileLocked(name); err != nil {
				return core.E("datanode.DeleteAll", core.Concat("failed to delete file: ", name), err)
			}
			found = true
		}
	}

	// Remove explicit directories under prefix
	for d := range m.directories {
		if d == p || core.HasPrefix(d, prefix) {
			delete(m.directories, d)
			found = true
		}
	}

	if !found {
		return core.E("datanode.DeleteAll", core.Concat("not found: ", p), fs.ErrNotExist)
	}
	return nil
}

// Rename documents the Rename operation.
//
//	result := m.Rename(...)
func (m *Medium) Rename(oldPath, newPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	oldPath = cleanPath(oldPath)
	newPath = cleanPath(newPath)

	// Check if source is a file
	info, err := m.dataNode.Stat(oldPath)
	if err != nil {
		return core.E("datanode.Rename", core.Concat("not found: ", oldPath), fs.ErrNotExist)
	}

	if !info.IsDir() {
		// Read old, write new, delete old
		data, err := m.readFileLocked(oldPath)
		if err != nil {
			return core.E("datanode.Rename", core.Concat("failed to read source file: ", oldPath), err)
		}
		m.dataNode.AddData(newPath, data)
		m.ensureDirsLocked(path.Dir(newPath))
		if err := m.removeFileLocked(oldPath); err != nil {
			return core.E("datanode.Rename", core.Concat("failed to remove source file: ", oldPath), err)
		}
		return nil
	}

	// Directory rename: move all files under oldPath to newPath
	oldPrefix := oldPath + "/"
	newPrefix := newPath + "/"

	entries, err := m.collectAllLocked()
	if err != nil {
		return core.E("datanode.Rename", core.Concat("failed to inspect tree: ", oldPath), err)
	}
	for _, name := range entries {
		if core.HasPrefix(name, oldPrefix) {
			newName := core.Concat(newPrefix, core.TrimPrefix(name, oldPrefix))
			data, err := m.readFileLocked(name)
			if err != nil {
				return core.E("datanode.Rename", core.Concat("failed to read source file: ", name), err)
			}
			m.dataNode.AddData(newName, data)
			if err := m.removeFileLocked(name); err != nil {
				return core.E("datanode.Rename", core.Concat("failed to remove source file: ", name), err)
			}
		}
	}

	// Move explicit directories
	dirsToMove := make(map[string]string)
	for d := range m.directories {
		if d == oldPath || core.HasPrefix(d, oldPrefix) {
			newD := core.Concat(newPath, core.TrimPrefix(d, oldPath))
			dirsToMove[d] = newD
		}
	}
	for old, nw := range dirsToMove {
		delete(m.directories, old)
		m.directories[nw] = true
	}

	return nil
}

// List documents the List operation.
//
//	result := m.List(...)
func (m *Medium) List(p string) ([]fs.DirEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p = cleanPath(p)

	entries, err := m.dataNode.ReadDir(p)
	if err != nil {
		// Check explicit directories
		if p == "" || m.directories[p] {
			return []fs.DirEntry{}, nil
		}
		return nil, core.E("datanode.List", core.Concat("not found: ", p), fs.ErrNotExist)
	}

	// Also include explicit subdirectories not discovered via files
	prefix := p
	if prefix != "" {
		prefix += "/"
	}
	seen := make(map[string]bool)
	for _, e := range entries {
		seen[e.Name()] = true
	}

	for d := range m.directories {
		if !core.HasPrefix(d, prefix) {
			continue
		}
		rest := core.TrimPrefix(d, prefix)
		if rest == "" {
			continue
		}
		first := core.SplitN(rest, "/", 2)[0]
		if !seen[first] {
			seen[first] = true
			entries = append(entries, &dirEntry{name: first})
		}
	}

	slices.SortFunc(entries, func(a, b fs.DirEntry) int {
		return cmp.Compare(a.Name(), b.Name())
	})

	return entries, nil
}

// Stat documents the Stat operation.
//
//	result := m.Stat(...)
func (m *Medium) Stat(p string) (fs.FileInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p = cleanPath(p)
	if p == "" {
		return &fileInfo{name: ".", isDir: true, mode: fs.ModeDir | 0755}, nil
	}

	info, err := m.dataNode.Stat(p)
	if err == nil {
		return info, nil
	}

	if m.directories[p] {
		return &fileInfo{name: path.Base(p), isDir: true, mode: fs.ModeDir | 0755}, nil
	}
	return nil, core.E("datanode.Stat", core.Concat("not found: ", p), fs.ErrNotExist)
}

// Open documents the Open operation.
//
//	result := m.Open(...)
func (m *Medium) Open(p string) (fs.File, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p = cleanPath(p)
	return m.dataNode.Open(p)
}

// Create documents the Create operation.
//
//	result := m.Create(...)
func (m *Medium) Create(p string) (goio.WriteCloser, error) {
	p = cleanPath(p)
	if p == "" {
		return nil, core.E("datanode.Create", "empty path", fs.ErrInvalid)
	}
	return &writeCloser{medium: m, path: p}, nil
}

// Append documents the Append operation.
//
//	result := m.Append(...)
func (m *Medium) Append(p string) (goio.WriteCloser, error) {
	p = cleanPath(p)
	if p == "" {
		return nil, core.E("datanode.Append", "empty path", fs.ErrInvalid)
	}

	// Read existing content
	var existing []byte
	m.mu.RLock()
	if m.IsFile(p) {
		data, err := m.readFileLocked(p)
		if err != nil {
			m.mu.RUnlock()
			return nil, core.E("datanode.Append", core.Concat("failed to read existing content: ", p), err)
		}
		existing = data
	}
	m.mu.RUnlock()

	return &writeCloser{medium: m, path: p, buf: existing}, nil
}

// ReadStream documents the ReadStream operation.
//
//	result := m.ReadStream(...)
func (m *Medium) ReadStream(p string) (goio.ReadCloser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p = cleanPath(p)
	f, err := m.dataNode.Open(p)
	if err != nil {
		return nil, core.E("datanode.ReadStream", core.Concat("not found: ", p), fs.ErrNotExist)
	}
	return f.(goio.ReadCloser), nil
}

// WriteStream documents the WriteStream operation.
//
//	result := m.WriteStream(...)
func (m *Medium) WriteStream(p string) (goio.WriteCloser, error) {
	return m.Create(p)
}

// Exists documents the Exists operation.
//
//	result := m.Exists(...)
func (m *Medium) Exists(p string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p = cleanPath(p)
	if p == "" {
		return true // root always exists
	}
	_, err := m.dataNode.Stat(p)
	if err == nil {
		return true
	}
	return m.directories[p]
}

// IsDir documents the IsDir operation.
//
//	result := m.IsDir(...)
func (m *Medium) IsDir(p string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p = cleanPath(p)
	if p == "" {
		return true
	}
	info, err := m.dataNode.Stat(p)
	if err == nil {
		return info.IsDir()
	}
	return m.directories[p]
}

// --- internal helpers ---

// hasPrefixLocked checks if any file path starts with prefix. Caller holds lock.
func (m *Medium) hasPrefixLocked(prefix string) (bool, error) {
	entries, err := m.collectAllLocked()
	if err != nil {
		return false, err
	}
	for _, name := range entries {
		if core.HasPrefix(name, prefix) {
			return true, nil
		}
	}
	for d := range m.directories {
		if core.HasPrefix(d, prefix) {
			return true, nil
		}
	}
	return false, nil
}

// collectAllLocked returns all file paths in the DataNode. Caller holds lock.
func (m *Medium) collectAllLocked() ([]string, error) {
	var names []string
	err := dataNodeWalkDir(m.dataNode, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			names = append(names, p)
		}
		return nil
	})
	return names, err
}

func (m *Medium) readFileLocked(name string) ([]byte, error) {
	f, err := dataNodeOpen(m.dataNode, name)
	if err != nil {
		return nil, err
	}
	data, readErr := dataNodeReadAll(f)
	closeErr := f.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return data, nil
}

// removeFileLocked removes a single file by rebuilding the DataNode.
// This is necessary because Borg's DataNode doesn't expose a Remove method.
// Caller must hold m.mu write lock.
func (m *Medium) removeFileLocked(target string) error {
	entries, err := m.collectAllLocked()
	if err != nil {
		return err
	}
	newDN := borgdatanode.New()
	for _, name := range entries {
		if name == target {
			continue
		}
		data, err := m.readFileLocked(name)
		if err != nil {
			return err
		}
		newDN.AddData(name, data)
	}
	m.dataNode = newDN
	return nil
}

// --- writeCloser buffers writes and flushes to DataNode on Close ---

type writeCloser struct {
	medium *Medium
	path   string
	buf    []byte
}

// Write documents the Write operation.
//
//	result := w.Write(...)
func (w *writeCloser) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	return len(p), nil
}

// Close documents the Close operation.
//
//	result := w.Close(...)
func (w *writeCloser) Close() error {
	w.medium.mu.Lock()
	defer w.medium.mu.Unlock()

	w.medium.dataNode.AddData(w.path, w.buf)
	w.medium.ensureDirsLocked(path.Dir(w.path))
	return nil
}

// --- fs types for explicit directories ---

type dirEntry struct {
	name string
}

// Name documents the Name operation.
//
//	result := d.Name(...)
func (d *dirEntry) Name() string { return d.name }

// IsDir documents the IsDir operation.
//
//	result := d.IsDir(...)
func (d *dirEntry) IsDir() bool { return true }

// Type documents the Type operation.
//
//	result := d.Type(...)
func (d *dirEntry) Type() fs.FileMode { return fs.ModeDir }

// Info documents the Info operation.
//
//	result := d.Info(...)
func (d *dirEntry) Info() (fs.FileInfo, error) {
	return &fileInfo{name: d.name, isDir: true, mode: fs.ModeDir | 0755}, nil
}

type fileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	isDir   bool
}

// Name documents the Name operation.
//
//	result := fi.Name(...)
func (fi *fileInfo) Name() string { return fi.name }

// Size documents the Size operation.
//
//	result := fi.Size(...)
func (fi *fileInfo) Size() int64 { return fi.size }

// Mode documents the Mode operation.
//
//	result := fi.Mode(...)
func (fi *fileInfo) Mode() fs.FileMode { return fi.mode }

// ModTime documents the ModTime operation.
//
//	result := fi.ModTime(...)
func (fi *fileInfo) ModTime() time.Time { return fi.modTime }

// IsDir documents the IsDir operation.
//
//	result := fi.IsDir(...)
func (fi *fileInfo) IsDir() bool { return fi.isDir }

// Sys documents the Sys operation.
//
//	result := fi.Sys(...)
func (fi *fileInfo) Sys() any { return nil }
