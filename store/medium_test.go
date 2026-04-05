package store

import (
	"io"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newKeyValueMedium(t *testing.T) *Medium {
	t.Helper()
	keyValueMedium, err := NewMedium(Options{Path: ":memory:"})
	require.NoError(t, err)
	t.Cleanup(func() { keyValueMedium.Close() })
	return keyValueMedium
}

func TestKeyValueMedium_ReadWrite_Good(t *testing.T) {
	keyValueMedium := newKeyValueMedium(t)

	err := keyValueMedium.Write("config/theme", "dark")
	require.NoError(t, err)

	value, err := keyValueMedium.Read("config/theme")
	require.NoError(t, err)
	assert.Equal(t, "dark", value)
}

func TestKeyValueMedium_Read_NoKey_Bad(t *testing.T) {
	keyValueMedium := newKeyValueMedium(t)
	_, err := keyValueMedium.Read("config")
	assert.Error(t, err)
}

func TestKeyValueMedium_Read_NotFound_Bad(t *testing.T) {
	keyValueMedium := newKeyValueMedium(t)
	_, err := keyValueMedium.Read("config/missing")
	assert.Error(t, err)
}

func TestKeyValueMedium_IsFile_Good(t *testing.T) {
	keyValueMedium := newKeyValueMedium(t)
	_ = keyValueMedium.Write("group/key", "val")

	assert.True(t, keyValueMedium.IsFile("group/key"))
	assert.False(t, keyValueMedium.IsFile("group/nope"))
	assert.False(t, keyValueMedium.IsFile("group"))
}

func TestKeyValueMedium_Delete_Good(t *testing.T) {
	keyValueMedium := newKeyValueMedium(t)
	_ = keyValueMedium.Write("group/key", "val")

	err := keyValueMedium.Delete("group/key")
	require.NoError(t, err)
	assert.False(t, keyValueMedium.IsFile("group/key"))
}

func TestKeyValueMedium_Delete_NonEmptyGroup_Bad(t *testing.T) {
	keyValueMedium := newKeyValueMedium(t)
	_ = keyValueMedium.Write("group/key", "val")

	err := keyValueMedium.Delete("group")
	assert.Error(t, err)
}

func TestKeyValueMedium_DeleteAll_Good(t *testing.T) {
	keyValueMedium := newKeyValueMedium(t)
	_ = keyValueMedium.Write("group/a", "1")
	_ = keyValueMedium.Write("group/b", "2")

	err := keyValueMedium.DeleteAll("group")
	require.NoError(t, err)
	assert.False(t, keyValueMedium.Exists("group"))
}

func TestKeyValueMedium_Rename_Good(t *testing.T) {
	keyValueMedium := newKeyValueMedium(t)
	_ = keyValueMedium.Write("old/key", "val")

	err := keyValueMedium.Rename("old/key", "new/key")
	require.NoError(t, err)

	value, err := keyValueMedium.Read("new/key")
	require.NoError(t, err)
	assert.Equal(t, "val", value)
	assert.False(t, keyValueMedium.IsFile("old/key"))
}

func TestKeyValueMedium_List_Groups_Good(t *testing.T) {
	keyValueMedium := newKeyValueMedium(t)
	_ = keyValueMedium.Write("alpha/a", "1")
	_ = keyValueMedium.Write("beta/b", "2")

	entries, err := keyValueMedium.List("")
	require.NoError(t, err)
	assert.Len(t, entries, 2)

	names := make(map[string]bool)
	for _, entry := range entries {
		names[entry.Name()] = true
		assert.True(t, entry.IsDir())
	}
	assert.True(t, names["alpha"])
	assert.True(t, names["beta"])
}

func TestKeyValueMedium_List_Keys_Good(t *testing.T) {
	keyValueMedium := newKeyValueMedium(t)
	_ = keyValueMedium.Write("group/a", "1")
	_ = keyValueMedium.Write("group/b", "22")

	entries, err := keyValueMedium.List("group")
	require.NoError(t, err)
	assert.Len(t, entries, 2)
}

func TestKeyValueMedium_Stat_Good(t *testing.T) {
	keyValueMedium := newKeyValueMedium(t)
	_ = keyValueMedium.Write("group/key", "hello")

	info, err := keyValueMedium.Stat("group")
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	info, err = keyValueMedium.Stat("group/key")
	require.NoError(t, err)
	assert.Equal(t, int64(5), info.Size())
	assert.False(t, info.IsDir())
}

func TestKeyValueMedium_Exists_IsDir_Good(t *testing.T) {
	keyValueMedium := newKeyValueMedium(t)
	_ = keyValueMedium.Write("group/key", "val")

	assert.True(t, keyValueMedium.Exists("group"))
	assert.True(t, keyValueMedium.Exists("group/key"))
	assert.True(t, keyValueMedium.IsDir("group"))
	assert.False(t, keyValueMedium.IsDir("group/key"))
	assert.False(t, keyValueMedium.Exists("nope"))
}

func TestKeyValueMedium_Open_Read_Good(t *testing.T) {
	keyValueMedium := newKeyValueMedium(t)
	_ = keyValueMedium.Write("group/key", "hello world")

	file, err := keyValueMedium.Open("group/key")
	require.NoError(t, err)
	defer file.Close()

	data, err := io.ReadAll(file)
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(data))
}

func TestKeyValueMedium_CreateClose_Good(t *testing.T) {
	keyValueMedium := newKeyValueMedium(t)

	writer, err := keyValueMedium.Create("group/key")
	require.NoError(t, err)
	_, _ = writer.Write([]byte("streamed"))
	require.NoError(t, writer.Close())

	value, err := keyValueMedium.Read("group/key")
	require.NoError(t, err)
	assert.Equal(t, "streamed", value)
}

func TestKeyValueMedium_Append_Good(t *testing.T) {
	keyValueMedium := newKeyValueMedium(t)
	_ = keyValueMedium.Write("group/key", "hello")

	writer, err := keyValueMedium.Append("group/key")
	require.NoError(t, err)
	_, _ = writer.Write([]byte(" world"))
	require.NoError(t, writer.Close())

	value, err := keyValueMedium.Read("group/key")
	require.NoError(t, err)
	assert.Equal(t, "hello world", value)
}

func TestKeyValueMedium_AsMedium_Good(t *testing.T) {
	keyValueStore := newKeyValueStore(t)

	keyValueMedium := keyValueStore.AsMedium()
	require.NoError(t, keyValueMedium.Write("group/key", "val"))

	value, err := keyValueStore.Get("group", "key")
	require.NoError(t, err)
	assert.Equal(t, "val", value)

	value, err = keyValueMedium.Read("group/key")
	require.NoError(t, err)
	assert.Equal(t, "val", value)
}

func TestKeyValueMedium_KeyValueStore_Good(t *testing.T) {
	keyValueMedium := newKeyValueMedium(t)

	assert.NotNil(t, keyValueMedium.KeyValueStore())
	assert.Same(t, keyValueMedium.KeyValueStore(), keyValueMedium.KeyValueStore())
}

func TestKeyValueMedium_EnsureDir_ReadWrite_Good(t *testing.T) {
	keyValueMedium := newKeyValueMedium(t)

	require.NoError(t, keyValueMedium.EnsureDir("ignored"))
	require.NoError(t, keyValueMedium.Write("group/key", "value"))

	value, err := keyValueMedium.Read("group/key")
	require.NoError(t, err)
	assert.Equal(t, "value", value)
}

func TestKeyValueMedium_StreamHelpers_Good(t *testing.T) {
	keyValueMedium := newKeyValueMedium(t)

	writer, err := keyValueMedium.WriteStream("group/key")
	require.NoError(t, err)
	_, err = writer.Write([]byte("streamed"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	reader, err := keyValueMedium.ReadStream("group/key")
	require.NoError(t, err)
	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, "streamed", string(data))
	require.NoError(t, reader.Close())

	file, err := keyValueMedium.Open("group/key")
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

	entries, err := keyValueMedium.List("group")
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
