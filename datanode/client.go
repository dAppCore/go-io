// Package datanode keeps io.Medium data in Borg's DataNode.
//
//	medium := datanode.New()
//	_ = medium.Write("jobs/run.log", "started")
//	snapshot, _ := medium.Snapshot()
//	restored, _ := datanode.FromTar(snapshot)
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
	dataNodeWalkDir = func(fileSystem fs.FS, root string, callback fs.WalkDirFunc) error {
		return fs.WalkDir(fileSystem, root, callback)
	}
	dataNodeOpen = func(dataNode *borgdatanode.DataNode, filePath string) (fs.File, error) {
		return dataNode.Open(filePath)
	}
	dataNodeReadAll = func(reader goio.Reader) ([]byte, error) {
		return goio.ReadAll(reader)
	}
)

// Example: medium := datanode.New()
// _ = medium.Write("jobs/run.log", "started")
// snapshot, _ := medium.Snapshot()
type Medium struct {
	dataNode     *borgdatanode.DataNode
	directorySet map[string]bool // explicit directories that exist without file contents
	mu           sync.RWMutex
}

func New() *Medium {
	return &Medium{
		dataNode:     borgdatanode.New(),
		directorySet: make(map[string]bool),
	}
}

// Example: sourceMedium := datanode.New()
// snapshot, _ := sourceMedium.Snapshot()
// restored, _ := datanode.FromTar(snapshot)
func FromTar(data []byte) (*Medium, error) {
	dataNode, err := borgdatanode.FromTar(data)
	if err != nil {
		return nil, core.E("datanode.FromTar", "failed to restore", err)
	}
	return &Medium{
		dataNode:     dataNode,
		directorySet: make(map[string]bool),
	}, nil
}

// Example: snapshot, _ := medium.Snapshot()
func (medium *Medium) Snapshot() ([]byte, error) {
	medium.mu.RLock()
	defer medium.mu.RUnlock()
	data, err := medium.dataNode.ToTar()
	if err != nil {
		return nil, core.E("datanode.Snapshot", "tar failed", err)
	}
	return data, nil
}

// Example: _ = medium.Restore(snapshot)
func (medium *Medium) Restore(data []byte) error {
	dataNode, err := borgdatanode.FromTar(data)
	if err != nil {
		return core.E("datanode.Restore", "tar failed", err)
	}
	medium.mu.Lock()
	defer medium.mu.Unlock()
	medium.dataNode = dataNode
	medium.directorySet = make(map[string]bool)
	return nil
}

// Example: dataNode := medium.DataNode()
func (medium *Medium) DataNode() *borgdatanode.DataNode {
	medium.mu.RLock()
	defer medium.mu.RUnlock()
	return medium.dataNode
}

// normaliseEntryPath normalises a path: strips the leading slash and cleans traversal.
func normaliseEntryPath(filePath string) string {
	filePath = core.TrimPrefix(filePath, "/")
	filePath = path.Clean(filePath)
	if filePath == "." {
		return ""
	}
	return filePath
}

// --- io.Medium interface ---

func (medium *Medium) Read(filePath string) (string, error) {
	medium.mu.RLock()
	defer medium.mu.RUnlock()

	filePath = normaliseEntryPath(filePath)
	file, err := medium.dataNode.Open(filePath)
	if err != nil {
		return "", core.E("datanode.Read", core.Concat("not found: ", filePath), fs.ErrNotExist)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return "", core.E("datanode.Read", core.Concat("stat failed: ", filePath), err)
	}
	if info.IsDir() {
		return "", core.E("datanode.Read", core.Concat("is a directory: ", filePath), fs.ErrInvalid)
	}

	data, err := goio.ReadAll(file)
	if err != nil {
		return "", core.E("datanode.Read", core.Concat("read failed: ", filePath), err)
	}
	return string(data), nil
}

func (medium *Medium) Write(filePath, content string) error {
	medium.mu.Lock()
	defer medium.mu.Unlock()

	filePath = normaliseEntryPath(filePath)
	if filePath == "" {
		return core.E("datanode.Write", "empty path", fs.ErrInvalid)
	}
	medium.dataNode.AddData(filePath, []byte(content))

	// ensure parent directories are tracked
	medium.ensureDirsLocked(path.Dir(filePath))
	return nil
}

func (medium *Medium) WriteMode(filePath, content string, mode fs.FileMode) error {
	return medium.Write(filePath, content)
}

func (medium *Medium) EnsureDir(filePath string) error {
	medium.mu.Lock()
	defer medium.mu.Unlock()

	filePath = normaliseEntryPath(filePath)
	if filePath == "" {
		return nil
	}
	medium.ensureDirsLocked(filePath)
	return nil
}

// ensureDirsLocked marks a directory and all ancestors as existing.
// Caller must hold medium.mu.
func (medium *Medium) ensureDirsLocked(directoryPath string) {
	for directoryPath != "" && directoryPath != "." {
		medium.directorySet[directoryPath] = true
		directoryPath = path.Dir(directoryPath)
		if directoryPath == "." {
			break
		}
	}
}

func (medium *Medium) IsFile(filePath string) bool {
	medium.mu.RLock()
	defer medium.mu.RUnlock()

	filePath = normaliseEntryPath(filePath)
	info, err := medium.dataNode.Stat(filePath)
	return err == nil && !info.IsDir()
}

func (medium *Medium) FileGet(filePath string) (string, error) {
	return medium.Read(filePath)
}

func (medium *Medium) FileSet(filePath, content string) error {
	return medium.Write(filePath, content)
}

func (medium *Medium) Delete(filePath string) error {
	medium.mu.Lock()
	defer medium.mu.Unlock()

	filePath = normaliseEntryPath(filePath)
	if filePath == "" {
		return core.E("datanode.Delete", "cannot delete root", fs.ErrPermission)
	}

	// Check if it's a file in the DataNode
	info, err := medium.dataNode.Stat(filePath)
	if err != nil {
		// Check explicit directories
		if medium.directorySet[filePath] {
			// Check if dir is empty
			hasChildren, err := medium.hasPrefixLocked(filePath + "/")
			if err != nil {
				return core.E("datanode.Delete", core.Concat("failed to inspect directory: ", filePath), err)
			}
			if hasChildren {
				return core.E("datanode.Delete", core.Concat("directory not empty: ", filePath), fs.ErrExist)
			}
			delete(medium.directorySet, filePath)
			return nil
		}
		return core.E("datanode.Delete", core.Concat("not found: ", filePath), fs.ErrNotExist)
	}

	if info.IsDir() {
		hasChildren, err := medium.hasPrefixLocked(filePath + "/")
		if err != nil {
			return core.E("datanode.Delete", core.Concat("failed to inspect directory: ", filePath), err)
		}
		if hasChildren {
			return core.E("datanode.Delete", core.Concat("directory not empty: ", filePath), fs.ErrExist)
		}
		delete(medium.directorySet, filePath)
		return nil
	}

	// Remove the file by creating a new DataNode without it
	if err := medium.removeFileLocked(filePath); err != nil {
		return core.E("datanode.Delete", core.Concat("failed to delete file: ", filePath), err)
	}
	return nil
}

func (medium *Medium) DeleteAll(filePath string) error {
	medium.mu.Lock()
	defer medium.mu.Unlock()

	filePath = normaliseEntryPath(filePath)
	if filePath == "" {
		return core.E("datanode.DeleteAll", "cannot delete root", fs.ErrPermission)
	}

	prefix := filePath + "/"
	found := false

	// Check if filePath itself is a file
	info, err := medium.dataNode.Stat(filePath)
	if err == nil && !info.IsDir() {
		if err := medium.removeFileLocked(filePath); err != nil {
			return core.E("datanode.DeleteAll", core.Concat("failed to delete file: ", filePath), err)
		}
		found = true
	}

	// Remove all files under prefix
	entries, err := medium.collectAllLocked()
	if err != nil {
		return core.E("datanode.DeleteAll", core.Concat("failed to inspect tree: ", filePath), err)
	}
	for _, name := range entries {
		if name == filePath || core.HasPrefix(name, prefix) {
			if err := medium.removeFileLocked(name); err != nil {
				return core.E("datanode.DeleteAll", core.Concat("failed to delete file: ", name), err)
			}
			found = true
		}
	}

	// Remove explicit directories under prefix
	for directoryPath := range medium.directorySet {
		if directoryPath == filePath || core.HasPrefix(directoryPath, prefix) {
			delete(medium.directorySet, directoryPath)
			found = true
		}
	}

	if !found {
		return core.E("datanode.DeleteAll", core.Concat("not found: ", filePath), fs.ErrNotExist)
	}
	return nil
}

func (medium *Medium) Rename(oldPath, newPath string) error {
	medium.mu.Lock()
	defer medium.mu.Unlock()

	oldPath = normaliseEntryPath(oldPath)
	newPath = normaliseEntryPath(newPath)

	// Check if source is a file
	info, err := medium.dataNode.Stat(oldPath)
	if err != nil {
		return core.E("datanode.Rename", core.Concat("not found: ", oldPath), fs.ErrNotExist)
	}

	if !info.IsDir() {
		// Read old, write new, delete old
		data, err := medium.readFileLocked(oldPath)
		if err != nil {
			return core.E("datanode.Rename", core.Concat("failed to read source file: ", oldPath), err)
		}
		medium.dataNode.AddData(newPath, data)
		medium.ensureDirsLocked(path.Dir(newPath))
		if err := medium.removeFileLocked(oldPath); err != nil {
			return core.E("datanode.Rename", core.Concat("failed to remove source file: ", oldPath), err)
		}
		return nil
	}

	// Directory rename: move all files under oldPath to newPath
	oldPrefix := oldPath + "/"
	newPrefix := newPath + "/"

	entries, err := medium.collectAllLocked()
	if err != nil {
		return core.E("datanode.Rename", core.Concat("failed to inspect tree: ", oldPath), err)
	}
	for _, name := range entries {
		if core.HasPrefix(name, oldPrefix) {
			newName := core.Concat(newPrefix, core.TrimPrefix(name, oldPrefix))
			data, err := medium.readFileLocked(name)
			if err != nil {
				return core.E("datanode.Rename", core.Concat("failed to read source file: ", name), err)
			}
			medium.dataNode.AddData(newName, data)
			if err := medium.removeFileLocked(name); err != nil {
				return core.E("datanode.Rename", core.Concat("failed to remove source file: ", name), err)
			}
		}
	}

	// Move explicit directories
	dirsToMove := make(map[string]string)
	for d := range medium.directorySet {
		if d == oldPath || core.HasPrefix(d, oldPrefix) {
			newD := core.Concat(newPath, core.TrimPrefix(d, oldPath))
			dirsToMove[d] = newD
		}
	}
	for old, nw := range dirsToMove {
		delete(medium.directorySet, old)
		medium.directorySet[nw] = true
	}

	return nil
}

func (medium *Medium) List(filePath string) ([]fs.DirEntry, error) {
	medium.mu.RLock()
	defer medium.mu.RUnlock()

	filePath = normaliseEntryPath(filePath)

	entries, err := medium.dataNode.ReadDir(filePath)
	if err != nil {
		// Check explicit directories
		if filePath == "" || medium.directorySet[filePath] {
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

	for d := range medium.directorySet {
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

func (medium *Medium) Stat(filePath string) (fs.FileInfo, error) {
	medium.mu.RLock()
	defer medium.mu.RUnlock()

	filePath = normaliseEntryPath(filePath)
	if filePath == "" {
		return &fileInfo{name: ".", isDir: true, mode: fs.ModeDir | 0755}, nil
	}

	info, err := medium.dataNode.Stat(filePath)
	if err == nil {
		return info, nil
	}

	if medium.directorySet[filePath] {
		return &fileInfo{name: path.Base(filePath), isDir: true, mode: fs.ModeDir | 0755}, nil
	}
	return nil, core.E("datanode.Stat", core.Concat("not found: ", filePath), fs.ErrNotExist)
}

func (medium *Medium) Open(filePath string) (fs.File, error) {
	medium.mu.RLock()
	defer medium.mu.RUnlock()

	filePath = normaliseEntryPath(filePath)
	return medium.dataNode.Open(filePath)
}

func (medium *Medium) Create(filePath string) (goio.WriteCloser, error) {
	filePath = normaliseEntryPath(filePath)
	if filePath == "" {
		return nil, core.E("datanode.Create", "empty path", fs.ErrInvalid)
	}
	return &writeCloser{medium: medium, path: filePath}, nil
}

func (medium *Medium) Append(filePath string) (goio.WriteCloser, error) {
	filePath = normaliseEntryPath(filePath)
	if filePath == "" {
		return nil, core.E("datanode.Append", "empty path", fs.ErrInvalid)
	}

	// Read existing content
	var existing []byte
	medium.mu.RLock()
	if medium.IsFile(filePath) {
		data, err := medium.readFileLocked(filePath)
		if err != nil {
			medium.mu.RUnlock()
			return nil, core.E("datanode.Append", core.Concat("failed to read existing content: ", filePath), err)
		}
		existing = data
	}
	medium.mu.RUnlock()

	return &writeCloser{medium: medium, path: filePath, buf: existing}, nil
}

func (medium *Medium) ReadStream(filePath string) (goio.ReadCloser, error) {
	medium.mu.RLock()
	defer medium.mu.RUnlock()

	filePath = normaliseEntryPath(filePath)
	file, err := medium.dataNode.Open(filePath)
	if err != nil {
		return nil, core.E("datanode.ReadStream", core.Concat("not found: ", filePath), fs.ErrNotExist)
	}
	return file.(goio.ReadCloser), nil
}

func (medium *Medium) WriteStream(filePath string) (goio.WriteCloser, error) {
	return medium.Create(filePath)
}

func (medium *Medium) Exists(filePath string) bool {
	medium.mu.RLock()
	defer medium.mu.RUnlock()

	filePath = normaliseEntryPath(filePath)
	if filePath == "" {
		return true // root always exists
	}
	_, err := medium.dataNode.Stat(filePath)
	if err == nil {
		return true
	}
	return medium.directorySet[filePath]
}

func (medium *Medium) IsDir(filePath string) bool {
	medium.mu.RLock()
	defer medium.mu.RUnlock()

	filePath = normaliseEntryPath(filePath)
	if filePath == "" {
		return true
	}
	info, err := medium.dataNode.Stat(filePath)
	if err == nil {
		return info.IsDir()
	}
	return medium.directorySet[filePath]
}

// --- internal helpers ---

// hasPrefixLocked checks if any file path starts with prefix. Caller holds lock.
func (medium *Medium) hasPrefixLocked(prefix string) (bool, error) {
	entries, err := medium.collectAllLocked()
	if err != nil {
		return false, err
	}
	for _, name := range entries {
		if core.HasPrefix(name, prefix) {
			return true, nil
		}
	}
	for d := range medium.directorySet {
		if core.HasPrefix(d, prefix) {
			return true, nil
		}
	}
	return false, nil
}

// collectAllLocked returns all file paths in the DataNode. Caller holds lock.
func (medium *Medium) collectAllLocked() ([]string, error) {
	var names []string
	err := dataNodeWalkDir(medium.dataNode, ".", func(filePath string, entry fs.DirEntry, err error) error {
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

func (medium *Medium) readFileLocked(filePath string) ([]byte, error) {
	file, err := dataNodeOpen(medium.dataNode, filePath)
	if err != nil {
		return nil, err
	}
	data, readErr := dataNodeReadAll(file)
	closeErr := file.Close()
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
// Caller must hold medium.mu write lock.
func (medium *Medium) removeFileLocked(target string) error {
	entries, err := medium.collectAllLocked()
	if err != nil {
		return err
	}
	newDataNode := borgdatanode.New()
	for _, name := range entries {
		if name == target {
			continue
		}
		data, err := medium.readFileLocked(name)
		if err != nil {
			return err
		}
		newDataNode.AddData(name, data)
	}
	medium.dataNode = newDataNode
	return nil
}

// --- writeCloser buffers writes and flushes to DataNode on Close ---

type writeCloser struct {
	medium *Medium
	path   string
	buf    []byte
}

func (writer *writeCloser) Write(data []byte) (int, error) {
	writer.buf = append(writer.buf, data...)
	return len(data), nil
}

func (writer *writeCloser) Close() error {
	writer.medium.mu.Lock()
	defer writer.medium.mu.Unlock()

	writer.medium.dataNode.AddData(writer.path, writer.buf)
	writer.medium.ensureDirsLocked(path.Dir(writer.path))
	return nil
}

// --- fs types for explicit directories ---

type dirEntry struct {
	name string
}

func (entry *dirEntry) Name() string { return entry.name }

func (entry *dirEntry) IsDir() bool { return true }

func (entry *dirEntry) Type() fs.FileMode { return fs.ModeDir }

func (entry *dirEntry) Info() (fs.FileInfo, error) {
	return &fileInfo{name: entry.name, isDir: true, mode: fs.ModeDir | 0755}, nil
}

type fileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	isDir   bool
}

func (info *fileInfo) Name() string { return info.name }

func (info *fileInfo) Size() int64 { return info.size }

func (info *fileInfo) Mode() fs.FileMode { return info.mode }

func (info *fileInfo) ModTime() time.Time { return info.modTime }

func (info *fileInfo) IsDir() bool { return info.isDir }

func (info *fileInfo) Sys() any { return nil }
