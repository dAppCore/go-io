package store

import (
	"database/sql"
	"errors"
	"io/fs"
	"text/template"

	core "dappco.re/go/core"
	_ "modernc.org/sqlite"
)

// Example: _, err := keyValueStore.Get("app", "theme")
var NotFoundError = errors.New("key not found")

// Example: keyValueStore, _ := store.New(store.Options{Path: ":memory:"})
type Store struct {
	database *sql.DB
}

type Options struct {
	Path string
}

// Example: keyValueStore, _ := store.New(store.Options{Path: ":memory:"})
// Example: _ = keyValueStore.Set("app", "theme", "midnight")
func New(options Options) (*Store, error) {
	if options.Path == "" {
		return nil, core.E("store.New", "database path is required", fs.ErrInvalid)
	}

	database, err := sql.Open("sqlite", options.Path)
	if err != nil {
		return nil, core.E("store.New", "open db", err)
	}
	if _, err := database.Exec("PRAGMA journal_mode=WAL"); err != nil {
		database.Close()
		return nil, core.E("store.New", "WAL mode", err)
	}
	if _, err := database.Exec(`CREATE TABLE IF NOT EXISTS entries (
		group_name TEXT NOT NULL,
		entry_key  TEXT NOT NULL,
		entry_value TEXT NOT NULL,
		PRIMARY KEY (group_name, entry_key)
	)`); err != nil {
		database.Close()
		return nil, core.E("store.New", "create schema", err)
	}
	return &Store{database: database}, nil
}

// Example: _ = keyValueStore.Close()
func (store *Store) Close() error {
	return store.database.Close()
}

// Example: theme, _ := keyValueStore.Get("app", "theme")
func (store *Store) Get(group, key string) (string, error) {
	var value string
	err := store.database.QueryRow("SELECT entry_value FROM entries WHERE group_name = ? AND entry_key = ?", group, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", core.E("store.Get", core.Concat("not found: ", group, "/", key), NotFoundError)
	}
	if err != nil {
		return "", core.E("store.Get", "query", err)
	}
	return value, nil
}

// Example: _ = keyValueStore.Set("app", "theme", "midnight")
func (store *Store) Set(group, key, value string) error {
	_, err := store.database.Exec(
		`INSERT INTO entries (group_name, entry_key, entry_value) VALUES (?, ?, ?)
		 ON CONFLICT(group_name, entry_key) DO UPDATE SET entry_value = excluded.entry_value`,
		group, key, value,
	)
	if err != nil {
		return core.E("store.Set", "exec", err)
	}
	return nil
}

// Example: _ = keyValueStore.Delete("app", "theme")
func (store *Store) Delete(group, key string) error {
	_, err := store.database.Exec("DELETE FROM entries WHERE group_name = ? AND entry_key = ?", group, key)
	if err != nil {
		return core.E("store.Delete", "exec", err)
	}
	return nil
}

// Example: count, _ := keyValueStore.Count("app")
func (store *Store) Count(group string) (int, error) {
	var count int
	err := store.database.QueryRow("SELECT COUNT(*) FROM entries WHERE group_name = ?", group).Scan(&count)
	if err != nil {
		return 0, core.E("store.Count", "query", err)
	}
	return count, nil
}

// Example: _ = keyValueStore.DeleteGroup("app")
func (store *Store) DeleteGroup(group string) error {
	_, err := store.database.Exec("DELETE FROM entries WHERE group_name = ?", group)
	if err != nil {
		return core.E("store.DeleteGroup", "exec", err)
	}
	return nil
}

// Example: values, _ := keyValueStore.GetAll("app")
func (store *Store) GetAll(group string) (map[string]string, error) {
	rows, err := store.database.Query("SELECT entry_key, entry_value FROM entries WHERE group_name = ?", group)
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

// Example: keyValueStore, _ := store.New(store.Options{Path: ":memory:"})
// Example: _ = keyValueStore.Set("user", "name", "alice")
// Example: out, _ := keyValueStore.Render("hello {{ .name }}", "user")
func (store *Store) Render(templateText, group string) (string, error) {
	rows, err := store.database.Query("SELECT entry_key, entry_value FROM entries WHERE group_name = ?", group)
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
