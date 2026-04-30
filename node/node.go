// Example: nodeTree := node.New()
// Example: nodeTree.AddData("config/app.yaml", []byte("port: 8080"))
// Example: snapshot, _ := nodeTree.ToTar()
// Example: restored, _ := node.FromTar(snapshot)
package node

import (
	"archive/tar" // AX-6-exception: tar archive transport has no core equivalent.
	goio "io"     // AX-6-exception: io interface types have no core equivalent; io.EOF preserves stream semantics.
	"io/fs"       // AX-6-exception: fs interface types have no core equivalent.
	"time"        // AX-6-exception: filesystem metadata timestamps have no core equivalent.

	core "dappco.re/go"
	coreio "dappco.re/go/io"
	"dappco.re/go/io/internal/fsutil"
)

const (
	opNodeExportFile    = "node.ExportFile"
	msgNodePathNotFound = "path not found: "
)

// Example: nodeTree := node.New()
// Example: nodeTree.AddData("config/app.yaml", []byte("port: 8080"))
// Example: snapshot, _ := nodeTree.ToTar()
// Example: restored, _ := node.FromTar(snapshot)
// Note: Node is not goroutine-safe. All methods must be called from a single goroutine,
// or the caller must provide external synchronisation.
type Node struct {
	files map[string]*dataFile
}

var _ coreio.Medium = (*Node)(nil)
var _ fs.ReadFileFS = (*Node)(nil)

// Example: nodeTree := node.New()
// Example: _ = nodeTree.Write("config/app.yaml", "port: 8080")
func New() *Node {
	return &Node{files: make(map[string]*dataFile)}
}

// AX-6-exception: core.NewBuffer is unavailable in the pinned core module; this is
// the minimal intrinsic writer needed by archive/tar.
type nodeArchiveBuffer struct {
	data []byte
}

func (buffer *nodeArchiveBuffer) Write(data []byte) (int, error) {
	buffer.data = append(buffer.data, data...)
	return len(data), nil
}

// Example: nodeTree.AddData("config/app.yaml", []byte("port: 8080"))
func (node *Node) AddData(name string, content []byte) {
	name = core.TrimPrefix(name, "/")
	if name == "" {
		return
	}
	if core.HasSuffix(name, "/") {
		return
	}
	node.files[name] = &dataFile{
		name:    name,
		content: content,
		modTime: time.Now(),
	}
}

// Example: snapshot, _ := nodeTree.ToTar()
func (node *Node) ToTar() ([]byte, error) {
	buffer := &nodeArchiveBuffer{}
	tarWriter := tar.NewWriter(buffer)

	for _, file := range node.files {
		hdr := &tar.Header{
			Name:    file.name,
			Mode:    0600,
			Size:    int64(len(file.content)),
			ModTime: file.modTime,
		}
		if err := tarWriter.WriteHeader(hdr); err != nil {
			return nil, err
		}
		if _, err := tarWriter.Write(file.content); err != nil {
			return nil, err
		}
	}

	if err := tarWriter.Close(); err != nil {
		return nil, err
	}

	return buffer.data, nil
}

// Example: restored, _ := node.FromTar(snapshot)
func FromTar(data []byte) (*Node, error) {
	restoredNode := New()
	if err := restoredNode.LoadTar(data); err != nil {
		return nil, err
	}
	return restoredNode, nil
}

// Example: _ = nodeTree.LoadTar(snapshot)
func (node *Node) LoadTar(data []byte) error {
	newFiles := make(map[string]*dataFile)
	tarReader := tar.NewReader(core.NewReader(string(data)))

	for {
		header, err := tarReader.Next()
		if err == goio.EOF {
			break
		}
		if err != nil {
			return err
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}
		file, err := dataFileFromTarEntry(tarReader, header)
		if err != nil {
			return err
		}
		if file != nil {
			newFiles[file.name] = file
		}
	}

	node.files = newFiles
	return nil
}

func dataFileFromTarEntry(tarReader *tar.Reader, header *tar.Header) (*dataFile, error) {
	contentResult := core.ReadAll(tarReader)
	if !contentResult.OK {
		if err, ok := contentResult.Value.(error); ok {
			return nil, core.E("node.LoadTar", "read tar entry", err)
		}
		return nil, core.E("node.LoadTar", "read tar entry", fs.ErrInvalid)
	}
	name := core.TrimPrefix(header.Name, "/")
	if name == "" || core.HasSuffix(name, "/") {
		return nil, nil
	}
	return &dataFile{
		name:    name,
		content: []byte(contentResult.Value.(string)),
		modTime: header.ModTime,
	}, nil
}

// Example: options := node.WalkOptions{MaxDepth: 1, SkipErrors: true}
type WalkOptions struct {
	MaxDepth   int
	Filter     func(entryPath string, entry fs.DirEntry) bool
	SkipErrors bool
}

// Example: _ = nodeTree.Walk(".", func(_ string, _ fs.DirEntry, _ error) error { return nil }, node.WalkOptions{MaxDepth: 1, SkipErrors: true})
func (node *Node) Walk(root string, walkFunc fs.WalkDirFunc, options WalkOptions) error {
	if options.SkipErrors {
		if _, err := node.Stat(root); err != nil {
			return nil
		}
	}

	return fs.WalkDir(node, root, func(entryPath string, entry fs.DirEntry, err error) error {
		if skip, skipErr := walkFilterDecision(entryPath, entry, err, options.Filter); skip {
			return skipErr
		}

		walkResult := walkFunc(entryPath, entry, err)

		if walkResult == nil && maxDepthReached(root, entryPath, entry, options.MaxDepth) {
			return fs.SkipDir
		}

		return walkResult
	})
}

func walkFilterDecision(entryPath string, entry fs.DirEntry, err error, filter func(string, fs.DirEntry) bool) (bool, error) {
	if filter == nil || err != nil || filter(entryPath, entry) {
		return false, nil
	}
	if entry != nil && entry.IsDir() {
		return true, fs.SkipDir
	}
	return true, nil
}

func maxDepthReached(root, entryPath string, entry fs.DirEntry, maxDepth int) bool {
	if maxDepth <= 0 || entry == nil || !entry.IsDir() || entryPath == root {
		return false
	}
	relativePath := core.TrimPrefix(entryPath, root)
	relativePath = core.TrimPrefix(relativePath, "/")
	return pathDepth(relativePath) >= maxDepth
}

func pathDepth(path string) int {
	depth := 0
	for _, part := range core.Split(path, "/") {
		if part != "" {
			depth++
		}
	}
	return depth
}

// Example: content, _ := nodeTree.ReadFile("config/app.yaml")
func (node *Node) ReadFile(name string) ([]byte, error) {
	name = core.TrimPrefix(name, "/")
	file, ok := node.files[name]
	if !ok {
		return nil, core.E("node.ReadFile", core.Concat(msgNodePathNotFound, name), fs.ErrNotExist)
	}
	result := make([]byte, len(file.content))
	copy(result, file.content)
	return result, nil
}

// ExportFile writes a node file to the local filesystem. It operates on coreio.Local directly
// and is intentionally local-only — use CopyTo for Medium-agnostic transfers.
// Example: _ = nodeTree.ExportFile("config/app.yaml", "backup/app.yaml", 0644)
func (node *Node) ExportFile(sourcePath, destinationPath string, permissions fs.FileMode) error {
	sourcePath = core.TrimPrefix(sourcePath, "/")
	file, ok := node.files[sourcePath]
	if !ok {
		info, err := node.Stat(sourcePath)
		if err != nil {
			return core.E(opNodeExportFile, core.Concat("source not found: ", sourcePath), fs.ErrNotExist)
		}
		if info.IsDir() {
			return core.E(opNodeExportFile, core.Concat("source is a directory: ", sourcePath), fs.ErrInvalid)
		}
		// unreachable: Stat only succeeds for directories when file is absent
		return core.E(opNodeExportFile, core.Concat("source not found: ", sourcePath), fs.ErrNotExist)
	}
	parent := core.PathDir(destinationPath)
	if parent != "." && parent != "" && parent != destinationPath && !coreio.Local.IsDir(parent) {
		return core.E(opNodeExportFile, core.Concat("parent directory not found: ", destinationPath), fs.ErrNotExist)
	}
	return coreio.Local.WriteMode(destinationPath, string(file.content), permissions)
}

// Example: _ = nodeTree.CopyTo(io.NewMemoryMedium(), "config", "backup/config")
func (node *Node) CopyTo(target coreio.Medium, sourcePath, destinationPath string) error {
	sourcePath = core.TrimPrefix(sourcePath, "/")
	if sourcePath == "." {
		sourcePath = ""
	}
	info, err := node.Stat(sourcePath)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		file, ok := node.files[sourcePath]
		if !ok {
			return core.E("node.CopyTo", core.Concat(msgNodePathNotFound, sourcePath), fs.ErrNotExist)
		}
		return target.Write(destinationPath, string(file.content))
	}

	prefix := sourcePath
	if prefix != "" && !core.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	for filePath, file := range node.files {
		if !core.HasPrefix(filePath, prefix) && filePath != sourcePath {
			continue
		}
		relativePath := core.TrimPrefix(filePath, prefix)
		copyDestinationPath := destinationPath
		if relativePath != "" {
			copyDestinationPath = core.Concat(destinationPath, "/", relativePath)
		}
		if err := target.Write(copyDestinationPath, string(file.content)); err != nil {
			return err
		}
	}
	return nil
}

// Example: file, _ := nodeTree.Open("config/app.yaml")
func (node *Node) Open(name string) (fs.File, error) {
	name = core.TrimPrefix(name, "/")
	if name == "." {
		name = ""
	}
	if dataFile, ok := node.files[name]; ok {
		return &dataFileReader{file: dataFile}, nil
	}
	if name == "" {
		return &dirFile{path: ".", modTime: time.Now()}, nil
	}
	prefix := name + "/"
	for filePath := range node.files {
		if core.HasPrefix(filePath, prefix) {
			return &dirFile{path: name, modTime: time.Now()}, nil
		}
	}
	return nil, core.E("node.Open", core.Concat(msgNodePathNotFound, name), fs.ErrNotExist)
}

// Example: info, _ := nodeTree.Stat("config/app.yaml")
func (node *Node) Stat(name string) (fs.FileInfo, error) {
	name = core.TrimPrefix(name, "/")
	if name == "." {
		name = ""
	}
	if dataFile, ok := node.files[name]; ok {
		return dataFile.Stat()
	}
	if name == "" {
		return &dirInfo{name: ".", modTime: time.Now()}, nil
	}
	prefix := name + "/"
	for filePath := range node.files {
		if core.HasPrefix(filePath, prefix) {
			return &dirInfo{name: core.PathBase(name), modTime: time.Now()}, nil
		}
	}
	return nil, core.E("node.Stat", core.Concat(msgNodePathNotFound, name), fs.ErrNotExist)
}

// Example: entries, _ := nodeTree.ReadDir("config")
func (node *Node) ReadDir(name string) ([]fs.DirEntry, error) {
	name = core.TrimPrefix(name, "/")
	if name == "." {
		name = ""
	}

	info, statErr := node.Stat(name)
	if statErr != nil {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrNotExist}
	}
	if !info.IsDir() {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrInvalid}
	}

	entries := []fs.DirEntry{}
	seen := make(map[string]bool)

	prefix := ""
	if name != "" {
		prefix = name + "/"
	}

	for filePath := range node.files {
		if !core.HasPrefix(filePath, prefix) {
			continue
		}

		relPath := core.TrimPrefix(filePath, prefix)
		firstComponent := core.SplitN(relPath, "/", 2)[0]

		if seen[firstComponent] {
			continue
		}
		seen[firstComponent] = true

		if core.Contains(relPath, "/") {
			directoryInfo := &dirInfo{name: firstComponent, modTime: time.Now()}
			entries = append(entries, fs.FileInfoToDirEntry(directoryInfo))
		} else {
			file := node.files[filePath]
			info, _ := file.Stat()
			entries = append(entries, fs.FileInfoToDirEntry(info))
		}
	}

	fsutil.SortDirEntriesByName(entries)

	return entries, nil
}

// Example: content, _ := nodeTree.Read("config/app.yaml")
func (node *Node) Read(filePath string) (string, error) {
	filePath = core.TrimPrefix(filePath, "/")
	file, ok := node.files[filePath]
	if !ok {
		return "", core.E("node.Read", core.Concat(msgNodePathNotFound, filePath), fs.ErrNotExist)
	}
	return string(file.content), nil
}

// Example: _ = nodeTree.Write("config/app.yaml", "port: 8080")
func (node *Node) Write(filePath, content string) error {
	filePath = core.TrimPrefix(filePath, "/")
	if filePath == "" || filePath == "." {
		return core.E("node.Write", "empty path", fs.ErrInvalid)
	}
	node.AddData(filePath, []byte(content))
	return nil
}

// Example: _ = nodeTree.WriteMode("keys/private.key", key, 0600)
// Note: mode is intentionally ignored — in-memory nodes have no filesystem permission model.
func (node *Node) WriteMode(filePath, content string, mode fs.FileMode) error {
	return node.Write(filePath, content)
}

// Example: _ = nodeTree.EnsureDir("config")
func (node *Node) EnsureDir(directoryPath string) error {
	return nil
}

// Example: exists := nodeTree.Exists("config/app.yaml")
func (node *Node) Exists(filePath string) bool {
	_, err := node.Stat(filePath)
	return err == nil
}

// Example: isFile := nodeTree.IsFile("config/app.yaml")
func (node *Node) IsFile(filePath string) bool {
	filePath = core.TrimPrefix(filePath, "/")
	_, ok := node.files[filePath]
	return ok
}

// Example: isDirectory := nodeTree.IsDir("config")
func (node *Node) IsDir(filePath string) bool {
	info, err := node.Stat(filePath)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// Example: _ = nodeTree.Delete("config/app.yaml")
func (node *Node) Delete(filePath string) error {
	filePath = core.TrimPrefix(filePath, "/")
	if _, ok := node.files[filePath]; ok {
		delete(node.files, filePath)
		return nil
	}
	return core.E("node.Delete", core.Concat(msgNodePathNotFound, filePath), fs.ErrNotExist)
}

// Example: _ = nodeTree.DeleteAll("logs/archive")
func (node *Node) DeleteAll(filePath string) error {
	filePath = core.TrimPrefix(filePath, "/")

	found := false
	if _, ok := node.files[filePath]; ok {
		delete(node.files, filePath)
		found = true
	}

	prefix := filePath + "/"
	for entryPath := range node.files {
		if core.HasPrefix(entryPath, prefix) {
			delete(node.files, entryPath)
			found = true
		}
	}

	if !found {
		return core.E("node.DeleteAll", core.Concat(msgNodePathNotFound, filePath), fs.ErrNotExist)
	}
	return nil
}

// Example: _ = nodeTree.Rename("drafts/todo.txt", "archive/todo.txt")
// Example: _ = nodeTree.Rename("drafts", "archive")
func (node *Node) Rename(oldPath, newPath string) error {
	oldPath = core.TrimPrefix(oldPath, "/")
	newPath = core.TrimPrefix(newPath, "/")

	if file, ok := node.files[oldPath]; ok {
		file.name = newPath
		node.files[newPath] = file
		delete(node.files, oldPath)
		return nil
	}

	// Directory rename: batch-rename all entries that share the prefix.
	oldPrefix := oldPath + "/"
	newPrefix := newPath + "/"
	renamed := 0
	toAdd := make(map[string]*dataFile)
	toDelete := make([]string, 0)
	for filePath, file := range node.files {
		if core.HasPrefix(filePath, oldPrefix) {
			updatedPath := core.Concat(newPrefix, core.TrimPrefix(filePath, oldPrefix))
			file.name = updatedPath
			toAdd[updatedPath] = file
			toDelete = append(toDelete, filePath)
			renamed++
		}
	}
	for _, p := range toDelete {
		delete(node.files, p)
	}
	for p, f := range toAdd {
		node.files[p] = f
	}
	if renamed == 0 {
		return core.E("node.Rename", core.Concat(msgNodePathNotFound, oldPath), fs.ErrNotExist)
	}
	return nil
}

// Example: entries, _ := nodeTree.List("config")
func (node *Node) List(filePath string) ([]fs.DirEntry, error) {
	filePath = core.TrimPrefix(filePath, "/")
	if filePath == "" || filePath == "." {
		return node.ReadDir(".")
	}
	return node.ReadDir(filePath)
}

// Example: writer, _ := nodeTree.Create("logs/app.log")
func (node *Node) Create(filePath string) (goio.WriteCloser, error) {
	filePath = core.TrimPrefix(filePath, "/")
	return &nodeWriter{node: node, path: filePath}, nil
}

// Example: writer, _ := nodeTree.Append("logs/app.log")
func (node *Node) Append(filePath string) (goio.WriteCloser, error) {
	filePath = core.TrimPrefix(filePath, "/")
	var existing []byte
	if file, ok := node.files[filePath]; ok {
		existing = make([]byte, len(file.content))
		copy(existing, file.content)
	}
	return &nodeWriter{node: node, path: filePath, buffer: existing}, nil
}

func (node *Node) ReadStream(filePath string) (goio.ReadCloser, error) {
	file, err := node.Open(filePath)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (node *Node) WriteStream(filePath string) (goio.WriteCloser, error) {
	return node.Create(filePath)
}

type nodeWriter struct {
	node   *Node
	path   string
	buffer []byte
}

func (writer *nodeWriter) Write(data []byte) (int, error) {
	writer.buffer = append(writer.buffer, data...)
	return len(data), nil
}

func (writer *nodeWriter) Close() error {
	if writer.path == "" || writer.path == "." {
		return core.E("node.nodeWriter.Close", "empty path", fs.ErrInvalid)
	}
	writer.node.files[writer.path] = &dataFile{
		name:    writer.path,
		content: writer.buffer,
		modTime: time.Now(),
	}
	return nil
}

type dataFile struct {
	name    string
	content []byte
	modTime time.Time
}

func (file *dataFile) Stat() (fs.FileInfo, error) { return &dataFileInfo{file: file}, nil }

func (file *dataFile) Read(buffer []byte) (int, error) { return 0, goio.EOF }

func (file *dataFile) Close() error { return nil }

type dataFileInfo struct{ file *dataFile }

func (info *dataFileInfo) Name() string { return core.PathBase(info.file.name) }

func (info *dataFileInfo) Size() int64 { return int64(len(info.file.content)) }

func (info *dataFileInfo) Mode() fs.FileMode { return 0444 }

func (info *dataFileInfo) ModTime() time.Time { return info.file.modTime }

func (info *dataFileInfo) IsDir() bool { return false }

func (info *dataFileInfo) Sys() any { return nil }

type dataFileReader struct {
	file   *dataFile
	offset int64
}

func (reader *dataFileReader) Stat() (fs.FileInfo, error) { return reader.file.Stat() }

func (reader *dataFileReader) Read(buffer []byte) (int, error) {
	if reader.offset >= int64(len(reader.file.content)) {
		return 0, goio.EOF
	}
	readCount := copy(buffer, reader.file.content[reader.offset:])
	reader.offset += int64(readCount)
	return readCount, nil
}

func (reader *dataFileReader) Close() error { return nil }

type dirInfo struct {
	name    string
	modTime time.Time
}

func (info *dirInfo) Name() string { return info.name }

func (info *dirInfo) Size() int64 { return 0 }

func (info *dirInfo) Mode() fs.FileMode { return fs.ModeDir | 0555 }

func (info *dirInfo) ModTime() time.Time { return info.modTime }

func (info *dirInfo) IsDir() bool { return true }

func (info *dirInfo) Sys() any { return nil }

type dirFile struct {
	path    string
	modTime time.Time
}

func (directory *dirFile) Stat() (fs.FileInfo, error) {
	return &dirInfo{name: core.PathBase(directory.path), modTime: directory.modTime}, nil
}

func (directory *dirFile) Read([]byte) (int, error) {
	return 0, core.E("node.dirFile.Read", core.Concat("cannot read directory: ", directory.path), &fs.PathError{Op: "read", Path: directory.path, Err: fs.ErrInvalid})
}

func (directory *dirFile) Close() error { return nil }

var _ fs.FS = (*Node)(nil)

var _ fs.StatFS = (*Node)(nil)
var _ fs.ReadDirFS = (*Node)(nil)

var _ goio.WriteCloser = (*nodeWriter)(nil)

var _ fs.File = (*dirFile)(nil)

var _ fs.File = (*dataFileReader)(nil)
