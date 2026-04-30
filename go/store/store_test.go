package store

import (
	core "dappco.re/go"
)

func newKeyValueStore(t *core.T) *KeyValueStore {
	t.Helper()

	keyValueStore, err := New(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	t.Cleanup(func() {
		core.RequireNoError(t, keyValueStore.Close())
	})
	return keyValueStore
}

func TestKeyValueStore_New_OptionsGood(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	core.AssertNotNil(t, keyValueStore)
	count, err := keyValueStore.Count("empty")
	core.RequireNoError(t, err)
	core.AssertEqual(t, 0, count)
}

func TestKeyValueStore_New_Options_Bad(t *core.T) {
	keyValueStore, err := New(Options{})
	core.AssertNil(t, keyValueStore)
	core.AssertError(t, err)
	if err == nil {
		t.Fatal("expected empty database path to fail")
	}
	core.AssertContains(t, err.Error(), "database path is required")
}

func TestKeyValueStore_SetGetGood(t *core.T) {
	keyValueStore := newKeyValueStore(t)

	err := keyValueStore.Set("config", "theme", "dark")
	core.RequireNoError(t, err)

	value, err := keyValueStore.Get("config", "theme")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "dark", value)
}

func TestKeyValueStore_Get_NotFound_Bad(t *core.T) {
	keyValueStore := newKeyValueStore(t)

	_, err := keyValueStore.Get("config", "missing")
	core.AssertErrorIs(t, err, NotFoundError)
}

func TestKeyValueStore_Delete_Good(t *core.T) {
	keyValueStore := newKeyValueStore(t)

	_ = keyValueStore.Set("config", "key", "val")
	err := keyValueStore.Delete("config", "key")
	core.RequireNoError(t, err)

	_, err = keyValueStore.Get("config", "key")
	core.AssertErrorIs(t, err, NotFoundError)
}

func TestKeyValueStore_Count_Good(t *core.T) {
	keyValueStore := newKeyValueStore(t)

	_ = keyValueStore.Set("group", "a", "1")
	_ = keyValueStore.Set("group", "b", "2")
	_ = keyValueStore.Set("other", "c", "3")

	count, err := keyValueStore.Count("group")
	core.RequireNoError(t, err)
	core.AssertEqual(t, 2, count)
}

func TestKeyValueStore_DeleteGroup_Good(t *core.T) {
	keyValueStore := newKeyValueStore(t)

	_ = keyValueStore.Set("group", "a", "1")
	_ = keyValueStore.Set("group", "b", "2")
	err := keyValueStore.DeleteGroup("group")
	core.RequireNoError(t, err)

	count, _ := keyValueStore.Count("group")
	core.AssertEqual(t, 0, count)
}

func TestKeyValueStore_GetAll_Good(t *core.T) {
	keyValueStore := newKeyValueStore(t)

	_ = keyValueStore.Set("group", "a", "1")
	_ = keyValueStore.Set("group", "b", "2")
	_ = keyValueStore.Set("other", "c", "3")

	all, err := keyValueStore.GetAll("group")
	core.RequireNoError(t, err)
	core.AssertEqual(t, map[string]string{"a": "1", "b": "2"}, all)
}

func TestKeyValueStore_GetAll_Empty_Good(t *core.T) {
	keyValueStore := newKeyValueStore(t)

	all, err := keyValueStore.GetAll("empty")
	core.RequireNoError(t, err)
	core.AssertEmpty(t, all)
}

func TestKeyValueStore_Render_Good(t *core.T) {
	keyValueStore := newKeyValueStore(t)

	_ = keyValueStore.Set("user", "pool", "pool.lthn.io:3333")
	_ = keyValueStore.Set("user", "wallet", "iz...")

	templateText := `{"pool":"{{ .pool }}","wallet":"{{ .wallet }}"}`
	renderedText, err := keyValueStore.Render(templateText, "user")
	core.RequireNoError(t, err)
	core.AssertContains(t, renderedText, "pool.lthn.io:3333")
	core.AssertContains(t, renderedText, "iz...")
}

func TestStore_New_Good(t *core.T) {
	keyValueStore, err := New(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	t.Cleanup(func() {
		core.RequireNoError(t, keyValueStore.Close())
	})

	core.AssertNotNil(t, keyValueStore)
	core.AssertNotNil(t, keyValueStore.database)
}

func TestStore_New_Bad(t *core.T) {
	keyValueStore, err := New(Options{})
	core.AssertError(t, err)
	core.AssertNil(t, keyValueStore)
}

func TestStore_New_Ugly(t *core.T) {
	keyValueStore, err := New(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	core.AssertNoError(t, keyValueStore.Close())
}

func TestStore_KeyValueStore_Close_Good(t *core.T) {
	keyValueStore, err := New(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	closeErr := keyValueStore.Close()
	core.AssertNoError(t, closeErr)
}

func TestStore_KeyValueStore_Close_Bad(t *core.T) {
	keyValueStore, err := New(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	core.RequireNoError(t, keyValueStore.Close())
	closeErr := keyValueStore.Close()
	core.AssertNoError(t, closeErr)
}

func TestStore_KeyValueStore_Close_Ugly(t *core.T) {
	keyValueStore, err := New(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	core.RequireNoError(t, keyValueStore.Set("group", "key", "value"))
	core.AssertNoError(t, keyValueStore.Close())
}

func TestStore_KeyValueStore_Get_Good(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	core.RequireNoError(t, keyValueStore.Set("group", "key", "value"))
	got, err := keyValueStore.Get("group", "key")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "value", got)
}

func TestStore_KeyValueStore_Get_Bad(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	got, err := keyValueStore.Get("group", "missing")
	core.AssertErrorIs(t, err, NotFoundError)
	core.AssertEqual(t, "", got)
}

func TestStore_KeyValueStore_Get_Ugly(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	core.RequireNoError(t, keyValueStore.Set("", "", "value"))
	got, err := keyValueStore.Get("", "")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "value", got)
}

func TestStore_KeyValueStore_Set_Good(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	err := keyValueStore.Set("group", "key", "value")
	core.AssertNoError(t, err)
	core.AssertEqual(t, 1, mustCountStoreEntries(t, keyValueStore, "group"))
}

func TestStore_KeyValueStore_Set_Bad(t *core.T) {
	keyValueStore, err := New(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	core.RequireNoError(t, keyValueStore.Close())
	setErr := keyValueStore.Set("group", "key", "value")
	core.AssertError(t, setErr)
}

func TestStore_KeyValueStore_Set_Ugly(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	core.RequireNoError(t, keyValueStore.Set("group", "key", "old"))
	err := keyValueStore.Set("group", "key", "new")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "new", mustGetStoreEntry(t, keyValueStore, "group", "key"))
}

func TestStore_KeyValueStore_Delete_Good(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	core.RequireNoError(t, keyValueStore.Set("group", "key", "value"))
	err := keyValueStore.Delete("group", "key")
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, mustCountStoreEntries(t, keyValueStore, "group"))
}

func TestStore_KeyValueStore_Delete_Bad(t *core.T) {
	keyValueStore, err := New(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	core.RequireNoError(t, keyValueStore.Close())
	deleteErr := keyValueStore.Delete("group", "key")
	core.AssertError(t, deleteErr)
}

func TestStore_KeyValueStore_Delete_Ugly(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	err := keyValueStore.Delete("group", "missing")
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, mustCountStoreEntries(t, keyValueStore, "group"))
}

func TestStore_KeyValueStore_Count_Good(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	core.RequireNoError(t, keyValueStore.Set("group", "a", "1"))
	core.RequireNoError(t, keyValueStore.Set("group", "b", "2"))
	count, err := keyValueStore.Count("group")
	core.AssertNoError(t, err)
	core.AssertEqual(t, 2, count)
}

func TestStore_KeyValueStore_Count_Bad(t *core.T) {
	keyValueStore, err := New(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	core.RequireNoError(t, keyValueStore.Close())
	count, countErr := keyValueStore.Count("group")
	core.AssertError(t, countErr)
	core.AssertEqual(t, 0, count)
}

func TestStore_KeyValueStore_Count_Ugly(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	count, err := keyValueStore.Count("missing")
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, count)
}

func TestStore_KeyValueStore_DeleteGroup_Good(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	core.RequireNoError(t, keyValueStore.Set("group", "a", "1"))
	err := keyValueStore.DeleteGroup("group")
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, mustCountStoreEntries(t, keyValueStore, "group"))
}

func TestStore_KeyValueStore_DeleteGroup_Bad(t *core.T) {
	keyValueStore, err := New(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	core.RequireNoError(t, keyValueStore.Close())
	deleteErr := keyValueStore.DeleteGroup("group")
	core.AssertError(t, deleteErr)
}

func TestStore_KeyValueStore_DeleteGroup_Ugly(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	err := keyValueStore.DeleteGroup("missing")
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, mustCountStoreEntries(t, keyValueStore, "missing"))
}

func TestStore_KeyValueStore_ListGroups_Good(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	core.RequireNoError(t, keyValueStore.Set("b", "key", "1"))
	core.RequireNoError(t, keyValueStore.Set("a", "key", "2"))
	groups, err := keyValueStore.ListGroups()
	core.AssertNoError(t, err)
	core.AssertEqual(t, []string{"a", "b"}, groups)
}

func TestStore_KeyValueStore_ListGroups_Bad(t *core.T) {
	keyValueStore, err := New(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	core.RequireNoError(t, keyValueStore.Close())
	groups, listErr := keyValueStore.ListGroups()
	core.AssertError(t, listErr)
	core.AssertNil(t, groups)
}

func TestStore_KeyValueStore_ListGroups_Ugly(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	groups, err := keyValueStore.ListGroups()
	core.AssertNoError(t, err)
	core.AssertEmpty(t, groups)
}

func TestStore_KeyValueStore_GetAll_Good(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	core.RequireNoError(t, keyValueStore.Set("group", "a", "1"))
	core.RequireNoError(t, keyValueStore.Set("group", "b", "2"))
	all, err := keyValueStore.GetAll("group")
	core.AssertNoError(t, err)
	core.AssertEqual(t, map[string]string{"a": "1", "b": "2"}, all)
}

func TestStore_KeyValueStore_GetAll_Bad(t *core.T) {
	keyValueStore, err := New(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	core.RequireNoError(t, keyValueStore.Close())
	all, getErr := keyValueStore.GetAll("group")
	core.AssertError(t, getErr)
	core.AssertNil(t, all)
}

func TestStore_KeyValueStore_GetAll_Ugly(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	all, err := keyValueStore.GetAll("missing")
	core.AssertNoError(t, err)
	core.AssertEmpty(t, all)
}

func TestStore_KeyValueStore_Render_Good(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	core.RequireNoError(t, keyValueStore.Set("user", "name", "Ada"))
	rendered, err := keyValueStore.Render("hello {{ .name }}", "user")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "hello Ada", rendered)
}

func TestStore_KeyValueStore_Render_Bad(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	rendered, err := keyValueStore.Render("hello {{", "user")
	core.AssertError(t, err)
	core.AssertEqual(t, "", rendered)
}

func TestStore_KeyValueStore_Render_Ugly(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	rendered, err := keyValueStore.Render("missing {{ .name }}", "missing")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "missing <no value>", rendered)
}
