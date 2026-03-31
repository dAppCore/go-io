package store

import (
	goio "io"
	"io/fs"
	"path"
	"time"

	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"
)

// Example: medium, _ := store.NewMedium(store.Options{Path: "config.db"})
// Example: _ = medium.Write("app/theme", "midnight")
// Example: entries, _ := medium.List("")
// Example: entries, _ := medium.List("app")
type Medium struct {
	store *Store
}

var _ coreio.Medium = (*Medium)(nil)

// Example: medium, _ := store.NewMedium(store.Options{Path: "config.db"})
// Example: _ = medium.Write("app/theme", "midnight")
func NewMedium(options Options) (*Medium, error) {
	store, err := New(options)
	if err != nil {
		return nil, err
	}
	return &Medium{store: store}, nil
}

// Example: medium := keyValueStore.AsMedium()
func (store *Store) AsMedium() *Medium {
	return &Medium{store: store}
}

// Example: keyValueStore := medium.Store()
func (medium *Medium) Store() *Store {
	return medium.store
}

// Example: _ = medium.Close()
func (medium *Medium) Close() error {
	return medium.store.Close()
}

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

func (medium *Medium) Read(entryPath string) (string, error) {
	group, key := splitGroupKeyPath(entryPath)
	if key == "" {
		return "", core.E("store.Read", "path must include group/key", fs.ErrInvalid)
	}
	return medium.store.Get(group, key)
}

func (medium *Medium) Write(entryPath, content string) error {
	group, key := splitGroupKeyPath(entryPath)
	if key == "" {
		return core.E("store.Write", "path must include group/key", fs.ErrInvalid)
	}
	return medium.store.Set(group, key, content)
}

// Example: _ = medium.WriteMode("app/theme", "midnight", 0600)
func (medium *Medium) WriteMode(entryPath, content string, mode fs.FileMode) error {
	return medium.Write(entryPath, content)
}

// Example: _ = medium.EnsureDir("app")
func (medium *Medium) EnsureDir(entryPath string) error {
	return nil
}

func (medium *Medium) IsFile(entryPath string) bool {
	group, key := splitGroupKeyPath(entryPath)
	if key == "" {
		return false
	}
	_, err := medium.store.Get(group, key)
	return err == nil
}

func (medium *Medium) FileGet(entryPath string) (string, error) {
	return medium.Read(entryPath)
}

func (medium *Medium) FileSet(entryPath, content string) error {
	return medium.Write(entryPath, content)
}

func (medium *Medium) Delete(entryPath string) error {
	group, key := splitGroupKeyPath(entryPath)
	if group == "" {
		return core.E("store.Delete", "path is required", fs.ErrInvalid)
	}
	if key == "" {
		entryCount, err := medium.store.Count(group)
		if err != nil {
			return err
		}
		if entryCount > 0 {
			return core.E("store.Delete", core.Concat("group not empty: ", group), fs.ErrExist)
		}
		return nil
	}
	return medium.store.Delete(group, key)
}

func (medium *Medium) DeleteAll(entryPath string) error {
	group, key := splitGroupKeyPath(entryPath)
	if group == "" {
		return core.E("store.DeleteAll", "path is required", fs.ErrInvalid)
	}
	if key == "" {
		return medium.store.DeleteGroup(group)
	}
	return medium.store.Delete(group, key)
}

func (medium *Medium) Rename(oldPath, newPath string) error {
	oldGroup, oldKey := splitGroupKeyPath(oldPath)
	newGroup, newKey := splitGroupKeyPath(newPath)
	if oldKey == "" || newKey == "" {
		return core.E("store.Rename", "both paths must include group/key", fs.ErrInvalid)
	}
	val, err := medium.store.Get(oldGroup, oldKey)
	if err != nil {
		return err
	}
	if err := medium.store.Set(newGroup, newKey, val); err != nil {
		return err
	}
	return medium.store.Delete(oldGroup, oldKey)
}

// Example: entries, _ := medium.List("app")
func (medium *Medium) List(entryPath string) ([]fs.DirEntry, error) {
	group, key := splitGroupKeyPath(entryPath)

	if group == "" {
		rows, err := medium.store.database.Query("SELECT DISTINCT grp FROM kv ORDER BY grp")
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
		if err := rows.Err(); err != nil {
			return nil, core.E("store.List", "rows", err)
		}
		return entries, nil
	}

	if key != "" {
		return nil, nil
	}

	all, err := medium.store.GetAll(group)
	if err != nil {
		return nil, err
	}
	var entries []fs.DirEntry
	for key, value := range all {
		entries = append(entries, &keyValueDirEntry{name: key, size: int64(len(value))})
	}
	return entries, nil
}

// Example: info, _ := medium.Stat("app/theme")
func (medium *Medium) Stat(entryPath string) (fs.FileInfo, error) {
	group, key := splitGroupKeyPath(entryPath)
	if group == "" {
		return nil, core.E("store.Stat", "path is required", fs.ErrInvalid)
	}
	if key == "" {
		entryCount, err := medium.store.Count(group)
		if err != nil {
			return nil, err
		}
		if entryCount == 0 {
			return nil, core.E("store.Stat", core.Concat("group not found: ", group), fs.ErrNotExist)
		}
		return &keyValueFileInfo{name: group, isDir: true}, nil
	}
	value, err := medium.store.Get(group, key)
	if err != nil {
		return nil, err
	}
	return &keyValueFileInfo{name: key, size: int64(len(value))}, nil
}

func (medium *Medium) Open(entryPath string) (fs.File, error) {
	group, key := splitGroupKeyPath(entryPath)
	if key == "" {
		return nil, core.E("store.Open", "path must include group/key", fs.ErrInvalid)
	}
	value, err := medium.store.Get(group, key)
	if err != nil {
		return nil, err
	}
	return &keyValueFile{name: key, content: []byte(value)}, nil
}

func (medium *Medium) Create(entryPath string) (goio.WriteCloser, error) {
	group, key := splitGroupKeyPath(entryPath)
	if key == "" {
		return nil, core.E("store.Create", "path must include group/key", fs.ErrInvalid)
	}
	return &keyValueWriteCloser{store: medium.store, group: group, key: key}, nil
}

func (medium *Medium) Append(entryPath string) (goio.WriteCloser, error) {
	group, key := splitGroupKeyPath(entryPath)
	if key == "" {
		return nil, core.E("store.Append", "path must include group/key", fs.ErrInvalid)
	}
	existingValue, _ := medium.store.Get(group, key)
	return &keyValueWriteCloser{store: medium.store, group: group, key: key, data: []byte(existingValue)}, nil
}

func (medium *Medium) ReadStream(entryPath string) (goio.ReadCloser, error) {
	group, key := splitGroupKeyPath(entryPath)
	if key == "" {
		return nil, core.E("store.ReadStream", "path must include group/key", fs.ErrInvalid)
	}
	val, err := medium.store.Get(group, key)
	if err != nil {
		return nil, err
	}
	return goio.NopCloser(core.NewReader(val)), nil
}

func (medium *Medium) WriteStream(entryPath string) (goio.WriteCloser, error) {
	return medium.Create(entryPath)
}

func (medium *Medium) Exists(entryPath string) bool {
	group, key := splitGroupKeyPath(entryPath)
	if group == "" {
		return false
	}
	if key == "" {
		entryCount, err := medium.store.Count(group)
		return err == nil && entryCount > 0
	}
	_, err := medium.store.Get(group, key)
	return err == nil
}

func (medium *Medium) IsDir(entryPath string) bool {
	group, key := splitGroupKeyPath(entryPath)
	if key != "" || group == "" {
		return false
	}
	entryCount, err := medium.store.Count(group)
	return err == nil && entryCount > 0
}

type keyValueFileInfo struct {
	name  string
	size  int64
	isDir bool
}

func (fileInfo *keyValueFileInfo) Name() string { return fileInfo.name }

func (fileInfo *keyValueFileInfo) Size() int64 { return fileInfo.size }

func (fileInfo *keyValueFileInfo) Mode() fs.FileMode {
	if fileInfo.isDir {
		return fs.ModeDir | 0755
	}
	return 0644
}

func (fileInfo *keyValueFileInfo) ModTime() time.Time { return time.Time{} }

func (fileInfo *keyValueFileInfo) IsDir() bool { return fileInfo.isDir }

func (fileInfo *keyValueFileInfo) Sys() any { return nil }

type keyValueDirEntry struct {
	name  string
	isDir bool
	size  int64
}

func (entry *keyValueDirEntry) Name() string { return entry.name }

func (entry *keyValueDirEntry) IsDir() bool { return entry.isDir }

func (entry *keyValueDirEntry) Type() fs.FileMode {
	if entry.isDir {
		return fs.ModeDir
	}
	return 0
}

func (entry *keyValueDirEntry) Info() (fs.FileInfo, error) {
	return &keyValueFileInfo{name: entry.name, size: entry.size, isDir: entry.isDir}, nil
}

type keyValueFile struct {
	name    string
	content []byte
	offset  int64
}

func (file *keyValueFile) Stat() (fs.FileInfo, error) {
	return &keyValueFileInfo{name: file.name, size: int64(len(file.content))}, nil
}

func (file *keyValueFile) Read(buffer []byte) (int, error) {
	if file.offset >= int64(len(file.content)) {
		return 0, goio.EOF
	}
	readCount := copy(buffer, file.content[file.offset:])
	file.offset += int64(readCount)
	return readCount, nil
}

func (file *keyValueFile) Close() error { return nil }

type keyValueWriteCloser struct {
	store *Store
	group string
	key   string
	data  []byte
}

func (writer *keyValueWriteCloser) Write(data []byte) (int, error) {
	writer.data = append(writer.data, data...)
	return len(data), nil
}

func (writer *keyValueWriteCloser) Close() error {
	return writer.store.Set(writer.group, writer.key, string(writer.data))
}
