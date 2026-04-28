package store

import (
	core "dappco.re/go"
	"io"
	"io/fs"
)

func newKeyValueMedium(t *core.T) *Medium {
	t.Helper()
	keyValueMedium, err := NewMedium(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	t.Cleanup(func() { keyValueMedium.Close() })
	return keyValueMedium
}

func TestKeyValueMedium_ReadWrite_Good(t *core.T) {
	keyValueMedium := newKeyValueMedium(t)

	err := keyValueMedium.Write("config/theme", "dark")
	core.RequireNoError(t, err)

	value, err := keyValueMedium.Read("config/theme")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "dark", value)
}

func TestKeyValueMedium_Read_NoKey_Bad(t *core.T) {
	keyValueMedium := newKeyValueMedium(t)
	_, err := keyValueMedium.Read("config")
	core.AssertError(t, err)
}

func TestKeyValueMedium_Read_NotFound_Bad(t *core.T) {
	keyValueMedium := newKeyValueMedium(t)
	_, err := keyValueMedium.Read("config/missing")
	core.AssertError(t, err)
}

func TestKeyValueMedium_IsFile_Good(t *core.T) {
	keyValueMedium := newKeyValueMedium(t)
	_ = keyValueMedium.Write("group/key", "val")

	core.AssertTrue(t, keyValueMedium.IsFile("group/key"))
	core.AssertFalse(t, keyValueMedium.IsFile("group/nope"))
	core.AssertFalse(t, keyValueMedium.IsFile("group"))
}

func TestKeyValueMedium_Delete_Good(t *core.T) {
	keyValueMedium := newKeyValueMedium(t)
	_ = keyValueMedium.Write("group/key", "val")

	err := keyValueMedium.Delete("group/key")
	core.RequireNoError(t, err)
	core.AssertFalse(t, keyValueMedium.IsFile("group/key"))
}

func TestKeyValueMedium_Delete_NonEmptyGroup_Bad(t *core.T) {
	keyValueMedium := newKeyValueMedium(t)
	_ = keyValueMedium.Write("group/key", "val")

	err := keyValueMedium.Delete("group")
	core.AssertError(t, err)
}

func TestKeyValueMedium_DeleteAll_Good(t *core.T) {
	keyValueMedium := newKeyValueMedium(t)
	_ = keyValueMedium.Write("group/a", "1")
	_ = keyValueMedium.Write("group/b", "2")

	err := keyValueMedium.DeleteAll("group")
	core.RequireNoError(t, err)
	core.AssertFalse(t, keyValueMedium.Exists("group"))
}

func TestKeyValueMedium_Rename_Good(t *core.T) {
	keyValueMedium := newKeyValueMedium(t)
	_ = keyValueMedium.Write("old/key", "val")

	err := keyValueMedium.Rename("old/key", "new/key")
	core.RequireNoError(t, err)

	value, err := keyValueMedium.Read("new/key")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "val", value)
	core.AssertFalse(t, keyValueMedium.IsFile("old/key"))
}

func TestKeyValueMedium_List_Groups_Good(t *core.T) {
	keyValueMedium := newKeyValueMedium(t)
	_ = keyValueMedium.Write("alpha/a", "1")
	_ = keyValueMedium.Write("beta/b", "2")

	entries, err := keyValueMedium.List("")
	core.RequireNoError(t, err)
	core.AssertLen(t, entries, 2)

	names := make(map[string]bool)
	for _, entry := range entries {
		names[entry.Name()] = true
		core.AssertTrue(t, entry.IsDir())
	}
	core.AssertTrue(t, names["alpha"])
	core.AssertTrue(t, names["beta"])
}

func TestKeyValueMedium_List_Keys_Good(t *core.T) {
	keyValueMedium := newKeyValueMedium(t)
	_ = keyValueMedium.Write("group/a", "1")
	_ = keyValueMedium.Write("group/b", "22")

	entries, err := keyValueMedium.List("group")
	core.RequireNoError(t, err)
	core.AssertLen(t, entries, 2)
}

func TestKeyValueMedium_Stat_Good(t *core.T) {
	keyValueMedium := newKeyValueMedium(t)
	_ = keyValueMedium.Write("group/key", "hello")

	info, err := keyValueMedium.Stat("group")
	core.RequireNoError(t, err)
	core.AssertTrue(t, info.IsDir())

	info, err = keyValueMedium.Stat("group/key")
	core.RequireNoError(t, err)
	core.AssertEqual(t, int64(5), info.Size())
	core.AssertFalse(t, info.IsDir())
}

func TestKeyValueMedium_Exists_IsDir_Good(t *core.T) {
	keyValueMedium := newKeyValueMedium(t)
	_ = keyValueMedium.Write("group/key", "val")

	core.AssertTrue(t, keyValueMedium.Exists("group"))
	core.AssertTrue(t, keyValueMedium.Exists("group/key"))
	core.AssertTrue(t, keyValueMedium.IsDir("group"))
	core.AssertFalse(t, keyValueMedium.IsDir("group/key"))
	core.AssertFalse(t, keyValueMedium.Exists("nope"))
}

func TestKeyValueMedium_Open_Read_Good(t *core.T) {
	keyValueMedium := newKeyValueMedium(t)
	_ = keyValueMedium.Write("group/key", "hello world")

	file, err := keyValueMedium.Open("group/key")
	core.RequireNoError(t, err)
	defer file.Close()

	data, err := io.ReadAll(file)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "hello world", string(data))
}

func TestKeyValueMedium_CreateClose_Good(t *core.T) {
	keyValueMedium := newKeyValueMedium(t)

	writer, err := keyValueMedium.Create("group/key")
	core.RequireNoError(t, err)
	_, _ = writer.Write([]byte("streamed"))
	core.RequireNoError(t, writer.Close())

	value, err := keyValueMedium.Read("group/key")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "streamed", value)
}

func TestKeyValueMedium_Append_Good(t *core.T) {
	keyValueMedium := newKeyValueMedium(t)
	_ = keyValueMedium.Write("group/key", "hello")

	writer, err := keyValueMedium.Append("group/key")
	core.RequireNoError(t, err)
	_, _ = writer.Write([]byte(" world"))
	core.RequireNoError(t, writer.Close())

	value, err := keyValueMedium.Read("group/key")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "hello world", value)
}

func TestKeyValueMedium_AsMedium_Good(t *core.T) {
	keyValueStore := newKeyValueStore(t)

	keyValueMedium := keyValueStore.AsMedium()
	core.RequireNoError(t, keyValueMedium.Write("group/key", "val"))

	value, err := keyValueStore.Get("group", "key")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "val", value)

	value, err = keyValueMedium.Read("group/key")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "val", value)
}

func TestKeyValueMedium_KeyValueStore_Good(t *core.T) {
	keyValueMedium := newKeyValueMedium(t)

	core.AssertNotNil(t, keyValueMedium.KeyValueStore())
	core.AssertSame(t, keyValueMedium.KeyValueStore(), keyValueMedium.KeyValueStore())
}

func TestKeyValueMedium_EnsureDir_ReadWrite_Good(t *core.T) {
	keyValueMedium := newKeyValueMedium(t)

	core.RequireNoError(t, keyValueMedium.EnsureDir("ignored"))
	core.RequireNoError(t, keyValueMedium.Write("group/key", "value"))

	value, err := keyValueMedium.Read("group/key")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "value", value)
}

func TestKeyValueMedium_StreamHelpers_Good(t *core.T) {
	keyValueMedium := newKeyValueMedium(t)

	writer, err := keyValueMedium.WriteStream("group/key")
	core.RequireNoError(t, err)
	_, err = writer.Write([]byte("streamed"))
	core.RequireNoError(t, err)
	core.RequireNoError(t, writer.Close())

	reader, err := keyValueMedium.ReadStream("group/key")
	core.RequireNoError(t, err)
	data, err := io.ReadAll(reader)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "streamed", string(data))
	core.RequireNoError(t, reader.Close())

	file, err := keyValueMedium.Open("group/key")
	core.RequireNoError(t, err)
	info, err := file.Stat()
	core.RequireNoError(t, err)
	core.AssertEqual(t, "key", info.Name())
	core.AssertEqual(t, int64(8), info.Size())
	core.AssertEqual(t, fs.FileMode(0644), info.Mode())
	core.AssertTrue(t, info.ModTime().IsZero())
	core.AssertFalse(t, info.IsDir())
	core.AssertNil(t, info.Sys())
	core.RequireNoError(t, file.Close())

	entries, err := keyValueMedium.List("group")
	core.RequireNoError(t, err)
	core.AssertLen(t, entries, 1)
	core.AssertEqual(t, "key", entries[0].Name())
	core.AssertFalse(t, entries[0].IsDir())
	core.AssertEqual(t, fs.FileMode(0), entries[0].Type())

	entryInfo, err := entries[0].Info()
	core.RequireNoError(t, err)
	core.AssertEqual(t, "key", entryInfo.Name())
	core.AssertEqual(t, int64(8), entryInfo.Size())
}
