// Package local provides a local filesystem implementation of the io.Medium interface.
package local

import (
	goio "io"
	"io/fs"
	"syscall"

	core "dappco.re/go/core"
)

// Medium is a local filesystem storage backend.
type Medium struct {
	filesystemRoot string
}

var unrestrictedFileSystem = (&core.Fs{}).NewUnrestricted()

// New creates a new local Medium rooted at the given directory.
// Pass "/" for full filesystem access, or a specific path to sandbox.
//
// Example usage:
//
//	medium, _ := local.New("/srv/app")
//	_ = medium.Write("config/app.yaml", "port: 8080")
func New(root string) (*Medium, error) {
	absoluteRoot := absolutePath(root)
	// Resolve symlinks so sandbox checks compare like-for-like.
	// On macOS, /var is a symlink to /private/var — without this,
	// resolving child paths resolves to /private/var/... while
	// root stays /var/..., causing false sandbox escape detections.
	if resolvedRoot, err := resolveSymlinksPath(absoluteRoot); err == nil {
		absoluteRoot = resolvedRoot
	}
	return &Medium{filesystemRoot: absoluteRoot}, nil
}

func dirSeparator() string {
	if sep := core.Env("DS"); sep != "" {
		return sep
	}
	return "/"
}

func normalisePath(path string) string {
	sep := dirSeparator()
	if sep == "/" {
		return core.Replace(path, "\\", sep)
	}
	return core.Replace(path, "/", sep)
}

func currentWorkingDir() string {
	if cwd := core.Env("DIR_CWD"); cwd != "" {
		return cwd
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

func resolveSymlinksPath(path string) (string, error) {
	return resolveSymlinksRecursive(absolutePath(path), map[string]struct{}{})
}

func resolveSymlinksRecursive(path string, seen map[string]struct{}) (string, error) {
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
		if !isSymlink(info.Mode) {
			current = next
			continue
		}

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
		if err != nil {
			return "", err
		}
		current = resolved
	}

	return current, nil
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
	core.Security("sandbox escape detected", "root", root, "path", path, "attempted", attempted, "user", username)
}

// sandboxedPath sanitises and returns the full filesystem path.
// Absolute paths are sandboxed under root (unless root is "/").
func (m *Medium) sandboxedPath(path string) string {
	if path == "" {
		return m.filesystemRoot
	}

	// If the path is relative and the medium is rooted at "/",
	// treat it as relative to the current working directory.
	// This makes io.Local behave more like the standard 'os' package.
	if m.filesystemRoot == dirSeparator() && !core.PathIsAbs(normalisePath(path)) {
		return core.Path(currentWorkingDir(), normalisePath(path))
	}

	// Use a cleaned absolute path to resolve all .. and . internally
	// before joining with the root. This is a standard way to sandbox paths.
	clean := cleanSandboxPath(path)

	// If root is "/", allow absolute paths through
	if m.filesystemRoot == dirSeparator() {
		return clean
	}

	// Join cleaned relative path with root
	return core.Path(m.filesystemRoot, core.TrimPrefix(clean, dirSeparator()))
}

// validatePath ensures the path is within the sandbox, following symlinks if they exist.
func (m *Medium) validatePath(path string) (string, error) {
	if m.filesystemRoot == dirSeparator() {
		return m.sandboxedPath(path), nil
	}

	// Split the cleaned path into components
	parts := splitPathParts(cleanSandboxPath(path))
	current := m.filesystemRoot

	for _, part := range parts {
		next := core.Path(current, part)
		realNext, err := resolveSymlinksPath(next)
		if err != nil {
			if core.Is(err, syscall.ENOENT) {
				// Part doesn't exist, we can't follow symlinks anymore.
				// Since the path is already Cleaned and current is safe,
				// appending a component to current will not escape.
				current = next
				continue
			}
			return "", err
		}

		// Verify the resolved part is still within the root
		if !isWithinRoot(m.filesystemRoot, realNext) {
			// Security event: sandbox escape attempt
			logSandboxEscape(m.filesystemRoot, path, realNext)
			return "", fs.ErrPermission
		}
		current = realNext
	}

	return current, nil
}

// Read returns file contents as string.
//
//	result := m.Read(...)
func (m *Medium) Read(path string) (string, error) {
	resolvedPath, err := m.validatePath(path)
	if err != nil {
		return "", err
	}
	return resultString("local.Read", core.Concat("read failed: ", path), unrestrictedFileSystem.Read(resolvedPath))
}

// Write saves content to file, creating parent directories as needed.
// Files are created with mode 0644. For sensitive files (keys, secrets),
// use WriteMode with 0600.
//
//	result := m.Write(...)
func (m *Medium) Write(path, content string) error {
	return m.WriteMode(path, content, 0644)
}

// WriteMode saves content to file with explicit permissions.
// Use 0600 for sensitive files (encryption output, private keys, auth hashes).
//
//	result := m.WriteMode(...)
func (m *Medium) WriteMode(path, content string, mode fs.FileMode) error {
	resolvedPath, err := m.validatePath(path)
	if err != nil {
		return err
	}
	return resultErr("local.WriteMode", core.Concat("write failed: ", path), unrestrictedFileSystem.WriteMode(resolvedPath, content, mode))
}

// EnsureDir creates directory if it doesn't exist.
//
//	result := m.EnsureDir(...)
func (m *Medium) EnsureDir(path string) error {
	resolvedPath, err := m.validatePath(path)
	if err != nil {
		return err
	}
	return resultErr("local.EnsureDir", core.Concat("ensure dir failed: ", path), unrestrictedFileSystem.EnsureDir(resolvedPath))
}

// IsDir returns true if path is a directory.
//
//	result := m.IsDir(...)
func (m *Medium) IsDir(path string) bool {
	if path == "" {
		return false
	}
	resolvedPath, err := m.validatePath(path)
	if err != nil {
		return false
	}
	return unrestrictedFileSystem.IsDir(resolvedPath)
}

// IsFile returns true if path is a regular file.
//
//	result := m.IsFile(...)
func (m *Medium) IsFile(path string) bool {
	if path == "" {
		return false
	}
	resolvedPath, err := m.validatePath(path)
	if err != nil {
		return false
	}
	return unrestrictedFileSystem.IsFile(resolvedPath)
}

// Exists returns true if path exists.
//
//	result := m.Exists(...)
func (m *Medium) Exists(path string) bool {
	resolvedPath, err := m.validatePath(path)
	if err != nil {
		return false
	}
	return unrestrictedFileSystem.Exists(resolvedPath)
}

// List returns directory entries.
//
//	result := m.List(...)
func (m *Medium) List(path string) ([]fs.DirEntry, error) {
	resolvedPath, err := m.validatePath(path)
	if err != nil {
		return nil, err
	}
	return resultDirEntries("local.List", core.Concat("list failed: ", path), unrestrictedFileSystem.List(resolvedPath))
}

// Stat returns file info.
//
//	result := m.Stat(...)
func (m *Medium) Stat(path string) (fs.FileInfo, error) {
	resolvedPath, err := m.validatePath(path)
	if err != nil {
		return nil, err
	}
	return resultFileInfo("local.Stat", core.Concat("stat failed: ", path), unrestrictedFileSystem.Stat(resolvedPath))
}

// Open opens the named file for reading.
//
//	result := m.Open(...)
func (m *Medium) Open(path string) (fs.File, error) {
	resolvedPath, err := m.validatePath(path)
	if err != nil {
		return nil, err
	}
	return resultFile("local.Open", core.Concat("open failed: ", path), unrestrictedFileSystem.Open(resolvedPath))
}

// Create creates or truncates the named file.
//
//	result := m.Create(...)
func (m *Medium) Create(path string) (goio.WriteCloser, error) {
	resolvedPath, err := m.validatePath(path)
	if err != nil {
		return nil, err
	}
	return resultWriteCloser("local.Create", core.Concat("create failed: ", path), unrestrictedFileSystem.Create(resolvedPath))
}

// Append opens the named file for appending, creating it if it doesn't exist.
//
//	result := m.Append(...)
func (m *Medium) Append(path string) (goio.WriteCloser, error) {
	resolvedPath, err := m.validatePath(path)
	if err != nil {
		return nil, err
	}
	return resultWriteCloser("local.Append", core.Concat("append failed: ", path), unrestrictedFileSystem.Append(resolvedPath))
}

// ReadStream returns a reader for the file content.
//
// This is a convenience wrapper around Open that exposes a streaming-oriented
// API, as required by the io.Medium interface, while Open provides the more
// general filesystem-level operation. Both methods are kept for semantic
// clarity and backward compatibility.
//
//	result := m.ReadStream(...)
func (m *Medium) ReadStream(path string) (goio.ReadCloser, error) {
	return m.Open(path)
}

// WriteStream returns a writer for the file content.
//
// This is a convenience wrapper around Create that exposes a streaming-oriented
// API, as required by the io.Medium interface, while Create provides the more
// general filesystem-level operation. Both methods are kept for semantic
// clarity and backward compatibility.
//
//	result := m.WriteStream(...)
func (m *Medium) WriteStream(path string) (goio.WriteCloser, error) {
	return m.Create(path)
}

// Delete removes a file or empty directory.
//
//	result := m.Delete(...)
func (m *Medium) Delete(path string) error {
	resolvedPath, err := m.validatePath(path)
	if err != nil {
		return err
	}
	if isProtectedPath(resolvedPath) {
		return core.E("local.Delete", core.Concat("refusing to delete protected path: ", resolvedPath), nil)
	}
	return resultErr("local.Delete", core.Concat("delete failed: ", path), unrestrictedFileSystem.Delete(resolvedPath))
}

// DeleteAll removes a file or directory recursively.
//
//	result := m.DeleteAll(...)
func (m *Medium) DeleteAll(path string) error {
	resolvedPath, err := m.validatePath(path)
	if err != nil {
		return err
	}
	if isProtectedPath(resolvedPath) {
		return core.E("local.DeleteAll", core.Concat("refusing to delete protected path: ", resolvedPath), nil)
	}
	return resultErr("local.DeleteAll", core.Concat("delete all failed: ", path), unrestrictedFileSystem.DeleteAll(resolvedPath))
}

// Rename moves a file or directory.
//
//	result := m.Rename(...)
func (m *Medium) Rename(oldPath, newPath string) error {
	oldResolvedPath, err := m.validatePath(oldPath)
	if err != nil {
		return err
	}
	newResolvedPath, err := m.validatePath(newPath)
	if err != nil {
		return err
	}
	return resultErr("local.Rename", core.Concat("rename failed: ", oldPath), unrestrictedFileSystem.Rename(oldResolvedPath, newResolvedPath))
}

func (m *Medium) FileGet(path string) (string, error) {
	return m.Read(path)
}

func (m *Medium) FileSet(path, content string) error {
	return m.Write(path, content)
}

func lstat(path string) (*syscall.Stat_t, error) {
	info := &syscall.Stat_t{}
	if err := syscall.Lstat(path, info); err != nil {
		return nil, err
	}
	return info, nil
}

func isSymlink(mode uint32) bool {
	return mode&syscall.S_IFMT == syscall.S_IFLNK
}

func readlink(path string) (string, error) {
	size := 256
	for {
		buf := make([]byte, size)
		n, err := syscall.Readlink(path, buf)
		if err != nil {
			return "", err
		}
		if n < len(buf) {
			return string(buf[:n]), nil
		}
		size *= 2
	}
}

func resultErr(operation, message string, result core.Result) error {
	if result.OK {
		return nil
	}
	if err, ok := result.Value.(error); ok {
		return core.E(operation, message, err)
	}
	return core.E(operation, message, nil)
}

func resultString(operation, message string, result core.Result) (string, error) {
	if !result.OK {
		return "", resultErr(operation, message, result)
	}
	value, ok := result.Value.(string)
	if !ok {
		return "", core.E(operation, "unexpected result type", nil)
	}
	return value, nil
}

func resultDirEntries(operation, message string, result core.Result) ([]fs.DirEntry, error) {
	if !result.OK {
		return nil, resultErr(operation, message, result)
	}
	entries, ok := result.Value.([]fs.DirEntry)
	if !ok {
		return nil, core.E(operation, "unexpected result type", nil)
	}
	return entries, nil
}

func resultFileInfo(operation, message string, result core.Result) (fs.FileInfo, error) {
	if !result.OK {
		return nil, resultErr(operation, message, result)
	}
	fileInfo, ok := result.Value.(fs.FileInfo)
	if !ok {
		return nil, core.E(operation, "unexpected result type", nil)
	}
	return fileInfo, nil
}

func resultFile(operation, message string, result core.Result) (fs.File, error) {
	if !result.OK {
		return nil, resultErr(operation, message, result)
	}
	file, ok := result.Value.(fs.File)
	if !ok {
		return nil, core.E(operation, "unexpected result type", nil)
	}
	return file, nil
}

func resultWriteCloser(operation, message string, result core.Result) (goio.WriteCloser, error) {
	if !result.OK {
		return nil, resultErr(operation, message, result)
	}
	writer, ok := result.Value.(goio.WriteCloser)
	if !ok {
		return nil, core.E(operation, "unexpected result type", nil)
	}
	return writer, nil
}
