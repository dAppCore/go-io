package io

import (
	"bytes"
	goio "io"
	"io/fs"
	"time"

	core "dappco.re/go/core"
	"dappco.re/go/core/io/local"
)

// Medium defines the standard interface for a storage backend.
// This allows for different implementations (e.g., local disk, S3, SFTP)
// to be used interchangeably.
type Medium interface {
	Read(path string) (string, error)

	Write(path, content string) error

	// WriteMode saves content with explicit file permissions.
	// Use 0600 for sensitive files (keys, secrets, encrypted output).
	WriteMode(path, content string, mode fs.FileMode) error

	EnsureDir(path string) error

	IsFile(path string) bool

	FileGet(path string) (string, error)

	FileSet(path, content string) error

	Delete(path string) error

	DeleteAll(path string) error

	Rename(oldPath, newPath string) error

	List(path string) ([]fs.DirEntry, error)

	Stat(path string) (fs.FileInfo, error)

	Open(path string) (fs.File, error)

	Create(path string) (goio.WriteCloser, error)

	Append(path string) (goio.WriteCloser, error)

	// ReadStream returns a reader for the file content.
	// Use this for large files to avoid loading the entire content into memory.
	ReadStream(path string) (goio.ReadCloser, error)

	// WriteStream returns a writer for the file content.
	// Use this for large files to avoid loading the entire content into memory.
	WriteStream(path string) (goio.WriteCloser, error)

	// Exists checks if a path exists (file or directory).
	Exists(path string) bool

	// IsDir checks if a path exists and is a directory.
	IsDir(path string) bool
}

// FileInfo provides a simple implementation of fs.FileInfo for mock testing.
type FileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	isDir   bool
}

func (fi FileInfo) Name() string { return fi.name }

func (fi FileInfo) Size() int64 { return fi.size }

func (fi FileInfo) Mode() fs.FileMode { return fi.mode }

func (fi FileInfo) ModTime() time.Time { return fi.modTime }

func (fi FileInfo) IsDir() bool { return fi.isDir }

func (fi FileInfo) Sys() any { return nil }

// DirEntry provides a simple implementation of fs.DirEntry for mock testing.
type DirEntry struct {
	name  string
	isDir bool
	mode  fs.FileMode
	info  fs.FileInfo
}

func (de DirEntry) Name() string { return de.name }

func (de DirEntry) IsDir() bool { return de.isDir }

func (de DirEntry) Type() fs.FileMode { return de.mode.Type() }

func (de DirEntry) Info() (fs.FileInfo, error) { return de.info, nil }

// Local is a pre-initialised medium for the local filesystem.
// It uses "/" as root, providing unsandboxed access to the filesystem.
// For sandboxed access, use NewSandboxed with a specific root path.
var Local Medium

var _ Medium = (*local.Medium)(nil)

func init() {
	var err error
	Local, err = local.New("/")
	if err != nil {
		core.Warn("io: failed to initialise Local medium, io.Local will be nil", "error", err)
	}
}

// Use NewSandboxed to confine file operations to a root directory.
// All file operations are restricted to paths within the root, and the root
// directory will be created if it does not exist.
//
// Example usage:
//
//	medium, _ := io.NewSandboxed("/srv/app")
//	_ = medium.Write("config/app.yaml", "port: 8080")
func NewSandboxed(root string) (Medium, error) {
	return local.New(root)
}

// --- Helper Functions ---

// Example: content, _ := io.Read(medium, "config/app.yaml")
func Read(m Medium, path string) (string, error) {
	return m.Read(path)
}

// Example: _ = io.Write(medium, "config/app.yaml", "port: 8080")
func Write(m Medium, path, content string) error {
	return m.Write(path, content)
}

// Example: reader, _ := io.ReadStream(medium, "logs/app.log")
func ReadStream(m Medium, path string) (goio.ReadCloser, error) {
	return m.ReadStream(path)
}

// Example: writer, _ := io.WriteStream(medium, "logs/app.log")
func WriteStream(m Medium, path string) (goio.WriteCloser, error) {
	return m.WriteStream(path)
}

// Example: _ = io.EnsureDir(medium, "config")
func EnsureDir(m Medium, path string) error {
	return m.EnsureDir(path)
}

// Example: ok := io.IsFile(medium, "config/app.yaml")
func IsFile(m Medium, path string) bool {
	return m.IsFile(path)
}

// Example: _ = io.Copy(source, "input.txt", destination, "backup/input.txt")
func Copy(source Medium, sourcePath string, destination Medium, destinationPath string) error {
	content, err := source.Read(sourcePath)
	if err != nil {
		return core.E("io.Copy", core.Concat("read failed: ", sourcePath), err)
	}
	if err := destination.Write(destinationPath, content); err != nil {
		return core.E("io.Copy", core.Concat("write failed: ", destinationPath), err)
	}
	return nil
}

// --- MockMedium ---

// MockMedium is an in-memory implementation of Medium for testing.
type MockMedium struct {
	Files    map[string]string
	Dirs     map[string]bool
	ModTimes map[string]time.Time
}

var _ Medium = (*MockMedium)(nil)

// Use NewMockMedium when tests need an in-memory Medium.
//
//	medium := io.NewMockMedium()
//	_ = medium.Write("config/app.yaml", "port: 8080")
func NewMockMedium() *MockMedium {
	return &MockMedium{
		Files:    make(map[string]string),
		Dirs:     make(map[string]bool),
		ModTimes: make(map[string]time.Time),
	}
}

func (m *MockMedium) Read(path string) (string, error) {
	content, ok := m.Files[path]
	if !ok {
		return "", core.E("io.MockMedium.Read", core.Concat("file not found: ", path), fs.ErrNotExist)
	}
	return content, nil
}

func (m *MockMedium) Write(path, content string) error {
	m.Files[path] = content
	m.ModTimes[path] = time.Now()
	return nil
}

func (m *MockMedium) WriteMode(path, content string, mode fs.FileMode) error {
	return m.Write(path, content)
}

func (m *MockMedium) EnsureDir(path string) error {
	m.Dirs[path] = true
	return nil
}

func (m *MockMedium) IsFile(path string) bool {
	_, ok := m.Files[path]
	return ok
}

func (m *MockMedium) FileGet(path string) (string, error) {
	return m.Read(path)
}

func (m *MockMedium) FileSet(path, content string) error {
	return m.Write(path, content)
}

func (m *MockMedium) Delete(path string) error {
	if _, ok := m.Files[path]; ok {
		delete(m.Files, path)
		return nil
	}
	if _, ok := m.Dirs[path]; ok {
		// Check if directory is empty (no files or subdirs with this prefix)
		prefix := path
		if !core.HasSuffix(prefix, "/") {
			prefix += "/"
		}
		for f := range m.Files {
			if core.HasPrefix(f, prefix) {
				return core.E("io.MockMedium.Delete", core.Concat("directory not empty: ", path), fs.ErrExist)
			}
		}
		for d := range m.Dirs {
			if d != path && core.HasPrefix(d, prefix) {
				return core.E("io.MockMedium.Delete", core.Concat("directory not empty: ", path), fs.ErrExist)
			}
		}
		delete(m.Dirs, path)
		return nil
	}
	return core.E("io.MockMedium.Delete", core.Concat("path not found: ", path), fs.ErrNotExist)
}

func (m *MockMedium) DeleteAll(path string) error {
	found := false
	if _, ok := m.Files[path]; ok {
		delete(m.Files, path)
		found = true
	}
	if _, ok := m.Dirs[path]; ok {
		delete(m.Dirs, path)
		found = true
	}

	// Delete all entries under this path
	prefix := path
	if !core.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	for f := range m.Files {
		if core.HasPrefix(f, prefix) {
			delete(m.Files, f)
			found = true
		}
	}
	for d := range m.Dirs {
		if core.HasPrefix(d, prefix) {
			delete(m.Dirs, d)
			found = true
		}
	}

	if !found {
		return core.E("io.MockMedium.DeleteAll", core.Concat("path not found: ", path), fs.ErrNotExist)
	}
	return nil
}

func (m *MockMedium) Rename(oldPath, newPath string) error {
	if content, ok := m.Files[oldPath]; ok {
		m.Files[newPath] = content
		delete(m.Files, oldPath)
		if mt, ok := m.ModTimes[oldPath]; ok {
			m.ModTimes[newPath] = mt
			delete(m.ModTimes, oldPath)
		}
		return nil
	}
	if _, ok := m.Dirs[oldPath]; ok {
		// Move directory and all contents
		m.Dirs[newPath] = true
		delete(m.Dirs, oldPath)

		oldPrefix := oldPath
		if !core.HasSuffix(oldPrefix, "/") {
			oldPrefix += "/"
		}
		newPrefix := newPath
		if !core.HasSuffix(newPrefix, "/") {
			newPrefix += "/"
		}

		// Collect files to move first (don't mutate during iteration)
		filesToMove := make(map[string]string)
		for f := range m.Files {
			if core.HasPrefix(f, oldPrefix) {
				newF := core.Concat(newPrefix, core.TrimPrefix(f, oldPrefix))
				filesToMove[f] = newF
			}
		}
		for oldF, newF := range filesToMove {
			m.Files[newF] = m.Files[oldF]
			delete(m.Files, oldF)
			if mt, ok := m.ModTimes[oldF]; ok {
				m.ModTimes[newF] = mt
				delete(m.ModTimes, oldF)
			}
		}

		// Collect directories to move first
		dirsToMove := make(map[string]string)
		for d := range m.Dirs {
			if core.HasPrefix(d, oldPrefix) {
				newD := core.Concat(newPrefix, core.TrimPrefix(d, oldPrefix))
				dirsToMove[d] = newD
			}
		}
		for oldD, newD := range dirsToMove {
			m.Dirs[newD] = true
			delete(m.Dirs, oldD)
		}
		return nil
	}
	return core.E("io.MockMedium.Rename", core.Concat("path not found: ", oldPath), fs.ErrNotExist)
}

func (m *MockMedium) Open(path string) (fs.File, error) {
	content, ok := m.Files[path]
	if !ok {
		return nil, core.E("io.MockMedium.Open", core.Concat("file not found: ", path), fs.ErrNotExist)
	}
	return &MockFile{
		name:    core.PathBase(path),
		content: []byte(content),
	}, nil
}

func (m *MockMedium) Create(path string) (goio.WriteCloser, error) {
	return &MockWriteCloser{
		medium: m,
		path:   path,
	}, nil
}

func (m *MockMedium) Append(path string) (goio.WriteCloser, error) {
	content := m.Files[path]
	return &MockWriteCloser{
		medium: m,
		path:   path,
		data:   []byte(content),
	}, nil
}

func (m *MockMedium) ReadStream(path string) (goio.ReadCloser, error) {
	return m.Open(path)
}

func (m *MockMedium) WriteStream(path string) (goio.WriteCloser, error) {
	return m.Create(path)
}

// MockFile implements fs.File for MockMedium.
type MockFile struct {
	name    string
	content []byte
	offset  int64
}

func (f *MockFile) Stat() (fs.FileInfo, error) {
	return FileInfo{
		name: f.name,
		size: int64(len(f.content)),
	}, nil
}

func (f *MockFile) Read(b []byte) (int, error) {
	if f.offset >= int64(len(f.content)) {
		return 0, goio.EOF
	}
	n := copy(b, f.content[f.offset:])
	f.offset += int64(n)
	return n, nil
}

func (f *MockFile) Close() error {
	return nil
}

// MockWriteCloser implements WriteCloser for MockMedium.
type MockWriteCloser struct {
	medium *MockMedium
	path   string
	data   []byte
}

func (w *MockWriteCloser) Write(p []byte) (int, error) {
	w.data = append(w.data, p...)
	return len(p), nil
}

func (w *MockWriteCloser) Close() error {
	w.medium.Files[w.path] = string(w.data)
	w.medium.ModTimes[w.path] = time.Now()
	return nil
}

func (m *MockMedium) List(path string) ([]fs.DirEntry, error) {
	if _, ok := m.Dirs[path]; !ok {
		// Check if it's the root or has children
		hasChildren := false
		prefix := path
		if path != "" && !core.HasSuffix(prefix, "/") {
			prefix += "/"
		}
		for f := range m.Files {
			if core.HasPrefix(f, prefix) {
				hasChildren = true
				break
			}
		}
		if !hasChildren {
			for d := range m.Dirs {
				if core.HasPrefix(d, prefix) {
					hasChildren = true
					break
				}
			}
		}
		if !hasChildren && path != "" {
			return nil, core.E("io.MockMedium.List", core.Concat("directory not found: ", path), fs.ErrNotExist)
		}
	}

	prefix := path
	if path != "" && !core.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	seen := make(map[string]bool)
	var entries []fs.DirEntry

	// Find immediate children (files)
	for f, content := range m.Files {
		if !core.HasPrefix(f, prefix) {
			continue
		}
		rest := core.TrimPrefix(f, prefix)
		if rest == "" || core.Contains(rest, "/") {
			// Skip if it's not an immediate child
			if idx := bytes.IndexByte([]byte(rest), '/'); idx != -1 {
				// This is a subdirectory
				dirName := rest[:idx]
				if !seen[dirName] {
					seen[dirName] = true
					entries = append(entries, DirEntry{
						name:  dirName,
						isDir: true,
						mode:  fs.ModeDir | 0755,
						info: FileInfo{
							name:  dirName,
							isDir: true,
							mode:  fs.ModeDir | 0755,
						},
					})
				}
			}
			continue
		}
		if !seen[rest] {
			seen[rest] = true
			entries = append(entries, DirEntry{
				name:  rest,
				isDir: false,
				mode:  0644,
				info: FileInfo{
					name: rest,
					size: int64(len(content)),
					mode: 0644,
				},
			})
		}
	}

	// Find immediate subdirectories
	for d := range m.Dirs {
		if !core.HasPrefix(d, prefix) {
			continue
		}
		rest := core.TrimPrefix(d, prefix)
		if rest == "" {
			continue
		}
		// Get only immediate child
		if idx := bytes.IndexByte([]byte(rest), '/'); idx != -1 {
			rest = rest[:idx]
		}
		if !seen[rest] {
			seen[rest] = true
			entries = append(entries, DirEntry{
				name:  rest,
				isDir: true,
				mode:  fs.ModeDir | 0755,
				info: FileInfo{
					name:  rest,
					isDir: true,
					mode:  fs.ModeDir | 0755,
				},
			})
		}
	}

	return entries, nil
}

func (m *MockMedium) Stat(path string) (fs.FileInfo, error) {
	if content, ok := m.Files[path]; ok {
		modTime, ok := m.ModTimes[path]
		if !ok {
			modTime = time.Now()
		}
		return FileInfo{
			name:    core.PathBase(path),
			size:    int64(len(content)),
			mode:    0644,
			modTime: modTime,
		}, nil
	}
	if _, ok := m.Dirs[path]; ok {
		return FileInfo{
			name:  core.PathBase(path),
			isDir: true,
			mode:  fs.ModeDir | 0755,
		}, nil
	}
	return nil, core.E("io.MockMedium.Stat", core.Concat("path not found: ", path), fs.ErrNotExist)
}

func (m *MockMedium) Exists(path string) bool {
	if _, ok := m.Files[path]; ok {
		return true
	}
	if _, ok := m.Dirs[path]; ok {
		return true
	}
	return false
}

func (m *MockMedium) IsDir(path string) bool {
	_, ok := m.Dirs[path]
	return ok
}
