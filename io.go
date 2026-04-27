package io

import (
	goio "io" // AX-6-exception: io interface types have no core equivalent; io.EOF preserves stream semantics.
	"io/fs"   // AX-6-exception: fs interface types have no core equivalent.
	"time"    // AX-6-exception: filesystem metadata timestamps have no core equivalent.

	core "dappco.re/go/core"
	"dappco.re/go/io/internal/fsutil"
	"dappco.re/go/io/local"
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

var _ fs.FileInfo = FileInfo{}

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

var _ fs.DirEntry = DirEntry{}

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

// Example: _ = io.Copy(sourceMedium, "input.txt", destinationMedium, "backup/input.txt")
func Copy(sourceMedium Medium, sourcePath string, destinationMedium Medium, destinationPath string) error {
	content, err := sourceMedium.Read(sourcePath)
	if err != nil {
		return core.E("io.Copy", core.Concat("read failed: ", sourcePath), err)
	}
	mode := fs.FileMode(0644)
	if info, err := sourceMedium.Stat(sourcePath); err == nil {
		mode = info.Mode()
	}
	if err := destinationMedium.WriteMode(destinationPath, content, mode); err != nil {
		return core.E("io.Copy", core.Concat("write failed: ", destinationPath), err)
	}
	return nil
}

// Example: medium := io.NewMemoryMedium()
// Example: _ = medium.Write("config/app.yaml", "port: 8080")
type MemoryMedium struct {
	fileContents      map[string]string
	fileModes         map[string]fs.FileMode
	directories       map[string]bool
	modificationTimes map[string]time.Time
}

var _ Medium = (*MemoryMedium)(nil)

// Example: medium := io.NewMemoryMedium()
// Example: _ = medium.Write("config/app.yaml", "port: 8080")
func NewMemoryMedium() *MemoryMedium {
	return &MemoryMedium{
		fileContents:      make(map[string]string),
		fileModes:         make(map[string]fs.FileMode),
		directories:       make(map[string]bool),
		modificationTimes: make(map[string]time.Time),
	}
}

func (medium *MemoryMedium) ensureAncestorDirectories(filePath string) {
	parentPath := core.PathDir(filePath)
	for parentPath != "." && parentPath != "" {
		medium.directories[parentPath] = true
		nextParentPath := core.PathDir(parentPath)
		if nextParentPath == parentPath {
			break
		}
		parentPath = nextParentPath
	}
}

func (medium *MemoryMedium) directoryExists(path string) bool {
	if path == "" {
		return false
	}
	if _, ok := medium.directories[path]; ok {
		return true
	}

	prefix := path
	if !core.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	for filePath := range medium.fileContents {
		if core.HasPrefix(filePath, prefix) {
			return true
		}
	}
	for directoryPath := range medium.directories {
		if directoryPath != path && core.HasPrefix(directoryPath, prefix) {
			return true
		}
	}

	return false
}

// Example: value, _ := io.NewMemoryMedium().Read("notes.txt")
func (medium *MemoryMedium) Read(path string) (string, error) {
	content, ok := medium.fileContents[path]
	if !ok {
		return "", core.E("io.MemoryMedium.Read", core.Concat("file not found: ", path), fs.ErrNotExist)
	}
	return content, nil
}

// Example: _ = io.NewMemoryMedium().Write("notes.txt", "hello")
func (medium *MemoryMedium) Write(path, content string) error {
	return medium.WriteMode(path, content, 0644)
}

// Example: _ = io.NewMemoryMedium().WriteMode("keys/private.key", "secret", 0600)
func (medium *MemoryMedium) WriteMode(filePath, content string, mode fs.FileMode) error {
	// Verify no ancestor directory component is stored as a file.
	ancestor := core.PathDir(filePath)
	for ancestor != "." && ancestor != "" {
		if _, ok := medium.fileContents[ancestor]; ok {
			return core.E("io.MemoryMedium.WriteMode", core.Concat("ancestor path is a file: ", ancestor), fs.ErrExist)
		}
		next := core.PathDir(ancestor)
		if next == ancestor {
			break
		}
		ancestor = next
	}
	if _, ok := medium.directories[filePath]; ok {
		return core.E("io.MemoryMedium.WriteMode", core.Concat("path is a directory: ", filePath), fs.ErrExist)
	}
	medium.ensureAncestorDirectories(filePath)
	medium.fileContents[filePath] = content
	medium.fileModes[filePath] = mode
	medium.modificationTimes[filePath] = time.Now()
	return nil
}

// Example: _ = io.NewMemoryMedium().EnsureDir("config/app")
func (medium *MemoryMedium) EnsureDir(path string) error {
	if _, ok := medium.fileContents[path]; ok {
		return core.E("io.MemoryMedium.EnsureDir", core.Concat("path is already a file: ", path), fs.ErrExist)
	}
	medium.ensureAncestorDirectories(path)
	medium.directories[path] = true
	return nil
}

// Example: ok := io.NewMemoryMedium().IsFile("notes.txt")
func (medium *MemoryMedium) IsFile(path string) bool {
	_, ok := medium.fileContents[path]
	return ok
}

// Example: _ = io.NewMemoryMedium().Delete("old.txt")
func (medium *MemoryMedium) Delete(path string) error {
	if _, ok := medium.fileContents[path]; ok {
		delete(medium.fileContents, path)
		delete(medium.fileModes, path)
		delete(medium.modificationTimes, path)
		return nil
	}
	if medium.directoryExists(path) {
		prefix := path
		if !core.HasSuffix(prefix, "/") {
			prefix += "/"
		}
		hasChildren := false
		for filePath := range medium.fileContents {
			if core.HasPrefix(filePath, prefix) {
				hasChildren = true
				break
			}
		}
		if !hasChildren {
			for directoryPath := range medium.directories {
				if directoryPath != path && core.HasPrefix(directoryPath, prefix) {
					hasChildren = true
					break
				}
			}
		}
		if hasChildren {
			return core.E("io.MemoryMedium.Delete", core.Concat("directory not empty: ", path), fs.ErrExist)
		}
		delete(medium.directories, path)
		return nil
	}
	return core.E("io.MemoryMedium.Delete", core.Concat("path not found: ", path), fs.ErrNotExist)
}

// Example: _ = io.NewMemoryMedium().DeleteAll("logs")
func (medium *MemoryMedium) DeleteAll(path string) error {
	found := false
	if _, ok := medium.fileContents[path]; ok {
		delete(medium.fileContents, path)
		delete(medium.fileModes, path)
		delete(medium.modificationTimes, path)
		found = true
	}
	if _, ok := medium.directories[path]; ok {
		delete(medium.directories, path)
		found = true
	}
	prefix := path
	if !core.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	for filePath := range medium.fileContents {
		if core.HasPrefix(filePath, prefix) {
			delete(medium.fileContents, filePath)
			delete(medium.fileModes, filePath)
			delete(medium.modificationTimes, filePath)
			found = true
		}
	}
	for directoryPath := range medium.directories {
		if core.HasPrefix(directoryPath, prefix) {
			delete(medium.directories, directoryPath)
			found = true
		}
	}

	if !found {
		return core.E("io.MemoryMedium.DeleteAll", core.Concat("path not found: ", path), fs.ErrNotExist)
	}
	return nil
}

// Example: _ = io.NewMemoryMedium().Rename("drafts/todo.txt", "archive/todo.txt")
func (medium *MemoryMedium) Rename(oldPath, newPath string) error {
	if content, ok := medium.fileContents[oldPath]; ok {
		medium.fileContents[newPath] = content
		delete(medium.fileContents, oldPath)
		if mode, ok := medium.fileModes[oldPath]; ok {
			medium.fileModes[newPath] = mode
			delete(medium.fileModes, oldPath)
		}
		if modTime, ok := medium.modificationTimes[oldPath]; ok {
			medium.modificationTimes[newPath] = modTime
			delete(medium.modificationTimes, oldPath)
		}
		return nil
	}
	if medium.directoryExists(oldPath) {
		medium.directories[newPath] = true
		if _, ok := medium.directories[oldPath]; ok {
			delete(medium.directories, oldPath)
		}

		oldPrefix := oldPath
		if !core.HasSuffix(oldPrefix, "/") {
			oldPrefix += "/"
		}
		newPrefix := newPath
		if !core.HasSuffix(newPrefix, "/") {
			newPrefix += "/"
		}

		filesToMove := make(map[string]string)
		for filePath := range medium.fileContents {
			if core.HasPrefix(filePath, oldPrefix) {
				newFilePath := core.Concat(newPrefix, core.TrimPrefix(filePath, oldPrefix))
				filesToMove[filePath] = newFilePath
			}
		}
		for oldFilePath, newFilePath := range filesToMove {
			medium.fileContents[newFilePath] = medium.fileContents[oldFilePath]
			delete(medium.fileContents, oldFilePath)
			if modTime, ok := medium.modificationTimes[oldFilePath]; ok {
				medium.modificationTimes[newFilePath] = modTime
				delete(medium.modificationTimes, oldFilePath)
			}
			if fileMode, ok := medium.fileModes[oldFilePath]; ok {
				medium.fileModes[newFilePath] = fileMode
				delete(medium.fileModes, oldFilePath)
			}
		}

		dirsToMove := make(map[string]string)
		for directoryPath := range medium.directories {
			if core.HasPrefix(directoryPath, oldPrefix) {
				newDirectoryPath := core.Concat(newPrefix, core.TrimPrefix(directoryPath, oldPrefix))
				dirsToMove[directoryPath] = newDirectoryPath
			}
		}
		for oldDirectoryPath, newDirectoryPath := range dirsToMove {
			medium.directories[newDirectoryPath] = true
			delete(medium.directories, oldDirectoryPath)
		}
		return nil
	}
	return core.E("io.MemoryMedium.Rename", core.Concat("path not found: ", oldPath), fs.ErrNotExist)
}

// Example: file, _ := io.NewMemoryMedium().Open("notes.txt")
func (medium *MemoryMedium) Open(path string) (fs.File, error) {
	content, ok := medium.fileContents[path]
	if !ok {
		return nil, core.E("io.MemoryMedium.Open", core.Concat("file not found: ", path), fs.ErrNotExist)
	}
	return &MemoryFile{
		name:    core.PathBase(path),
		content: []byte(content),
		mode:    medium.modeForPath(path),
		modTime: medium.modificationTimeForPath(path),
	}, nil
}

// Example: writer, _ := io.NewMemoryMedium().Create("notes.txt")
func (medium *MemoryMedium) Create(path string) (goio.WriteCloser, error) {
	return &MemoryWriteCloser{
		medium: medium,
		path:   path,
		mode:   0644,
	}, nil
}

// Example: writer, _ := io.NewMemoryMedium().Append("notes.txt")
func (medium *MemoryMedium) Append(path string) (goio.WriteCloser, error) {
	content := medium.fileContents[path]
	return &MemoryWriteCloser{
		medium: medium,
		path:   path,
		data:   []byte(content),
		mode:   medium.modeForPath(path),
	}, nil
}

// Example: reader, _ := io.NewMemoryMedium().ReadStream("notes.txt")
func (medium *MemoryMedium) ReadStream(path string) (goio.ReadCloser, error) {
	return medium.Open(path)
}

// Example: writer, _ := io.NewMemoryMedium().WriteStream("notes.txt")
func (medium *MemoryMedium) WriteStream(path string) (goio.WriteCloser, error) {
	return medium.Create(path)
}

// Example: file, _ := io.NewMemoryMedium().Open("notes.txt")
type MemoryFile struct {
	name    string
	content []byte
	offset  int64
	mode    fs.FileMode
	modTime time.Time
}

var _ fs.File = (*MemoryFile)(nil)
var _ goio.ReadCloser = (*MemoryFile)(nil)

func (file *MemoryFile) Stat() (fs.FileInfo, error) {
	return NewFileInfo(file.name, int64(len(file.content)), file.mode, file.modTime, false), nil
}

func (file *MemoryFile) Read(buffer []byte) (int, error) {
	if file.offset >= int64(len(file.content)) {
		return 0, goio.EOF
	}
	readCount := copy(buffer, file.content[file.offset:])
	file.offset += int64(readCount)
	return readCount, nil
}

func (file *MemoryFile) Close() error {
	return nil
}

// Example: writer, _ := io.NewMemoryMedium().Create("notes.txt")
type MemoryWriteCloser struct {
	medium *MemoryMedium
	path   string
	data   []byte
	mode   fs.FileMode
}

var _ goio.WriteCloser = (*MemoryWriteCloser)(nil)

func (writeCloser *MemoryWriteCloser) Write(data []byte) (int, error) {
	writeCloser.data = append(writeCloser.data, data...)
	return len(data), nil
}

func (writeCloser *MemoryWriteCloser) Close() error {
	if _, ok := writeCloser.medium.directories[writeCloser.path]; ok {
		return core.E("io.MemoryWriteCloser.Close", core.Concat("path is a directory: ", writeCloser.path), fs.ErrExist)
	}
	writeCloser.medium.ensureAncestorDirectories(writeCloser.path)
	writeCloser.medium.fileContents[writeCloser.path] = string(writeCloser.data)
	writeCloser.medium.fileModes[writeCloser.path] = writeCloser.mode
	writeCloser.medium.modificationTimes[writeCloser.path] = time.Now()
	return nil
}

func (medium *MemoryMedium) modeForPath(path string) fs.FileMode {
	if mode, ok := medium.fileModes[path]; ok {
		return mode
	}
	return 0644
}

func (medium *MemoryMedium) modificationTimeForPath(path string) time.Time {
	if modTime, ok := medium.modificationTimes[path]; ok {
		return modTime
	}
	return time.Time{}
}

// Example: entries, _ := io.NewMemoryMedium().List("config")
func (medium *MemoryMedium) List(path string) ([]fs.DirEntry, error) {
	if _, ok := medium.directories[path]; !ok {
		hasChildren := false
		prefix := path
		if path != "" && !core.HasSuffix(prefix, "/") {
			prefix += "/"
		}
		for filePath := range medium.fileContents {
			if core.HasPrefix(filePath, prefix) {
				hasChildren = true
				break
			}
		}
		if !hasChildren {
			for directoryPath := range medium.directories {
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

	for filePath, content := range medium.fileContents {
		if !core.HasPrefix(filePath, prefix) {
			continue
		}
		rest := core.TrimPrefix(filePath, prefix)
		if rest == "" || core.Contains(rest, "/") {
			if parts := core.SplitN(rest, "/", 2); len(parts) == 2 {
				dirName := parts[0]
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
			filePath := core.Concat(prefix, rest)
			entries = append(entries, NewDirEntry(
				rest,
				false,
				medium.modeForPath(filePath),
				NewFileInfo(rest, int64(len(content)), medium.modeForPath(filePath), medium.modificationTimeForPath(filePath), false),
			))
		}
	}

	for directoryPath := range medium.directories {
		if !core.HasPrefix(directoryPath, prefix) {
			continue
		}
		rest := core.TrimPrefix(directoryPath, prefix)
		if rest == "" {
			continue
		}
		if parts := core.SplitN(rest, "/", 2); len(parts) == 2 {
			rest = parts[0]
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

	fsutil.SortDirEntriesByName(entries)

	return entries, nil
}

// Example: info, _ := io.NewMemoryMedium().Stat("notes.txt")
func (medium *MemoryMedium) Stat(path string) (fs.FileInfo, error) {
	if content, ok := medium.fileContents[path]; ok {
		modTime, ok := medium.modificationTimes[path]
		if !ok {
			modTime = time.Now()
		}
		return NewFileInfo(core.PathBase(path), int64(len(content)), medium.modeForPath(path), modTime, false), nil
	}
	if medium.directoryExists(path) {
		return NewFileInfo(core.PathBase(path), 0, fs.ModeDir|0755, time.Time{}, true), nil
	}
	return nil, core.E("io.MemoryMedium.Stat", core.Concat("path not found: ", path), fs.ErrNotExist)
}

// Example: ok := io.NewMemoryMedium().Exists("notes.txt")
func (medium *MemoryMedium) Exists(path string) bool {
	if _, ok := medium.fileContents[path]; ok {
		return true
	}
	return medium.directoryExists(path)
}

// Example: ok := io.NewMemoryMedium().IsDir("config")
func (medium *MemoryMedium) IsDir(path string) bool {
	return medium.directoryExists(path)
}
