package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()

	keyValueStore, err := New(Options{Path: ":memory:"})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, keyValueStore.Close())
	})
	return keyValueStore
}

func TestStore_New_Options_Good(t *testing.T) {
	keyValueStore := newTestStore(t)
	assert.NotNil(t, keyValueStore)
}

func TestStore_New_Options_Bad(t *testing.T) {
	_, err := New(Options{})
	assert.Error(t, err)
}

func TestStore_SetGet_Good(t *testing.T) {
	keyValueStore := newTestStore(t)

	err := keyValueStore.Set("config", "theme", "dark")
	require.NoError(t, err)

	val, err := keyValueStore.Get("config", "theme")
	require.NoError(t, err)
	assert.Equal(t, "dark", val)
}

func TestStore_Get_NotFound_Bad(t *testing.T) {
	keyValueStore := newTestStore(t)

	_, err := keyValueStore.Get("config", "missing")
	assert.ErrorIs(t, err, NotFoundError)
}

func TestStore_Delete_Good(t *testing.T) {
	keyValueStore := newTestStore(t)

	_ = keyValueStore.Set("config", "key", "val")
	err := keyValueStore.Delete("config", "key")
	require.NoError(t, err)

	_, err = keyValueStore.Get("config", "key")
	assert.ErrorIs(t, err, NotFoundError)
}

func TestStore_Count_Good(t *testing.T) {
	keyValueStore := newTestStore(t)

	_ = keyValueStore.Set("group", "a", "1")
	_ = keyValueStore.Set("group", "b", "2")
	_ = keyValueStore.Set("other", "c", "3")

	count, err := keyValueStore.Count("group")
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestStore_DeleteGroup_Good(t *testing.T) {
	keyValueStore := newTestStore(t)

	_ = keyValueStore.Set("group", "a", "1")
	_ = keyValueStore.Set("group", "b", "2")
	err := keyValueStore.DeleteGroup("group")
	require.NoError(t, err)

	count, _ := keyValueStore.Count("group")
	assert.Equal(t, 0, count)
}

func TestStore_GetAll_Good(t *testing.T) {
	keyValueStore := newTestStore(t)

	_ = keyValueStore.Set("group", "a", "1")
	_ = keyValueStore.Set("group", "b", "2")
	_ = keyValueStore.Set("other", "c", "3")

	all, err := keyValueStore.GetAll("group")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"a": "1", "b": "2"}, all)
}

func TestStore_GetAll_Empty_Good(t *testing.T) {
	keyValueStore := newTestStore(t)

	all, err := keyValueStore.GetAll("empty")
	require.NoError(t, err)
	assert.Empty(t, all)
}

func TestStore_Render_Good(t *testing.T) {
	keyValueStore := newTestStore(t)

	_ = keyValueStore.Set("user", "pool", "pool.lthn.io:3333")
	_ = keyValueStore.Set("user", "wallet", "iz...")

	tmpl := `{"pool":"{{ .pool }}","wallet":"{{ .wallet }}"}`
	out, err := keyValueStore.Render(tmpl, "user")
	require.NoError(t, err)
	assert.Contains(t, out, "pool.lthn.io:3333")
	assert.Contains(t, out, "iz...")
}
