package store

import (
	"io"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestKeyValueMedium(t *testing.T) *Medium {
	t.Helper()
	medium, err := NewMedium(Options{Path: ":memory:"})
	require.NoError(t, err)
	t.Cleanup(func() { medium.Close() })
	return medium
}

func TestKeyValueMedium_ReadWrite_Good(t *testing.T) {
	m := newTestKeyValueMedium(t)

	err := m.Write("config/theme", "dark")
	require.NoError(t, err)

	val, err := m.Read("config/theme")
	require.NoError(t, err)
	assert.Equal(t, "dark", val)
}

func TestKeyValueMedium_Read_NoKey_Bad(t *testing.T) {
	m := newTestKeyValueMedium(t)
	_, err := m.Read("config")
	assert.Error(t, err)
}

func TestKeyValueMedium_Read_NotFound_Bad(t *testing.T) {
	m := newTestKeyValueMedium(t)
	_, err := m.Read("config/missing")
	assert.Error(t, err)
}

func TestKeyValueMedium_IsFile_Good(t *testing.T) {
	m := newTestKeyValueMedium(t)
	_ = m.Write("group/key", "val")

	assert.True(t, m.IsFile("group/key"))
	assert.False(t, m.IsFile("group/nope"))
	assert.False(t, m.IsFile("group"))
}

func TestKeyValueMedium_Delete_Good(t *testing.T) {
	m := newTestKeyValueMedium(t)
	_ = m.Write("group/key", "val")

	err := m.Delete("group/key")
	require.NoError(t, err)
	assert.False(t, m.IsFile("group/key"))
}

func TestKeyValueMedium_Delete_NonEmptyGroup_Bad(t *testing.T) {
	m := newTestKeyValueMedium(t)
	_ = m.Write("group/key", "val")

	err := m.Delete("group")
	assert.Error(t, err)
}

func TestKeyValueMedium_DeleteAll_Good(t *testing.T) {
	m := newTestKeyValueMedium(t)
	_ = m.Write("group/a", "1")
	_ = m.Write("group/b", "2")

	err := m.DeleteAll("group")
	require.NoError(t, err)
	assert.False(t, m.Exists("group"))
}

func TestKeyValueMedium_Rename_Good(t *testing.T) {
	m := newTestKeyValueMedium(t)
	_ = m.Write("old/key", "val")

	err := m.Rename("old/key", "new/key")
	require.NoError(t, err)

	val, err := m.Read("new/key")
	require.NoError(t, err)
	assert.Equal(t, "val", val)
	assert.False(t, m.IsFile("old/key"))
}

func TestKeyValueMedium_List_Groups_Good(t *testing.T) {
	m := newTestKeyValueMedium(t)
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

func TestKeyValueMedium_List_Keys_Good(t *testing.T) {
	m := newTestKeyValueMedium(t)
	_ = m.Write("group/a", "1")
	_ = m.Write("group/b", "22")

	entries, err := m.List("group")
	require.NoError(t, err)
	assert.Len(t, entries, 2)
}

func TestKeyValueMedium_Stat_Good(t *testing.T) {
	m := newTestKeyValueMedium(t)
	_ = m.Write("group/key", "hello")

	info, err := m.Stat("group")
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	info, err = m.Stat("group/key")
	require.NoError(t, err)
	assert.Equal(t, int64(5), info.Size())
	assert.False(t, info.IsDir())
}

func TestKeyValueMedium_Exists_IsDir_Good(t *testing.T) {
	m := newTestKeyValueMedium(t)
	_ = m.Write("group/key", "val")

	assert.True(t, m.Exists("group"))
	assert.True(t, m.Exists("group/key"))
	assert.True(t, m.IsDir("group"))
	assert.False(t, m.IsDir("group/key"))
	assert.False(t, m.Exists("nope"))
}

func TestKeyValueMedium_Open_Read_Good(t *testing.T) {
	m := newTestKeyValueMedium(t)
	_ = m.Write("group/key", "hello world")

	f, err := m.Open("group/key")
	require.NoError(t, err)
	defer f.Close()

	data, err := io.ReadAll(f)
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(data))
}

func TestKeyValueMedium_CreateClose_Good(t *testing.T) {
	m := newTestKeyValueMedium(t)

	w, err := m.Create("group/key")
	require.NoError(t, err)
	_, _ = w.Write([]byte("streamed"))
	require.NoError(t, w.Close())

	val, err := m.Read("group/key")
	require.NoError(t, err)
	assert.Equal(t, "streamed", val)
}

func TestKeyValueMedium_Append_Good(t *testing.T) {
	m := newTestKeyValueMedium(t)
	_ = m.Write("group/key", "hello")

	w, err := m.Append("group/key")
	require.NoError(t, err)
	_, _ = w.Write([]byte(" world"))
	require.NoError(t, w.Close())

	val, err := m.Read("group/key")
	require.NoError(t, err)
	assert.Equal(t, "hello world", val)
}

func TestKeyValueMedium_AsMedium_Good(t *testing.T) {
	s := newTestStore(t)

	m := s.AsMedium()
	require.NoError(t, m.Write("group/key", "val"))

	val, err := s.Get("group", "key")
	require.NoError(t, err)
	assert.Equal(t, "val", val)

	val, err = m.Read("group/key")
	require.NoError(t, err)
	assert.Equal(t, "val", val)
}

func TestKeyValueMedium_Store_Good(t *testing.T) {
	m := newTestKeyValueMedium(t)

	assert.NotNil(t, m.Store())
	assert.Same(t, m.Store(), m.Store())
}

func TestKeyValueMedium_EnsureDir_ReadWrite_Good(t *testing.T) {
	m := newTestKeyValueMedium(t)

	require.NoError(t, m.EnsureDir("ignored"))
	require.NoError(t, m.Write("group/key", "value"))

	value, err := m.Read("group/key")
	require.NoError(t, err)
	assert.Equal(t, "value", value)
}

func TestKeyValueMedium_StreamHelpers_Good(t *testing.T) {
	m := newTestKeyValueMedium(t)

	writer, err := m.WriteStream("group/key")
	require.NoError(t, err)
	_, err = writer.Write([]byte("streamed"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	reader, err := m.ReadStream("group/key")
	require.NoError(t, err)
	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, "streamed", string(data))
	require.NoError(t, reader.Close())

	file, err := m.Open("group/key")
	require.NoError(t, err)
	info, err := file.Stat()
	require.NoError(t, err)
	assert.Equal(t, "key", info.Name())
	assert.Equal(t, int64(8), info.Size())
	assert.Equal(t, fs.FileMode(0644), info.Mode())
	assert.True(t, info.ModTime().IsZero())
	assert.False(t, info.IsDir())
	assert.Nil(t, info.Sys())
	require.NoError(t, file.Close())

	entries, err := m.List("group")
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "key", entries[0].Name())
	assert.False(t, entries[0].IsDir())
	assert.Equal(t, fs.FileMode(0), entries[0].Type())

	entryInfo, err := entries[0].Info()
	require.NoError(t, err)
	assert.Equal(t, "key", entryInfo.Name())
	assert.Equal(t, int64(8), entryInfo.Size())
}
