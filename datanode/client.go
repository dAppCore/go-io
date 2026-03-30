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

// Use New when you need an in-memory Medium that snapshots to tar.
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

// Use FromTar(snapshot) to restore a Medium from tar bytes.
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

// Example: snapshot, _ := medium.Snapshot()
// Use this for crash reports, workspace packaging, or TIM creation.
func (m *Medium) Snapshot() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data, err := m.dataNode.ToTar()
	if err != nil {
		return nil, core.E("datanode.Snapshot", "tar failed", err)
	}
	return data, nil
}

// Example: _ = medium.Restore(snapshot)
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

// Example: dataNode := medium.DataNode()
// Use this to wrap the filesystem in a TIM container.
func (m *Medium) DataNode() *borgdatanode.DataNode {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.dataNode
}

// cleanPath normalises a path: strips leading slash, cleans traversal.
func cleanPath(filePath string) string {
	filePath = core.TrimPrefix(filePath, "/")
	filePath = path.Clean(filePath)
	if filePath == "." {
		return ""
	}
	return filePath
}

// --- io.Medium interface ---

func (m *Medium) Read(filePath string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	filePath = cleanPath(filePath)
	f, err := m.dataNode.Open(filePath)
	if err != nil {
		return "", core.E("datanode.Read", core.Concat("not found: ", filePath), fs.ErrNotExist)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", core.E("datanode.Read", core.Concat("stat failed: ", filePath), err)
	}
	if info.IsDir() {
		return "", core.E("datanode.Read", core.Concat("is a directory: ", filePath), fs.ErrInvalid)
	}

	data, err := goio.ReadAll(f)
	if err != nil {
		return "", core.E("datanode.Read", core.Concat("read failed: ", filePath), err)
	}
	return string(data), nil
}

func (m *Medium) Write(filePath, content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	filePath = cleanPath(filePath)
	if filePath == "" {
		return core.E("datanode.Write", "empty path", fs.ErrInvalid)
	}
	m.dataNode.AddData(filePath, []byte(content))

	// ensure parent directories are tracked
	m.ensureDirsLocked(path.Dir(filePath))
	return nil
}

func (m *Medium) WriteMode(filePath, content string, mode fs.FileMode) error {
	return m.Write(filePath, content)
}

func (m *Medium) EnsureDir(filePath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	filePath = cleanPath(filePath)
	if filePath == "" {
		return nil
	}
	m.ensureDirsLocked(filePath)
	return nil
}

// ensureDirsLocked marks a directory and all ancestors as existing.
// Caller must hold m.mu.
func (m *Medium) ensureDirsLocked(directoryPath string) {
	for directoryPath != "" && directoryPath != "." {
		m.directories[directoryPath] = true
		directoryPath = path.Dir(directoryPath)
		if directoryPath == "." {
			break
		}
	}
}

func (m *Medium) IsFile(filePath string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	filePath = cleanPath(filePath)
	info, err := m.dataNode.Stat(filePath)
	return err == nil && !info.IsDir()
}

func (m *Medium) FileGet(filePath string) (string, error) {
	return m.Read(filePath)
}

func (m *Medium) FileSet(filePath, content string) error {
	return m.Write(filePath, content)
}

func (m *Medium) Delete(filePath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	filePath = cleanPath(filePath)
	if filePath == "" {
		return core.E("datanode.Delete", "cannot delete root", fs.ErrPermission)
	}

	// Check if it's a file in the DataNode
	info, err := m.dataNode.Stat(filePath)
	if err != nil {
		// Check explicit directories
		if m.directories[filePath] {
			// Check if dir is empty
			hasChildren, err := m.hasPrefixLocked(filePath + "/")
			if err != nil {
				return core.E("datanode.Delete", core.Concat("failed to inspect directory: ", filePath), err)
			}
			if hasChildren {
				return core.E("datanode.Delete", core.Concat("directory not empty: ", filePath), fs.ErrExist)
			}
			delete(m.directories, filePath)
			return nil
		}
		return core.E("datanode.Delete", core.Concat("not found: ", filePath), fs.ErrNotExist)
	}

	if info.IsDir() {
		hasChildren, err := m.hasPrefixLocked(filePath + "/")
		if err != nil {
			return core.E("datanode.Delete", core.Concat("failed to inspect directory: ", filePath), err)
		}
		if hasChildren {
			return core.E("datanode.Delete", core.Concat("directory not empty: ", filePath), fs.ErrExist)
		}
		delete(m.directories, filePath)
		return nil
	}

	// Remove the file by creating a new DataNode without it
	if err := m.removeFileLocked(filePath); err != nil {
		return core.E("datanode.Delete", core.Concat("failed to delete file: ", filePath), err)
	}
	return nil
}

func (m *Medium) DeleteAll(filePath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	filePath = cleanPath(filePath)
	if filePath == "" {
		return core.E("datanode.DeleteAll", "cannot delete root", fs.ErrPermission)
	}

	prefix := filePath + "/"
	found := false

	// Check if filePath itself is a file
	info, err := m.dataNode.Stat(filePath)
	if err == nil && !info.IsDir() {
		if err := m.removeFileLocked(filePath); err != nil {
			return core.E("datanode.DeleteAll", core.Concat("failed to delete file: ", filePath), err)
		}
		found = true
	}

	// Remove all files under prefix
	entries, err := m.collectAllLocked()
	if err != nil {
		return core.E("datanode.DeleteAll", core.Concat("failed to inspect tree: ", filePath), err)
	}
	for _, name := range entries {
		if name == filePath || core.HasPrefix(name, prefix) {
			if err := m.removeFileLocked(name); err != nil {
				return core.E("datanode.DeleteAll", core.Concat("failed to delete file: ", name), err)
			}
			found = true
		}
	}

	// Remove explicit directories under prefix
	for directoryPath := range m.directories {
		if directoryPath == filePath || core.HasPrefix(directoryPath, prefix) {
			delete(m.directories, directoryPath)
			found = true
		}
	}

	if !found {
		return core.E("datanode.DeleteAll", core.Concat("not found: ", filePath), fs.ErrNotExist)
	}
	return nil
}

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

func (m *Medium) List(filePath string) ([]fs.DirEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	filePath = cleanPath(filePath)

	entries, err := m.dataNode.ReadDir(filePath)
	if err != nil {
		// Check explicit directories
		if filePath == "" || m.directories[filePath] {
			return []fs.DirEntry{}, nil
		}
		return nil, core.E("datanode.List", core.Concat("not found: ", filePath), fs.ErrNotExist)
	}

	// Also include explicit subdirectories not discovered via files
	prefix := filePath
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

func (m *Medium) Stat(filePath string) (fs.FileInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	filePath = cleanPath(filePath)
	if filePath == "" {
		return &fileInfo{name: ".", isDir: true, mode: fs.ModeDir | 0755}, nil
	}

	info, err := m.dataNode.Stat(filePath)
	if err == nil {
		return info, nil
	}

	if m.directories[filePath] {
		return &fileInfo{name: path.Base(filePath), isDir: true, mode: fs.ModeDir | 0755}, nil
	}
	return nil, core.E("datanode.Stat", core.Concat("not found: ", filePath), fs.ErrNotExist)
}

func (m *Medium) Open(filePath string) (fs.File, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	filePath = cleanPath(filePath)
	return m.dataNode.Open(filePath)
}

func (m *Medium) Create(filePath string) (goio.WriteCloser, error) {
	filePath = cleanPath(filePath)
	if filePath == "" {
		return nil, core.E("datanode.Create", "empty path", fs.ErrInvalid)
	}
	return &writeCloser{medium: m, path: filePath}, nil
}

func (m *Medium) Append(filePath string) (goio.WriteCloser, error) {
	filePath = cleanPath(filePath)
	if filePath == "" {
		return nil, core.E("datanode.Append", "empty path", fs.ErrInvalid)
	}

	// Read existing content
	var existing []byte
	m.mu.RLock()
	if m.IsFile(filePath) {
		data, err := m.readFileLocked(filePath)
		if err != nil {
			m.mu.RUnlock()
			return nil, core.E("datanode.Append", core.Concat("failed to read existing content: ", filePath), err)
		}
		existing = data
	}
	m.mu.RUnlock()

	return &writeCloser{medium: m, path: filePath, buf: existing}, nil
}

func (m *Medium) ReadStream(filePath string) (goio.ReadCloser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	filePath = cleanPath(filePath)
	f, err := m.dataNode.Open(filePath)
	if err != nil {
		return nil, core.E("datanode.ReadStream", core.Concat("not found: ", filePath), fs.ErrNotExist)
	}
	return f.(goio.ReadCloser), nil
}

func (m *Medium) WriteStream(filePath string) (goio.WriteCloser, error) {
	return m.Create(filePath)
}

func (m *Medium) Exists(filePath string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	filePath = cleanPath(filePath)
	if filePath == "" {
		return true // root always exists
	}
	_, err := m.dataNode.Stat(filePath)
	if err == nil {
		return true
	}
	return m.directories[filePath]
}

func (m *Medium) IsDir(filePath string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	filePath = cleanPath(filePath)
	if filePath == "" {
		return true
	}
	info, err := m.dataNode.Stat(filePath)
	if err == nil {
		return info.IsDir()
	}
	return m.directories[filePath]
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
	err := dataNodeWalkDir(m.dataNode, ".", func(filePath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			names = append(names, filePath)
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

func (w *writeCloser) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	return len(p), nil
}

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

func (d *dirEntry) Name() string { return d.name }

func (d *dirEntry) IsDir() bool { return true }

func (d *dirEntry) Type() fs.FileMode { return fs.ModeDir }

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

func (fi *fileInfo) Name() string { return fi.name }

func (fi *fileInfo) Size() int64 { return fi.size }

func (fi *fileInfo) Mode() fs.FileMode { return fi.mode }

func (fi *fileInfo) ModTime() time.Time { return fi.modTime }

func (fi *fileInfo) IsDir() bool { return fi.isDir }

func (fi *fileInfo) Sys() any { return nil }
