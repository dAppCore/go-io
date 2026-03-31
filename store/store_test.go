package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestKeyValueStore(t *testing.T) *KeyValueStore {
	t.Helper()

	keyValueStore, err := New(Options{Path: ":memory:"})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, keyValueStore.Close())
	})
	return keyValueStore
}

func TestKeyValueStore_New_Options_Good(t *testing.T) {
	keyValueStore := newTestKeyValueStore(t)
	assert.NotNil(t, keyValueStore)
}

func TestKeyValueStore_New_Options_Bad(t *testing.T) {
	_, err := New(Options{})
	assert.Error(t, err)
}

func TestKeyValueStore_SetGet_Good(t *testing.T) {
	keyValueStore := newTestKeyValueStore(t)

	err := keyValueStore.Set("config", "theme", "dark")
	require.NoError(t, err)

	value, err := keyValueStore.Get("config", "theme")
	require.NoError(t, err)
	assert.Equal(t, "dark", value)
}

func TestKeyValueStore_Get_NotFound_Bad(t *testing.T) {
	keyValueStore := newTestKeyValueStore(t)

	_, err := keyValueStore.Get("config", "missing")
	assert.ErrorIs(t, err, NotFoundError)
}

func TestKeyValueStore_Delete_Good(t *testing.T) {
	keyValueStore := newTestKeyValueStore(t)

	_ = keyValueStore.Set("config", "key", "val")
	err := keyValueStore.Delete("config", "key")
	require.NoError(t, err)

	_, err = keyValueStore.Get("config", "key")
	assert.ErrorIs(t, err, NotFoundError)
}

func TestKeyValueStore_Count_Good(t *testing.T) {
	keyValueStore := newTestKeyValueStore(t)

	_ = keyValueStore.Set("group", "a", "1")
	_ = keyValueStore.Set("group", "b", "2")
	_ = keyValueStore.Set("other", "c", "3")

	count, err := keyValueStore.Count("group")
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestKeyValueStore_DeleteGroup_Good(t *testing.T) {
	keyValueStore := newTestKeyValueStore(t)

	_ = keyValueStore.Set("group", "a", "1")
	_ = keyValueStore.Set("group", "b", "2")
	err := keyValueStore.DeleteGroup("group")
	require.NoError(t, err)

	count, _ := keyValueStore.Count("group")
	assert.Equal(t, 0, count)
}

func TestKeyValueStore_GetAll_Good(t *testing.T) {
	keyValueStore := newTestKeyValueStore(t)

	_ = keyValueStore.Set("group", "a", "1")
	_ = keyValueStore.Set("group", "b", "2")
	_ = keyValueStore.Set("other", "c", "3")

	all, err := keyValueStore.GetAll("group")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"a": "1", "b": "2"}, all)
}

func TestKeyValueStore_GetAll_Empty_Good(t *testing.T) {
	keyValueStore := newTestKeyValueStore(t)

	all, err := keyValueStore.GetAll("empty")
	require.NoError(t, err)
	assert.Empty(t, all)
}

func TestKeyValueStore_Render_Good(t *testing.T) {
	keyValueStore := newTestKeyValueStore(t)

	_ = keyValueStore.Set("user", "pool", "pool.lthn.io:3333")
	_ = keyValueStore.Set("user", "wallet", "iz...")

	templateText := `{"pool":"{{ .pool }}","wallet":"{{ .wallet }}"}`
	renderedText, err := keyValueStore.Render(templateText, "user")
	require.NoError(t, err)
	assert.Contains(t, renderedText, "pool.lthn.io:3333")
	assert.Contains(t, renderedText, "iz...")
}
