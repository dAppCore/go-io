package store

import (
	core "dappco.re/go"
	"io"
	"io/fs"
)

const (
	storeGroupKeyPath = "group/key"
	storeOldKeyPath   = "old/key"
	storeHelloContent = "hello world"
)

func newKeyValueMedium(t *core.T) *Medium {
	t.Helper()
	keyValueMedium, err := NewMedium(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	t.Cleanup(func() { _ = keyValueMedium.Close() })
	return keyValueMedium
}

func TestKeyValueMedium_ReadWriteGood(t *core.T) {
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
	_ = keyValueMedium.Write(storeGroupKeyPath, "val")

	core.AssertTrue(t, keyValueMedium.IsFile(storeGroupKeyPath))
	core.AssertFalse(t, keyValueMedium.IsFile("group/nope"))
	core.AssertFalse(t, keyValueMedium.IsFile("group"))
}

func TestKeyValueMedium_Delete_Good(t *core.T) {
	keyValueMedium := newKeyValueMedium(t)
	_ = keyValueMedium.Write(storeGroupKeyPath, "val")

	err := keyValueMedium.Delete(storeGroupKeyPath)
	core.RequireNoError(t, err)
	core.AssertFalse(t, keyValueMedium.IsFile(storeGroupKeyPath))
}

func TestKeyValueMedium_Delete_NonEmptyGroup_Bad(t *core.T) {
	keyValueMedium := newKeyValueMedium(t)
	_ = keyValueMedium.Write(storeGroupKeyPath, "val")

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
	_ = keyValueMedium.Write(storeOldKeyPath, "val")

	err := keyValueMedium.Rename(storeOldKeyPath, "new/key")
	core.RequireNoError(t, err)

	value, err := keyValueMedium.Read("new/key")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "val", value)
	core.AssertFalse(t, keyValueMedium.IsFile(storeOldKeyPath))
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
	_ = keyValueMedium.Write(storeGroupKeyPath, "hello")

	info, err := keyValueMedium.Stat("group")
	core.RequireNoError(t, err)
	core.AssertTrue(t, info.IsDir())

	info, err = keyValueMedium.Stat(storeGroupKeyPath)
	core.RequireNoError(t, err)
	core.AssertEqual(t, int64(5), info.Size())
	core.AssertFalse(t, info.IsDir())
}

func TestKeyValueMedium_Exists_IsDir_Good(t *core.T) {
	keyValueMedium := newKeyValueMedium(t)
	_ = keyValueMedium.Write(storeGroupKeyPath, "val")

	core.AssertTrue(t, keyValueMedium.Exists("group"))
	core.AssertTrue(t, keyValueMedium.Exists(storeGroupKeyPath))
	core.AssertTrue(t, keyValueMedium.IsDir("group"))
	core.AssertFalse(t, keyValueMedium.IsDir(storeGroupKeyPath))
	core.AssertFalse(t, keyValueMedium.Exists("nope"))
}

func TestKeyValueMedium_Open_Read_Good(t *core.T) {
	keyValueMedium := newKeyValueMedium(t)
	_ = keyValueMedium.Write(storeGroupKeyPath, storeHelloContent)

	file, err := keyValueMedium.Open(storeGroupKeyPath)
	core.RequireNoError(t, err)
	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(file)
	core.RequireNoError(t, err)
	core.AssertEqual(t, storeHelloContent, string(data))
}

func TestKeyValueMedium_CreateCloseGood(t *core.T) {
	keyValueMedium := newKeyValueMedium(t)

	writer, err := keyValueMedium.Create(storeGroupKeyPath)
	core.RequireNoError(t, err)
	_, _ = writer.Write([]byte("streamed"))
	core.RequireNoError(t, writer.Close())

	value, err := keyValueMedium.Read(storeGroupKeyPath)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "streamed", value)
}

func TestKeyValueMedium_Append_Good(t *core.T) {
	keyValueMedium := newKeyValueMedium(t)
	_ = keyValueMedium.Write(storeGroupKeyPath, "hello")

	writer, err := keyValueMedium.Append(storeGroupKeyPath)
	core.RequireNoError(t, err)
	_, _ = writer.Write([]byte(" world"))
	core.RequireNoError(t, writer.Close())

	value, err := keyValueMedium.Read(storeGroupKeyPath)
	core.RequireNoError(t, err)
	core.AssertEqual(t, storeHelloContent, value)
}

func TestKeyValueMedium_AsMedium_Good(t *core.T) {
	keyValueStore := newKeyValueStore(t)

	keyValueMedium := keyValueStore.AsMedium()
	core.RequireNoError(t, keyValueMedium.Write(storeGroupKeyPath, "val"))

	value, err := keyValueStore.Get("group", "key")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "val", value)

	value, err = keyValueMedium.Read(storeGroupKeyPath)
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
	core.RequireNoError(t, keyValueMedium.Write(storeGroupKeyPath, "value"))

	value, err := keyValueMedium.Read(storeGroupKeyPath)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "value", value)
}

func TestKeyValueMedium_StreamHelpersGood(t *core.T) {
	keyValueMedium := newKeyValueMedium(t)

	writer, err := keyValueMedium.WriteStream(storeGroupKeyPath)
	core.RequireNoError(t, err)
	_, err = writer.Write([]byte("streamed"))
	core.RequireNoError(t, err)
	core.RequireNoError(t, writer.Close())

	reader, err := keyValueMedium.ReadStream(storeGroupKeyPath)
	core.RequireNoError(t, err)
	data, err := io.ReadAll(reader)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "streamed", string(data))
	core.RequireNoError(t, reader.Close())

	file, err := keyValueMedium.Open(storeGroupKeyPath)
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

func newStoreMediumFixture(t *core.T) *Medium {
	t.Helper()

	keyValueStore := newKeyValueStore(t)
	return keyValueStore.AsMedium()
}

func mustCountStoreEntries(t *core.T, keyValueStore *KeyValueStore, group string) int {
	t.Helper()

	count, err := keyValueStore.Count(group)
	core.RequireNoError(t, err)
	return count
}

func mustGetStoreEntry(t *core.T, keyValueStore *KeyValueStore, group, key string) string {
	t.Helper()

	value, err := keyValueStore.Get(group, key)
	core.RequireNoError(t, err)
	return value
}

func TestMedium_KeyValueStore_AsMedium_Good(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	medium := keyValueStore.AsMedium()
	core.AssertNotNil(t, medium)
	core.AssertSame(t, keyValueStore, medium.KeyValueStore())
}

func TestMedium_KeyValueStore_AsMedium_Bad(t *core.T) {
	var keyValueStore *KeyValueStore
	medium := keyValueStore.AsMedium()
	core.AssertNotNil(t, medium)
	core.AssertNil(t, medium.KeyValueStore())
}

func TestMedium_KeyValueStore_AsMedium_Ugly(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	medium := keyValueStore.AsMedium()
	core.RequireNoError(t, medium.Write(storeGroupKeyPath, "value"))
	core.AssertTrue(t, medium.Exists(storeGroupKeyPath))
}

func TestMedium_NewMedium_Good(t *core.T) {
	medium, err := NewMedium(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	defer func() { _ = medium.Close() }()
	core.AssertNotNil(t, medium.KeyValueStore())
}

func TestMedium_NewMedium_Bad(t *core.T) {
	medium, err := NewMedium(Options{})
	core.AssertError(t, err)
	core.AssertNil(t, medium)
}

func TestMedium_NewMedium_Ugly(t *core.T) {
	medium, err := NewMedium(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	core.AssertNoError(t, medium.Close())
}

func TestMedium_Medium_KeyValueStore_Good(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	medium := keyValueStore.AsMedium()
	got := medium.KeyValueStore()
	core.AssertSame(t, keyValueStore, got)
}

func TestMedium_Medium_KeyValueStore_Bad(t *core.T) {
	medium := &Medium{}
	got := medium.KeyValueStore()
	core.AssertNil(t, got)
}

func TestMedium_Medium_KeyValueStore_Ugly(t *core.T) {
	medium := newStoreMediumFixture(t)
	core.RequireNoError(t, medium.Write(storeGroupKeyPath, "value"))
	got := medium.KeyValueStore()
	core.AssertEqual(t, "value", mustGetStoreEntry(t, got, "group", "key"))
}

func TestMedium_Medium_Close_Good(t *core.T) {
	medium, err := NewMedium(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	closeErr := medium.Close()
	core.AssertNoError(t, closeErr)
}

func TestMedium_Medium_Close_Bad(t *core.T) {
	medium := &Medium{}
	core.AssertPanics(t, func() { _ = medium.Close() })
	core.AssertNil(t, medium.keyValueStore)
}

func TestMedium_Medium_Close_Ugly(t *core.T) {
	medium, err := NewMedium(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	core.RequireNoError(t, medium.Write(storeGroupKeyPath, "value"))
	core.AssertNoError(t, medium.Close())
}

func TestMedium_Medium_Read_Good(t *core.T) {
	medium := newStoreMediumFixture(t)
	core.RequireNoError(t, medium.Write(storeGroupKeyPath, "value"))
	got, err := medium.Read(storeGroupKeyPath)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "value", got)
}

func TestMedium_Medium_Read_Bad(t *core.T) {
	medium := newStoreMediumFixture(t)
	got, err := medium.Read("group")
	core.AssertError(t, err)
	core.AssertEqual(t, "", got)
}

func TestMedium_Medium_Read_Ugly(t *core.T) {
	medium := newStoreMediumFixture(t)
	core.RequireNoError(t, medium.Write("/group/key", "value"))
	got, err := medium.Read(storeGroupKeyPath)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "value", got)
}

func TestMedium_Medium_Write_Good(t *core.T) {
	medium := newStoreMediumFixture(t)
	err := medium.Write(storeGroupKeyPath, "value")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile(storeGroupKeyPath))
}

func TestMedium_Medium_Write_Bad(t *core.T) {
	medium := newStoreMediumFixture(t)
	err := medium.Write("group", "value")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.IsFile("group"))
}

func TestMedium_Medium_Write_Ugly(t *core.T) {
	medium := newStoreMediumFixture(t)
	err := medium.Write("/group/key", "")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists(storeGroupKeyPath))
}

func TestMedium_Medium_WriteMode_Good(t *core.T) {
	medium := newStoreMediumFixture(t)
	err := medium.WriteMode(storeGroupKeyPath, "value", 0600)
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile(storeGroupKeyPath))
}

func TestMedium_Medium_WriteMode_Bad(t *core.T) {
	medium := newStoreMediumFixture(t)
	err := medium.WriteMode("group", "value", 0600)
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("group"))
}

func TestMedium_Medium_WriteMode_Ugly(t *core.T) {
	medium := newStoreMediumFixture(t)
	err := medium.WriteMode(storeGroupKeyPath, "value", 0)
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists(storeGroupKeyPath))
}

func TestMedium_Medium_EnsureDir_Good(t *core.T) {
	medium := newStoreMediumFixture(t)
	err := medium.EnsureDir("group")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("group"))
}

func TestMedium_Medium_EnsureDir_Bad(t *core.T) {
	medium := newStoreMediumFixture(t)
	err := medium.EnsureDir("")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists(""))
}

func TestMedium_Medium_EnsureDir_Ugly(t *core.T) {
	medium := newStoreMediumFixture(t)
	err := medium.EnsureDir("a/b/c")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("a/b/c"))
}

func TestMedium_Medium_IsFile_Good(t *core.T) {
	medium := newStoreMediumFixture(t)
	core.RequireNoError(t, medium.Write(storeGroupKeyPath, "value"))
	got := medium.IsFile(storeGroupKeyPath)
	core.AssertTrue(t, got)
}

func TestMedium_Medium_IsFile_Bad(t *core.T) {
	medium := newStoreMediumFixture(t)
	got := medium.IsFile("group")
	core.AssertFalse(t, got)
}

func TestMedium_Medium_IsFile_Ugly(t *core.T) {
	medium := newStoreMediumFixture(t)
	core.RequireNoError(t, medium.Write(storeGroupKeyPath, "value"))
	got := medium.IsFile("/group/key")
	core.AssertTrue(t, got)
}

func TestMedium_Medium_Delete_Good(t *core.T) {
	medium := newStoreMediumFixture(t)
	core.RequireNoError(t, medium.Write(storeGroupKeyPath, "value"))
	err := medium.Delete(storeGroupKeyPath)
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists(storeGroupKeyPath))
}

func TestMedium_Medium_Delete_Bad(t *core.T) {
	medium := newStoreMediumFixture(t)
	err := medium.Delete("")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists(""))
}

func TestMedium_Medium_Delete_Ugly(t *core.T) {
	medium := newStoreMediumFixture(t)
	err := medium.Delete("missing/key")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("missing/key"))
}

func TestMedium_Medium_DeleteAll_Good(t *core.T) {
	medium := newStoreMediumFixture(t)
	core.RequireNoError(t, medium.Write("group/a", "1"))
	core.RequireNoError(t, medium.Write("group/b", "2"))
	err := medium.DeleteAll("group")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("group/a"))
}

func TestMedium_Medium_DeleteAll_Bad(t *core.T) {
	medium := newStoreMediumFixture(t)
	err := medium.DeleteAll("")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists(""))
}

func TestMedium_Medium_DeleteAll_Ugly(t *core.T) {
	medium := newStoreMediumFixture(t)
	err := medium.DeleteAll("missing")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("missing"))
}

func TestMedium_Medium_Rename_Good(t *core.T) {
	medium := newStoreMediumFixture(t)
	core.RequireNoError(t, medium.Write("group/old", "value"))
	err := medium.Rename("group/old", "group/new")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("group/new"))
}

func TestMedium_Medium_Rename_Bad(t *core.T) {
	medium := newStoreMediumFixture(t)
	err := medium.Rename("group", "group/new")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("group/new"))
}

func TestMedium_Medium_Rename_Ugly(t *core.T) {
	medium := newStoreMediumFixture(t)
	core.RequireNoError(t, medium.Write(storeGroupKeyPath, "value"))
	err := medium.Rename(storeGroupKeyPath, "other/key")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("other/key"))
}

func TestMedium_Medium_List_Good(t *core.T) {
	medium := newStoreMediumFixture(t)
	core.RequireNoError(t, medium.Write(storeGroupKeyPath, "value"))
	entries, err := medium.List("group")
	core.AssertNoError(t, err)
	core.AssertLen(t, entries, 1)
}

func TestMedium_Medium_List_Bad(t *core.T) {
	medium := newStoreMediumFixture(t)
	core.RequireNoError(t, medium.Write(storeGroupKeyPath, "value"))
	entries, err := medium.List(storeGroupKeyPath)
	core.AssertErrorIs(t, err, ErrNotDirectory)
	core.AssertNil(t, entries)
}

func TestMedium_Medium_List_Ugly(t *core.T) {
	medium := newStoreMediumFixture(t)
	entries, err := medium.List("")
	core.AssertNoError(t, err)
	core.AssertEmpty(t, entries)
}

func TestMedium_Medium_Stat_Good(t *core.T) {
	medium := newStoreMediumFixture(t)
	core.RequireNoError(t, medium.Write(storeGroupKeyPath, "value"))
	info, err := medium.Stat(storeGroupKeyPath)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "key", info.Name())
}

func TestMedium_Medium_Stat_Bad(t *core.T) {
	medium := newStoreMediumFixture(t)
	info, err := medium.Stat("")
	core.AssertError(t, err)
	core.AssertNil(t, info)
}

func TestMedium_Medium_Stat_Ugly(t *core.T) {
	medium := newStoreMediumFixture(t)
	core.RequireNoError(t, medium.Write(storeGroupKeyPath, "value"))
	info, err := medium.Stat("group")
	core.AssertNoError(t, err)
	core.AssertTrue(t, info.IsDir())
}

func TestMedium_Medium_Open_Good(t *core.T) {
	medium := newStoreMediumFixture(t)
	core.RequireNoError(t, medium.Write(storeGroupKeyPath, "value"))
	file, err := medium.Open(storeGroupKeyPath)
	core.AssertNoError(t, err)
	core.AssertNotNil(t, file)
	core.RequireNoError(t, file.Close())
}

func TestMedium_Medium_Open_Bad(t *core.T) {
	medium := newStoreMediumFixture(t)
	file, err := medium.Open("group")
	core.AssertError(t, err)
	core.AssertNil(t, file)
}

func TestMedium_Medium_Open_Ugly(t *core.T) {
	medium := newStoreMediumFixture(t)
	core.RequireNoError(t, medium.Write("group/empty", ""))
	file, err := medium.Open("group/empty")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, file)
}

func TestMedium_Medium_Create_Good(t *core.T) {
	medium := newStoreMediumFixture(t)
	writer, err := medium.Create(storeGroupKeyPath)
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("value"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestMedium_Medium_Create_Bad(t *core.T) {
	medium := newStoreMediumFixture(t)
	writer, err := medium.Create("group")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestMedium_Medium_Create_Ugly(t *core.T) {
	medium := newStoreMediumFixture(t)
	writer, err := medium.Create("/group/key")
	core.RequireNoError(t, err)
	core.AssertNoError(t, writer.Close())
	core.AssertTrue(t, medium.Exists(storeGroupKeyPath))
}

func TestMedium_Medium_Append_Good(t *core.T) {
	medium := newStoreMediumFixture(t)
	core.RequireNoError(t, medium.Write(storeGroupKeyPath, "a"))
	writer, err := medium.Append(storeGroupKeyPath)
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("b"))
	core.RequireNoError(t, writeErr)
	core.RequireNoError(t, writer.Close())
}

func TestMedium_Medium_Append_Bad(t *core.T) {
	medium := newStoreMediumFixture(t)
	writer, err := medium.Append("group")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestMedium_Medium_Append_Ugly(t *core.T) {
	medium := newStoreMediumFixture(t)
	writer, err := medium.Append("group/new")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("created"))
	core.RequireNoError(t, writeErr)
	core.RequireNoError(t, writer.Close())
}

func TestMedium_Medium_ReadStream_Good(t *core.T) {
	medium := newStoreMediumFixture(t)
	core.RequireNoError(t, medium.Write(storeGroupKeyPath, "value"))
	reader, err := medium.ReadStream(storeGroupKeyPath)
	core.RequireNoError(t, err)
	defer func() { _ = reader.Close() }()
	data, readErr := io.ReadAll(reader)
	core.AssertNoError(t, readErr)
	core.AssertEqual(t, "value", string(data))
}

func TestMedium_Medium_ReadStream_Bad(t *core.T) {
	medium := newStoreMediumFixture(t)
	reader, err := medium.ReadStream("group")
	core.AssertError(t, err)
	core.AssertNil(t, reader)
}

func TestMedium_Medium_ReadStream_Ugly(t *core.T) {
	medium := newStoreMediumFixture(t)
	core.RequireNoError(t, medium.Write("group/empty", ""))
	reader, err := medium.ReadStream("group/empty")
	core.RequireNoError(t, err)
	core.AssertNoError(t, reader.Close())
}

func TestMedium_Medium_WriteStream_Good(t *core.T) {
	medium := newStoreMediumFixture(t)
	writer, err := medium.WriteStream(storeGroupKeyPath)
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("value"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestMedium_Medium_WriteStream_Bad(t *core.T) {
	medium := newStoreMediumFixture(t)
	writer, err := medium.WriteStream("group")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestMedium_Medium_WriteStream_Ugly(t *core.T) {
	medium := newStoreMediumFixture(t)
	writer, err := medium.WriteStream("/group/key")
	core.RequireNoError(t, err)
	core.AssertNoError(t, writer.Close())
	core.AssertTrue(t, medium.Exists(storeGroupKeyPath))
}

func TestMedium_Medium_Exists_Good(t *core.T) {
	medium := newStoreMediumFixture(t)
	core.RequireNoError(t, medium.Write(storeGroupKeyPath, "value"))
	got := medium.Exists(storeGroupKeyPath)
	core.AssertTrue(t, got)
}

func TestMedium_Medium_Exists_Bad(t *core.T) {
	medium := newStoreMediumFixture(t)
	got := medium.Exists("")
	core.AssertFalse(t, got)
}

func TestMedium_Medium_Exists_Ugly(t *core.T) {
	medium := newStoreMediumFixture(t)
	core.RequireNoError(t, medium.Write(storeGroupKeyPath, "value"))
	got := medium.Exists("group")
	core.AssertTrue(t, got)
}

func TestMedium_Medium_IsDir_Good(t *core.T) {
	medium := newStoreMediumFixture(t)
	core.RequireNoError(t, medium.Write(storeGroupKeyPath, "value"))
	got := medium.IsDir("group")
	core.AssertTrue(t, got)
}

func TestMedium_Medium_IsDir_Bad(t *core.T) {
	medium := newStoreMediumFixture(t)
	got := medium.IsDir("")
	core.AssertFalse(t, got)
}

func TestMedium_Medium_IsDir_Ugly(t *core.T) {
	medium := newStoreMediumFixture(t)
	core.RequireNoError(t, medium.Write(storeGroupKeyPath, "value"))
	got := medium.IsDir(storeGroupKeyPath)
	core.AssertFalse(t, got)
}

func TestMedium_ValueFileInfo_Name_Good(t *core.T) {
	info := &keyValueFileInfo{name: "key"}
	got := info.Name()
	core.AssertEqual(t, "key", got)
}

func TestMedium_ValueFileInfo_Name_Bad(t *core.T) {
	info := &keyValueFileInfo{}
	got := info.Name()
	core.AssertEqual(t, "", got)
}

func TestMedium_ValueFileInfo_Name_Ugly(t *core.T) {
	info := &keyValueFileInfo{name: "."}
	got := info.Name()
	core.AssertEqual(t, ".", got)
}

func TestMedium_ValueFileInfo_Size_Good(t *core.T) {
	info := &keyValueFileInfo{size: 5}
	got := info.Size()
	core.AssertEqual(t, int64(5), got)
}

func TestMedium_ValueFileInfo_Size_Bad(t *core.T) {
	info := &keyValueFileInfo{size: -1}
	got := info.Size()
	core.AssertEqual(t, int64(-1), got)
}

func TestMedium_ValueFileInfo_Size_Ugly(t *core.T) {
	info := &keyValueFileInfo{}
	got := info.Size()
	core.AssertEqual(t, int64(0), got)
}

func TestMedium_ValueFileInfo_Mode_Good(t *core.T) {
	info := &keyValueFileInfo{name: "key"}
	got := info.Mode()
	core.AssertEqual(t, fs.FileMode(0644), got)
}

func TestMedium_ValueFileInfo_Mode_Bad(t *core.T) {
	info := &keyValueFileInfo{isDir: true}
	got := info.Mode()
	core.AssertTrue(t, got.IsDir())
}

func TestMedium_ValueFileInfo_Mode_Ugly(t *core.T) {
	info := &keyValueFileInfo{}
	got := info.Mode()
	core.AssertEqual(t, fs.FileMode(0644), got)
}

func TestMedium_ValueFileInfo_ModTime_Good(t *core.T) {
	info := &keyValueFileInfo{name: "key"}
	got := info.ModTime()
	core.AssertTrue(t, got.IsZero())
}

func TestMedium_ValueFileInfo_ModTime_Bad(t *core.T) {
	info := &keyValueFileInfo{}
	got := info.ModTime()
	core.AssertTrue(t, got.IsZero())
}

func TestMedium_ValueFileInfo_ModTime_Ugly(t *core.T) {
	info := &keyValueFileInfo{isDir: true}
	got := info.ModTime()
	core.AssertTrue(t, got.IsZero())
}

func TestMedium_ValueFileInfo_IsDir_Good(t *core.T) {
	info := &keyValueFileInfo{isDir: true}
	got := info.IsDir()
	core.AssertTrue(t, got)
}

func TestMedium_ValueFileInfo_IsDir_Bad(t *core.T) {
	info := &keyValueFileInfo{}
	got := info.IsDir()
	core.AssertFalse(t, got)
}

func TestMedium_ValueFileInfo_IsDir_Ugly(t *core.T) {
	info := &keyValueFileInfo{name: "group", isDir: false}
	got := info.IsDir()
	core.AssertFalse(t, got)
}

func TestMedium_ValueFileInfo_Sys_Good(t *core.T) {
	info := &keyValueFileInfo{name: "key"}
	got := info.Sys()
	core.AssertNil(t, got)
}

func TestMedium_ValueFileInfo_Sys_Bad(t *core.T) {
	info := &keyValueFileInfo{}
	got := info.Sys()
	core.AssertNil(t, got)
}

func TestMedium_ValueFileInfo_Sys_Ugly(t *core.T) {
	info := &keyValueFileInfo{isDir: true}
	got := info.Sys()
	core.AssertNil(t, got)
}

func TestMedium_ValueDirEntry_Name_Good(t *core.T) {
	entry := &keyValueDirEntry{name: "key"}
	got := entry.Name()
	core.AssertEqual(t, "key", got)
}

func TestMedium_ValueDirEntry_Name_Bad(t *core.T) {
	entry := &keyValueDirEntry{}
	got := entry.Name()
	core.AssertEqual(t, "", got)
}

func TestMedium_ValueDirEntry_Name_Ugly(t *core.T) {
	entry := &keyValueDirEntry{name: "."}
	got := entry.Name()
	core.AssertEqual(t, ".", got)
}

func TestMedium_ValueDirEntry_IsDir_Good(t *core.T) {
	entry := &keyValueDirEntry{isDir: true}
	got := entry.IsDir()
	core.AssertTrue(t, got)
}

func TestMedium_ValueDirEntry_IsDir_Bad(t *core.T) {
	entry := &keyValueDirEntry{}
	got := entry.IsDir()
	core.AssertFalse(t, got)
}

func TestMedium_ValueDirEntry_IsDir_Ugly(t *core.T) {
	entry := &keyValueDirEntry{name: "key", isDir: false}
	got := entry.IsDir()
	core.AssertFalse(t, got)
}

func TestMedium_ValueDirEntry_Type_Good(t *core.T) {
	entry := &keyValueDirEntry{isDir: true}
	got := entry.Type()
	core.AssertTrue(t, got.IsDir())
}

func TestMedium_ValueDirEntry_Type_Bad(t *core.T) {
	entry := &keyValueDirEntry{}
	got := entry.Type()
	core.AssertEqual(t, fs.FileMode(0), got)
}

func TestMedium_ValueDirEntry_Type_Ugly(t *core.T) {
	entry := &keyValueDirEntry{name: "key", isDir: false}
	got := entry.Type()
	core.AssertEqual(t, fs.FileMode(0), got)
}

func TestMedium_ValueDirEntry_Info_Good(t *core.T) {
	entry := &keyValueDirEntry{name: "key", size: 5}
	info, err := entry.Info()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "key", info.Name())
}

func TestMedium_ValueDirEntry_Info_Bad(t *core.T) {
	entry := &keyValueDirEntry{}
	info, err := entry.Info()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "", info.Name())
}

func TestMedium_ValueDirEntry_Info_Ugly(t *core.T) {
	entry := &keyValueDirEntry{name: "group", isDir: true}
	info, err := entry.Info()
	core.AssertNoError(t, err)
	core.AssertTrue(t, info.IsDir())
}

func TestMedium_ValueFile_Stat_Good(t *core.T) {
	file := &keyValueFile{name: "key", content: []byte("value")}
	info, err := file.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "key", info.Name())
}

func TestMedium_ValueFile_Stat_Bad(t *core.T) {
	file := &keyValueFile{}
	info, err := file.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "", info.Name())
}

func TestMedium_ValueFile_Stat_Ugly(t *core.T) {
	file := &keyValueFile{name: "empty"}
	info, err := file.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, int64(0), info.Size())
}

func TestMedium_ValueFile_Read_Good(t *core.T) {
	file := &keyValueFile{content: []byte("value")}
	buffer := make([]byte, 5)
	count, err := file.Read(buffer)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "value", string(buffer[:count]))
}

func TestMedium_ValueFile_Read_Bad(t *core.T) {
	file := &keyValueFile{content: []byte("x"), offset: 1}
	buffer := make([]byte, 1)
	count, err := file.Read(buffer)
	core.AssertErrorIs(t, err, io.EOF)
	core.AssertEqual(t, 0, count)
}

func TestMedium_ValueFile_Read_Ugly(t *core.T) {
	file := &keyValueFile{content: []byte("value")}
	buffer := make([]byte, 2)
	count, err := file.Read(buffer)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "va", string(buffer[:count]))
}

func TestMedium_ValueFile_Close_Good(t *core.T) {
	file := &keyValueFile{name: "key"}
	err := file.Close()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "key", file.name)
}

func TestMedium_ValueFile_Close_Bad(t *core.T) {
	file := &keyValueFile{}
	err := file.Close()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "", file.name)
}

func TestMedium_ValueFile_Close_Ugly(t *core.T) {
	file := &keyValueFile{offset: 99}
	err := file.Close()
	core.AssertNoError(t, err)
	core.AssertEqual(t, int64(99), file.offset)
}

func TestMedium_ValueWriteCloser_Write_Good(t *core.T) {
	writer := &keyValueWriteCloser{}
	count, err := writer.Write([]byte("value"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, len("value"), count)
}

func TestMedium_ValueWriteCloser_Write_Bad(t *core.T) {
	writer := &keyValueWriteCloser{}
	count, err := writer.Write(nil)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, count)
}

func TestMedium_ValueWriteCloser_Write_Ugly(t *core.T) {
	writer := &keyValueWriteCloser{data: []byte("a")}
	count, err := writer.Write([]byte("b"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, 1, count)
}

func TestMedium_ValueWriteCloser_Close_Good(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	writer := &keyValueWriteCloser{keyValueStore: keyValueStore, group: "group", key: "key", data: []byte("value")}
	err := writer.Close()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "value", mustGetStoreEntry(t, keyValueStore, "group", "key"))
}

func TestMedium_ValueWriteCloser_Close_Bad(t *core.T) {
	writer := &keyValueWriteCloser{}
	core.AssertPanics(t, func() { _ = writer.Close() })
	core.AssertNil(t, writer.keyValueStore)
}

func TestMedium_ValueWriteCloser_Close_Ugly(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	writer := &keyValueWriteCloser{keyValueStore: keyValueStore, group: "", key: "", data: []byte("value")}
	err := writer.Close()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "value", mustGetStoreEntry(t, keyValueStore, "", ""))
}
