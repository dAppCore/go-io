// Example: medium, _ := sqlite.New(sqlite.Options{Path: ":memory:"})
// Example: _ = medium.Write("config/app.yaml", "port: 8080")
package sqlite

import (
	"bytes"
	"database/sql"
	goio "io"
	"io/fs"
	"path"
	"time"

	core "dappco.re/go/core"
	coreio "dappco.re/go/io"

	_ "modernc.org/sqlite"
)

// Example: medium, _ := sqlite.New(sqlite.Options{Path: ":memory:"})
// Example: _ = medium.Write("config/app.yaml", "port: 8080")
type Medium struct {
	database *sql.DB
	table    string
}

var _ coreio.Medium = (*Medium)(nil)

// Example: medium, _ := sqlite.New(sqlite.Options{Path: ":memory:", Table: "files"})
type Options struct {
	Path  string
	Table string
}

func normaliseTableName(table string) string {
	if table == "" {
		return "files"
	}
	return table
}

// isValidTableName reports whether name consists only of ASCII letters, digits, and underscores,
// starting with a letter or underscore. This prevents SQL-injection via table-name concatenation.
func isValidTableName(name string) bool {
	if name == "" {
		return false
	}
	for i, ch := range name {
		switch {
		case ch >= 'a' && ch <= 'z', ch >= 'A' && ch <= 'Z', ch == '_':
			// always valid
		case ch >= '0' && ch <= '9':
			if i == 0 {
				return false // must not start with a digit
			}
		default:
			return false
		}
	}
	return true
}

// Example: medium, _ := sqlite.New(sqlite.Options{Path: ":memory:", Table: "files"})
// Example: _ = medium.Write("config/app.yaml", "port: 8080")
func New(options Options) (*Medium, error) {
	if options.Path == "" {
		return nil, core.E("sqlite.New", "database path is required", fs.ErrInvalid)
	}

	tableName := normaliseTableName(options.Table)
	if !isValidTableName(tableName) {
		return nil, core.E("sqlite.New", core.Concat("table name contains invalid characters: ", tableName), fs.ErrInvalid)
	}

	medium := &Medium{table: tableName}

	database, err := sql.Open("sqlite", options.Path)
	if err != nil {
		return nil, core.E("sqlite.New", "failed to open database", err)
	}

	if _, err := database.Exec("PRAGMA journal_mode=WAL"); err != nil {
		database.Close()
		return nil, core.E("sqlite.New", "failed to set WAL mode", err)
	}

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

// Example: _ = medium.Close()
func (medium *Medium) Close() error {
	if medium.database != nil {
		return medium.database.Close()
	}
	return nil
}

func normaliseEntryPath(filePath string) string {
	clean := path.Clean("/" + filePath)
	if clean == "/" {
		return ""
	}
	return core.TrimPrefix(clean, "/")
}

// Example: content, _ := medium.Read("config/app.yaml")
func (medium *Medium) Read(filePath string) (string, error) {
	key := normaliseEntryPath(filePath)
	if key == "" {
		return "", core.E("sqlite.Read", "path is required", fs.ErrInvalid)
	}

	var content []byte
	var isDir bool
	err := medium.database.QueryRow(
		`SELECT content, is_dir FROM `+medium.table+` WHERE path = ?`, key,
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

// Example: _ = medium.Write("config/app.yaml", "port: 8080")
func (medium *Medium) Write(filePath, content string) error {
	return medium.WriteMode(filePath, content, 0644)
}

// Example: _ = medium.WriteMode("keys/private.key", key, 0600)
func (medium *Medium) WriteMode(filePath, content string, mode fs.FileMode) error {
	key := normaliseEntryPath(filePath)
	if key == "" {
		return core.E("sqlite.WriteMode", "path is required", fs.ErrInvalid)
	}

	_, err := medium.database.Exec(
		`INSERT INTO `+medium.table+` (path, content, mode, is_dir, mtime) VALUES (?, ?, ?, FALSE, ?)
		 ON CONFLICT(path) DO UPDATE SET content = excluded.content, mode = excluded.mode, is_dir = FALSE, mtime = excluded.mtime`,
		key, []byte(content), int(mode), time.Now().UTC(),
	)
	if err != nil {
		return core.E("sqlite.WriteMode", core.Concat("insert failed: ", key), err)
	}
	return nil
}

// Example: _ = medium.EnsureDir("config")
func (medium *Medium) EnsureDir(filePath string) error {
	key := normaliseEntryPath(filePath)
	if key == "" {
		return nil
	}

	_, err := medium.database.Exec(
		`INSERT INTO `+medium.table+` (path, content, mode, is_dir, mtime) VALUES (?, '', 493, TRUE, ?)
		 ON CONFLICT(path) DO NOTHING`,
		key, time.Now().UTC(),
	)
	if err != nil {
		return core.E("sqlite.EnsureDir", core.Concat("insert failed: ", key), err)
	}
	return nil
}

// Example: isFile := medium.IsFile("config/app.yaml")
func (medium *Medium) IsFile(filePath string) bool {
	key := normaliseEntryPath(filePath)
	if key == "" {
		return false
	}

	var isDir bool
	err := medium.database.QueryRow(
		`SELECT is_dir FROM `+medium.table+` WHERE path = ?`, key,
	).Scan(&isDir)
	if err != nil {
		return false
	}
	return !isDir
}

// Example: _ = medium.Delete("config/app.yaml")
func (medium *Medium) Delete(filePath string) error {
	key := normaliseEntryPath(filePath)
	if key == "" {
		return core.E("sqlite.Delete", "path is required", fs.ErrInvalid)
	}

	var isDir bool
	err := medium.database.QueryRow(
		`SELECT is_dir FROM `+medium.table+` WHERE path = ?`, key,
	).Scan(&isDir)
	if err == sql.ErrNoRows {
		return core.E("sqlite.Delete", core.Concat("path not found: ", key), fs.ErrNotExist)
	}
	if err != nil {
		return core.E("sqlite.Delete", core.Concat("query failed: ", key), err)
	}

	if isDir {
		prefix := key + "/"
		var count int
		err := medium.database.QueryRow(
			`SELECT COUNT(*) FROM `+medium.table+` WHERE path LIKE ? AND path != ?`, prefix+"%", key,
		).Scan(&count)
		if err != nil {
			return core.E("sqlite.Delete", core.Concat("count failed: ", key), err)
		}
		if count > 0 {
			return core.E("sqlite.Delete", core.Concat("directory not empty: ", key), fs.ErrExist)
		}
	}

	execResult, err := medium.database.Exec(`DELETE FROM `+medium.table+` WHERE path = ?`, key)
	if err != nil {
		return core.E("sqlite.Delete", core.Concat("delete failed: ", key), err)
	}
	rowsAffected, _ := execResult.RowsAffected()
	if rowsAffected == 0 {
		return core.E("sqlite.Delete", core.Concat("path not found: ", key), fs.ErrNotExist)
	}
	return nil
}

// Example: _ = medium.DeleteAll("config")
func (medium *Medium) DeleteAll(filePath string) error {
	key := normaliseEntryPath(filePath)
	if key == "" {
		return core.E("sqlite.DeleteAll", "path is required", fs.ErrInvalid)
	}

	prefix := key + "/"

	execResult, err := medium.database.Exec(
		`DELETE FROM `+medium.table+` WHERE path = ? OR path LIKE ?`,
		key, prefix+"%",
	)
	if err != nil {
		return core.E("sqlite.DeleteAll", core.Concat("delete failed: ", key), err)
	}
	rowsAffected, _ := execResult.RowsAffected()
	if rowsAffected == 0 {
		return core.E("sqlite.DeleteAll", core.Concat("path not found: ", key), fs.ErrNotExist)
	}
	return nil
}

// Example: _ = medium.Rename("drafts/todo.txt", "archive/todo.txt")
func (medium *Medium) Rename(oldPath, newPath string) error {
	oldKey := normaliseEntryPath(oldPath)
	newKey := normaliseEntryPath(newPath)
	if oldKey == "" || newKey == "" {
		return core.E("sqlite.Rename", "both old and new paths are required", fs.ErrInvalid)
	}

	tx, err := medium.database.Begin()
	if err != nil {
		return core.E("sqlite.Rename", "begin tx failed", err)
	}
	defer tx.Rollback()

	var content []byte
	var mode int
	var isDir bool
	var mtime time.Time
	err = tx.QueryRow(
		`SELECT content, mode, is_dir, mtime FROM `+medium.table+` WHERE path = ?`, oldKey,
	).Scan(&content, &mode, &isDir, &mtime)
	if err == sql.ErrNoRows {
		return core.E("sqlite.Rename", core.Concat("source not found: ", oldKey), fs.ErrNotExist)
	}
	if err != nil {
		return core.E("sqlite.Rename", core.Concat("query failed: ", oldKey), err)
	}

	_, err = tx.Exec(
		`INSERT INTO `+medium.table+` (path, content, mode, is_dir, mtime) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(path) DO UPDATE SET content = excluded.content, mode = excluded.mode, is_dir = excluded.is_dir, mtime = excluded.mtime`,
		newKey, content, mode, isDir, mtime,
	)
	if err != nil {
		return core.E("sqlite.Rename", core.Concat("insert at new path failed: ", newKey), err)
	}

	_, err = tx.Exec(`DELETE FROM `+medium.table+` WHERE path = ?`, oldKey)
	if err != nil {
		return core.E("sqlite.Rename", core.Concat("delete old path failed: ", oldKey), err)
	}

	if isDir {
		oldPrefix := oldKey + "/"
		newPrefix := newKey + "/"

		childRows, err := tx.Query(
			`SELECT path, content, mode, is_dir, mtime FROM `+medium.table+` WHERE path LIKE ?`,
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
		for childRows.Next() {
			var childEntry child
			if err := childRows.Scan(&childEntry.path, &childEntry.content, &childEntry.mode, &childEntry.isDir, &childEntry.mtime); err != nil {
				childRows.Close()
				return core.E("sqlite.Rename", "scan child failed", err)
			}
			children = append(children, childEntry)
		}
		childRows.Close()

		for _, childEntry := range children {
			newChildPath := core.Concat(newPrefix, core.TrimPrefix(childEntry.path, oldPrefix))
			_, err = tx.Exec(
				`INSERT INTO `+medium.table+` (path, content, mode, is_dir, mtime) VALUES (?, ?, ?, ?, ?)
				 ON CONFLICT(path) DO UPDATE SET content = excluded.content, mode = excluded.mode, is_dir = excluded.is_dir, mtime = excluded.mtime`,
				newChildPath, childEntry.content, childEntry.mode, childEntry.isDir, childEntry.mtime,
			)
			if err != nil {
				return core.E("sqlite.Rename", "insert child failed", err)
			}
		}

		_, err = tx.Exec(`DELETE FROM `+medium.table+` WHERE path LIKE ?`, oldPrefix+"%")
		if err != nil {
			return core.E("sqlite.Rename", "delete old children failed", err)
		}
	}

	return tx.Commit()
}

// Example: entries, _ := medium.List("config")
func (medium *Medium) List(filePath string) ([]fs.DirEntry, error) {
	prefix := normaliseEntryPath(filePath)
	if prefix != "" {
		prefix += "/"
	}

	rows, err := medium.database.Query(
		`SELECT path, content, mode, is_dir, mtime FROM `+medium.table+` WHERE path LIKE ?`,
		prefix+"%",
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

		parts := core.SplitN(rest, "/", 2)
		if len(parts) == 2 {
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

	if err := rows.Err(); err != nil {
		return nil, core.E("sqlite.List", "rows", err)
	}
	return entries, nil
}

// Example: info, _ := medium.Stat("config/app.yaml")
func (medium *Medium) Stat(filePath string) (fs.FileInfo, error) {
	key := normaliseEntryPath(filePath)
	if key == "" {
		return nil, core.E("sqlite.Stat", "path is required", fs.ErrInvalid)
	}

	var content []byte
	var mode int
	var isDir bool
	var mtime time.Time
	err := medium.database.QueryRow(
		`SELECT content, mode, is_dir, mtime FROM `+medium.table+` WHERE path = ?`, key,
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

// Example: file, _ := medium.Open("config/app.yaml")
func (medium *Medium) Open(filePath string) (fs.File, error) {
	key := normaliseEntryPath(filePath)
	if key == "" {
		return nil, core.E("sqlite.Open", "path is required", fs.ErrInvalid)
	}

	var content []byte
	var mode int
	var isDir bool
	var mtime time.Time
	err := medium.database.QueryRow(
		`SELECT content, mode, is_dir, mtime FROM `+medium.table+` WHERE path = ?`, key,
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

// Example: writer, _ := medium.Create("logs/app.log")
func (medium *Medium) Create(filePath string) (goio.WriteCloser, error) {
	key := normaliseEntryPath(filePath)
	if key == "" {
		return nil, core.E("sqlite.Create", "path is required", fs.ErrInvalid)
	}
	return &sqliteWriteCloser{
		medium: medium,
		path:   key,
	}, nil
}

// Example: writer, _ := medium.Append("logs/app.log")
func (medium *Medium) Append(filePath string) (goio.WriteCloser, error) {
	key := normaliseEntryPath(filePath)
	if key == "" {
		return nil, core.E("sqlite.Append", "path is required", fs.ErrInvalid)
	}

	var existing []byte
	err := medium.database.QueryRow(
		`SELECT content FROM `+medium.table+` WHERE path = ? AND is_dir = FALSE`, key,
	).Scan(&existing)
	if err != nil && err != sql.ErrNoRows {
		return nil, core.E("sqlite.Append", core.Concat("query failed: ", key), err)
	}

	return &sqliteWriteCloser{
		medium: medium,
		path:   key,
		data:   existing,
	}, nil
}

// Example: reader, _ := medium.ReadStream("logs/app.log")
func (medium *Medium) ReadStream(filePath string) (goio.ReadCloser, error) {
	key := normaliseEntryPath(filePath)
	if key == "" {
		return nil, core.E("sqlite.ReadStream", "path is required", fs.ErrInvalid)
	}

	var content []byte
	var isDir bool
	err := medium.database.QueryRow(
		`SELECT content, is_dir FROM `+medium.table+` WHERE path = ?`, key,
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

// Example: writer, _ := medium.WriteStream("logs/app.log")
func (medium *Medium) WriteStream(filePath string) (goio.WriteCloser, error) {
	return medium.Create(filePath)
}

// Example: exists := medium.Exists("config/app.yaml")
func (medium *Medium) Exists(filePath string) bool {
	key := normaliseEntryPath(filePath)
	if key == "" {
		return true
	}

	var count int
	err := medium.database.QueryRow(
		`SELECT COUNT(*) FROM `+medium.table+` WHERE path = ?`, key,
	).Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}

// Example: isDirectory := medium.IsDir("config")
func (medium *Medium) IsDir(filePath string) bool {
	key := normaliseEntryPath(filePath)
	if key == "" {
		return false
	}

	var isDir bool
	err := medium.database.QueryRow(
		`SELECT is_dir FROM `+medium.table+` WHERE path = ?`, key,
	).Scan(&isDir)
	if err != nil {
		return false
	}
	return isDir
}

type fileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	isDir   bool
}

func (info *fileInfo) Name() string { return info.name }

func (info *fileInfo) Size() int64 { return info.size }

func (info *fileInfo) Mode() fs.FileMode { return info.mode }

func (info *fileInfo) ModTime() time.Time { return info.modTime }

func (info *fileInfo) IsDir() bool { return info.isDir }

func (info *fileInfo) Sys() any { return nil }

type dirEntry struct {
	name  string
	isDir bool
	mode  fs.FileMode
	info  fs.FileInfo
}

func (entry *dirEntry) Name() string { return entry.name }

func (entry *dirEntry) IsDir() bool { return entry.isDir }

func (entry *dirEntry) Type() fs.FileMode { return entry.mode.Type() }

func (entry *dirEntry) Info() (fs.FileInfo, error) { return entry.info, nil }

type sqliteFile struct {
	name    string
	content []byte
	offset  int64
	mode    fs.FileMode
	modTime time.Time
}

func (file *sqliteFile) Stat() (fs.FileInfo, error) {
	return &fileInfo{
		name:    file.name,
		size:    int64(len(file.content)),
		mode:    file.mode,
		modTime: file.modTime,
	}, nil
}

func (file *sqliteFile) Read(buffer []byte) (int, error) {
	if file.offset >= int64(len(file.content)) {
		return 0, goio.EOF
	}
	bytesRead := copy(buffer, file.content[file.offset:])
	file.offset += int64(bytesRead)
	return bytesRead, nil
}

func (file *sqliteFile) Close() error {
	return nil
}

type sqliteWriteCloser struct {
	medium *Medium
	path   string
	data   []byte
	mode   fs.FileMode
}

func (writer *sqliteWriteCloser) Write(data []byte) (int, error) {
	writer.data = append(writer.data, data...)
	return len(data), nil
}

func (writer *sqliteWriteCloser) Close() error {
	mode := writer.mode
	if mode == 0 {
		mode = 0644
	}
	_, err := writer.medium.database.Exec(
		`INSERT INTO `+writer.medium.table+` (path, content, mode, is_dir, mtime) VALUES (?, ?, ?, FALSE, ?)
		 ON CONFLICT(path) DO UPDATE SET content = excluded.content, mode = excluded.mode, is_dir = FALSE, mtime = excluded.mtime`,
		writer.path, writer.data, int(mode), time.Now().UTC(),
	)
	if err != nil {
		return core.E("sqlite.WriteCloser.Close", core.Concat("store failed: ", writer.path), err)
	}
	return nil
}
