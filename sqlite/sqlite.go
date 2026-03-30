// Package sqlite persists io.Medium content in a SQLite database.
//
//	medium, _ := sqlite.New(sqlite.Options{Path: ":memory:"})
//	_ = medium.Write("config/app.yaml", "port: 8080")
package sqlite

import (
	"bytes"
	"database/sql"
	goio "io"
	"io/fs"
	"path"
	"time"

	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"

	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

// Medium is a SQLite-backed storage backend implementing the io.Medium interface.
type Medium struct {
	database *sql.DB
	table    string
}

var _ coreio.Medium = (*Medium)(nil)

// Options configures a SQLite-backed Medium.
type Options struct {
	// Path is the SQLite database path. Use ":memory:" for tests.
	Path string
	// Table is the table name used for file storage. Empty defaults to "files".
	Table string
}

func normaliseTableName(table string) string {
	if table == "" {
		return "files"
	}
	return table
}

// New opens a SQLite-backed Medium at the provided database path.
//
//	medium, _ := sqlite.New(sqlite.Options{Path: ":memory:", Table: "files"})
//	_ = medium.Write("config/app.yaml", "port: 8080")
func New(options Options) (*Medium, error) {
	if options.Path == "" {
		return nil, core.E("sqlite.New", "database path is required", nil)
	}

	medium := &Medium{table: normaliseTableName(options.Table)}

	database, err := sql.Open("sqlite", options.Path)
	if err != nil {
		return nil, core.E("sqlite.New", "failed to open database", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := database.Exec("PRAGMA journal_mode=WAL"); err != nil {
		database.Close()
		return nil, core.E("sqlite.New", "failed to set WAL mode", err)
	}

	// Create the schema
	createSQL := `CREATE TABLE IF NOT EXISTS ` + medium.table + ` (
		path    TEXT PRIMARY KEY,
		content BLOB NOT NULL,
		mode    INTEGER DEFAULT 420,
		is_dir  BOOLEAN DEFAULT FALSE,
		mtime   DATETIME DEFAULT CURRENT_TIMESTAMP
	)`
	if _, err := database.Exec(createSQL); err != nil {
		database.Close()
		return nil, core.E("sqlite.New", "failed to create table", err)
	}

	medium.database = database
	return medium, nil
}

// Close closes the underlying database connection.
func (m *Medium) Close() error {
	if m.database != nil {
		return m.database.Close()
	}
	return nil
}

// normaliseEntryPath normalises a path for consistent storage.
// Uses a leading "/" before Clean to sandbox traversal attempts.
func normaliseEntryPath(filePath string) string {
	clean := path.Clean("/" + filePath)
	if clean == "/" {
		return ""
	}
	return core.TrimPrefix(clean, "/")
}

func (m *Medium) Read(filePath string) (string, error) {
	key := normaliseEntryPath(filePath)
	if key == "" {
		return "", core.E("sqlite.Read", "path is required", fs.ErrInvalid)
	}

	var content []byte
	var isDir bool
	err := m.database.QueryRow(
		`SELECT content, is_dir FROM `+m.table+` WHERE path = ?`, key,
	).Scan(&content, &isDir)
	if err == sql.ErrNoRows {
		return "", core.E("sqlite.Read", core.Concat("file not found: ", key), fs.ErrNotExist)
	}
	if err != nil {
		return "", core.E("sqlite.Read", core.Concat("query failed: ", key), err)
	}
	if isDir {
		return "", core.E("sqlite.Read", core.Concat("path is a directory: ", key), fs.ErrInvalid)
	}
	return string(content), nil
}

func (m *Medium) Write(filePath, content string) error {
	return m.WriteMode(filePath, content, 0644)
}

// WriteMode saves the given content with explicit permissions.
func (m *Medium) WriteMode(filePath, content string, mode fs.FileMode) error {
	key := normaliseEntryPath(filePath)
	if key == "" {
		return core.E("sqlite.WriteMode", "path is required", fs.ErrInvalid)
	}

	_, err := m.database.Exec(
		`INSERT INTO `+m.table+` (path, content, mode, is_dir, mtime) VALUES (?, ?, ?, FALSE, ?)
		 ON CONFLICT(path) DO UPDATE SET content = excluded.content, mode = excluded.mode, is_dir = FALSE, mtime = excluded.mtime`,
		key, []byte(content), int(mode), time.Now().UTC(),
	)
	if err != nil {
		return core.E("sqlite.WriteMode", core.Concat("insert failed: ", key), err)
	}
	return nil
}

// EnsureDir makes sure a directory exists, creating it if necessary.
func (m *Medium) EnsureDir(filePath string) error {
	key := normaliseEntryPath(filePath)
	if key == "" {
		// Root always "exists"
		return nil
	}

	_, err := m.database.Exec(
		`INSERT INTO `+m.table+` (path, content, mode, is_dir, mtime) VALUES (?, '', 493, TRUE, ?)
		 ON CONFLICT(path) DO NOTHING`,
		key, time.Now().UTC(),
	)
	if err != nil {
		return core.E("sqlite.EnsureDir", core.Concat("insert failed: ", key), err)
	}
	return nil
}

func (m *Medium) IsFile(filePath string) bool {
	key := normaliseEntryPath(filePath)
	if key == "" {
		return false
	}

	var isDir bool
	err := m.database.QueryRow(
		`SELECT is_dir FROM `+m.table+` WHERE path = ?`, key,
	).Scan(&isDir)
	if err != nil {
		return false
	}
	return !isDir
}

func (m *Medium) FileGet(filePath string) (string, error) {
	return m.Read(filePath)
}

func (m *Medium) FileSet(filePath, content string) error {
	return m.Write(filePath, content)
}

// Delete removes a file or empty directory.
func (m *Medium) Delete(filePath string) error {
	key := normaliseEntryPath(filePath)
	if key == "" {
		return core.E("sqlite.Delete", "path is required", fs.ErrInvalid)
	}

	// Check if it's a directory with children
	var isDir bool
	err := m.database.QueryRow(
		`SELECT is_dir FROM `+m.table+` WHERE path = ?`, key,
	).Scan(&isDir)
	if err == sql.ErrNoRows {
		return core.E("sqlite.Delete", core.Concat("path not found: ", key), fs.ErrNotExist)
	}
	if err != nil {
		return core.E("sqlite.Delete", core.Concat("query failed: ", key), err)
	}

	if isDir {
		// Check for children
		prefix := key + "/"
		var count int
		err := m.database.QueryRow(
			`SELECT COUNT(*) FROM `+m.table+` WHERE path LIKE ? AND path != ?`, prefix+"%", key,
		).Scan(&count)
		if err != nil {
			return core.E("sqlite.Delete", core.Concat("count failed: ", key), err)
		}
		if count > 0 {
			return core.E("sqlite.Delete", core.Concat("directory not empty: ", key), fs.ErrExist)
		}
	}

	res, err := m.database.Exec(`DELETE FROM `+m.table+` WHERE path = ?`, key)
	if err != nil {
		return core.E("sqlite.Delete", core.Concat("delete failed: ", key), err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return core.E("sqlite.Delete", core.Concat("path not found: ", key), fs.ErrNotExist)
	}
	return nil
}

// DeleteAll removes a file or directory and all its contents recursively.
func (m *Medium) DeleteAll(filePath string) error {
	key := normaliseEntryPath(filePath)
	if key == "" {
		return core.E("sqlite.DeleteAll", "path is required", fs.ErrInvalid)
	}

	prefix := key + "/"

	// Delete the exact path and all children
	res, err := m.database.Exec(
		`DELETE FROM `+m.table+` WHERE path = ? OR path LIKE ?`,
		key, prefix+"%",
	)
	if err != nil {
		return core.E("sqlite.DeleteAll", core.Concat("delete failed: ", key), err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return core.E("sqlite.DeleteAll", core.Concat("path not found: ", key), fs.ErrNotExist)
	}
	return nil
}

// Rename moves a file or directory from oldPath to newPath.
func (m *Medium) Rename(oldPath, newPath string) error {
	oldKey := normaliseEntryPath(oldPath)
	newKey := normaliseEntryPath(newPath)
	if oldKey == "" || newKey == "" {
		return core.E("sqlite.Rename", "both old and new paths are required", fs.ErrInvalid)
	}

	tx, err := m.database.Begin()
	if err != nil {
		return core.E("sqlite.Rename", "begin tx failed", err)
	}
	defer tx.Rollback()

	// Check if source exists
	var content []byte
	var mode int
	var isDir bool
	var mtime time.Time
	err = tx.QueryRow(
		`SELECT content, mode, is_dir, mtime FROM `+m.table+` WHERE path = ?`, oldKey,
	).Scan(&content, &mode, &isDir, &mtime)
	if err == sql.ErrNoRows {
		return core.E("sqlite.Rename", core.Concat("source not found: ", oldKey), fs.ErrNotExist)
	}
	if err != nil {
		return core.E("sqlite.Rename", core.Concat("query failed: ", oldKey), err)
	}

	// Insert or replace at new path
	_, err = tx.Exec(
		`INSERT INTO `+m.table+` (path, content, mode, is_dir, mtime) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(path) DO UPDATE SET content = excluded.content, mode = excluded.mode, is_dir = excluded.is_dir, mtime = excluded.mtime`,
		newKey, content, mode, isDir, mtime,
	)
	if err != nil {
		return core.E("sqlite.Rename", core.Concat("insert at new path failed: ", newKey), err)
	}

	// Delete old path
	_, err = tx.Exec(`DELETE FROM `+m.table+` WHERE path = ?`, oldKey)
	if err != nil {
		return core.E("sqlite.Rename", core.Concat("delete old path failed: ", oldKey), err)
	}

	// If it's a directory, move all children
	if isDir {
		oldPrefix := oldKey + "/"
		newPrefix := newKey + "/"

		rows, err := tx.Query(
			`SELECT path, content, mode, is_dir, mtime FROM `+m.table+` WHERE path LIKE ?`,
			oldPrefix+"%",
		)
		if err != nil {
			return core.E("sqlite.Rename", "query children failed", err)
		}

		type child struct {
			path    string
			content []byte
			mode    int
			isDir   bool
			mtime   time.Time
		}
		var children []child
		for rows.Next() {
			var c child
			if err := rows.Scan(&c.path, &c.content, &c.mode, &c.isDir, &c.mtime); err != nil {
				rows.Close()
				return core.E("sqlite.Rename", "scan child failed", err)
			}
			children = append(children, c)
		}
		rows.Close()

		for _, c := range children {
			newChildPath := core.Concat(newPrefix, core.TrimPrefix(c.path, oldPrefix))
			_, err = tx.Exec(
				`INSERT INTO `+m.table+` (path, content, mode, is_dir, mtime) VALUES (?, ?, ?, ?, ?)
				 ON CONFLICT(path) DO UPDATE SET content = excluded.content, mode = excluded.mode, is_dir = excluded.is_dir, mtime = excluded.mtime`,
				newChildPath, c.content, c.mode, c.isDir, c.mtime,
			)
			if err != nil {
				return core.E("sqlite.Rename", "insert child failed", err)
			}
		}

		// Delete old children
		_, err = tx.Exec(`DELETE FROM `+m.table+` WHERE path LIKE ?`, oldPrefix+"%")
		if err != nil {
			return core.E("sqlite.Rename", "delete old children failed", err)
		}
	}

	return tx.Commit()
}

// List returns the directory entries for the given path.
func (m *Medium) List(filePath string) ([]fs.DirEntry, error) {
	prefix := normaliseEntryPath(filePath)
	if prefix != "" {
		prefix += "/"
	}

	// Query all paths under the prefix
	rows, err := m.database.Query(
		`SELECT path, content, mode, is_dir, mtime FROM `+m.table+` WHERE path LIKE ? OR path LIKE ?`,
		prefix+"%", prefix+"%",
	)
	if err != nil {
		return nil, core.E("sqlite.List", "query failed", err)
	}
	defer rows.Close()

	seen := make(map[string]bool)
	var entries []fs.DirEntry

	for rows.Next() {
		var rowPath string
		var content []byte
		var mode int
		var isDir bool
		var mtime time.Time
		if err := rows.Scan(&rowPath, &content, &mode, &isDir, &mtime); err != nil {
			return nil, core.E("sqlite.List", "scan failed", err)
		}

		rest := core.TrimPrefix(rowPath, prefix)
		if rest == "" {
			continue
		}

		// Check if this is a direct child or nested
		parts := core.SplitN(rest, "/", 2)
		if len(parts) == 2 {
			// Nested - register as a directory
			dirName := parts[0]
			if !seen[dirName] {
				seen[dirName] = true
				entries = append(entries, &dirEntry{
					name:  dirName,
					isDir: true,
					mode:  fs.ModeDir | 0755,
					info: &fileInfo{
						name:  dirName,
						isDir: true,
						mode:  fs.ModeDir | 0755,
					},
				})
			}
		} else {
			// Direct child
			if !seen[rest] {
				seen[rest] = true
				entries = append(entries, &dirEntry{
					name:  rest,
					isDir: isDir,
					mode:  fs.FileMode(mode),
					info: &fileInfo{
						name:    rest,
						size:    int64(len(content)),
						mode:    fs.FileMode(mode),
						modTime: mtime,
						isDir:   isDir,
					},
				})
			}
		}
	}

	return entries, rows.Err()
}

func (m *Medium) Stat(filePath string) (fs.FileInfo, error) {
	key := normaliseEntryPath(filePath)
	if key == "" {
		return nil, core.E("sqlite.Stat", "path is required", fs.ErrInvalid)
	}

	var content []byte
	var mode int
	var isDir bool
	var mtime time.Time
	err := m.database.QueryRow(
		`SELECT content, mode, is_dir, mtime FROM `+m.table+` WHERE path = ?`, key,
	).Scan(&content, &mode, &isDir, &mtime)
	if err == sql.ErrNoRows {
		return nil, core.E("sqlite.Stat", core.Concat("path not found: ", key), fs.ErrNotExist)
	}
	if err != nil {
		return nil, core.E("sqlite.Stat", core.Concat("query failed: ", key), err)
	}

	name := path.Base(key)
	return &fileInfo{
		name:    name,
		size:    int64(len(content)),
		mode:    fs.FileMode(mode),
		modTime: mtime,
		isDir:   isDir,
	}, nil
}

func (m *Medium) Open(filePath string) (fs.File, error) {
	key := normaliseEntryPath(filePath)
	if key == "" {
		return nil, core.E("sqlite.Open", "path is required", fs.ErrInvalid)
	}

	var content []byte
	var mode int
	var isDir bool
	var mtime time.Time
	err := m.database.QueryRow(
		`SELECT content, mode, is_dir, mtime FROM `+m.table+` WHERE path = ?`, key,
	).Scan(&content, &mode, &isDir, &mtime)
	if err == sql.ErrNoRows {
		return nil, core.E("sqlite.Open", core.Concat("file not found: ", key), fs.ErrNotExist)
	}
	if err != nil {
		return nil, core.E("sqlite.Open", core.Concat("query failed: ", key), err)
	}
	if isDir {
		return nil, core.E("sqlite.Open", core.Concat("path is a directory: ", key), fs.ErrInvalid)
	}

	return &sqliteFile{
		name:    path.Base(key),
		content: content,
		mode:    fs.FileMode(mode),
		modTime: mtime,
	}, nil
}

func (m *Medium) Create(filePath string) (goio.WriteCloser, error) {
	key := normaliseEntryPath(filePath)
	if key == "" {
		return nil, core.E("sqlite.Create", "path is required", fs.ErrInvalid)
	}
	return &sqliteWriteCloser{
		medium: m,
		path:   key,
	}, nil
}

func (m *Medium) Append(filePath string) (goio.WriteCloser, error) {
	key := normaliseEntryPath(filePath)
	if key == "" {
		return nil, core.E("sqlite.Append", "path is required", fs.ErrInvalid)
	}

	var existing []byte
	err := m.database.QueryRow(
		`SELECT content FROM `+m.table+` WHERE path = ? AND is_dir = FALSE`, key,
	).Scan(&existing)
	if err != nil && err != sql.ErrNoRows {
		return nil, core.E("sqlite.Append", core.Concat("query failed: ", key), err)
	}

	return &sqliteWriteCloser{
		medium: m,
		path:   key,
		data:   existing,
	}, nil
}

func (m *Medium) ReadStream(filePath string) (goio.ReadCloser, error) {
	key := normaliseEntryPath(filePath)
	if key == "" {
		return nil, core.E("sqlite.ReadStream", "path is required", fs.ErrInvalid)
	}

	var content []byte
	var isDir bool
	err := m.database.QueryRow(
		`SELECT content, is_dir FROM `+m.table+` WHERE path = ?`, key,
	).Scan(&content, &isDir)
	if err == sql.ErrNoRows {
		return nil, core.E("sqlite.ReadStream", core.Concat("file not found: ", key), fs.ErrNotExist)
	}
	if err != nil {
		return nil, core.E("sqlite.ReadStream", core.Concat("query failed: ", key), err)
	}
	if isDir {
		return nil, core.E("sqlite.ReadStream", core.Concat("path is a directory: ", key), fs.ErrInvalid)
	}

	return goio.NopCloser(bytes.NewReader(content)), nil
}

func (m *Medium) WriteStream(filePath string) (goio.WriteCloser, error) {
	return m.Create(filePath)
}

func (m *Medium) Exists(filePath string) bool {
	key := normaliseEntryPath(filePath)
	if key == "" {
		// Root always exists
		return true
	}

	var count int
	err := m.database.QueryRow(
		`SELECT COUNT(*) FROM `+m.table+` WHERE path = ?`, key,
	).Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}

func (m *Medium) IsDir(filePath string) bool {
	key := normaliseEntryPath(filePath)
	if key == "" {
		return false
	}

	var isDir bool
	err := m.database.QueryRow(
		`SELECT is_dir FROM `+m.table+` WHERE path = ?`, key,
	).Scan(&isDir)
	if err != nil {
		return false
	}
	return isDir
}

// --- Internal types ---

// fileInfo implements fs.FileInfo for SQLite entries.
type fileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	isDir   bool
}

func (fi *fileInfo) Name() string { return fi.name }

func (fi *fileInfo) Size() int64 { return fi.size }

func (fi *fileInfo) Mode() fs.FileMode { return fi.mode }

func (fi *fileInfo) ModTime() time.Time { return fi.modTime }

func (fi *fileInfo) IsDir() bool { return fi.isDir }

func (fi *fileInfo) Sys() any { return nil }

// dirEntry implements fs.DirEntry for SQLite listings.
type dirEntry struct {
	name  string
	isDir bool
	mode  fs.FileMode
	info  fs.FileInfo
}

func (de *dirEntry) Name() string { return de.name }

func (de *dirEntry) IsDir() bool { return de.isDir }

func (de *dirEntry) Type() fs.FileMode { return de.mode.Type() }

func (de *dirEntry) Info() (fs.FileInfo, error) { return de.info, nil }

// sqliteFile implements fs.File for SQLite entries.
type sqliteFile struct {
	name    string
	content []byte
	offset  int64
	mode    fs.FileMode
	modTime time.Time
}

func (f *sqliteFile) Stat() (fs.FileInfo, error) {
	return &fileInfo{
		name:    f.name,
		size:    int64(len(f.content)),
		mode:    f.mode,
		modTime: f.modTime,
	}, nil
}

func (f *sqliteFile) Read(b []byte) (int, error) {
	if f.offset >= int64(len(f.content)) {
		return 0, goio.EOF
	}
	n := copy(b, f.content[f.offset:])
	f.offset += int64(n)
	return n, nil
}

func (f *sqliteFile) Close() error {
	return nil
}

// sqliteWriteCloser buffers writes and stores to SQLite on Close.
type sqliteWriteCloser struct {
	medium *Medium
	path   string
	data   []byte
}

func (w *sqliteWriteCloser) Write(p []byte) (int, error) {
	w.data = append(w.data, p...)
	return len(p), nil
}

func (w *sqliteWriteCloser) Close() error {
	_, err := w.medium.database.Exec(
		`INSERT INTO `+w.medium.table+` (path, content, mode, is_dir, mtime) VALUES (?, ?, 420, FALSE, ?)
		 ON CONFLICT(path) DO UPDATE SET content = excluded.content, is_dir = FALSE, mtime = excluded.mtime`,
		w.path, w.data, time.Now().UTC(),
	)
	if err != nil {
		return core.E("sqlite.WriteCloser.Close", core.Concat("store failed: ", w.path), err)
	}
	return nil
}
