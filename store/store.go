package store

import (
	"database/sql"
	"io/fs"
	"text/template" // Note: AX-6 intrinsic - structural for KeyValueStore.Render templating; core exposes no template primitive.

	core "dappco.re/go"
	_ "modernc.org/sqlite"
)

// NotFoundError is the sentinel returned when a key does not exist in the store.
// Callers test for it with core.Is. It is defined with core.NewError so that
// identity comparison works correctly across package boundaries.
// Example: _, err := keyValueStore.Get("app", "theme"); core.Is(err, store.NotFoundError)
var NotFoundError = core.NewError("store: key not found")

const (
	opStoreNew        = "store.New"
	opStoreListGroups = "store.ListGroups"
	opStoreGetAll     = "store.GetAll"
	opStoreRender     = "store.Render"
)

func closeStoreDatabase(database *sql.DB, operation string) {
	if err := database.Close(); err != nil {
		core.Warn("store database close failed", "op", operation, "err", err)
	}
}

func closeStoreRows(rows *sql.Rows, operation string) {
	if err := rows.Close(); err != nil {
		core.Warn("store rows close failed", "op", operation, "err", err)
	}
}

// Example: keyValueStore, _ := store.New(store.Options{Path: ":memory:"})
type KeyValueStore struct {
	database *sql.DB
}

// Example: keyValueStore, _ := store.New(store.Options{Path: ":memory:"})
type Options struct {
	Path string
}

// Example: keyValueStore, _ := store.New(store.Options{Path: ":memory:"})
// Example: _ = keyValueStore.Set("app", "theme", "midnight")
func New(options Options) (*KeyValueStore, error) {
	if options.Path == "" {
		return nil, core.E(opStoreNew, "database path is required", fs.ErrInvalid)
	}

	database, err := sql.Open("sqlite", options.Path)
	if err != nil {
		return nil, core.E(opStoreNew, "open db", err)
	}
	if _, err := database.Exec("PRAGMA journal_mode=WAL"); err != nil {
		closeStoreDatabase(database, opStoreNew)
		return nil, core.E(opStoreNew, "WAL mode", err)
	}
	if _, err := database.Exec(`CREATE TABLE IF NOT EXISTS entries (
		group_name TEXT NOT NULL,
		entry_key  TEXT NOT NULL,
		entry_value TEXT NOT NULL,
		PRIMARY KEY (group_name, entry_key)
	)`); err != nil {
		closeStoreDatabase(database, opStoreNew)
		return nil, core.E(opStoreNew, "create schema", err)
	}
	return &KeyValueStore{database: database}, nil
}

// Example: _ = keyValueStore.Close()
func (keyValueStore *KeyValueStore) Close() error {
	return keyValueStore.database.Close()
}

// Example: theme, _ := keyValueStore.Get("app", "theme")
func (keyValueStore *KeyValueStore) Get(group, key string) (string, error) {
	var value string
	err := keyValueStore.database.QueryRow("SELECT entry_value FROM entries WHERE group_name = ? AND entry_key = ?", group, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", core.E("store.Get", core.Concat("not found: ", group, "/", key), NotFoundError)
	}
	if err != nil {
		return "", core.E("store.Get", "query", err)
	}
	return value, nil
}

// Example: _ = keyValueStore.Set("app", "theme", "midnight")
func (keyValueStore *KeyValueStore) Set(group, key, value string) error {
	_, err := keyValueStore.database.Exec(
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
func (keyValueStore *KeyValueStore) Delete(group, key string) error {
	_, err := keyValueStore.database.Exec("DELETE FROM entries WHERE group_name = ? AND entry_key = ?", group, key)
	if err != nil {
		return core.E("store.Delete", "exec", err)
	}
	return nil
}

// Example: count, _ := keyValueStore.Count("app")
func (keyValueStore *KeyValueStore) Count(group string) (int, error) {
	var count int
	err := keyValueStore.database.QueryRow("SELECT COUNT(*) FROM entries WHERE group_name = ?", group).Scan(&count)
	if err != nil {
		return 0, core.E("store.Count", "query", err)
	}
	return count, nil
}

// Example: _ = keyValueStore.DeleteGroup("app")
func (keyValueStore *KeyValueStore) DeleteGroup(group string) error {
	_, err := keyValueStore.database.Exec("DELETE FROM entries WHERE group_name = ?", group)
	if err != nil {
		return core.E("store.DeleteGroup", "exec", err)
	}
	return nil
}

// Example: groups, _ := keyValueStore.ListGroups()
func (keyValueStore *KeyValueStore) ListGroups() ([]string, error) {
	rows, err := keyValueStore.database.Query("SELECT DISTINCT group_name FROM entries ORDER BY group_name")
	if err != nil {
		return nil, core.E(opStoreListGroups, "query groups", err)
	}
	defer closeStoreRows(rows, opStoreListGroups)

	var groups []string
	for rows.Next() {
		var groupName string
		if err := rows.Scan(&groupName); err != nil {
			return nil, core.E(opStoreListGroups, "scan", err)
		}
		groups = append(groups, groupName)
	}
	if err := rows.Err(); err != nil {
		return nil, core.E(opStoreListGroups, "rows", err)
	}
	return groups, nil
}

// Example: values, _ := keyValueStore.GetAll("app")
func (keyValueStore *KeyValueStore) GetAll(group string) (map[string]string, error) {
	rows, err := keyValueStore.database.Query("SELECT entry_key, entry_value FROM entries WHERE group_name = ?", group)
	if err != nil {
		return nil, core.E(opStoreGetAll, "query", err)
	}
	defer closeStoreRows(rows, opStoreGetAll)

	result := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, core.E(opStoreGetAll, "scan", err)
		}
		result[key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, core.E(opStoreGetAll, "rows", err)
	}
	return result, nil
}

// Example: keyValueStore, _ := store.New(store.Options{Path: ":memory:"})
// Example: _ = keyValueStore.Set("user", "name", "alice")
// Example: renderedText, _ := keyValueStore.Render("hello {{ .name }}", "user")
func (keyValueStore *KeyValueStore) Render(templateText, group string) (string, error) {
	rows, err := keyValueStore.database.Query("SELECT entry_key, entry_value FROM entries WHERE group_name = ?", group)
	if err != nil {
		return "", core.E(opStoreRender, "query", err)
	}
	defer closeStoreRows(rows, opStoreRender)

	templateValues := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return "", core.E(opStoreRender, "scan", err)
		}
		templateValues[key] = value
	}
	if err := rows.Err(); err != nil {
		return "", core.E(opStoreRender, "rows", err)
	}

	renderTemplate, err := template.New("render").Parse(templateText)
	if err != nil {
		return "", core.E(opStoreRender, "parse template", err)
	}
	builder := core.NewBuilder()
	if err := renderTemplate.Execute(builder, templateValues); err != nil {
		return "", core.E(opStoreRender, "execute template", err)
	}
	return builder.String(), nil
}
