package store

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestMedium(t *testing.T) *Medium {
	t.Helper()
	m, err := NewMedium(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { m.Close() })
	return m
}

func TestMedium_ReadWrite_Good(t *testing.T) {
	m := newTestMedium(t)

	err := m.Write("config/theme", "dark")
	require.NoError(t, err)

	val, err := m.Read("config/theme")
	require.NoError(t, err)
	assert.Equal(t, "dark", val)
}

func TestMedium_Read_Bad_NoKey(t *testing.T) {
	m := newTestMedium(t)
	_, err := m.Read("config")
	assert.Error(t, err)
}

func TestMedium_Read_Bad_NotFound(t *testing.T) {
	m := newTestMedium(t)
	_, err := m.Read("config/missing")
	assert.Error(t, err)
}

func TestMedium_IsFile_Good(t *testing.T) {
	m := newTestMedium(t)
	_ = m.Write("grp/key", "val")

	assert.True(t, m.IsFile("grp/key"))
	assert.False(t, m.IsFile("grp/nope"))
	assert.False(t, m.IsFile("grp"))
}

func TestMedium_Delete_Good(t *testing.T) {
	m := newTestMedium(t)
	_ = m.Write("grp/key", "val")

	err := m.Delete("grp/key")
	require.NoError(t, err)
	assert.False(t, m.IsFile("grp/key"))
}

func TestMedium_Delete_Bad_NonEmptyGroup(t *testing.T) {
	m := newTestMedium(t)
	_ = m.Write("grp/key", "val")

	err := m.Delete("grp")
	assert.Error(t, err)
}

func TestMedium_DeleteAll_Good(t *testing.T) {
	m := newTestMedium(t)
	_ = m.Write("grp/a", "1")
	_ = m.Write("grp/b", "2")

	err := m.DeleteAll("grp")
	require.NoError(t, err)
	assert.False(t, m.Exists("grp"))
}

func TestMedium_Rename_Good(t *testing.T) {
	m := newTestMedium(t)
	_ = m.Write("old/key", "val")

	err := m.Rename("old/key", "new/key")
	require.NoError(t, err)

	val, err := m.Read("new/key")
	require.NoError(t, err)
	assert.Equal(t, "val", val)
	assert.False(t, m.IsFile("old/key"))
}

func TestMedium_List_Good_Groups(t *testing.T) {
	m := newTestMedium(t)
	_ = m.Write("alpha/a", "1")
	_ = m.Write("beta/b", "2")

	entries, err := m.List("")
	require.NoError(t, err)
	assert.Len(t, entries, 2)

	names := make(map[string]bool)
	for _, e := range entries {
		names[e.Name()] = true
		assert.True(t, e.IsDir())
	}
	assert.True(t, names["alpha"])
	assert.True(t, names["beta"])
}

func TestMedium_List_Good_Keys(t *testing.T) {
	m := newTestMedium(t)
	_ = m.Write("grp/a", "1")
	_ = m.Write("grp/b", "22")

	entries, err := m.List("grp")
	require.NoError(t, err)
	assert.Len(t, entries, 2)
}

func TestMedium_Stat_Good(t *testing.T) {
	m := newTestMedium(t)
	_ = m.Write("grp/key", "hello")

	// Stat group
	info, err := m.Stat("grp")
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Stat key
	info, err = m.Stat("grp/key")
	require.NoError(t, err)
	assert.Equal(t, int64(5), info.Size())
	assert.False(t, info.IsDir())
}

func TestMedium_Exists_IsDir_Good(t *testing.T) {
	m := newTestMedium(t)
	_ = m.Write("grp/key", "val")

	assert.True(t, m.Exists("grp"))
	assert.True(t, m.Exists("grp/key"))
	assert.True(t, m.IsDir("grp"))
	assert.False(t, m.IsDir("grp/key"))
	assert.False(t, m.Exists("nope"))
}

func TestMedium_Open_Read_Good(t *testing.T) {
	m := newTestMedium(t)
	_ = m.Write("grp/key", "hello world")

	f, err := m.Open("grp/key")
	require.NoError(t, err)
	defer f.Close()

	data, err := io.ReadAll(f)
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(data))
}

func TestMedium_CreateClose_Good(t *testing.T) {
	m := newTestMedium(t)

	w, err := m.Create("grp/key")
	require.NoError(t, err)
	_, _ = w.Write([]byte("streamed"))
	require.NoError(t, w.Close())

	val, err := m.Read("grp/key")
	require.NoError(t, err)
	assert.Equal(t, "streamed", val)
}

func TestMedium_Append_Good(t *testing.T) {
	m := newTestMedium(t)
	_ = m.Write("grp/key", "hello")

	w, err := m.Append("grp/key")
	require.NoError(t, err)
	_, _ = w.Write([]byte(" world"))
	require.NoError(t, w.Close())

	val, err := m.Read("grp/key")
	require.NoError(t, err)
	assert.Equal(t, "hello world", val)
}

func TestMedium_AsMedium_Good(t *testing.T) {
	s, err := New(":memory:")
	require.NoError(t, err)
	defer s.Close()

	m := s.AsMedium()
	require.NoError(t, m.Write("grp/key", "val"))

	// Accessible through both APIs
	val, err := s.Get("grp", "key")
	require.NoError(t, err)
	assert.Equal(t, "val", val)

	val, err = m.Read("grp/key")
	require.NoError(t, err)
	assert.Equal(t, "val", val)
}
