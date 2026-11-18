// Example: medium, _ := local.New("/srv/app")
// Example: _ = medium.Write("config/app.yaml", "port: 8080")
// Example: content, _ := medium.Read("config/app.yaml")
package local

import (
	"cmp"
	goio "io"
	"io/fs"
	"os" // AX-6-exception: os.Root is the Go security primitive that anchors every local Medium operation to one directory handle.
	"runtime"
	"slices"
	"sync"

	core "github.com/dappcore/go"
)

// Example: medium, _ := local.New("/srv/app")
// Example: _ = medium.Write("config/app.yaml", "port: 8080")
type Medium struct {
	filesystemRoot      string
	root                *os.Root
	closeOnce           sync.Once
	closeErr            error
	beforeRootOperation func()
}

var _ fs.FS = (*Medium)(nil)
var _ goio.Closer = (*Medium)(nil)

const (
	opLocalDelete    = "local.Delete"
	opLocalDeleteAll = "local.DeleteAll"
)

// Example: medium, _ := local.New("/srv/app")
// Example: _ = medium.Write("config/app.yaml", "port: 8080")
func New(root string) (
	*Medium,
	error,
) {
	if runtime.GOOS == "js" || runtime.GOOS == "plan9" {
		return nil, core.E(
			"local.New",
			core.Concat("os.Root cannot enforce rename-safe containment on ", runtime.GOOS),
			fs.ErrPermission,
		)
	}

	absoluteRoot, err := absolutePath(root)
	if err != nil {
		return nil, core.E("local.New", "resolve root", err)
	}
	if resolved := core.PathEvalSymlinks(absoluteRoot); resolved.OK {
		if value, ok := resolved.Value.(string); ok {
			absoluteRoot = value
		}
	}

	rootHandle, err := os.OpenRoot(absoluteRoot)
	if err != nil {
		return nil, core.E("local.New", "open root", err)
	}
	return &Medium{
		filesystemRoot: absoluteRoot,
		root:           rootHandle,
	}, nil
}

func dirSeparator() string {
	if separator := core.Env("CORE_PATH_SEPARATOR"); separator != "" {
		return separator
	}
	if separator := core.Env("DS"); separator != "" {
		return separator
	}
	return "/"
}

func normalisePath(path string) string {
	separator := dirSeparator()
	if separator == "/" {
		return core.Replace(path, "\\", separator)
	}
	return core.Replace(path, "/", separator)
}

func currentWorkingDir() string {
	if workingDirectory := core.Env("CORE_WORKING_DIRECTORY"); workingDirectory != "" {
		return workingDirectory
	}
	if workingDirectory := core.Env("DIR_CWD"); workingDirectory != "" {
		return workingDirectory
	}
	if result := core.Getwd(); result.OK {
		if workingDirectory, ok := result.Value.(string); ok {
			return workingDirectory
		}
	}
	return "."
}

func absolutePath(path string) (
	string,
	error,
) {
	path = normalisePath(path)
	if path == "" {
		path = "."
	}
	result := core.PathAbs(path)
	if !result.OK {
		if err, ok := result.Value.(error); ok {
			return "", err
		}
		return "", core.E("local.absolutePath", "unexpected absolute path result", nil)
	}
	absolute, ok := result.Value.(string)
	if !ok {
		return "", core.E("local.absolutePath", "unexpected absolute path type", nil)
	}
	return absolute, nil
}

// rootPath converts legacy absolute and traversal-heavy caller paths into a
// clean name relative to the already-open root. It does not inspect the host
// filesystem; containment is enforced when os.Root resolves the name.
func (medium *Medium) rootPath(path string) (
	string,
	error,
) {
	if medium == nil {
		return "", core.E("local.rootPath", "medium is nil", fs.ErrInvalid)
	}
	if core.Contains(path, "\x00") {
		return "", core.E("local.rootPath", "path contains NUL", fs.ErrInvalid)
	}

	path = core.PathToSlash(normalisePath(path))
	if medium.filesystemRoot == dirSeparator() && path != "" && !core.PathIsAbs(path) {
		path = core.PathToSlash(core.PathJoin(currentWorkingDir(), path))
	}

	clean := core.CleanPath("/"+core.TrimPrefix(path, "/"), "/")
	relative := core.TrimPrefix(clean, "/")
	if relative == "" {
		return ".", nil
	}
	return relative, nil
}

func canonicalPath(path string) string {
	if path == "" {
		return ""
	}
	absolute, err := absolutePath(path)
	if err != nil {
		return ""
	}
	if resolved := core.PathEvalSymlinks(absolute); resolved.OK {
		if value, ok := resolved.Value.(string); ok {
			return value
		}
	}
	return absolute
}

func isProtectedPath(fullPath string) bool {
	fullPath = canonicalPath(fullPath)
	protected := map[string]struct{}{
		canonicalPath(dirSeparator()): {},
	}
	for _, home := range []string{core.Env("HOME"), core.Env("DIR_HOME")} {
		if home == "" {
			continue
		}
		protected[canonicalPath(home)] = struct{}{}
	}
	_, ok := protected[fullPath]
	return ok
}

func (medium *Medium) fullPath(path string) string {
	if path == "." {
		return medium.filesystemRoot
	}
	return core.PathJoin(medium.filesystemRoot, path)
}

func (medium *Medium) prepare(path string) (
	*os.Root,
	string,
	error,
) {
	name, err := medium.rootPath(path)
	if err != nil {
		return nil, "", err
	}
	if medium.root == nil {
		return nil, "", core.E("local.prepare", "medium is closed or uninitialised", fs.ErrClosed)
	}
	medium.runBeforeRootOperation()
	return medium.root, name, nil
}

func ensureParent(root *os.Root, name string) error {
	parent := core.PathToSlash(core.PathDir(name))
	if parent == "" || parent == "." {
		return nil
	}
	return root.MkdirAll(parent, 0755)
}

func (medium *Medium) Read(path string) (
	string,
	error,
) {
	root, name, err := medium.prepare(path)
	if err != nil {
		return "", err
	}
	content, err := root.ReadFile(name)
	if err != nil {
		return "", core.E("local.Read", core.Concat("read failed: ", path), err)
	}
	return string(content), nil
}

func (medium *Medium) Write(path, content string) error { // legacy error contract
	return medium.WriteMode(path, content, 0644)
}

func (medium *Medium) WriteMode(path, content string, mode fs.FileMode) error { // legacy error contract
	root, name, err := medium.prepare(path)
	if err != nil {
		return err
	}
	if err := ensureParent(root, name); err != nil {
		return core.E("local.WriteMode", core.Concat("create parent failed: ", path), err)
	}
	if err := root.WriteFile(name, []byte(content), mode); err != nil {
		return core.E("local.WriteMode", core.Concat("write failed: ", path), err)
	}
	return nil
}

// Example: _ = medium.EnsureDir("config/app")
func (medium *Medium) EnsureDir(path string) error { // legacy error contract
	root, name, err := medium.prepare(path)
	if err != nil {
		return err
	}
	if err := root.MkdirAll(name, 0755); err != nil {
		return core.E("local.EnsureDir", core.Concat("ensure dir failed: ", path), err)
	}
	return nil
}

// Example: isDirectory := medium.IsDir("config")
func (medium *Medium) IsDir(path string) bool {
	if path == "" {
		return false
	}
	info, err := medium.Stat(path)
	return err == nil && info.IsDir()
}

// Example: isFile := medium.IsFile("config/app.yaml")
func (medium *Medium) IsFile(path string) bool {
	if path == "" {
		return false
	}
	info, err := medium.Stat(path)
	return err == nil && !info.IsDir()
}

// Example: exists := medium.Exists("config/app.yaml")
func (medium *Medium) Exists(path string) bool {
	_, err := medium.Stat(path)
	return err == nil
}

// Example: entries, _ := medium.List("config")
func (medium *Medium) List(path string) (
	[]fs.DirEntry,
	error,
) {
	root, name, err := medium.prepare(path)
	if err != nil {
		return nil, err
	}
	entries, err := fs.ReadDir(root.FS(), name)
	if err != nil {
		return nil, core.E("local.List", core.Concat("list failed: ", path), err)
	}
	slices.SortFunc(entries, func(a, b fs.DirEntry) int {
		return cmp.Compare(a.Name(), b.Name())
	})
	return entries, nil
}

// Example: info, _ := medium.Stat("config/app.yaml")
func (medium *Medium) Stat(path string) (
	fs.FileInfo,
	error,
) {
	root, name, err := medium.prepare(path)
	if err != nil {
		return nil, err
	}
	info, err := root.Stat(name)
	if err != nil {
		return nil, core.E("local.Stat", core.Concat("stat failed: ", path), err)
	}
	return info, nil
}

// Example: file, _ := medium.Open("config/app.yaml")
func (medium *Medium) Open(path string) (
	fs.File,
	error,
) {
	root, name, err := medium.prepare(path)
	if err != nil {
		return nil, err
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, core.E("local.Open", core.Concat("open failed: ", path), err)
	}
	return file, nil
}

// Example: writer, _ := medium.Create("logs/app.log")
func (medium *Medium) Create(path string) (
	goio.WriteCloser,
	error,
) {
	root, name, err := medium.prepare(path)
	if err != nil {
		return nil, err
	}
	if err := ensureParent(root, name); err != nil {
		return nil, core.E("local.Create", core.Concat("create parent failed: ", path), err)
	}
	writer, err := root.OpenFile(name, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return nil, core.E("local.Create", core.Concat("create failed: ", path), err)
	}
	return writer, nil
}

// Example: writer, _ := medium.Append("logs/app.log")
func (medium *Medium) Append(path string) (
	goio.WriteCloser,
	error,
) {
	root, name, err := medium.prepare(path)
	if err != nil {
		return nil, err
	}
	if err := ensureParent(root, name); err != nil {
		return nil, core.E("local.Append", core.Concat("create parent failed: ", path), err)
	}
	writer, err := root.OpenFile(name, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, core.E("local.Append", core.Concat("append failed: ", path), err)
	}
	return writer, nil
}

// Example: reader, _ := medium.ReadStream("logs/app.log")
func (medium *Medium) ReadStream(path string) (
	goio.ReadCloser,
	error,
) {
	file, err := medium.Open(path)
	if err != nil {
		return nil, err
	}
	reader, ok := file.(goio.ReadCloser)
	if !ok {
		_ = file.Close()
		return nil, core.E("local.ReadStream", "opened file is not closable", fs.ErrInvalid)
	}
	return reader, nil
}

// Example: writer, _ := medium.WriteStream("logs/app.log")
func (medium *Medium) WriteStream(path string) (
	goio.WriteCloser,
	error,
) {
	return medium.Create(path)
}

// Example: _ = medium.Delete("config/app.yaml")
func (medium *Medium) Delete(path string) error { // legacy error contract
	root, name, err := medium.prepare(path)
	if err != nil {
		return err
	}
	if name == "." {
		return core.E(opLocalDelete, "refusing to delete sandbox root", fs.ErrPermission)
	}
	if protected := medium.fullPath(name); isProtectedPath(protected) {
		return core.E(opLocalDelete, core.Concat("refusing to delete protected path: ", protected), fs.ErrPermission)
	}
	if err := root.Remove(name); err != nil {
		return core.E(opLocalDelete, core.Concat("delete failed: ", path), err)
	}
	return nil
}

// Example: _ = medium.DeleteAll("logs/archive")
func (medium *Medium) DeleteAll(path string) error { // legacy error contract
	root, name, err := medium.prepare(path)
	if err != nil {
		return err
	}
	if name == "." {
		return core.E(opLocalDeleteAll, "refusing to delete sandbox root", fs.ErrPermission)
	}
	if protected := medium.fullPath(name); isProtectedPath(protected) {
		return core.E(opLocalDeleteAll, core.Concat("refusing to delete protected path: ", protected), fs.ErrPermission)
	}
	if err := root.RemoveAll(name); err != nil {
		return core.E(opLocalDeleteAll, core.Concat("delete all failed: ", path), err)
	}
	return nil
}

// Example: _ = medium.Rename("drafts/todo.txt", "archive/todo.txt")
func (medium *Medium) Rename(oldPath, newPath string) error { // legacy error contract
	oldName, err := medium.rootPath(oldPath)
	if err != nil {
		return err
	}
	newName, err := medium.rootPath(newPath)
	if err != nil {
		return err
	}
	medium.runBeforeRootOperation()
	if err := ensureParent(medium.root, newName); err != nil {
		return core.E("local.Rename", core.Concat("create destination parent failed: ", newPath), err)
	}
	if err := medium.root.Rename(oldName, newName); err != nil {
		return core.E("local.Rename", core.Concat("rename failed: ", oldPath), err)
	}
	return nil
}

// Close releases the directory handle that enforces Medium containment.
func (medium *Medium) Close() error {
	if medium == nil {
		return core.E("local.Close", "medium is nil", fs.ErrInvalid)
	}
	medium.closeOnce.Do(func() {
		if medium.root != nil {
			medium.closeErr = medium.root.Close()
		}
	})
	return medium.closeErr
}

func (medium *Medium) runBeforeRootOperation() {
	if medium == nil || medium.beforeRootOperation == nil {
		return
	}
	hook := medium.beforeRootOperation
	medium.beforeRootOperation = nil
	hook()
}
