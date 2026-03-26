// Package local provides a local filesystem implementation of the io.Medium interface.
package local

import (
	"fmt"
	goio "io"
	"io/fs"
	"os"
	"strings"
	"time"

	core "dappco.re/go/core"
	coreerr "forge.lthn.ai/core/go-log"
)

// Medium is a local filesystem storage backend.
type Medium struct {
	root string
}

// New creates a new local Medium rooted at the given directory.
// Pass "/" for full filesystem access, or a specific path to sandbox.
//
// Example usage:
//
//	m, _ := local.New("/srv/app")
//	_ = m.Write("config/app.yaml", "port: 8080")
func New(root string) (*Medium, error) {
	abs := absolutePath(root)
	// Resolve symlinks so sandbox checks compare like-for-like.
	// On macOS, /var is a symlink to /private/var — without this,
	// resolving child paths resolves to /private/var/... while
	// root stays /var/..., causing false sandbox escape detections.
	if resolved, err := resolveSymlinksPath(abs); err == nil {
		abs = resolved
	}
	return &Medium{root: abs}, nil
}

func dirSeparator() string {
	if sep := core.Env("DS"); sep != "" {
		return sep
	}
	return string(os.PathSeparator)
}

func normalisePath(p string) string {
	sep := dirSeparator()
	if sep == "/" {
		return strings.ReplaceAll(p, "\\", sep)
	}
	return strings.ReplaceAll(p, "/", sep)
}

func currentWorkingDir() string {
	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		return cwd
	}
	if cwd := core.Env("DIR_CWD"); cwd != "" {
		return cwd
	}
	return "."
}

func absolutePath(p string) string {
	p = normalisePath(p)
	if core.PathIsAbs(p) {
		return core.Path(p)
	}
	return core.Path(currentWorkingDir(), p)
}

func cleanSandboxPath(p string) string {
	return core.Path(dirSeparator() + normalisePath(p))
}

func splitPathParts(p string) []string {
	trimmed := strings.TrimPrefix(p, dirSeparator())
	if trimmed == "" {
		return nil
	}
	var parts []string
	for _, part := range strings.Split(trimmed, dirSeparator()) {
		if part == "" {
			continue
		}
		parts = append(parts, part)
	}
	return parts
}

func resolveSymlinksPath(p string) (string, error) {
	return resolveSymlinksRecursive(absolutePath(p), map[string]struct{}{})
}

func resolveSymlinksRecursive(p string, seen map[string]struct{}) (string, error) {
	p = core.Path(p)
	if p == dirSeparator() {
		return p, nil
	}

	current := dirSeparator()
	for _, part := range splitPathParts(p) {
		next := core.Path(current, part)
		info, err := os.Lstat(next)
		if err != nil {
			if os.IsNotExist(err) {
				current = next
				continue
			}
			return "", err
		}
		if info.Mode()&os.ModeSymlink == 0 {
			current = next
			continue
		}

		target, err := os.Readlink(next)
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
			return "", coreerr.E("local.resolveSymlinksPath", "symlink cycle: "+target, os.ErrInvalid)
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
	return target == root || strings.HasPrefix(target, root+dirSeparator())
}

func canonicalPath(p string) string {
	if p == "" {
		return ""
	}
	if resolved, err := resolveSymlinksPath(p); err == nil {
		return resolved
	}
	return absolutePath(p)
}

func isProtectedPath(full string) bool {
	full = canonicalPath(full)
	protected := map[string]struct{}{
		canonicalPath(dirSeparator()): {},
	}
	for _, home := range []string{core.Env("HOME"), core.Env("DIR_HOME")} {
		if home == "" {
			continue
		}
		protected[canonicalPath(home)] = struct{}{}
	}
	_, ok := protected[full]
	return ok
}

func logSandboxEscape(root, path, attempted string) {
	username := core.Env("USER")
	if username == "" {
		username = "unknown"
	}
	fmt.Fprintf(os.Stderr, "[%s] SECURITY sandbox escape detected root=%s path=%s attempted=%s user=%s\n",
		time.Now().Format(time.RFC3339), root, path, attempted, username)
}

// path sanitises and returns the full path.
// Absolute paths are sandboxed under root (unless root is "/").
func (m *Medium) path(p string) string {
	if p == "" {
		return m.root
	}

	// If the path is relative and the medium is rooted at "/",
	// treat it as relative to the current working directory.
	// This makes io.Local behave more like the standard 'os' package.
	if m.root == dirSeparator() && !core.PathIsAbs(normalisePath(p)) {
		return core.Path(currentWorkingDir(), normalisePath(p))
	}

	// Use a cleaned absolute path to resolve all .. and . internally
	// before joining with the root. This is a standard way to sandbox paths.
	clean := cleanSandboxPath(p)

	// If root is "/", allow absolute paths through
	if m.root == dirSeparator() {
		return clean
	}

	// Join cleaned relative path with root
	return core.Path(m.root, strings.TrimPrefix(clean, dirSeparator()))
}

// validatePath ensures the path is within the sandbox, following symlinks if they exist.
func (m *Medium) validatePath(p string) (string, error) {
	if m.root == dirSeparator() {
		return m.path(p), nil
	}

	// Split the cleaned path into components
	parts := splitPathParts(cleanSandboxPath(p))
	current := m.root

	for _, part := range parts {
		next := core.Path(current, part)
		realNext, err := resolveSymlinksPath(next)
		if err != nil {
			if os.IsNotExist(err) {
				// Part doesn't exist, we can't follow symlinks anymore.
				// Since the path is already Cleaned and current is safe,
				// appending a component to current will not escape.
				current = next
				continue
			}
			return "", err
		}

		// Verify the resolved part is still within the root
		if !isWithinRoot(m.root, realNext) {
			// Security event: sandbox escape attempt
			logSandboxEscape(m.root, p, realNext)
			return "", os.ErrPermission // Path escapes sandbox
		}
		current = realNext
	}

	return current, nil
}

// Read returns file contents as string.
func (m *Medium) Read(p string) (string, error) {
	full, err := m.validatePath(p)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Write saves content to file, creating parent directories as needed.
// Files are created with mode 0644. For sensitive files (keys, secrets),
// use WriteMode with 0600.
func (m *Medium) Write(p, content string) error {
	return m.WriteMode(p, content, 0644)
}

// WriteMode saves content to file with explicit permissions.
// Use 0600 for sensitive files (encryption output, private keys, auth hashes).
func (m *Medium) WriteMode(p, content string, mode fs.FileMode) error {
	full, err := m.validatePath(p)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(core.PathDir(full), 0755); err != nil {
		return err
	}
	return os.WriteFile(full, []byte(content), mode)
}

// EnsureDir creates directory if it doesn't exist.
func (m *Medium) EnsureDir(p string) error {
	full, err := m.validatePath(p)
	if err != nil {
		return err
	}
	return os.MkdirAll(full, 0755)
}

// IsDir returns true if path is a directory.
func (m *Medium) IsDir(p string) bool {
	if p == "" {
		return false
	}
	full, err := m.validatePath(p)
	if err != nil {
		return false
	}
	info, err := os.Stat(full)
	return err == nil && info.IsDir()
}

// IsFile returns true if path is a regular file.
func (m *Medium) IsFile(p string) bool {
	if p == "" {
		return false
	}
	full, err := m.validatePath(p)
	if err != nil {
		return false
	}
	info, err := os.Stat(full)
	return err == nil && info.Mode().IsRegular()
}

// Exists returns true if path exists.
func (m *Medium) Exists(p string) bool {
	full, err := m.validatePath(p)
	if err != nil {
		return false
	}
	_, err = os.Stat(full)
	return err == nil
}

// List returns directory entries.
func (m *Medium) List(p string) ([]fs.DirEntry, error) {
	full, err := m.validatePath(p)
	if err != nil {
		return nil, err
	}
	return os.ReadDir(full)
}

// Stat returns file info.
func (m *Medium) Stat(p string) (fs.FileInfo, error) {
	full, err := m.validatePath(p)
	if err != nil {
		return nil, err
	}
	return os.Stat(full)
}

// Open opens the named file for reading.
func (m *Medium) Open(p string) (fs.File, error) {
	full, err := m.validatePath(p)
	if err != nil {
		return nil, err
	}
	return os.Open(full)
}

// Create creates or truncates the named file.
func (m *Medium) Create(p string) (goio.WriteCloser, error) {
	full, err := m.validatePath(p)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(core.PathDir(full), 0755); err != nil {
		return nil, err
	}
	return os.Create(full)
}

// Append opens the named file for appending, creating it if it doesn't exist.
func (m *Medium) Append(p string) (goio.WriteCloser, error) {
	full, err := m.validatePath(p)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(core.PathDir(full), 0755); err != nil {
		return nil, err
	}
	return os.OpenFile(full, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
}

// ReadStream returns a reader for the file content.
//
// This is a convenience wrapper around Open that exposes a streaming-oriented
// API, as required by the io.Medium interface, while Open provides the more
// general filesystem-level operation. Both methods are kept for semantic
// clarity and backward compatibility.
func (m *Medium) ReadStream(path string) (goio.ReadCloser, error) {
	return m.Open(path)
}

// WriteStream returns a writer for the file content.
//
// This is a convenience wrapper around Create that exposes a streaming-oriented
// API, as required by the io.Medium interface, while Create provides the more
// general filesystem-level operation. Both methods are kept for semantic
// clarity and backward compatibility.
func (m *Medium) WriteStream(path string) (goio.WriteCloser, error) {
	return m.Create(path)
}

// Delete removes a file or empty directory.
func (m *Medium) Delete(p string) error {
	full, err := m.validatePath(p)
	if err != nil {
		return err
	}
	if isProtectedPath(full) {
		return coreerr.E("local.Delete", "refusing to delete protected path: "+full, nil)
	}
	return os.Remove(full)
}

// DeleteAll removes a file or directory recursively.
func (m *Medium) DeleteAll(p string) error {
	full, err := m.validatePath(p)
	if err != nil {
		return err
	}
	if isProtectedPath(full) {
		return coreerr.E("local.DeleteAll", "refusing to delete protected path: "+full, nil)
	}
	return os.RemoveAll(full)
}

// Rename moves a file or directory.
func (m *Medium) Rename(oldPath, newPath string) error {
	oldFull, err := m.validatePath(oldPath)
	if err != nil {
		return err
	}
	newFull, err := m.validatePath(newPath)
	if err != nil {
		return err
	}
	return os.Rename(oldFull, newFull)
}

// FileGet is an alias for Read.
func (m *Medium) FileGet(p string) (string, error) {
	return m.Read(p)
}

// FileSet is an alias for Write.
func (m *Medium) FileSet(p, content string) error {
	return m.Write(p, content)
}
