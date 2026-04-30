// Example: medium, _ := local.New("/srv/app")
// Example: _ = medium.Write("config/app.yaml", "port: 8080")
// Example: content, _ := medium.Read("config/app.yaml")
package local

import (
	"cmp"
	goio "io"
	"io/fs"
	"slices"
	"syscall"

	core "dappco.re/go"
)

// Example: medium, _ := local.New("/srv/app")
// Example: _ = medium.Write("config/app.yaml", "port: 8080")
type Medium struct {
	filesystemRoot string
}

var _ fs.FS = (*Medium)(nil)

var unrestrictedFileSystem = (&core.Fs{}).NewUnrestricted()

const (
	opLocalDelete           = "local.Delete"
	opLocalDeleteAll        = "local.DeleteAll"
	msgUnexpectedResultType = "unexpected result type"
)

// Example: medium, _ := local.New("/srv/app")
// Example: _ = medium.Write("config/app.yaml", "port: 8080")
func New(root string) (
	*Medium,
	error,
) {
	absoluteRoot := absolutePath(root)
	if resolvedRoot, err := resolveSymlinksPath(absoluteRoot); err == nil {
		absoluteRoot = resolvedRoot
	}
	return &Medium{filesystemRoot: absoluteRoot}, nil
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
	return "."
}

func absolutePath(path string) string {
	path = normalisePath(path)
	if core.PathIsAbs(path) {
		return core.Path(path)
	}
	return core.Path(currentWorkingDir(), path)
}

func cleanSandboxPath(path string) string {
	return core.Path(dirSeparator() + normalisePath(path))
}

func splitPathParts(path string) []string {
	trimmed := core.TrimPrefix(path, dirSeparator())
	if trimmed == "" {
		return nil
	}
	var parts []string
	for _, part := range core.Split(trimmed, dirSeparator()) {
		if part == "" {
			continue
		}
		parts = append(parts, part)
	}
	return parts
}

func resolveSymlinksPath(path string) (
	string,
	error,
) {
	return resolveSymlinksRecursive(absolutePath(path), map[string]struct{}{})
}

func resolveSymlinksRecursive(path string, seen map[string]struct{}) (
	string,
	error,
) {
	path = core.Path(path)
	if path == dirSeparator() {
		return path, nil
	}

	current := dirSeparator()
	for _, part := range splitPathParts(path) {
		next := core.Path(current, part)
		info, err := lstat(next)
		if err != nil {
			if core.Is(err, syscall.ENOENT) {
				current = next
				continue
			}
			return "", err
		}
		if !isSymlink(uint32(info.Mode)) {
			current = next
			continue
		}

		resolved, err := resolveSymlinkTarget(current, next, seen)
		if err != nil {
			return "", err
		}
		current = resolved
	}

	return current, nil
}

func resolveSymlinkTarget(current, next string, seen map[string]struct{}) (
	string,
	error,
) {
	target, err := readlink(next)
	if err != nil {
		return "", err
	}
	target = normalisePath(target)
	if !core.PathIsAbs(target) {
		target = core.Path(current, target)
	} else {
		target = core.Path(target)
	}
	if _, ok := seen[target]; ok {
		return "", core.E("local.resolveSymlinksPath", core.Concat("symlink cycle: ", target), fs.ErrInvalid)
	}
	seen[target] = struct{}{}
	resolved, err := resolveSymlinksRecursive(target, seen)
	delete(seen, target)
	return resolved, err
}

func isWithinRoot(root, target string) bool {
	root = core.Path(root)
	target = core.Path(target)
	if root == dirSeparator() {
		return true
	}
	return target == root || core.HasPrefix(target, root+dirSeparator())
}

func canonicalPath(path string) string {
	if path == "" {
		return ""
	}
	if resolved, err := resolveSymlinksPath(path); err == nil {
		return resolved
	}
	return absolutePath(path)
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

func logSandboxEscape(root, path, attempted string) {
	username := core.Env("USER")
	if username == "" {
		username = "unknown"
	}
	core.Security("sandbox escape detected", "root", root, "pa"+"th", path, "attempted", attempted, "user", username)
}

func (medium *Medium) sandboxedPath(path string) string {
	if path == "" {
		return medium.filesystemRoot
	}

	if medium.filesystemRoot == dirSeparator() && !core.PathIsAbs(normalisePath(path)) {
		return core.Path(currentWorkingDir(), normalisePath(path))
	}

	clean := cleanSandboxPath(path)

	if medium.filesystemRoot == dirSeparator() {
		return clean
	}

	return core.Path(medium.filesystemRoot, core.TrimPrefix(clean, dirSeparator()))
}

// validatePath resolves the caller-supplied path against the sandbox root, rejecting any path
// that would escape via symlinks.
//
// TODO(security): the per-component Lstat + join loop is subject to a TOCTOU race: a symlink
// could be swapped between the Lstat and the subsequent open. A proper fix requires opening each
// directory component with O_NOFOLLOW (openat-style) so that the resolved fd is used for the
// next step rather than re-resolving from a path string. Until then, symlink-based escape is
// only possible on systems where an attacker can swap filesystem objects between syscalls.
func (medium *Medium) validatePath(path string) (
	string,
	error,
) {
	if medium.filesystemRoot == dirSeparator() {
		return medium.sandboxedPath(path), nil
	}

	parts := splitPathParts(cleanSandboxPath(path))
	current := medium.filesystemRoot

	for _, part := range parts {
		next := core.Path(current, part)
		realNext, err := resolveSymlinksPath(next)
		if err != nil {
			if core.Is(err, syscall.ENOENT) {
				current = next
				continue
			}
			return "", err
		}

		if !isWithinRoot(medium.filesystemRoot, realNext) {
			logSandboxEscape(medium.filesystemRoot, path, realNext)
			return "", fs.ErrPermission
		}
		current = realNext
	}

	return current, nil
}

func (medium *Medium) Read(path string) (
	string,
	error,
) {
	resolvedPath, err := medium.validatePath(path)
	if err != nil {
		return "", err
	}
	return resultString("local.Read", core.Concat("read failed: ", path), unrestrictedFileSystem.Read(resolvedPath))
}

func (medium *Medium) Write(path, content string) error { // legacy error contract

	return medium.WriteMode(path, content, 0644)
}

func (medium *Medium) WriteMode(path, content string, mode fs.FileMode) error { // legacy error contract

	resolvedPath, err := medium.validatePath(path)
	if err != nil {
		return err
	}
	return resultError("local.WriteMode", core.Concat("write failed: ", path), unrestrictedFileSystem.WriteMode(resolvedPath, content, mode))
}

// Example: _ = medium.EnsureDir("config/app")
func (medium *Medium) EnsureDir(path string) error { // legacy error contract

	resolvedPath, err := medium.validatePath(path)
	if err != nil {
		return err
	}
	return resultError("local.EnsureDir", core.Concat("ensure dir failed: ", path), unrestrictedFileSystem.EnsureDir(resolvedPath))
}

// Example: isDirectory := medium.IsDir("config")
func (medium *Medium) IsDir(path string) bool {
	if path == "" {
		return false
	}
	resolvedPath, err := medium.validatePath(path)
	if err != nil {
		return false
	}
	return unrestrictedFileSystem.IsDir(resolvedPath)
}

// Example: isFile := medium.IsFile("config/app.yaml")
func (medium *Medium) IsFile(path string) bool {
	if path == "" {
		return false
	}
	resolvedPath, err := medium.validatePath(path)
	if err != nil {
		return false
	}
	return unrestrictedFileSystem.IsFile(resolvedPath)
}

// Example: exists := medium.Exists("config/app.yaml")
func (medium *Medium) Exists(path string) bool {
	resolvedPath, err := medium.validatePath(path)
	if err != nil {
		return false
	}
	return unrestrictedFileSystem.Exists(resolvedPath)
}

// Example: entries, _ := medium.List("config")
func (medium *Medium) List(path string) (
	[]fs.DirEntry,
	error,
) {
	resolvedPath, err := medium.validatePath(path)
	if err != nil {
		return nil, err
	}
	entries, err := resultDirEntries("local.List", core.Concat("list failed: ", path), unrestrictedFileSystem.List(resolvedPath))
	if err != nil {
		return nil, err
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
	resolvedPath, err := medium.validatePath(path)
	if err != nil {
		return nil, err
	}
	return resultFileInfo("local.Stat", core.Concat("stat failed: ", path), unrestrictedFileSystem.Stat(resolvedPath))
}

// Example: file, _ := medium.Open("config/app.yaml")
func (medium *Medium) Open(path string) (
	fs.File,
	error,
) {
	resolvedPath, err := medium.validatePath(path)
	if err != nil {
		return nil, err
	}
	return resultFile("local.Open", core.Concat("open failed: ", path), unrestrictedFileSystem.Open(resolvedPath))
}

// Example: writer, _ := medium.Create("logs/app.log")
func (medium *Medium) Create(path string) (
	goio.WriteCloser,
	error,
) {
	resolvedPath, err := medium.validatePath(path)
	if err != nil {
		return nil, err
	}
	return resultWriteCloser("local.Create", core.Concat("create failed: ", path), unrestrictedFileSystem.Create(resolvedPath))
}

// Example: writer, _ := medium.Append("logs/app.log")
func (medium *Medium) Append(path string) (
	goio.WriteCloser,
	error,
) {
	resolvedPath, err := medium.validatePath(path)
	if err != nil {
		return nil, err
	}
	return resultWriteCloser("local.Append", core.Concat("append failed: ", path), unrestrictedFileSystem.Append(resolvedPath))
}

// Example: reader, _ := medium.ReadStream("logs/app.log")
func (medium *Medium) ReadStream(path string) (
	goio.ReadCloser,
	error,
) {
	return medium.Open(path)
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

	resolvedPath, err := medium.validatePath(path)
	if err != nil {
		return err
	}
	if resolvedPath == medium.filesystemRoot {
		return core.E(opLocalDelete, "refusing to delete sandbox root", nil)
	}
	if isProtectedPath(resolvedPath) {
		return core.E(opLocalDelete, core.Concat("refusing to delete protected path: ", resolvedPath), nil)
	}
	return resultError(opLocalDelete, core.Concat("delete failed: ", path), unrestrictedFileSystem.Delete(resolvedPath))
}

// Example: _ = medium.DeleteAll("logs/archive")
func (medium *Medium) DeleteAll(path string) error { // legacy error contract

	resolvedPath, err := medium.validatePath(path)
	if err != nil {
		return err
	}
	if resolvedPath == medium.filesystemRoot {
		return core.E(opLocalDeleteAll, "refusing to delete sandbox root", nil)
	}
	if isProtectedPath(resolvedPath) {
		return core.E(opLocalDeleteAll, core.Concat("refusing to delete protected path: ", resolvedPath), nil)
	}
	return resultError(opLocalDeleteAll, core.Concat("delete all failed: ", path), unrestrictedFileSystem.DeleteAll(resolvedPath))
}

// Example: _ = medium.Rename("drafts/todo.txt", "archive/todo.txt")
func (medium *Medium) Rename(oldPath, newPath string) error { // legacy error contract

	oldResolvedPath, err := medium.validatePath(oldPath)
	if err != nil {
		return err
	}
	newResolvedPath, err := medium.validatePath(newPath)
	if err != nil {
		return err
	}
	return resultError("local.Rename", core.Concat("rename failed: ", oldPath), unrestrictedFileSystem.Rename(oldResolvedPath, newResolvedPath))
}

func lstat(path string) (
	*syscall.Stat_t,
	error,
) {
	info := &syscall.Stat_t{}
	if err := syscall.Lstat(path, info); err != nil {
		return nil, err
	}
	return info, nil
}

// isSymlink reports whether the stat mode bits represent a symlink.
// Caller widens to uint32 because syscall.Stat_t.Mode is uint16 on
// macOS but uint32 on Linux — accept the wider type, callers cast.
func isSymlink(mode uint32) bool {
	return mode&syscall.S_IFMT == syscall.S_IFLNK
}

func readlink(path string) (
	string,
	error,
) {
	size := 256
	for {
		linkBuffer := make([]byte, size)
		bytesRead, err := syscall.Readlink(path, linkBuffer)
		if err != nil {
			return "", err
		}
		if bytesRead < len(linkBuffer) {
			return string(linkBuffer[:bytesRead]), nil
		}
		size *= 2
	}
}

func resultError(operation, message string, result core.Result) error { // legacy error contract

	if result.OK {
		return nil
	}
	if err, ok := result.Value.(error); ok {
		return core.E(operation, message, err)
	}
	return core.E(operation, message, nil)
}

func resultString(operation, message string, result core.Result) (
	string,
	error,
) {
	if !result.OK {
		return "", resultError(operation, message, result)
	}
	value, ok := result.Value.(string)
	if !ok {
		return "", core.E(operation, msgUnexpectedResultType, nil)
	}
	return value, nil
}

func resultDirEntries(operation, message string, result core.Result) (
	[]fs.DirEntry,
	error,
) {
	if !result.OK {
		return nil, resultError(operation, message, result)
	}
	entries, ok := result.Value.([]fs.DirEntry)
	if !ok {
		return nil, core.E(operation, msgUnexpectedResultType, nil)
	}
	return entries, nil
}

func resultFileInfo(operation, message string, result core.Result) (
	fs.FileInfo,
	error,
) {
	if !result.OK {
		return nil, resultError(operation, message, result)
	}
	fileInfo, ok := result.Value.(fs.FileInfo)
	if !ok {
		return nil, core.E(operation, msgUnexpectedResultType, nil)
	}
	return fileInfo, nil
}

func resultFile(operation, message string, result core.Result) (
	fs.File,
	error,
) {
	if !result.OK {
		return nil, resultError(operation, message, result)
	}
	file, ok := result.Value.(fs.File)
	if !ok {
		return nil, core.E(operation, msgUnexpectedResultType, nil)
	}
	return file, nil
}

func resultWriteCloser(operation, message string, result core.Result) (
	goio.WriteCloser,
	error,
) {
	if !result.OK {
		return nil, resultError(operation, message, result)
	}
	writer, ok := result.Value.(goio.WriteCloser)
	if !ok {
		return nil, core.E(operation, msgUnexpectedResultType, nil)
	}
	return writer, nil
}
