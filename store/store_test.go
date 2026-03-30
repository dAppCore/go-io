package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()

	s, err := New(Options{Path: ":memory:"})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, s.Close())
	})
	return s
}

func TestStore_New_Options_Good(t *testing.T) {
	s := newTestStore(t)
	assert.NotNil(t, s)
}

func TestStore_New_Options_Bad(t *testing.T) {
	_, err := New(Options{})
	assert.Error(t, err)
}

func TestStore_SetGet_Good(t *testing.T) {
	s := newTestStore(t)

	err := s.Set("config", "theme", "dark")
	require.NoError(t, err)

	val, err := s.Get("config", "theme")
	require.NoError(t, err)
	assert.Equal(t, "dark", val)
}

func TestStore_Get_NotFound_Bad(t *testing.T) {
	s := newTestStore(t)

	_, err := s.Get("config", "missing")
	assert.ErrorIs(t, err, NotFoundError)
}

func TestStore_Delete_Good(t *testing.T) {
	s := newTestStore(t)

	_ = s.Set("config", "key", "val")
	err := s.Delete("config", "key")
	require.NoError(t, err)

	_, err = s.Get("config", "key")
	assert.ErrorIs(t, err, NotFoundError)
}

func TestStore_Count_Good(t *testing.T) {
	s := newTestStore(t)

	_ = s.Set("grp", "a", "1")
	_ = s.Set("grp", "b", "2")
	_ = s.Set("other", "c", "3")

	n, err := s.Count("grp")
	require.NoError(t, err)
	assert.Equal(t, 2, n)
}

func TestStore_DeleteGroup_Good(t *testing.T) {
	s := newTestStore(t)

	_ = s.Set("grp", "a", "1")
	_ = s.Set("grp", "b", "2")
	err := s.DeleteGroup("grp")
	require.NoError(t, err)

	n, _ := s.Count("grp")
	assert.Equal(t, 0, n)
}

func TestStore_GetAll_Good(t *testing.T) {
	s := newTestStore(t)

	_ = s.Set("grp", "a", "1")
	_ = s.Set("grp", "b", "2")
	_ = s.Set("other", "c", "3")

	all, err := s.GetAll("grp")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"a": "1", "b": "2"}, all)
}

func TestStore_GetAll_Empty_Good(t *testing.T) {
	s := newTestStore(t)

	all, err := s.GetAll("empty")
	require.NoError(t, err)
	assert.Empty(t, all)
}

func TestStore_Render_Good(t *testing.T) {
	s := newTestStore(t)

	_ = s.Set("user", "pool", "pool.lthn.io:3333")
	_ = s.Set("user", "wallet", "iz...")

	tmpl := `{"pool":"{{ .pool }}","wallet":"{{ .wallet }}"}`
	out, err := s.Render(tmpl, "user")
	require.NoError(t, err)
	assert.Contains(t, out, "pool.lthn.io:3333")
	assert.Contains(t, out, "iz...")
}
