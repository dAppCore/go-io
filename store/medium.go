package store

import (
	goio "io"
	"io/fs"
	"path"
	"time"

	core "dappco.re/go/core"
)

// Medium wraps a Store to satisfy the io.Medium interface.
// Paths are mapped as group/key — first segment is the group,
// the rest is the key. List("") returns groups as directories,
// List("group") returns keys as files.
type Medium struct {
	s *Store
}

// NewMedium creates an io.Medium backed by a KV store at the given SQLite path.
//
// Example usage:
//
//	m, _ := store.NewMedium("config.db")
//	_ = m.Write("app/theme", "midnight")
func NewMedium(dbPath string) (*Medium, error) {
	s, err := New(dbPath)
	if err != nil {
		return nil, err
	}
	return &Medium{s: s}, nil
}

// AsMedium returns a Medium adapter for an existing Store.
//
//	result := s.AsMedium(...)
func (s *Store) AsMedium() *Medium {
	return &Medium{s: s}
}

// Store returns the underlying KV store for direct access.
//
//	result := m.Store(...)
func (m *Medium) Store() *Store {
	return m.s
}

// Close closes the underlying store.
//
//	result := m.Close(...)
func (m *Medium) Close() error {
	return m.s.Close()
}

// splitPath splits a medium-style path into group and key.
// First segment = group, remainder = key.
func splitPath(p string) (group, key string) {
	clean := path.Clean(p)
	clean = core.TrimPrefix(clean, "/")
	if clean == "" || clean == "." {
		return "", ""
	}
	parts := core.SplitN(clean, "/", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

// Read retrieves the value at group/key.
//
//	result := m.Read(...)
func (m *Medium) Read(p string) (string, error) {
	group, key := splitPath(p)
	if key == "" {
		return "", core.E("store.Read", "path must include group/key", fs.ErrInvalid)
	}
	return m.s.Get(group, key)
}

// Write stores a value at group/key.
//
//	result := m.Write(...)
func (m *Medium) Write(p, content string) error {
	group, key := splitPath(p)
	if key == "" {
		return core.E("store.Write", "path must include group/key", fs.ErrInvalid)
	}
	return m.s.Set(group, key, content)
}

// EnsureDir is a no-op — groups are created implicitly on Set.
//
//	result := m.EnsureDir(...)
func (m *Medium) EnsureDir(_ string) error {
	return nil
}

// IsFile returns true if a group/key pair exists.
//
//	result := m.IsFile(...)
func (m *Medium) IsFile(p string) bool {
	group, key := splitPath(p)
	if key == "" {
		return false
	}
	_, err := m.s.Get(group, key)
	return err == nil
}

// FileGet is an alias for Read.
//
//	result := m.FileGet(...)
func (m *Medium) FileGet(p string) (string, error) {
	return m.Read(p)
}

// FileSet is an alias for Write.
//
//	result := m.FileSet(...)
func (m *Medium) FileSet(p, content string) error {
	return m.Write(p, content)
}

// Delete removes a key, or checks that a group is empty.
//
//	result := m.Delete(...)
func (m *Medium) Delete(p string) error {
	group, key := splitPath(p)
	if group == "" {
		return core.E("store.Delete", "path is required", fs.ErrInvalid)
	}
	if key == "" {
		n, err := m.s.Count(group)
		if err != nil {
			return err
		}
		if n > 0 {
			return core.E("store.Delete", core.Concat("group not empty: ", group), fs.ErrExist)
		}
		return nil
	}
	return m.s.Delete(group, key)
}

// DeleteAll removes a key, or all keys in a group.
//
//	result := m.DeleteAll(...)
func (m *Medium) DeleteAll(p string) error {
	group, key := splitPath(p)
	if group == "" {
		return core.E("store.DeleteAll", "path is required", fs.ErrInvalid)
	}
	if key == "" {
		return m.s.DeleteGroup(group)
	}
	return m.s.Delete(group, key)
}

// Rename moves a key from one path to another.
//
//	result := m.Rename(...)
func (m *Medium) Rename(oldPath, newPath string) error {
	og, ok := splitPath(oldPath)
	ng, nk := splitPath(newPath)
	if ok == "" || nk == "" {
		return core.E("store.Rename", "both paths must include group/key", fs.ErrInvalid)
	}
	val, err := m.s.Get(og, ok)
	if err != nil {
		return err
	}
	if err := m.s.Set(ng, nk, val); err != nil {
		return err
	}
	return m.s.Delete(og, ok)
}

// List returns directory entries. Empty path returns groups.
// A group path returns keys in that group.
//
//	result := m.List(...)
func (m *Medium) List(p string) ([]fs.DirEntry, error) {
	group, key := splitPath(p)

	if group == "" {
		rows, err := m.s.db.Query("SELECT DISTINCT grp FROM kv ORDER BY grp")
		if err != nil {
			return nil, core.E("store.List", "query groups", err)
		}
		defer rows.Close()

		var entries []fs.DirEntry
		for rows.Next() {
			var g string
			if err := rows.Scan(&g); err != nil {
				return nil, core.E("store.List", "scan", err)
			}
			entries = append(entries, &kvDirEntry{name: g, isDir: true})
		}
		return entries, rows.Err()
	}

	if key != "" {
		return nil, nil // leaf node, nothing beneath
	}

	all, err := m.s.GetAll(group)
	if err != nil {
		return nil, err
	}
	var entries []fs.DirEntry
	for k, v := range all {
		entries = append(entries, &kvDirEntry{name: k, size: int64(len(v))})
	}
	return entries, nil
}

// Stat returns file info for a group (dir) or key (file).
//
//	result := m.Stat(...)
func (m *Medium) Stat(p string) (fs.FileInfo, error) {
	group, key := splitPath(p)
	if group == "" {
		return nil, core.E("store.Stat", "path is required", fs.ErrInvalid)
	}
	if key == "" {
		n, err := m.s.Count(group)
		if err != nil {
			return nil, err
		}
		if n == 0 {
			return nil, core.E("store.Stat", core.Concat("group not found: ", group), fs.ErrNotExist)
		}
		return &kvFileInfo{name: group, isDir: true}, nil
	}
	val, err := m.s.Get(group, key)
	if err != nil {
		return nil, err
	}
	return &kvFileInfo{name: key, size: int64(len(val))}, nil
}

// Open opens a key for reading.
//
//	result := m.Open(...)
func (m *Medium) Open(p string) (fs.File, error) {
	group, key := splitPath(p)
	if key == "" {
		return nil, core.E("store.Open", "path must include group/key", fs.ErrInvalid)
	}
	val, err := m.s.Get(group, key)
	if err != nil {
		return nil, err
	}
	return &kvFile{name: key, content: []byte(val)}, nil
}

// Create creates or truncates a key. Content is stored on Close.
//
//	result := m.Create(...)
func (m *Medium) Create(p string) (goio.WriteCloser, error) {
	group, key := splitPath(p)
	if key == "" {
		return nil, core.E("store.Create", "path must include group/key", fs.ErrInvalid)
	}
	return &kvWriteCloser{s: m.s, group: group, key: key}, nil
}

// Append opens a key for appending. Content is stored on Close.
//
//	result := m.Append(...)
func (m *Medium) Append(p string) (goio.WriteCloser, error) {
	group, key := splitPath(p)
	if key == "" {
		return nil, core.E("store.Append", "path must include group/key", fs.ErrInvalid)
	}
	existing, _ := m.s.Get(group, key)
	return &kvWriteCloser{s: m.s, group: group, key: key, data: []byte(existing)}, nil
}

// ReadStream returns a reader for the value.
//
//	result := m.ReadStream(...)
func (m *Medium) ReadStream(p string) (goio.ReadCloser, error) {
	group, key := splitPath(p)
	if key == "" {
		return nil, core.E("store.ReadStream", "path must include group/key", fs.ErrInvalid)
	}
	val, err := m.s.Get(group, key)
	if err != nil {
		return nil, err
	}
	return goio.NopCloser(core.NewReader(val)), nil
}

// WriteStream returns a writer. Content is stored on Close.
//
//	result := m.WriteStream(...)
func (m *Medium) WriteStream(p string) (goio.WriteCloser, error) {
	return m.Create(p)
}

// Exists returns true if a group or key exists.
//
//	result := m.Exists(...)
func (m *Medium) Exists(p string) bool {
	group, key := splitPath(p)
	if group == "" {
		return false
	}
	if key == "" {
		n, err := m.s.Count(group)
		return err == nil && n > 0
	}
	_, err := m.s.Get(group, key)
	return err == nil
}

// IsDir returns true if the path is a group with entries.
//
//	result := m.IsDir(...)
func (m *Medium) IsDir(p string) bool {
	group, key := splitPath(p)
	if key != "" || group == "" {
		return false
	}
	n, err := m.s.Count(group)
	return err == nil && n > 0
}

// --- fs helper types ---

type kvFileInfo struct {
	name  string
	size  int64
	isDir bool
}

// Name documents the Name operation.
//
//	result := fi.Name(...)
func (fi *kvFileInfo) Name() string { return fi.name }

// Size documents the Size operation.
//
//	result := fi.Size(...)
func (fi *kvFileInfo) Size() int64 { return fi.size }

// Mode documents the Mode operation.
//
//	result := fi.Mode(...)
func (fi *kvFileInfo) Mode() fs.FileMode {
	if fi.isDir {
		return fs.ModeDir | 0755
	}
	return 0644
}

// ModTime documents the ModTime operation.
//
//	result := fi.ModTime(...)
func (fi *kvFileInfo) ModTime() time.Time { return time.Time{} }

// IsDir documents the IsDir operation.
//
//	result := fi.IsDir(...)
func (fi *kvFileInfo) IsDir() bool { return fi.isDir }

// Sys documents the Sys operation.
//
//	result := fi.Sys(...)
func (fi *kvFileInfo) Sys() any { return nil }

type kvDirEntry struct {
	name  string
	isDir bool
	size  int64
}

// Name documents the Name operation.
//
//	result := de.Name(...)
func (de *kvDirEntry) Name() string { return de.name }

// IsDir documents the IsDir operation.
//
//	result := de.IsDir(...)
func (de *kvDirEntry) IsDir() bool { return de.isDir }

// Type documents the Type operation.
//
//	result := de.Type(...)
func (de *kvDirEntry) Type() fs.FileMode {
	if de.isDir {
		return fs.ModeDir
	}
	return 0
}

// Info documents the Info operation.
//
//	result := de.Info(...)
func (de *kvDirEntry) Info() (fs.FileInfo, error) {
	return &kvFileInfo{name: de.name, size: de.size, isDir: de.isDir}, nil
}

type kvFile struct {
	name    string
	content []byte
	offset  int64
}

// Stat documents the Stat operation.
//
//	result := f.Stat(...)
func (f *kvFile) Stat() (fs.FileInfo, error) {
	return &kvFileInfo{name: f.name, size: int64(len(f.content))}, nil
}

// Read documents the Read operation.
//
//	result := f.Read(...)
func (f *kvFile) Read(b []byte) (int, error) {
	if f.offset >= int64(len(f.content)) {
		return 0, goio.EOF
	}
	n := copy(b, f.content[f.offset:])
	f.offset += int64(n)
	return n, nil
}

// Close documents the Close operation.
//
//	result := f.Close(...)
func (f *kvFile) Close() error { return nil }

type kvWriteCloser struct {
	s     *Store
	group string
	key   string
	data  []byte
}

// Write documents the Write operation.
//
//	result := w.Write(...)
func (w *kvWriteCloser) Write(p []byte) (int, error) {
	w.data = append(w.data, p...)
	return len(p), nil
}

// Close documents the Close operation.
//
//	result := w.Close(...)
func (w *kvWriteCloser) Close() error {
	return w.s.Set(w.group, w.key, string(w.data))
}
