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
	database *sql.DB
}

// Use New to open a SQLite-backed key-value store.
// Use ":memory:" for tests.
//
// Example usage:
//
//	kvStore, _ := store.New(":memory:")
//	_ = kvStore.Set("app", "theme", "midnight")
func New(dbPath string) (*Store, error) {
	database, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, core.E("store.New", "open db", err)
	}
	if _, err := database.Exec("PRAGMA journal_mode=WAL"); err != nil {
		database.Close()
		return nil, core.E("store.New", "WAL mode", err)
	}
	if _, err := database.Exec(`CREATE TABLE IF NOT EXISTS kv (
		grp   TEXT NOT NULL,
		key   TEXT NOT NULL,
		value TEXT NOT NULL,
		PRIMARY KEY (grp, key)
	)`); err != nil {
		database.Close()
		return nil, core.E("store.New", "create schema", err)
	}
	return &Store{database: database}, nil
}

// Example: _ = kvStore.Close()
func (s *Store) Close() error {
	return s.database.Close()
}

// Example: theme, _ := kvStore.Get("app", "theme")
func (s *Store) Get(group, key string) (string, error) {
	var value string
	err := s.database.QueryRow("SELECT value FROM kv WHERE grp = ? AND key = ?", group, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", core.E("store.Get", core.Concat("not found: ", group, "/", key), ErrNotFound)
	}
	if err != nil {
		return "", core.E("store.Get", "query", err)
	}
	return value, nil
}

// Example: _ = kvStore.Set("app", "theme", "midnight")
func (s *Store) Set(group, key, value string) error {
	_, err := s.database.Exec(
		`INSERT INTO kv (grp, key, value) VALUES (?, ?, ?)
		 ON CONFLICT(grp, key) DO UPDATE SET value = excluded.value`,
		group, key, value,
	)
	if err != nil {
		return core.E("store.Set", "exec", err)
	}
	return nil
}

// Example: _ = kvStore.Delete("app", "theme")
func (s *Store) Delete(group, key string) error {
	_, err := s.database.Exec("DELETE FROM kv WHERE grp = ? AND key = ?", group, key)
	if err != nil {
		return core.E("store.Delete", "exec", err)
	}
	return nil
}

// Example: count, _ := kvStore.Count("app")
func (s *Store) Count(group string) (int, error) {
	var count int
	err := s.database.QueryRow("SELECT COUNT(*) FROM kv WHERE grp = ?", group).Scan(&count)
	if err != nil {
		return 0, core.E("store.Count", "query", err)
	}
	return count, nil
}

// Example: _ = kvStore.DeleteGroup("app")
func (s *Store) DeleteGroup(group string) error {
	_, err := s.database.Exec("DELETE FROM kv WHERE grp = ?", group)
	if err != nil {
		return core.E("store.DeleteGroup", "exec", err)
	}
	return nil
}

// Example: values, _ := kvStore.GetAll("app")
func (s *Store) GetAll(group string) (map[string]string, error) {
	rows, err := s.database.Query("SELECT key, value FROM kv WHERE grp = ?", group)
	if err != nil {
		return nil, core.E("store.GetAll", "query", err)
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, core.E("store.GetAll", "scan", err)
		}
		result[key] = value
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
//	kvStore, _ := store.New(":memory:")
//	_ = kvStore.Set("user", "name", "alice")
//	out, _ := kvStore.Render("hello {{ .name }}", "user")
func (s *Store) Render(templateText, group string) (string, error) {
	rows, err := s.database.Query("SELECT key, value FROM kv WHERE grp = ?", group)
	if err != nil {
		return "", core.E("store.Render", "query", err)
	}
	defer rows.Close()

	templateValues := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return "", core.E("store.Render", "scan", err)
		}
		templateValues[key] = value
	}
	if err := rows.Err(); err != nil {
		return "", core.E("store.Render", "rows", err)
	}

	tmpl, err := template.New("render").Parse(templateText)
	if err != nil {
		return "", core.E("store.Render", "parse template", err)
	}
	builder := core.NewBuilder()
	if err := tmpl.Execute(builder, templateValues); err != nil {
		return "", core.E("store.Render", "execute template", err)
	}
	return builder.String(), nil
}
