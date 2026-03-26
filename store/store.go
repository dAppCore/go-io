package store

import (
	"database/sql"
	"text/template"

	core "dappco.re/go/core"
	_ "modernc.org/sqlite"
)

// ErrNotFound is returned when a key does not exist in the store.
var ErrNotFound = core.E("store.ErrNotFound", "key not found", nil)

// Store is a group-namespaced key-value store backed by SQLite.
type Store struct {
	db *sql.DB
}

// New creates a Store at the given SQLite path. Use ":memory:" for tests.
//
// Example usage:
//
//	s, _ := store.New(":memory:")
//	_ = s.Set("app", "theme", "midnight")
func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, core.E("store.New", "open db", err)
	}
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, core.E("store.New", "WAL mode", err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS kv (
		grp   TEXT NOT NULL,
		key   TEXT NOT NULL,
		value TEXT NOT NULL,
		PRIMARY KEY (grp, key)
	)`); err != nil {
		db.Close()
		return nil, core.E("store.New", "create schema", err)
	}
	return &Store{db: db}, nil
}

// Close closes the underlying database.
//
//	result := s.Close(...)
func (s *Store) Close() error {
	return s.db.Close()
}

// Get retrieves a value by group and key.
//
//	result := s.Get(...)
func (s *Store) Get(group, key string) (string, error) {
	var val string
	err := s.db.QueryRow("SELECT value FROM kv WHERE grp = ? AND key = ?", group, key).Scan(&val)
	if err == sql.ErrNoRows {
		return "", core.E("store.Get", core.Concat("not found: ", group, "/", key), ErrNotFound)
	}
	if err != nil {
		return "", core.E("store.Get", "query", err)
	}
	return val, nil
}

// Set stores a value by group and key, overwriting if exists.
//
//	result := s.Set(...)
func (s *Store) Set(group, key, value string) error {
	_, err := s.db.Exec(
		`INSERT INTO kv (grp, key, value) VALUES (?, ?, ?)
		 ON CONFLICT(grp, key) DO UPDATE SET value = excluded.value`,
		group, key, value,
	)
	if err != nil {
		return core.E("store.Set", "exec", err)
	}
	return nil
}

// Delete removes a single key from a group.
//
//	result := s.Delete(...)
func (s *Store) Delete(group, key string) error {
	_, err := s.db.Exec("DELETE FROM kv WHERE grp = ? AND key = ?", group, key)
	if err != nil {
		return core.E("store.Delete", "exec", err)
	}
	return nil
}

// Count returns the number of keys in a group.
//
//	result := s.Count(...)
func (s *Store) Count(group string) (int, error) {
	var n int
	err := s.db.QueryRow("SELECT COUNT(*) FROM kv WHERE grp = ?", group).Scan(&n)
	if err != nil {
		return 0, core.E("store.Count", "query", err)
	}
	return n, nil
}

// DeleteGroup removes all keys in a group.
//
//	result := s.DeleteGroup(...)
func (s *Store) DeleteGroup(group string) error {
	_, err := s.db.Exec("DELETE FROM kv WHERE grp = ?", group)
	if err != nil {
		return core.E("store.DeleteGroup", "exec", err)
	}
	return nil
}

// GetAll returns all key-value pairs in a group.
//
//	result := s.GetAll(...)
func (s *Store) GetAll(group string) (map[string]string, error) {
	rows, err := s.db.Query("SELECT key, value FROM kv WHERE grp = ?", group)
	if err != nil {
		return nil, core.E("store.GetAll", "query", err)
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, core.E("store.GetAll", "scan", err)
		}
		result[k] = v
	}
	if err := rows.Err(); err != nil {
		return nil, core.E("store.GetAll", "rows", err)
	}
	return result, nil
}

// Render loads all key-value pairs from a group and renders a Go template.
//
// Example usage:
//
//	_ = s.Set("user", "name", "alice")
//	out, _ := s.Render("hello {{ .name }}", "user")
func (s *Store) Render(tmplStr, group string) (string, error) {
	rows, err := s.db.Query("SELECT key, value FROM kv WHERE grp = ?", group)
	if err != nil {
		return "", core.E("store.Render", "query", err)
	}
	defer rows.Close()

	vars := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return "", core.E("store.Render", "scan", err)
		}
		vars[k] = v
	}
	if err := rows.Err(); err != nil {
		return "", core.E("store.Render", "rows", err)
	}

	tmpl, err := template.New("render").Parse(tmplStr)
	if err != nil {
		return "", core.E("store.Render", "parse template", err)
	}
	b := core.NewBuilder()
	if err := tmpl.Execute(b, vars); err != nil {
		return "", core.E("store.Render", "execute template", err)
	}
	return b.String(), nil
}
