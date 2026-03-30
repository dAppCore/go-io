// Example: nodeTree := node.New()
// Example: nodeTree.AddData("config/app.yaml", []byte("port: 8080"))
// Example: snapshot, _ := nodeTree.ToTar()
// Example: restored, _ := node.FromTar(snapshot)
package node

import (
	"archive/tar"
	"bytes"
	"cmp"
	goio "io"
	"io/fs"
	"path"
	"slices"
	"time"

	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"
)

// Example: nodeTree := node.New()
// Example: nodeTree.AddData("config/app.yaml", []byte("port: 8080"))
// Example: snapshot, _ := nodeTree.ToTar()
// Example: restored, _ := node.FromTar(snapshot)
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
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	for _, file := range node.files {
		hdr := &tar.Header{
			Name:    file.name,
			Mode:    0600,
			Size:    int64(len(file.content)),
			ModTime: file.modTime,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return nil, err
		}
		if _, err := tw.Write(file.content); err != nil {
			return nil, err
		}
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
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
	tr := tar.NewReader(bytes.NewReader(data))

	for {
		header, err := tr.Next()
		if err == goio.EOF {
			break
		}
		if err != nil {
			return err
		}

		if header.Typeflag == tar.TypeReg {
			content, err := goio.ReadAll(tr)
			if err != nil {
				return core.E("node.LoadTar", "read tar entry", err)
			}
			name := core.TrimPrefix(header.Name, "/")
			if name == "" || core.HasSuffix(name, "/") {
				continue
			}
			newFiles[name] = &dataFile{
				name:    name,
				content: content,
				modTime: header.ModTime,
			}
		}
	}

	node.files = newFiles
	return nil
}

// Example: _ = nodeTree.WalkNode("config", func(_ string, _ fs.DirEntry, _ error) error { return nil })
func (node *Node) WalkNode(root string, fn fs.WalkDirFunc) error {
	return fs.WalkDir(node, root, fn)
}

// Example: options := node.WalkOptions{MaxDepth: 1, SkipErrors: true}
type WalkOptions struct {
	MaxDepth   int
	Filter     func(entryPath string, entry fs.DirEntry) bool
	SkipErrors bool
}

// Example: _ = nodeTree.WalkWithOptions(".", callback, node.WalkOptions{MaxDepth: 1, SkipErrors: true})
func (node *Node) WalkWithOptions(root string, fn fs.WalkDirFunc, options WalkOptions) error {
	if options.SkipErrors {
		if _, err := node.Stat(root); err != nil {
			return nil
		}
	}

	return fs.WalkDir(node, root, func(entryPath string, entry fs.DirEntry, err error) error {
		if options.Filter != nil && err == nil {
			if !options.Filter(entryPath, entry) {
				if entry != nil && entry.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
		}

		result := fn(entryPath, entry, err)

		if result == nil && options.MaxDepth > 0 && entry != nil && entry.IsDir() && entryPath != root {
			rel := core.TrimPrefix(entryPath, root)
			rel = core.TrimPrefix(rel, "/")
			depth := len(core.Split(rel, "/"))
			if depth >= options.MaxDepth {
				return fs.SkipDir
			}
		}

		return result
	})
}

// Example: content, _ := nodeTree.ReadFile("config/app.yaml")
func (node *Node) ReadFile(name string) ([]byte, error) {
	name = core.TrimPrefix(name, "/")
	file, ok := node.files[name]
	if !ok {
		return nil, core.E("node.ReadFile", core.Concat("path not found: ", name), fs.ErrNotExist)
	}
	result := make([]byte, len(file.content))
	copy(result, file.content)
	return result, nil
}

// Example: _ = nodeTree.CopyFile("config/app.yaml", "/tmp/app.yaml", 0644)
func (node *Node) CopyFile(sourcePath, destinationPath string, perm fs.FileMode) error {
	sourcePath = core.TrimPrefix(sourcePath, "/")
	file, ok := node.files[sourcePath]
	if !ok {
		info, err := node.Stat(sourcePath)
		if err != nil {
			return core.E("node.CopyFile", core.Concat("source not found: ", sourcePath), fs.ErrNotExist)
		}
		if info.IsDir() {
			return core.E("node.CopyFile", core.Concat("source is a directory: ", sourcePath), fs.ErrInvalid)
		}
		return core.E("node.CopyFile", core.Concat("source not found: ", sourcePath), fs.ErrNotExist)
	}
	parent := core.PathDir(destinationPath)
	if parent != "." && parent != "" && parent != destinationPath && !coreio.Local.IsDir(parent) {
		return &fs.PathError{Op: "copyfile", Path: destinationPath, Err: fs.ErrNotExist}
	}
	return coreio.Local.WriteMode(destinationPath, string(file.content), perm)
}

// Example: _ = nodeTree.CopyTo(io.NewMemoryMedium(), "config", "backup/config")
func (node *Node) CopyTo(target coreio.Medium, sourcePath, destPath string) error {
	sourcePath = core.TrimPrefix(sourcePath, "/")
	info, err := node.Stat(sourcePath)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		file, ok := node.files[sourcePath]
		if !ok {
			return core.E("node.CopyTo", core.Concat("path not found: ", sourcePath), fs.ErrNotExist)
		}
		return target.Write(destPath, string(file.content))
	}

	prefix := sourcePath
	if prefix != "" && !core.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	for filePath, file := range node.files {
		if !core.HasPrefix(filePath, prefix) && filePath != sourcePath {
			continue
		}
		rel := core.TrimPrefix(filePath, prefix)
		dest := destPath
		if rel != "" {
			dest = core.Concat(destPath, "/", rel)
		}
		if err := target.Write(dest, string(file.content)); err != nil {
			return err
		}
	}
	return nil
}

func (node *Node) Open(name string) (fs.File, error) {
	name = core.TrimPrefix(name, "/")
	if dataFile, ok := node.files[name]; ok {
		return &dataFileReader{file: dataFile}, nil
	}
	prefix := name + "/"
	if name == "." || name == "" {
		prefix = ""
	}
	for filePath := range node.files {
		if core.HasPrefix(filePath, prefix) {
			return &dirFile{path: name, modTime: time.Now()}, nil
		}
	}
	return nil, core.E("node.Open", core.Concat("path not found: ", name), fs.ErrNotExist)
}

func (node *Node) Stat(name string) (fs.FileInfo, error) {
	name = core.TrimPrefix(name, "/")
	if dataFile, ok := node.files[name]; ok {
		return dataFile.Stat()
	}
	prefix := name + "/"
	if name == "." || name == "" {
		prefix = ""
	}
	for filePath := range node.files {
		if core.HasPrefix(filePath, prefix) {
			return &dirInfo{name: path.Base(name), modTime: time.Now()}, nil
		}
	}
	return nil, core.E("node.Stat", core.Concat("path not found: ", name), fs.ErrNotExist)
}

func (node *Node) ReadDir(name string) ([]fs.DirEntry, error) {
	name = core.TrimPrefix(name, "/")
	if name == "." {
		name = ""
	}

	if info, err := node.Stat(name); err == nil && !info.IsDir() {
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

	slices.SortFunc(entries, func(a, b fs.DirEntry) int {
		return cmp.Compare(a.Name(), b.Name())
	})

	return entries, nil
}

func (node *Node) Read(filePath string) (string, error) {
	filePath = core.TrimPrefix(filePath, "/")
	file, ok := node.files[filePath]
	if !ok {
		return "", core.E("node.Read", core.Concat("path not found: ", filePath), fs.ErrNotExist)
	}
	return string(file.content), nil
}

func (node *Node) Write(filePath, content string) error {
	node.AddData(filePath, []byte(content))
	return nil
}

func (node *Node) WriteMode(filePath, content string, mode fs.FileMode) error {
	return node.Write(filePath, content)
}

func (node *Node) FileGet(filePath string) (string, error) {
	return node.Read(filePath)
}

func (node *Node) FileSet(filePath, content string) error {
	return node.Write(filePath, content)
}

// Example: _ = nodeTree.EnsureDir("config")
func (node *Node) EnsureDir(_ string) error {
	return nil
}

func (node *Node) Exists(filePath string) bool {
	_, err := node.Stat(filePath)
	return err == nil
}

func (node *Node) IsFile(filePath string) bool {
	filePath = core.TrimPrefix(filePath, "/")
	_, ok := node.files[filePath]
	return ok
}

func (node *Node) IsDir(filePath string) bool {
	info, err := node.Stat(filePath)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func (node *Node) Delete(filePath string) error {
	filePath = core.TrimPrefix(filePath, "/")
	if _, ok := node.files[filePath]; ok {
		delete(node.files, filePath)
		return nil
	}
	return core.E("node.Delete", core.Concat("path not found: ", filePath), fs.ErrNotExist)
}

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
		return core.E("node.DeleteAll", core.Concat("path not found: ", filePath), fs.ErrNotExist)
	}
	return nil
}

func (node *Node) Rename(oldPath, newPath string) error {
	oldPath = core.TrimPrefix(oldPath, "/")
	newPath = core.TrimPrefix(newPath, "/")

	file, ok := node.files[oldPath]
	if !ok {
		return core.E("node.Rename", core.Concat("path not found: ", oldPath), fs.ErrNotExist)
	}

	file.name = newPath
	node.files[newPath] = file
	delete(node.files, oldPath)
	return nil
}

func (node *Node) List(filePath string) ([]fs.DirEntry, error) {
	filePath = core.TrimPrefix(filePath, "/")
	if filePath == "" || filePath == "." {
		return node.ReadDir(".")
	}
	return node.ReadDir(filePath)
}

func (node *Node) Create(filePath string) (goio.WriteCloser, error) {
	filePath = core.TrimPrefix(filePath, "/")
	return &nodeWriter{node: node, path: filePath}, nil
}

func (node *Node) Append(filePath string) (goio.WriteCloser, error) {
	filePath = core.TrimPrefix(filePath, "/")
	var existing []byte
	if file, ok := node.files[filePath]; ok {
		existing = make([]byte, len(file.content))
		copy(existing, file.content)
	}
	return &nodeWriter{node: node, path: filePath, buf: existing}, nil
}

func (node *Node) ReadStream(filePath string) (goio.ReadCloser, error) {
	file, err := node.Open(filePath)
	if err != nil {
		return nil, err
	}
	return goio.NopCloser(file), nil
}

func (node *Node) WriteStream(filePath string) (goio.WriteCloser, error) {
	return node.Create(filePath)
}

type nodeWriter struct {
	node *Node
	path string
	buf  []byte
}

func (writer *nodeWriter) Write(data []byte) (int, error) {
	writer.buf = append(writer.buf, data...)
	return len(data), nil
}

func (writer *nodeWriter) Close() error {
	writer.node.files[writer.path] = &dataFile{
		name:    writer.path,
		content: writer.buf,
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

func (file *dataFile) Read(_ []byte) (int, error) { return 0, goio.EOF }

func (file *dataFile) Close() error { return nil }

type dataFileInfo struct{ file *dataFile }

func (info *dataFileInfo) Name() string { return path.Base(info.file.name) }

func (info *dataFileInfo) Size() int64 { return int64(len(info.file.content)) }

func (info *dataFileInfo) Mode() fs.FileMode { return 0444 }

func (info *dataFileInfo) ModTime() time.Time { return info.file.modTime }

func (info *dataFileInfo) IsDir() bool { return false }

func (info *dataFileInfo) Sys() any { return nil }

type dataFileReader struct {
	file   *dataFile
	reader *bytes.Reader
}

func (reader *dataFileReader) Stat() (fs.FileInfo, error) { return reader.file.Stat() }

func (reader *dataFileReader) Read(buffer []byte) (int, error) {
	if reader.reader == nil {
		reader.reader = bytes.NewReader(reader.file.content)
	}
	return reader.reader.Read(buffer)
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
	return &dirInfo{name: path.Base(directory.path), modTime: directory.modTime}, nil
}

func (directory *dirFile) Read([]byte) (int, error) {
	return 0, core.E("node.dirFile.Read", core.Concat("cannot read directory: ", directory.path), &fs.PathError{Op: "read", Path: directory.path, Err: fs.ErrInvalid})
}

func (directory *dirFile) Close() error { return nil }

var _ fs.FS = (*Node)(nil)

var _ fs.StatFS = (*Node)(nil)
var _ fs.ReadDirFS = (*Node)(nil)

var _ goio.ReadCloser = goio.NopCloser(nil)

var _ goio.WriteCloser = (*nodeWriter)(nil)

var _ fs.File = (*dirFile)(nil)

var _ fs.File = (*dataFileReader)(nil)
