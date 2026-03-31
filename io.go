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
// Example: _ = medium.Write("config/app.yaml", "port: 8080")
// Example: backup, _ := io.NewSandboxed("/srv/backup")
// Example: _ = io.Copy(medium, "data/report.json", backup, "daily/report.json")
type Medium interface {
	// Example: content, _ := medium.Read("config/app.yaml")
	Read(path string) (string, error)

	// Example: _ = medium.Write("config/app.yaml", "port: 8080")
	Write(path, content string) error

	// Example: _ = medium.WriteMode("keys/private.key", key, 0600)
	WriteMode(path, content string, mode fs.FileMode) error

	// Example: _ = medium.EnsureDir("config/app")
	EnsureDir(path string) error

	// Example: isFile := medium.IsFile("config/app.yaml")
	IsFile(path string) bool

	// Example: _ = medium.Delete("config/app.yaml")
	Delete(path string) error

	// Example: _ = medium.DeleteAll("logs/archive")
	DeleteAll(path string) error

	// Example: _ = medium.Rename("drafts/todo.txt", "archive/todo.txt")
	Rename(oldPath, newPath string) error

	// Example: entries, _ := medium.List("config")
	List(path string) ([]fs.DirEntry, error)

	// Example: info, _ := medium.Stat("config/app.yaml")
	Stat(path string) (fs.FileInfo, error)

	// Example: file, _ := medium.Open("config/app.yaml")
	Open(path string) (fs.File, error)

	// Example: writer, _ := medium.Create("logs/app.log")
	Create(path string) (goio.WriteCloser, error)

	// Example: writer, _ := medium.Append("logs/app.log")
	Append(path string) (goio.WriteCloser, error)

	// Example: reader, _ := medium.ReadStream("logs/app.log")
	ReadStream(path string) (goio.ReadCloser, error)

	// Example: writer, _ := medium.WriteStream("logs/app.log")
	WriteStream(path string) (goio.WriteCloser, error)

	// Example: exists := medium.Exists("config/app.yaml")
	Exists(path string) bool

	// Example: isDirectory := medium.IsDir("config")
	IsDir(path string) bool
}

// Example: info := io.NewFileInfo("app.yaml", 8, 0644, time.Unix(0, 0), false)
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

// Example: info := io.NewFileInfo("app.yaml", 8, 0644, time.Unix(0, 0), false)
// Example: entry := io.NewDirEntry("app.yaml", false, 0644, info)
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

// Example: info := io.NewFileInfo("app.yaml", 8, 0644, time.Unix(0, 0), false)
func NewFileInfo(name string, size int64, mode fs.FileMode, modTime time.Time, isDir bool) FileInfo {
	return FileInfo{
		name:    name,
		size:    size,
		mode:    mode,
		modTime: modTime,
		isDir:   isDir,
	}
}

// Example: info := io.NewFileInfo("app.yaml", 8, 0644, time.Unix(0, 0), false)
// Example: entry := io.NewDirEntry("app.yaml", false, 0644, info)
func NewDirEntry(name string, isDir bool, mode fs.FileMode, info fs.FileInfo) DirEntry {
	return DirEntry{
		name:  name,
		isDir: isDir,
		mode:  mode,
		info:  info,
	}
}

// Example: _ = io.Local.Read("/etc/hostname")
var Local Medium

var _ Medium = (*local.Medium)(nil)

func init() {
	var err error
	Local, err = local.New("/")
	if err != nil {
		core.Warn("io.Local init failed", "error", err)
	}
}

// Example: medium, _ := io.NewSandboxed("/srv/app")
// Example: _ = medium.Write("config/app.yaml", "port: 8080")
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

// Example: isFile := io.IsFile(medium, "config/app.yaml")
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

// Example: medium := io.NewMemoryMedium()
// Example: _ = medium.Write("config/app.yaml", "port: 8080")
type MemoryMedium struct {
	files    map[string]string
	dirs     map[string]bool
	modTimes map[string]time.Time
}

var _ Medium = (*MemoryMedium)(nil)

// Example: medium := io.NewMemoryMedium()
// Example: _ = medium.Write("config/app.yaml", "port: 8080")
func NewMemoryMedium() *MemoryMedium {
	return &MemoryMedium{
		files:    make(map[string]string),
		dirs:     make(map[string]bool),
		modTimes: make(map[string]time.Time),
	}
}

func (medium *MemoryMedium) Read(path string) (string, error) {
	content, ok := medium.files[path]
	if !ok {
		return "", core.E("io.MemoryMedium.Read", core.Concat("file not found: ", path), fs.ErrNotExist)
	}
	return content, nil
}

func (medium *MemoryMedium) Write(path, content string) error {
	medium.files[path] = content
	medium.modTimes[path] = time.Now()
	return nil
}

func (medium *MemoryMedium) WriteMode(path, content string, mode fs.FileMode) error {
	return medium.Write(path, content)
}

func (medium *MemoryMedium) EnsureDir(path string) error {
	medium.dirs[path] = true
	return nil
}

func (medium *MemoryMedium) IsFile(path string) bool {
	_, ok := medium.files[path]
	return ok
}

func (medium *MemoryMedium) Delete(path string) error {
	if _, ok := medium.files[path]; ok {
		delete(medium.files, path)
		return nil
	}
	if _, ok := medium.dirs[path]; ok {
		prefix := path
		if !core.HasSuffix(prefix, "/") {
			prefix += "/"
		}
		for filePath := range medium.files {
			if core.HasPrefix(filePath, prefix) {
				return core.E("io.MemoryMedium.Delete", core.Concat("directory not empty: ", path), fs.ErrExist)
			}
		}
		for directoryPath := range medium.dirs {
			if directoryPath != path && core.HasPrefix(directoryPath, prefix) {
				return core.E("io.MemoryMedium.Delete", core.Concat("directory not empty: ", path), fs.ErrExist)
			}
		}
		delete(medium.dirs, path)
		return nil
	}
	return core.E("io.MemoryMedium.Delete", core.Concat("path not found: ", path), fs.ErrNotExist)
}

func (medium *MemoryMedium) DeleteAll(path string) error {
	found := false
	if _, ok := medium.files[path]; ok {
		delete(medium.files, path)
		found = true
	}
	if _, ok := medium.dirs[path]; ok {
		delete(medium.dirs, path)
		found = true
	}
	prefix := path
	if !core.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	for filePath := range medium.files {
		if core.HasPrefix(filePath, prefix) {
			delete(medium.files, filePath)
			found = true
		}
	}
	for directoryPath := range medium.dirs {
		if core.HasPrefix(directoryPath, prefix) {
			delete(medium.dirs, directoryPath)
			found = true
		}
	}

	if !found {
		return core.E("io.MemoryMedium.DeleteAll", core.Concat("path not found: ", path), fs.ErrNotExist)
	}
	return nil
}

func (medium *MemoryMedium) Rename(oldPath, newPath string) error {
	if content, ok := medium.files[oldPath]; ok {
		medium.files[newPath] = content
		delete(medium.files, oldPath)
		if modTime, ok := medium.modTimes[oldPath]; ok {
			medium.modTimes[newPath] = modTime
			delete(medium.modTimes, oldPath)
		}
		return nil
	}
	if _, ok := medium.dirs[oldPath]; ok {
		medium.dirs[newPath] = true
		delete(medium.dirs, oldPath)

		oldPrefix := oldPath
		if !core.HasSuffix(oldPrefix, "/") {
			oldPrefix += "/"
		}
		newPrefix := newPath
		if !core.HasSuffix(newPrefix, "/") {
			newPrefix += "/"
		}

		filesToMove := make(map[string]string)
		for filePath := range medium.files {
			if core.HasPrefix(filePath, oldPrefix) {
				newFilePath := core.Concat(newPrefix, core.TrimPrefix(filePath, oldPrefix))
				filesToMove[filePath] = newFilePath
			}
		}
		for oldFilePath, newFilePath := range filesToMove {
			medium.files[newFilePath] = medium.files[oldFilePath]
			delete(medium.files, oldFilePath)
			if modTime, ok := medium.modTimes[oldFilePath]; ok {
				medium.modTimes[newFilePath] = modTime
				delete(medium.modTimes, oldFilePath)
			}
		}

		dirsToMove := make(map[string]string)
		for directoryPath := range medium.dirs {
			if core.HasPrefix(directoryPath, oldPrefix) {
				newDirectoryPath := core.Concat(newPrefix, core.TrimPrefix(directoryPath, oldPrefix))
				dirsToMove[directoryPath] = newDirectoryPath
			}
		}
		for oldDirectoryPath, newDirectoryPath := range dirsToMove {
			medium.dirs[newDirectoryPath] = true
			delete(medium.dirs, oldDirectoryPath)
		}
		return nil
	}
	return core.E("io.MemoryMedium.Rename", core.Concat("path not found: ", oldPath), fs.ErrNotExist)
}

func (medium *MemoryMedium) Open(path string) (fs.File, error) {
	content, ok := medium.files[path]
	if !ok {
		return nil, core.E("io.MemoryMedium.Open", core.Concat("file not found: ", path), fs.ErrNotExist)
	}
	return &memoryFile{
		name:    core.PathBase(path),
		content: []byte(content),
	}, nil
}

func (medium *MemoryMedium) Create(path string) (goio.WriteCloser, error) {
	return &memoryWriteCloser{
		medium: medium,
		path:   path,
	}, nil
}

func (medium *MemoryMedium) Append(path string) (goio.WriteCloser, error) {
	content := medium.files[path]
	return &memoryWriteCloser{
		medium: medium,
		path:   path,
		data:   []byte(content),
	}, nil
}

func (medium *MemoryMedium) ReadStream(path string) (goio.ReadCloser, error) {
	return medium.Open(path)
}

func (medium *MemoryMedium) WriteStream(path string) (goio.WriteCloser, error) {
	return medium.Create(path)
}

type memoryFile struct {
	name    string
	content []byte
	offset  int64
}

func (file *memoryFile) Stat() (fs.FileInfo, error) {
	return NewFileInfo(file.name, int64(len(file.content)), 0, time.Time{}, false), nil
}

func (file *memoryFile) Read(buffer []byte) (int, error) {
	if file.offset >= int64(len(file.content)) {
		return 0, goio.EOF
	}
	readCount := copy(buffer, file.content[file.offset:])
	file.offset += int64(readCount)
	return readCount, nil
}

func (file *memoryFile) Close() error {
	return nil
}

type memoryWriteCloser struct {
	medium *MemoryMedium
	path   string
	data   []byte
}

func (writeCloser *memoryWriteCloser) Write(data []byte) (int, error) {
	writeCloser.data = append(writeCloser.data, data...)
	return len(data), nil
}

func (writeCloser *memoryWriteCloser) Close() error {
	writeCloser.medium.files[writeCloser.path] = string(writeCloser.data)
	writeCloser.medium.modTimes[writeCloser.path] = time.Now()
	return nil
}

func (medium *MemoryMedium) List(path string) ([]fs.DirEntry, error) {
	if _, ok := medium.dirs[path]; !ok {
		hasChildren := false
		prefix := path
		if path != "" && !core.HasSuffix(prefix, "/") {
			prefix += "/"
		}
		for filePath := range medium.files {
			if core.HasPrefix(filePath, prefix) {
				hasChildren = true
				break
			}
		}
		if !hasChildren {
			for directoryPath := range medium.dirs {
				if core.HasPrefix(directoryPath, prefix) {
					hasChildren = true
					break
				}
			}
		}
		if !hasChildren && path != "" {
			return nil, core.E("io.MemoryMedium.List", core.Concat("directory not found: ", path), fs.ErrNotExist)
		}
	}

	prefix := path
	if path != "" && !core.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	seen := make(map[string]bool)
	var entries []fs.DirEntry

	for filePath, content := range medium.files {
		if !core.HasPrefix(filePath, prefix) {
			continue
		}
		rest := core.TrimPrefix(filePath, prefix)
		if rest == "" || core.Contains(rest, "/") {
			if idx := bytes.IndexByte([]byte(rest), '/'); idx != -1 {
				dirName := rest[:idx]
				if !seen[dirName] {
					seen[dirName] = true
					entries = append(entries, NewDirEntry(
						dirName,
						true,
						fs.ModeDir|0755,
						NewFileInfo(dirName, 0, fs.ModeDir|0755, time.Time{}, true),
					))
				}
			}
			continue
		}
		if !seen[rest] {
			seen[rest] = true
			entries = append(entries, NewDirEntry(
				rest,
				false,
				0644,
				NewFileInfo(rest, int64(len(content)), 0644, time.Time{}, false),
			))
		}
	}

	for directoryPath := range medium.dirs {
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
			entries = append(entries, NewDirEntry(
				rest,
				true,
				fs.ModeDir|0755,
				NewFileInfo(rest, 0, fs.ModeDir|0755, time.Time{}, true),
			))
		}
	}

	return entries, nil
}

func (medium *MemoryMedium) Stat(path string) (fs.FileInfo, error) {
	if content, ok := medium.files[path]; ok {
		modTime, ok := medium.modTimes[path]
		if !ok {
			modTime = time.Now()
		}
		return NewFileInfo(core.PathBase(path), int64(len(content)), 0644, modTime, false), nil
	}
	if _, ok := medium.dirs[path]; ok {
		return NewFileInfo(core.PathBase(path), 0, fs.ModeDir|0755, time.Time{}, true), nil
	}
	return nil, core.E("io.MemoryMedium.Stat", core.Concat("path not found: ", path), fs.ErrNotExist)
}

func (medium *MemoryMedium) Exists(path string) bool {
	if _, ok := medium.files[path]; ok {
		return true
	}
	if _, ok := medium.dirs[path]; ok {
		return true
	}
	return false
}

func (medium *MemoryMedium) IsDir(path string) bool {
	_, ok := medium.dirs[path]
	return ok
}
