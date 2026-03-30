package store

import (
	goio "io"
	"io/fs"
	"path"
	"time"

	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"
)

// Medium wraps a Store to satisfy the io.Medium interface.
// Paths are mapped as group/key - the first segment is the group,
// the rest is the key. List("") returns groups as directories,
// List("group") returns keys as files.
type Medium struct {
	store *Store
}

var _ coreio.Medium = (*Medium)(nil)

// NewMedium exposes a Store as an io.Medium.
//
//	medium, _ := store.NewMedium(store.Options{Path: "config.db"})
//	_ = medium.Write("app/theme", "midnight")
func NewMedium(options Options) (*Medium, error) {
	store, err := New(options)
	if err != nil {
		return nil, err
	}
	return &Medium{store: store}, nil
}

// Example: medium := keyValueStore.AsMedium()
func (s *Store) AsMedium() *Medium {
	return &Medium{store: s}
}

// Example: keyValueStore := medium.Store()
func (m *Medium) Store() *Store {
	return m.store
}

// Example: _ = medium.Close()
func (m *Medium) Close() error {
	return m.store.Close()
}

// splitGroupKeyPath splits a group/key path into store components.
func splitGroupKeyPath(entryPath string) (group, key string) {
	clean := path.Clean(entryPath)
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

func (m *Medium) Read(entryPath string) (string, error) {
	group, key := splitGroupKeyPath(entryPath)
	if key == "" {
		return "", core.E("store.Read", "path must include group/key", fs.ErrInvalid)
	}
	return m.store.Get(group, key)
}

func (m *Medium) Write(entryPath, content string) error {
	group, key := splitGroupKeyPath(entryPath)
	if key == "" {
		return core.E("store.Write", "path must include group/key", fs.ErrInvalid)
	}
	return m.store.Set(group, key, content)
}

// WriteMode ignores the requested mode because key-value entries do not store POSIX permissions.
func (m *Medium) WriteMode(entryPath, content string, _ fs.FileMode) error {
	return m.Write(entryPath, content)
}

// EnsureDir is a no-op — groups are created implicitly on Set.
func (m *Medium) EnsureDir(_ string) error {
	return nil
}

func (m *Medium) IsFile(entryPath string) bool {
	group, key := splitGroupKeyPath(entryPath)
	if key == "" {
		return false
	}
	_, err := m.store.Get(group, key)
	return err == nil
}

func (m *Medium) FileGet(entryPath string) (string, error) {
	return m.Read(entryPath)
}

func (m *Medium) FileSet(entryPath, content string) error {
	return m.Write(entryPath, content)
}

func (m *Medium) Delete(entryPath string) error {
	group, key := splitGroupKeyPath(entryPath)
	if group == "" {
		return core.E("store.Delete", "path is required", fs.ErrInvalid)
	}
	if key == "" {
		entryCount, err := m.store.Count(group)
		if err != nil {
			return err
		}
		if entryCount > 0 {
			return core.E("store.Delete", core.Concat("group not empty: ", group), fs.ErrExist)
		}
		return nil
	}
	return m.store.Delete(group, key)
}

func (m *Medium) DeleteAll(entryPath string) error {
	group, key := splitGroupKeyPath(entryPath)
	if group == "" {
		return core.E("store.DeleteAll", "path is required", fs.ErrInvalid)
	}
	if key == "" {
		return m.store.DeleteGroup(group)
	}
	return m.store.Delete(group, key)
}

func (m *Medium) Rename(oldPath, newPath string) error {
	oldGroup, oldKey := splitGroupKeyPath(oldPath)
	newGroup, newKey := splitGroupKeyPath(newPath)
	if oldKey == "" || newKey == "" {
		return core.E("store.Rename", "both paths must include group/key", fs.ErrInvalid)
	}
	val, err := m.store.Get(oldGroup, oldKey)
	if err != nil {
		return err
	}
	if err := m.store.Set(newGroup, newKey, val); err != nil {
		return err
	}
	return m.store.Delete(oldGroup, oldKey)
}

// List returns directory entries. Empty path returns groups.
// A group path returns keys in that group.
func (m *Medium) List(entryPath string) ([]fs.DirEntry, error) {
	group, key := splitGroupKeyPath(entryPath)

	if group == "" {
		rows, err := m.store.database.Query("SELECT DISTINCT grp FROM kv ORDER BY grp")
		if err != nil {
			return nil, core.E("store.List", "query groups", err)
		}
		defer rows.Close()

		var entries []fs.DirEntry
		for rows.Next() {
			var groupName string
			if err := rows.Scan(&groupName); err != nil {
				return nil, core.E("store.List", "scan", err)
			}
			entries = append(entries, &keyValueDirEntry{name: groupName, isDir: true})
		}
		return entries, rows.Err()
	}

	if key != "" {
		return nil, nil // leaf node, nothing beneath
	}

	all, err := m.store.GetAll(group)
	if err != nil {
		return nil, err
	}
	var entries []fs.DirEntry
	for key, value := range all {
		entries = append(entries, &keyValueDirEntry{name: key, size: int64(len(value))})
	}
	return entries, nil
}

// Stat returns file info for a group (dir) or key (file).
func (m *Medium) Stat(entryPath string) (fs.FileInfo, error) {
	group, key := splitGroupKeyPath(entryPath)
	if group == "" {
		return nil, core.E("store.Stat", "path is required", fs.ErrInvalid)
	}
	if key == "" {
		entryCount, err := m.store.Count(group)
		if err != nil {
			return nil, err
		}
		if entryCount == 0 {
			return nil, core.E("store.Stat", core.Concat("group not found: ", group), fs.ErrNotExist)
		}
		return &keyValueFileInfo{name: group, isDir: true}, nil
	}
	val, err := m.store.Get(group, key)
	if err != nil {
		return nil, err
	}
	return &keyValueFileInfo{name: key, size: int64(len(val))}, nil
}

func (m *Medium) Open(entryPath string) (fs.File, error) {
	group, key := splitGroupKeyPath(entryPath)
	if key == "" {
		return nil, core.E("store.Open", "path must include group/key", fs.ErrInvalid)
	}
	val, err := m.store.Get(group, key)
	if err != nil {
		return nil, err
	}
	return &keyValueFile{name: key, content: []byte(val)}, nil
}

func (m *Medium) Create(entryPath string) (goio.WriteCloser, error) {
	group, key := splitGroupKeyPath(entryPath)
	if key == "" {
		return nil, core.E("store.Create", "path must include group/key", fs.ErrInvalid)
	}
	return &keyValueWriteCloser{store: m.store, group: group, key: key}, nil
}

func (m *Medium) Append(entryPath string) (goio.WriteCloser, error) {
	group, key := splitGroupKeyPath(entryPath)
	if key == "" {
		return nil, core.E("store.Append", "path must include group/key", fs.ErrInvalid)
	}
	existing, _ := m.store.Get(group, key)
	return &keyValueWriteCloser{store: m.store, group: group, key: key, data: []byte(existing)}, nil
}

func (m *Medium) ReadStream(entryPath string) (goio.ReadCloser, error) {
	group, key := splitGroupKeyPath(entryPath)
	if key == "" {
		return nil, core.E("store.ReadStream", "path must include group/key", fs.ErrInvalid)
	}
	val, err := m.store.Get(group, key)
	if err != nil {
		return nil, err
	}
	return goio.NopCloser(core.NewReader(val)), nil
}

func (m *Medium) WriteStream(entryPath string) (goio.WriteCloser, error) {
	return m.Create(entryPath)
}

func (m *Medium) Exists(entryPath string) bool {
	group, key := splitGroupKeyPath(entryPath)
	if group == "" {
		return false
	}
	if key == "" {
		entryCount, err := m.store.Count(group)
		return err == nil && entryCount > 0
	}
	_, err := m.store.Get(group, key)
	return err == nil
}

func (m *Medium) IsDir(entryPath string) bool {
	group, key := splitGroupKeyPath(entryPath)
	if key != "" || group == "" {
		return false
	}
	entryCount, err := m.store.Count(group)
	return err == nil && entryCount > 0
}

// --- fs helper types ---

type keyValueFileInfo struct {
	name  string
	size  int64
	isDir bool
}

func (fi *keyValueFileInfo) Name() string { return fi.name }

func (fi *keyValueFileInfo) Size() int64 { return fi.size }

func (fi *keyValueFileInfo) Mode() fs.FileMode {
	if fi.isDir {
		return fs.ModeDir | 0755
	}
	return 0644
}

func (fi *keyValueFileInfo) ModTime() time.Time { return time.Time{} }

func (fi *keyValueFileInfo) IsDir() bool { return fi.isDir }

func (fi *keyValueFileInfo) Sys() any { return nil }

type keyValueDirEntry struct {
	name  string
	isDir bool
	size  int64
}

func (de *keyValueDirEntry) Name() string { return de.name }

func (de *keyValueDirEntry) IsDir() bool { return de.isDir }

func (de *keyValueDirEntry) Type() fs.FileMode {
	if de.isDir {
		return fs.ModeDir
	}
	return 0
}

func (de *keyValueDirEntry) Info() (fs.FileInfo, error) {
	return &keyValueFileInfo{name: de.name, size: de.size, isDir: de.isDir}, nil
}

type keyValueFile struct {
	name    string
	content []byte
	offset  int64
}

func (f *keyValueFile) Stat() (fs.FileInfo, error) {
	return &keyValueFileInfo{name: f.name, size: int64(len(f.content))}, nil
}

func (f *keyValueFile) Read(b []byte) (int, error) {
	if f.offset >= int64(len(f.content)) {
		return 0, goio.EOF
	}
	n := copy(b, f.content[f.offset:])
	f.offset += int64(n)
	return n, nil
}

func (f *keyValueFile) Close() error { return nil }

type keyValueWriteCloser struct {
	store *Store
	group string
	key   string
	data  []byte
}

func (w *keyValueWriteCloser) Write(p []byte) (int, error) {
	w.data = append(w.data, p...)
	return len(p), nil
}

func (w *keyValueWriteCloser) Close() error {
	return w.store.Set(w.group, w.key, string(w.data))
}
