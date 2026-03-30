// Package node provides an in-memory filesystem implementation of io.Medium
// ported from Borg's DataNode. It stores files in memory with implicit
// directory structure and supports tar serialisation.
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

// Node is an in-memory filesystem that implements coreio.Node (and therefore
// coreio.Medium). Directories are implicit -- they exist whenever a file path
// contains a "/".
type Node struct {
	files map[string]*dataFile
}

// compile-time interface checks
var _ coreio.Medium = (*Node)(nil)
var _ fs.ReadFileFS = (*Node)(nil)

// New creates a new, empty Node.
//
// Example usage:
//
//	nodeTree := node.New()
//	nodeTree.AddData("config/app.yaml", []byte("port: 8080"))
func New() *Node {
	return &Node{files: make(map[string]*dataFile)}
}

// ---------- Node-specific methods ----------

// AddData stages content in the in-memory filesystem.
//
//	result := n.AddData(...)
func (n *Node) AddData(name string, content []byte) {
	name = core.TrimPrefix(name, "/")
	if name == "" {
		return
	}
	// Directories are implicit, so we don't store them.
	if core.HasSuffix(name, "/") {
		return
	}
	n.files[name] = &dataFile{
		name:    name,
		content: content,
		modTime: time.Now(),
	}
}

// ToTar serialises the entire in-memory tree to a tar archive.
//
//	result := n.ToTar(...)
func (n *Node) ToTar() ([]byte, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	for _, file := range n.files {
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

// FromTar creates a new Node from a tar archive.
//
//	result := node.FromTar(...)
func FromTar(data []byte) (*Node, error) {
	n := New()
	if err := n.LoadTar(data); err != nil {
		return nil, err
	}
	return n, nil
}

// LoadTar replaces the in-memory tree with the contents of a tar archive.
//
//	result := n.LoadTar(...)
func (n *Node) LoadTar(data []byte) error {
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

	n.files = newFiles
	return nil
}

// WalkNode walks the in-memory tree, calling fn for each entry.
//
//	result := n.WalkNode(...)
func (n *Node) WalkNode(root string, fn fs.WalkDirFunc) error {
	return fs.WalkDir(n, root, fn)
}

// WalkOptions configures the behaviour of Walk.
type WalkOptions struct {
	// MaxDepth limits how many directory levels to descend. 0 means unlimited.
	MaxDepth int
	// Filter, if set, is called for each entry. Return true to include the
	// entry (and descend into it if it is a directory).
	Filter func(path string, d fs.DirEntry) bool
	// SkipErrors suppresses errors (e.g. nonexistent root) instead of
	// propagating them through the callback.
	SkipErrors bool
}

// Walk walks the in-memory tree with optional WalkOptions.
//
//	result := n.Walk(...)
func (n *Node) Walk(root string, fn fs.WalkDirFunc, opts ...WalkOptions) error {
	var opt WalkOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	if opt.SkipErrors {
		// If root doesn't exist, silently return nil.
		if _, err := n.Stat(root); err != nil {
			return nil
		}
	}

	return fs.WalkDir(n, root, func(p string, d fs.DirEntry, err error) error {
		if opt.Filter != nil && err == nil {
			if !opt.Filter(p, d) {
				if d != nil && d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
		}

		// Call the user's function first so the entry is visited.
		result := fn(p, d, err)

		// After visiting a directory at MaxDepth, prevent descending further.
		if result == nil && opt.MaxDepth > 0 && d != nil && d.IsDir() && p != root {
			rel := core.TrimPrefix(p, root)
			rel = core.TrimPrefix(rel, "/")
			depth := len(core.Split(rel, "/"))
			if depth >= opt.MaxDepth {
				return fs.SkipDir
			}
		}

		return result
	})
}

// ReadFile returns the content of the named file as a byte slice.
// Implements fs.ReadFileFS.
//
//	result := n.ReadFile(...)
func (n *Node) ReadFile(name string) ([]byte, error) {
	name = core.TrimPrefix(name, "/")
	f, ok := n.files[name]
	if !ok {
		return nil, core.E("node.ReadFile", core.Concat("path not found: ", name), fs.ErrNotExist)
	}
	// Return a copy to prevent callers from mutating internal state.
	result := make([]byte, len(f.content))
	copy(result, f.content)
	return result, nil
}

// CopyFile copies a file from the in-memory tree to the local filesystem.
//
//	result := n.CopyFile(...)
func (n *Node) CopyFile(src, dst string, perm fs.FileMode) error {
	src = core.TrimPrefix(src, "/")
	f, ok := n.files[src]
	if !ok {
		// Check if it's a directory — can't copy directories this way.
		info, err := n.Stat(src)
		if err != nil {
			return core.E("node.CopyFile", core.Concat("source not found: ", src), fs.ErrNotExist)
		}
		if info.IsDir() {
			return core.E("node.CopyFile", core.Concat("source is a directory: ", src), fs.ErrInvalid)
		}
		return core.E("node.CopyFile", core.Concat("source not found: ", src), fs.ErrNotExist)
	}
	parent := core.PathDir(dst)
	if parent != "." && parent != "" && parent != dst && !coreio.Local.IsDir(parent) {
		return &fs.PathError{Op: "copyfile", Path: dst, Err: fs.ErrNotExist}
	}
	return coreio.Local.WriteMode(dst, string(f.content), perm)
}

// CopyTo copies a file (or directory tree) from the node to any Medium.
//
// Example usage:
//
//	dst := io.NewMockMedium()
//	_ = n.CopyTo(dst, "config", "backup/config")
func (n *Node) CopyTo(target coreio.Medium, sourcePath, destPath string) error {
	sourcePath = core.TrimPrefix(sourcePath, "/")
	info, err := n.Stat(sourcePath)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		// Single file copy
		f, ok := n.files[sourcePath]
		if !ok {
			return core.E("node.CopyTo", core.Concat("path not found: ", sourcePath), fs.ErrNotExist)
		}
		return target.Write(destPath, string(f.content))
	}

	// Directory: walk and copy all files underneath
	prefix := sourcePath
	if prefix != "" && !core.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	for p, f := range n.files {
		if !core.HasPrefix(p, prefix) && p != sourcePath {
			continue
		}
		rel := core.TrimPrefix(p, prefix)
		dest := destPath
		if rel != "" {
			dest = core.Concat(destPath, "/", rel)
		}
		if err := target.Write(dest, string(f.content)); err != nil {
			return err
		}
	}
	return nil
}

// ---------- Medium interface: fs.FS methods ----------

// Open opens a file from the Node. Implements fs.FS.
//
//	result := n.Open(...)
func (n *Node) Open(name string) (fs.File, error) {
	name = core.TrimPrefix(name, "/")
	if file, ok := n.files[name]; ok {
		return &dataFileReader{file: file}, nil
	}
	// Check if it's a directory
	prefix := name + "/"
	if name == "." || name == "" {
		prefix = ""
	}
	for p := range n.files {
		if core.HasPrefix(p, prefix) {
			return &dirFile{path: name, modTime: time.Now()}, nil
		}
	}
	return nil, core.E("node.Open", core.Concat("path not found: ", name), fs.ErrNotExist)
}

// Stat returns file information for the given path.
//
//	result := n.Stat(...)
func (n *Node) Stat(name string) (fs.FileInfo, error) {
	name = core.TrimPrefix(name, "/")
	if file, ok := n.files[name]; ok {
		return file.Stat()
	}
	// Check if it's a directory
	prefix := name + "/"
	if name == "." || name == "" {
		prefix = ""
	}
	for p := range n.files {
		if core.HasPrefix(p, prefix) {
			return &dirInfo{name: path.Base(name), modTime: time.Now()}, nil
		}
	}
	return nil, core.E("node.Stat", core.Concat("path not found: ", name), fs.ErrNotExist)
}

// ReadDir reads and returns all directory entries for the named directory.
//
//	result := n.ReadDir(...)
func (n *Node) ReadDir(name string) ([]fs.DirEntry, error) {
	name = core.TrimPrefix(name, "/")
	if name == "." {
		name = ""
	}

	// Disallow reading a file as a directory.
	if info, err := n.Stat(name); err == nil && !info.IsDir() {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrInvalid}
	}

	entries := []fs.DirEntry{}
	seen := make(map[string]bool)

	prefix := ""
	if name != "" {
		prefix = name + "/"
	}

	for p := range n.files {
		if !core.HasPrefix(p, prefix) {
			continue
		}

		relPath := core.TrimPrefix(p, prefix)
		firstComponent := core.SplitN(relPath, "/", 2)[0]

		if seen[firstComponent] {
			continue
		}
		seen[firstComponent] = true

		if core.Contains(relPath, "/") {
			dir := &dirInfo{name: firstComponent, modTime: time.Now()}
			entries = append(entries, fs.FileInfoToDirEntry(dir))
		} else {
			file := n.files[p]
			info, _ := file.Stat()
			entries = append(entries, fs.FileInfoToDirEntry(info))
		}
	}

	slices.SortFunc(entries, func(a, b fs.DirEntry) int {
		return cmp.Compare(a.Name(), b.Name())
	})

	return entries, nil
}

// ---------- Medium interface: read/write ----------

// Read retrieves the content of a file as a string.
//
//	result := n.Read(...)
func (n *Node) Read(p string) (string, error) {
	p = core.TrimPrefix(p, "/")
	f, ok := n.files[p]
	if !ok {
		return "", core.E("node.Read", core.Concat("path not found: ", p), fs.ErrNotExist)
	}
	return string(f.content), nil
}

// Write saves the given content to a file, overwriting it if it exists.
//
//	result := n.Write(...)
func (n *Node) Write(p, content string) error {
	n.AddData(p, []byte(content))
	return nil
}

// WriteMode saves content with explicit permissions (no-op for in-memory node).
//
//	result := n.WriteMode(...)
func (n *Node) WriteMode(p, content string, mode fs.FileMode) error {
	return n.Write(p, content)
}

// FileGet is an alias for Read.
//
//	result := n.FileGet(...)
func (n *Node) FileGet(p string) (string, error) {
	return n.Read(p)
}

// FileSet is an alias for Write.
//
//	result := n.FileSet(...)
func (n *Node) FileSet(p, content string) error {
	return n.Write(p, content)
}

// EnsureDir is a no-op because directories are implicit in Node.
//
//	result := n.EnsureDir(...)
func (n *Node) EnsureDir(_ string) error {
	return nil
}

// ---------- Medium interface: existence checks ----------

// Exists checks if a path exists (file or directory).
//
//	result := n.Exists(...)
func (n *Node) Exists(p string) bool {
	_, err := n.Stat(p)
	return err == nil
}

// IsFile checks if a path exists and is a regular file.
//
//	result := n.IsFile(...)
func (n *Node) IsFile(p string) bool {
	p = core.TrimPrefix(p, "/")
	_, ok := n.files[p]
	return ok
}

// IsDir checks if a path exists and is a directory.
//
//	result := n.IsDir(...)
func (n *Node) IsDir(p string) bool {
	info, err := n.Stat(p)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// ---------- Medium interface: mutations ----------

// Delete removes a single file.
//
//	result := n.Delete(...)
func (n *Node) Delete(p string) error {
	p = core.TrimPrefix(p, "/")
	if _, ok := n.files[p]; ok {
		delete(n.files, p)
		return nil
	}
	return core.E("node.Delete", core.Concat("path not found: ", p), fs.ErrNotExist)
}

// DeleteAll removes a file or directory and all children.
//
//	result := n.DeleteAll(...)
func (n *Node) DeleteAll(p string) error {
	p = core.TrimPrefix(p, "/")

	found := false
	if _, ok := n.files[p]; ok {
		delete(n.files, p)
		found = true
	}

	prefix := p + "/"
	for k := range n.files {
		if core.HasPrefix(k, prefix) {
			delete(n.files, k)
			found = true
		}
	}

	if !found {
		return core.E("node.DeleteAll", core.Concat("path not found: ", p), fs.ErrNotExist)
	}
	return nil
}

// Rename moves a file from oldPath to newPath.
//
//	result := n.Rename(...)
func (n *Node) Rename(oldPath, newPath string) error {
	oldPath = core.TrimPrefix(oldPath, "/")
	newPath = core.TrimPrefix(newPath, "/")

	f, ok := n.files[oldPath]
	if !ok {
		return core.E("node.Rename", core.Concat("path not found: ", oldPath), fs.ErrNotExist)
	}

	f.name = newPath
	n.files[newPath] = f
	delete(n.files, oldPath)
	return nil
}

// List returns directory entries for the given path.
//
//	result := n.List(...)
func (n *Node) List(p string) ([]fs.DirEntry, error) {
	p = core.TrimPrefix(p, "/")
	if p == "" || p == "." {
		return n.ReadDir(".")
	}
	return n.ReadDir(p)
}

// ---------- Medium interface: streams ----------

// Create creates or truncates the named file, returning a WriteCloser.
// Content is committed to the Node on Close.
//
//	result := n.Create(...)
func (n *Node) Create(p string) (goio.WriteCloser, error) {
	p = core.TrimPrefix(p, "/")
	return &nodeWriter{node: n, path: p}, nil
}

// Append opens the named file for appending, creating it if needed.
// Content is committed to the Node on Close.
//
//	result := n.Append(...)
func (n *Node) Append(p string) (goio.WriteCloser, error) {
	p = core.TrimPrefix(p, "/")
	var existing []byte
	if f, ok := n.files[p]; ok {
		existing = make([]byte, len(f.content))
		copy(existing, f.content)
	}
	return &nodeWriter{node: n, path: p, buf: existing}, nil
}

// ReadStream returns a ReadCloser for the file content.
//
//	result := n.ReadStream(...)
func (n *Node) ReadStream(p string) (goio.ReadCloser, error) {
	f, err := n.Open(p)
	if err != nil {
		return nil, err
	}
	return goio.NopCloser(f), nil
}

// WriteStream returns a WriteCloser for the file content.
//
//	result := n.WriteStream(...)
func (n *Node) WriteStream(p string) (goio.WriteCloser, error) {
	return n.Create(p)
}

// ---------- Internal types ----------

// nodeWriter buffers writes and commits them to the Node on Close.
type nodeWriter struct {
	node *Node
	path string
	buf  []byte
}

// Write documents the Write operation.
//
//	result := w.Write(...)
func (w *nodeWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	return len(p), nil
}

// Close documents the Close operation.
//
//	result := w.Close(...)
func (w *nodeWriter) Close() error {
	w.node.files[w.path] = &dataFile{
		name:    w.path,
		content: w.buf,
		modTime: time.Now(),
	}
	return nil
}

// dataFile represents a file in the Node.
type dataFile struct {
	name    string
	content []byte
	modTime time.Time
}

// Stat documents the Stat operation.
//
//	result := d.Stat(...)
func (d *dataFile) Stat() (fs.FileInfo, error) { return &dataFileInfo{file: d}, nil }

// Read documents the Read operation.
//
//	result := d.Read(...)
func (d *dataFile) Read(_ []byte) (int, error) { return 0, goio.EOF }

// Close documents the Close operation.
//
//	result := d.Close(...)
func (d *dataFile) Close() error { return nil }

// dataFileInfo implements fs.FileInfo for a dataFile.
type dataFileInfo struct{ file *dataFile }

// Name documents the Name operation.
//
//	result := d.Name(...)
func (d *dataFileInfo) Name() string { return path.Base(d.file.name) }

// Size documents the Size operation.
//
//	result := d.Size(...)
func (d *dataFileInfo) Size() int64 { return int64(len(d.file.content)) }

// Mode documents the Mode operation.
//
//	result := d.Mode(...)
func (d *dataFileInfo) Mode() fs.FileMode { return 0444 }

// ModTime documents the ModTime operation.
//
//	result := d.ModTime(...)
func (d *dataFileInfo) ModTime() time.Time { return d.file.modTime }

// IsDir documents the IsDir operation.
//
//	result := d.IsDir(...)
func (d *dataFileInfo) IsDir() bool { return false }

// Sys documents the Sys operation.
//
//	result := d.Sys(...)
func (d *dataFileInfo) Sys() any { return nil }

// dataFileReader implements fs.File for reading a dataFile.
type dataFileReader struct {
	file   *dataFile
	reader *bytes.Reader
}

// Stat documents the Stat operation.
//
//	result := d.Stat(...)
func (d *dataFileReader) Stat() (fs.FileInfo, error) { return d.file.Stat() }

// Read documents the Read operation.
//
//	result := d.Read(...)
func (d *dataFileReader) Read(p []byte) (int, error) {
	if d.reader == nil {
		d.reader = bytes.NewReader(d.file.content)
	}
	return d.reader.Read(p)
}

// Close documents the Close operation.
//
//	result := d.Close(...)
func (d *dataFileReader) Close() error { return nil }

// dirInfo implements fs.FileInfo for an implicit directory.
type dirInfo struct {
	name    string
	modTime time.Time
}

// Name documents the Name operation.
//
//	result := d.Name(...)
func (d *dirInfo) Name() string { return d.name }

// Size documents the Size operation.
//
//	result := d.Size(...)
func (d *dirInfo) Size() int64 { return 0 }

// Mode documents the Mode operation.
//
//	result := d.Mode(...)
func (d *dirInfo) Mode() fs.FileMode { return fs.ModeDir | 0555 }

// ModTime documents the ModTime operation.
//
//	result := d.ModTime(...)
func (d *dirInfo) ModTime() time.Time { return d.modTime }

// IsDir documents the IsDir operation.
//
//	result := d.IsDir(...)
func (d *dirInfo) IsDir() bool { return true }

// Sys documents the Sys operation.
//
//	result := d.Sys(...)
func (d *dirInfo) Sys() any { return nil }

// dirFile implements fs.File for a directory.
type dirFile struct {
	path    string
	modTime time.Time
}

// Stat documents the Stat operation.
//
//	result := d.Stat(...)
func (d *dirFile) Stat() (fs.FileInfo, error) {
	return &dirInfo{name: path.Base(d.path), modTime: d.modTime}, nil
}

// Read documents the Read operation.
//
//	result := d.Read(...)
func (d *dirFile) Read([]byte) (int, error) {
	return 0, core.E("node.dirFile.Read", core.Concat("cannot read directory: ", d.path), &fs.PathError{Op: "read", Path: d.path, Err: fs.ErrInvalid})
}

// Close documents the Close operation.
//
//	result := d.Close(...)
func (d *dirFile) Close() error { return nil }

// Ensure Node implements fs.FS so WalkDir works.
var _ fs.FS = (*Node)(nil)

// Ensure Node also satisfies fs.StatFS and fs.ReadDirFS for WalkDir.
var _ fs.StatFS = (*Node)(nil)
var _ fs.ReadDirFS = (*Node)(nil)

// Unexported helper: ensure ReadStream result also satisfies fs.File
// (for cases where callers do a type assertion).
var _ goio.ReadCloser = goio.NopCloser(nil)

// Ensure nodeWriter satisfies goio.WriteCloser.
var _ goio.WriteCloser = (*nodeWriter)(nil)

// Ensure dirFile satisfies fs.File.
var _ fs.File = (*dirFile)(nil)

// Ensure dataFileReader satisfies fs.File.
var _ fs.File = (*dataFileReader)(nil)

// ReadDirFile is not needed since fs.WalkDir works via ReadDirFS on the FS itself,
// but we need the Node to satisfy fs.ReadDirFS.

// ensure all internal compile-time checks are grouped above
// no further type assertions needed
