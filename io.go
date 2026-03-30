package io

import (
	"bytes"
	goio "io"
	"io/fs"
	"time"

	core "dappco.re/go/core"
	"dappco.re/go/core/io/local"
)

// Example: medium, _ := io.NewSandboxed("/srv/app")
// _ = medium.Write("config/app.yaml", "port: 8080")
// backup, _ := io.NewSandboxed("/srv/backup")
// _ = io.Copy(medium, "data/report.json", backup, "daily/report.json")
type Medium interface {
	Read(path string) (string, error)

	Write(path, content string) error

	// Example: _ = medium.WriteMode("keys/private.key", key, 0600)
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

	// Example: reader, _ := medium.ReadStream("logs/app.log")
	ReadStream(path string) (goio.ReadCloser, error)

	// Example: writer, _ := medium.WriteStream("logs/app.log")
	WriteStream(path string) (goio.WriteCloser, error)

	// Example: ok := medium.Exists("config/app.yaml")
	Exists(path string) bool

	// Example: ok := medium.IsDir("config")
	IsDir(path string) bool
}

// Example: info := io.FileInfo{name: "app.yaml", size: 8, mode: 0644}
type FileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	isDir   bool
}

func (info FileInfo) Name() string { return info.name }

func (info FileInfo) Size() int64 { return info.size }

func (info FileInfo) Mode() fs.FileMode { return info.mode }

func (info FileInfo) ModTime() time.Time { return info.modTime }

func (info FileInfo) IsDir() bool { return info.isDir }

func (info FileInfo) Sys() any { return nil }

// Example: entry := io.DirEntry{name: "app.yaml", mode: 0644}
type DirEntry struct {
	name  string
	isDir bool
	mode  fs.FileMode
	info  fs.FileInfo
}

func (entry DirEntry) Name() string { return entry.name }

func (entry DirEntry) IsDir() bool { return entry.isDir }

func (entry DirEntry) Type() fs.FileMode { return entry.mode.Type() }

func (entry DirEntry) Info() (fs.FileInfo, error) { return entry.info, nil }

// Example: _ = io.Local.Read("/etc/hostname")
var Local Medium

var _ Medium = (*local.Medium)(nil)

func init() {
	var err error
	Local, err = local.New("/")
	if err != nil {
		core.Warn("io: failed to initialise Local medium, io.Local will be nil", "error", err)
	}
}

// Example: medium, _ := io.NewSandboxed("/srv/app")
// _ = medium.Write("config/app.yaml", "port: 8080")
func NewSandboxed(root string) (Medium, error) {
	return local.New(root)
}

// Example: content, _ := io.Read(medium, "config/app.yaml")
func Read(medium Medium, path string) (string, error) {
	return medium.Read(path)
}

// Example: _ = io.Write(medium, "config/app.yaml", "port: 8080")
func Write(medium Medium, path, content string) error {
	return medium.Write(path, content)
}

// Example: reader, _ := io.ReadStream(medium, "logs/app.log")
func ReadStream(medium Medium, path string) (goio.ReadCloser, error) {
	return medium.ReadStream(path)
}

// Example: writer, _ := io.WriteStream(medium, "logs/app.log")
func WriteStream(medium Medium, path string) (goio.WriteCloser, error) {
	return medium.WriteStream(path)
}

// Example: _ = io.EnsureDir(medium, "config")
func EnsureDir(medium Medium, path string) error {
	return medium.EnsureDir(path)
}

// Example: ok := io.IsFile(medium, "config/app.yaml")
func IsFile(medium Medium, path string) bool {
	return medium.IsFile(path)
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

// Example: medium := io.NewMockMedium()
// _ = medium.Write("config/app.yaml", "port: 8080")
type MockMedium struct {
	Files    map[string]string
	Dirs     map[string]bool
	ModTimes map[string]time.Time
}

var _ Medium = (*MockMedium)(nil)

// Example: medium := io.NewMockMedium()
// _ = medium.Write("config/app.yaml", "port: 8080")
func NewMockMedium() *MockMedium {
	return &MockMedium{
		Files:    make(map[string]string),
		Dirs:     make(map[string]bool),
		ModTimes: make(map[string]time.Time),
	}
}

func (medium *MockMedium) Read(path string) (string, error) {
	content, ok := medium.Files[path]
	if !ok {
		return "", core.E("io.MockMedium.Read", core.Concat("file not found: ", path), fs.ErrNotExist)
	}
	return content, nil
}

func (medium *MockMedium) Write(path, content string) error {
	medium.Files[path] = content
	medium.ModTimes[path] = time.Now()
	return nil
}

func (medium *MockMedium) WriteMode(path, content string, mode fs.FileMode) error {
	return medium.Write(path, content)
}

func (medium *MockMedium) EnsureDir(path string) error {
	medium.Dirs[path] = true
	return nil
}

func (medium *MockMedium) IsFile(path string) bool {
	_, ok := medium.Files[path]
	return ok
}

func (medium *MockMedium) FileGet(path string) (string, error) {
	return medium.Read(path)
}

func (medium *MockMedium) FileSet(path, content string) error {
	return medium.Write(path, content)
}

func (medium *MockMedium) Delete(path string) error {
	if _, ok := medium.Files[path]; ok {
		delete(medium.Files, path)
		return nil
	}
	if _, ok := medium.Dirs[path]; ok {
		prefix := path
		if !core.HasSuffix(prefix, "/") {
			prefix += "/"
		}
		for filePath := range medium.Files {
			if core.HasPrefix(filePath, prefix) {
				return core.E("io.MockMedium.Delete", core.Concat("directory not empty: ", path), fs.ErrExist)
			}
		}
		for directoryPath := range medium.Dirs {
			if directoryPath != path && core.HasPrefix(directoryPath, prefix) {
				return core.E("io.MockMedium.Delete", core.Concat("directory not empty: ", path), fs.ErrExist)
			}
		}
		delete(medium.Dirs, path)
		return nil
	}
	return core.E("io.MockMedium.Delete", core.Concat("path not found: ", path), fs.ErrNotExist)
}

func (medium *MockMedium) DeleteAll(path string) error {
	found := false
	if _, ok := medium.Files[path]; ok {
		delete(medium.Files, path)
		found = true
	}
	if _, ok := medium.Dirs[path]; ok {
		delete(medium.Dirs, path)
		found = true
	}
	prefix := path
	if !core.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	for filePath := range medium.Files {
		if core.HasPrefix(filePath, prefix) {
			delete(medium.Files, filePath)
			found = true
		}
	}
	for directoryPath := range medium.Dirs {
		if core.HasPrefix(directoryPath, prefix) {
			delete(medium.Dirs, directoryPath)
			found = true
		}
	}

	if !found {
		return core.E("io.MockMedium.DeleteAll", core.Concat("path not found: ", path), fs.ErrNotExist)
	}
	return nil
}

func (medium *MockMedium) Rename(oldPath, newPath string) error {
	if content, ok := medium.Files[oldPath]; ok {
		medium.Files[newPath] = content
		delete(medium.Files, oldPath)
		if modTime, ok := medium.ModTimes[oldPath]; ok {
			medium.ModTimes[newPath] = modTime
			delete(medium.ModTimes, oldPath)
		}
		return nil
	}
	if _, ok := medium.Dirs[oldPath]; ok {
		medium.Dirs[newPath] = true
		delete(medium.Dirs, oldPath)

		oldPrefix := oldPath
		if !core.HasSuffix(oldPrefix, "/") {
			oldPrefix += "/"
		}
		newPrefix := newPath
		if !core.HasSuffix(newPrefix, "/") {
			newPrefix += "/"
		}

		filesToMove := make(map[string]string)
		for filePath := range medium.Files {
			if core.HasPrefix(filePath, oldPrefix) {
				newFilePath := core.Concat(newPrefix, core.TrimPrefix(filePath, oldPrefix))
				filesToMove[filePath] = newFilePath
			}
		}
		for oldFilePath, newFilePath := range filesToMove {
			medium.Files[newFilePath] = medium.Files[oldFilePath]
			delete(medium.Files, oldFilePath)
			if modTime, ok := medium.ModTimes[oldFilePath]; ok {
				medium.ModTimes[newFilePath] = modTime
				delete(medium.ModTimes, oldFilePath)
			}
		}

		dirsToMove := make(map[string]string)
		for directoryPath := range medium.Dirs {
			if core.HasPrefix(directoryPath, oldPrefix) {
				newDirectoryPath := core.Concat(newPrefix, core.TrimPrefix(directoryPath, oldPrefix))
				dirsToMove[directoryPath] = newDirectoryPath
			}
		}
		for oldDirectoryPath, newDirectoryPath := range dirsToMove {
			medium.Dirs[newDirectoryPath] = true
			delete(medium.Dirs, oldDirectoryPath)
		}
		return nil
	}
	return core.E("io.MockMedium.Rename", core.Concat("path not found: ", oldPath), fs.ErrNotExist)
}

func (medium *MockMedium) Open(path string) (fs.File, error) {
	content, ok := medium.Files[path]
	if !ok {
		return nil, core.E("io.MockMedium.Open", core.Concat("file not found: ", path), fs.ErrNotExist)
	}
	return &MockFile{
		name:    core.PathBase(path),
		content: []byte(content),
	}, nil
}

func (medium *MockMedium) Create(path string) (goio.WriteCloser, error) {
	return &MockWriteCloser{
		medium: medium,
		path:   path,
	}, nil
}

func (medium *MockMedium) Append(path string) (goio.WriteCloser, error) {
	content := medium.Files[path]
	return &MockWriteCloser{
		medium: medium,
		path:   path,
		data:   []byte(content),
	}, nil
}

func (medium *MockMedium) ReadStream(path string) (goio.ReadCloser, error) {
	return medium.Open(path)
}

func (medium *MockMedium) WriteStream(path string) (goio.WriteCloser, error) {
	return medium.Create(path)
}

// MockFile implements fs.File for MockMedium.
type MockFile struct {
	name    string
	content []byte
	offset  int64
}

func (file *MockFile) Stat() (fs.FileInfo, error) {
	return FileInfo{
		name: file.name,
		size: int64(len(file.content)),
	}, nil
}

func (file *MockFile) Read(buffer []byte) (int, error) {
	if file.offset >= int64(len(file.content)) {
		return 0, goio.EOF
	}
	readCount := copy(buffer, file.content[file.offset:])
	file.offset += int64(readCount)
	return readCount, nil
}

func (file *MockFile) Close() error {
	return nil
}

// MockWriteCloser implements WriteCloser for MockMedium.
type MockWriteCloser struct {
	medium *MockMedium
	path   string
	data   []byte
}

func (writeCloser *MockWriteCloser) Write(data []byte) (int, error) {
	writeCloser.data = append(writeCloser.data, data...)
	return len(data), nil
}

func (writeCloser *MockWriteCloser) Close() error {
	writeCloser.medium.Files[writeCloser.path] = string(writeCloser.data)
	writeCloser.medium.ModTimes[writeCloser.path] = time.Now()
	return nil
}

func (medium *MockMedium) List(path string) ([]fs.DirEntry, error) {
	if _, ok := medium.Dirs[path]; !ok {
		hasChildren := false
		prefix := path
		if path != "" && !core.HasSuffix(prefix, "/") {
			prefix += "/"
		}
		for filePath := range medium.Files {
			if core.HasPrefix(filePath, prefix) {
				hasChildren = true
				break
			}
		}
		if !hasChildren {
			for directoryPath := range medium.Dirs {
				if core.HasPrefix(directoryPath, prefix) {
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

	for filePath, content := range medium.Files {
		if !core.HasPrefix(filePath, prefix) {
			continue
		}
		rest := core.TrimPrefix(filePath, prefix)
		if rest == "" || core.Contains(rest, "/") {
			if idx := bytes.IndexByte([]byte(rest), '/'); idx != -1 {
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

	for directoryPath := range medium.Dirs {
		if !core.HasPrefix(directoryPath, prefix) {
			continue
		}
		rest := core.TrimPrefix(directoryPath, prefix)
		if rest == "" {
			continue
		}
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

func (medium *MockMedium) Stat(path string) (fs.FileInfo, error) {
	if content, ok := medium.Files[path]; ok {
		modTime, ok := medium.ModTimes[path]
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
	if _, ok := medium.Dirs[path]; ok {
		return FileInfo{
			name:  core.PathBase(path),
			isDir: true,
			mode:  fs.ModeDir | 0755,
		}, nil
	}
	return nil, core.E("io.MockMedium.Stat", core.Concat("path not found: ", path), fs.ErrNotExist)
}

func (medium *MockMedium) Exists(path string) bool {
	if _, ok := medium.Files[path]; ok {
		return true
	}
	if _, ok := medium.Dirs[path]; ok {
		return true
	}
	return false
}

func (medium *MockMedium) IsDir(path string) bool {
	_, ok := medium.Dirs[path]
	return ok
}
