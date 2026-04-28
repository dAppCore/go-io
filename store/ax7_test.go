package store

import (
	goio "io"
	"io/fs"

	core "dappco.re/go"
)

func newAX7Medium(t *core.T) *Medium {
	t.Helper()

	keyValueStore := newKeyValueStore(t)
	return keyValueStore.AsMedium()
}

func TestAX7_New_Good(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	core.AssertNotNil(t, keyValueStore)
	core.AssertNotNil(t, keyValueStore.database)
}

func TestAX7_New_Bad(t *core.T) {
	keyValueStore, err := New(Options{})
	core.AssertError(t, err)
	core.AssertNil(t, keyValueStore)
}

func TestAX7_New_Ugly(t *core.T) {
	keyValueStore, err := New(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	core.AssertNoError(t, keyValueStore.Close())
}

func TestAX7_KeyValueStore_Close_Good(t *core.T) {
	keyValueStore, err := New(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	closeErr := keyValueStore.Close()
	core.AssertNoError(t, closeErr)
}

func TestAX7_KeyValueStore_Close_Bad(t *core.T) {
	keyValueStore, err := New(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	core.RequireNoError(t, keyValueStore.Close())
	closeErr := keyValueStore.Close()
	core.AssertNoError(t, closeErr)
}

func TestAX7_KeyValueStore_Close_Ugly(t *core.T) {
	keyValueStore, err := New(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	core.RequireNoError(t, keyValueStore.Set("group", "key", "value"))
	core.AssertNoError(t, keyValueStore.Close())
}

func TestAX7_KeyValueStore_Get_Good(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	core.RequireNoError(t, keyValueStore.Set("group", "key", "value"))
	got, err := keyValueStore.Get("group", "key")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "value", got)
}

func TestAX7_KeyValueStore_Get_Bad(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	got, err := keyValueStore.Get("group", "missing")
	core.AssertErrorIs(t, err, NotFoundError)
	core.AssertEqual(t, "", got)
}

func TestAX7_KeyValueStore_Get_Ugly(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	core.RequireNoError(t, keyValueStore.Set("", "", "value"))
	got, err := keyValueStore.Get("", "")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "value", got)
}

func TestAX7_KeyValueStore_Set_Good(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	err := keyValueStore.Set("group", "key", "value")
	core.AssertNoError(t, err)
	core.AssertEqual(t, 1, mustCountAX7(t, keyValueStore, "group"))
}

func TestAX7_KeyValueStore_Set_Bad(t *core.T) {
	keyValueStore, err := New(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	core.RequireNoError(t, keyValueStore.Close())
	setErr := keyValueStore.Set("group", "key", "value")
	core.AssertError(t, setErr)
}

func TestAX7_KeyValueStore_Set_Ugly(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	core.RequireNoError(t, keyValueStore.Set("group", "key", "old"))
	err := keyValueStore.Set("group", "key", "new")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "new", mustGetAX7(t, keyValueStore, "group", "key"))
}

func TestAX7_KeyValueStore_Delete_Good(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	core.RequireNoError(t, keyValueStore.Set("group", "key", "value"))
	err := keyValueStore.Delete("group", "key")
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, mustCountAX7(t, keyValueStore, "group"))
}

func TestAX7_KeyValueStore_Delete_Bad(t *core.T) {
	keyValueStore, err := New(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	core.RequireNoError(t, keyValueStore.Close())
	deleteErr := keyValueStore.Delete("group", "key")
	core.AssertError(t, deleteErr)
}

func TestAX7_KeyValueStore_Delete_Ugly(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	err := keyValueStore.Delete("group", "missing")
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, mustCountAX7(t, keyValueStore, "group"))
}

func TestAX7_KeyValueStore_Count_Good(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	core.RequireNoError(t, keyValueStore.Set("group", "a", "1"))
	core.RequireNoError(t, keyValueStore.Set("group", "b", "2"))
	count, err := keyValueStore.Count("group")
	core.AssertNoError(t, err)
	core.AssertEqual(t, 2, count)
}

func TestAX7_KeyValueStore_Count_Bad(t *core.T) {
	keyValueStore, err := New(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	core.RequireNoError(t, keyValueStore.Close())
	count, countErr := keyValueStore.Count("group")
	core.AssertError(t, countErr)
	core.AssertEqual(t, 0, count)
}

func TestAX7_KeyValueStore_Count_Ugly(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	count, err := keyValueStore.Count("missing")
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, count)
}

func TestAX7_KeyValueStore_DeleteGroup_Good(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	core.RequireNoError(t, keyValueStore.Set("group", "a", "1"))
	err := keyValueStore.DeleteGroup("group")
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, mustCountAX7(t, keyValueStore, "group"))
}

func TestAX7_KeyValueStore_DeleteGroup_Bad(t *core.T) {
	keyValueStore, err := New(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	core.RequireNoError(t, keyValueStore.Close())
	deleteErr := keyValueStore.DeleteGroup("group")
	core.AssertError(t, deleteErr)
}

func TestAX7_KeyValueStore_DeleteGroup_Ugly(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	err := keyValueStore.DeleteGroup("missing")
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, mustCountAX7(t, keyValueStore, "missing"))
}

func TestAX7_KeyValueStore_ListGroups_Good(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	core.RequireNoError(t, keyValueStore.Set("b", "key", "1"))
	core.RequireNoError(t, keyValueStore.Set("a", "key", "2"))
	groups, err := keyValueStore.ListGroups()
	core.AssertNoError(t, err)
	core.AssertEqual(t, []string{"a", "b"}, groups)
}

func TestAX7_KeyValueStore_ListGroups_Bad(t *core.T) {
	keyValueStore, err := New(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	core.RequireNoError(t, keyValueStore.Close())
	groups, listErr := keyValueStore.ListGroups()
	core.AssertError(t, listErr)
	core.AssertNil(t, groups)
}

func TestAX7_KeyValueStore_ListGroups_Ugly(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	groups, err := keyValueStore.ListGroups()
	core.AssertNoError(t, err)
	core.AssertEmpty(t, groups)
}

func TestAX7_KeyValueStore_GetAll_Good(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	core.RequireNoError(t, keyValueStore.Set("group", "a", "1"))
	core.RequireNoError(t, keyValueStore.Set("group", "b", "2"))
	all, err := keyValueStore.GetAll("group")
	core.AssertNoError(t, err)
	core.AssertEqual(t, map[string]string{"a": "1", "b": "2"}, all)
}

func TestAX7_KeyValueStore_GetAll_Bad(t *core.T) {
	keyValueStore, err := New(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	core.RequireNoError(t, keyValueStore.Close())
	all, getErr := keyValueStore.GetAll("group")
	core.AssertError(t, getErr)
	core.AssertNil(t, all)
}

func TestAX7_KeyValueStore_GetAll_Ugly(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	all, err := keyValueStore.GetAll("missing")
	core.AssertNoError(t, err)
	core.AssertEmpty(t, all)
}

func TestAX7_KeyValueStore_Render_Good(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	core.RequireNoError(t, keyValueStore.Set("user", "name", "Ada"))
	rendered, err := keyValueStore.Render("hello {{ .name }}", "user")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "hello Ada", rendered)
}

func TestAX7_KeyValueStore_Render_Bad(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	rendered, err := keyValueStore.Render("hello {{", "user")
	core.AssertError(t, err)
	core.AssertEqual(t, "", rendered)
}

func TestAX7_KeyValueStore_Render_Ugly(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	rendered, err := keyValueStore.Render("missing {{ .name }}", "missing")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "missing <no value>", rendered)
}

func TestAX7_KeyValueStore_AsMedium_Good(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	medium := keyValueStore.AsMedium()
	core.AssertNotNil(t, medium)
	core.AssertSame(t, keyValueStore, medium.KeyValueStore())
}

func TestAX7_KeyValueStore_AsMedium_Bad(t *core.T) {
	var keyValueStore *KeyValueStore
	medium := keyValueStore.AsMedium()
	core.AssertNotNil(t, medium)
	core.AssertNil(t, medium.KeyValueStore())
}

func TestAX7_KeyValueStore_AsMedium_Ugly(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	medium := keyValueStore.AsMedium()
	core.RequireNoError(t, medium.Write("group/key", "value"))
	core.AssertTrue(t, medium.Exists("group/key"))
}

func TestAX7_NewMedium_Good(t *core.T) {
	medium, err := NewMedium(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	defer medium.Close()
	core.AssertNotNil(t, medium.KeyValueStore())
}

func TestAX7_NewMedium_Bad(t *core.T) {
	medium, err := NewMedium(Options{})
	core.AssertError(t, err)
	core.AssertNil(t, medium)
}

func TestAX7_NewMedium_Ugly(t *core.T) {
	medium, err := NewMedium(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	core.AssertNoError(t, medium.Close())
}

func TestAX7_Medium_KeyValueStore_Good(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	medium := keyValueStore.AsMedium()
	got := medium.KeyValueStore()
	core.AssertSame(t, keyValueStore, got)
}

func TestAX7_Medium_KeyValueStore_Bad(t *core.T) {
	medium := &Medium{}
	got := medium.KeyValueStore()
	core.AssertNil(t, got)
}

func TestAX7_Medium_KeyValueStore_Ugly(t *core.T) {
	medium := newAX7Medium(t)
	core.RequireNoError(t, medium.Write("group/key", "value"))
	got := medium.KeyValueStore()
	core.AssertEqual(t, "value", mustGetAX7(t, got, "group", "key"))
}

func TestAX7_Medium_Close_Good(t *core.T) {
	medium, err := NewMedium(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	closeErr := medium.Close()
	core.AssertNoError(t, closeErr)
}

func TestAX7_Medium_Close_Bad(t *core.T) {
	medium := &Medium{}
	core.AssertPanics(t, func() { _ = medium.Close() })
	core.AssertNil(t, medium.keyValueStore)
}

func TestAX7_Medium_Close_Ugly(t *core.T) {
	medium, err := NewMedium(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	core.RequireNoError(t, medium.Write("group/key", "value"))
	core.AssertNoError(t, medium.Close())
}

func TestAX7_Medium_Read_Good(t *core.T) {
	medium := newAX7Medium(t)
	core.RequireNoError(t, medium.Write("group/key", "value"))
	got, err := medium.Read("group/key")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "value", got)
}

func TestAX7_Medium_Read_Bad(t *core.T) {
	medium := newAX7Medium(t)
	got, err := medium.Read("group")
	core.AssertError(t, err)
	core.AssertEqual(t, "", got)
}

func TestAX7_Medium_Read_Ugly(t *core.T) {
	medium := newAX7Medium(t)
	core.RequireNoError(t, medium.Write("/group/key", "value"))
	got, err := medium.Read("group/key")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "value", got)
}

func TestAX7_Medium_Write_Good(t *core.T) {
	medium := newAX7Medium(t)
	err := medium.Write("group/key", "value")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("group/key"))
}

func TestAX7_Medium_Write_Bad(t *core.T) {
	medium := newAX7Medium(t)
	err := medium.Write("group", "value")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.IsFile("group"))
}

func TestAX7_Medium_Write_Ugly(t *core.T) {
	medium := newAX7Medium(t)
	err := medium.Write("/group/key", "")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("group/key"))
}

func TestAX7_Medium_WriteMode_Good(t *core.T) {
	medium := newAX7Medium(t)
	err := medium.WriteMode("group/key", "value", 0600)
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("group/key"))
}

func TestAX7_Medium_WriteMode_Bad(t *core.T) {
	medium := newAX7Medium(t)
	err := medium.WriteMode("group", "value", 0600)
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("group"))
}

func TestAX7_Medium_WriteMode_Ugly(t *core.T) {
	medium := newAX7Medium(t)
	err := medium.WriteMode("group/key", "value", 0)
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("group/key"))
}

func TestAX7_Medium_EnsureDir_Good(t *core.T) {
	medium := newAX7Medium(t)
	err := medium.EnsureDir("group")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("group"))
}

func TestAX7_Medium_EnsureDir_Bad(t *core.T) {
	medium := newAX7Medium(t)
	err := medium.EnsureDir("")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists(""))
}

func TestAX7_Medium_EnsureDir_Ugly(t *core.T) {
	medium := newAX7Medium(t)
	err := medium.EnsureDir("a/b/c")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("a/b/c"))
}

func TestAX7_Medium_IsFile_Good(t *core.T) {
	medium := newAX7Medium(t)
	core.RequireNoError(t, medium.Write("group/key", "value"))
	got := medium.IsFile("group/key")
	core.AssertTrue(t, got)
}

func TestAX7_Medium_IsFile_Bad(t *core.T) {
	medium := newAX7Medium(t)
	got := medium.IsFile("group")
	core.AssertFalse(t, got)
}

func TestAX7_Medium_IsFile_Ugly(t *core.T) {
	medium := newAX7Medium(t)
	core.RequireNoError(t, medium.Write("group/key", "value"))
	got := medium.IsFile("/group/key")
	core.AssertTrue(t, got)
}

func TestAX7_Medium_Delete_Good(t *core.T) {
	medium := newAX7Medium(t)
	core.RequireNoError(t, medium.Write("group/key", "value"))
	err := medium.Delete("group/key")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("group/key"))
}

func TestAX7_Medium_Delete_Bad(t *core.T) {
	medium := newAX7Medium(t)
	err := medium.Delete("")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists(""))
}

func TestAX7_Medium_Delete_Ugly(t *core.T) {
	medium := newAX7Medium(t)
	err := medium.Delete("missing/key")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("missing/key"))
}

func TestAX7_Medium_DeleteAll_Good(t *core.T) {
	medium := newAX7Medium(t)
	core.RequireNoError(t, medium.Write("group/a", "1"))
	core.RequireNoError(t, medium.Write("group/b", "2"))
	err := medium.DeleteAll("group")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("group/a"))
}

func TestAX7_Medium_DeleteAll_Bad(t *core.T) {
	medium := newAX7Medium(t)
	err := medium.DeleteAll("")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists(""))
}

func TestAX7_Medium_DeleteAll_Ugly(t *core.T) {
	medium := newAX7Medium(t)
	err := medium.DeleteAll("missing")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("missing"))
}

func TestAX7_Medium_Rename_Good(t *core.T) {
	medium := newAX7Medium(t)
	core.RequireNoError(t, medium.Write("group/old", "value"))
	err := medium.Rename("group/old", "group/new")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("group/new"))
}

func TestAX7_Medium_Rename_Bad(t *core.T) {
	medium := newAX7Medium(t)
	err := medium.Rename("group", "group/new")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("group/new"))
}

func TestAX7_Medium_Rename_Ugly(t *core.T) {
	medium := newAX7Medium(t)
	core.RequireNoError(t, medium.Write("group/key", "value"))
	err := medium.Rename("group/key", "other/key")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("other/key"))
}

func TestAX7_Medium_List_Good(t *core.T) {
	medium := newAX7Medium(t)
	core.RequireNoError(t, medium.Write("group/key", "value"))
	entries, err := medium.List("group")
	core.AssertNoError(t, err)
	core.AssertLen(t, entries, 1)
}

func TestAX7_Medium_List_Bad(t *core.T) {
	medium := newAX7Medium(t)
	core.RequireNoError(t, medium.Write("group/key", "value"))
	entries, err := medium.List("group/key")
	core.AssertErrorIs(t, err, ErrNotDirectory)
	core.AssertNil(t, entries)
}

func TestAX7_Medium_List_Ugly(t *core.T) {
	medium := newAX7Medium(t)
	entries, err := medium.List("")
	core.AssertNoError(t, err)
	core.AssertEmpty(t, entries)
}

func TestAX7_Medium_Stat_Good(t *core.T) {
	medium := newAX7Medium(t)
	core.RequireNoError(t, medium.Write("group/key", "value"))
	info, err := medium.Stat("group/key")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "key", info.Name())
}

func TestAX7_Medium_Stat_Bad(t *core.T) {
	medium := newAX7Medium(t)
	info, err := medium.Stat("")
	core.AssertError(t, err)
	core.AssertNil(t, info)
}

func TestAX7_Medium_Stat_Ugly(t *core.T) {
	medium := newAX7Medium(t)
	core.RequireNoError(t, medium.Write("group/key", "value"))
	info, err := medium.Stat("group")
	core.AssertNoError(t, err)
	core.AssertTrue(t, info.IsDir())
}

func TestAX7_Medium_Open_Good(t *core.T) {
	medium := newAX7Medium(t)
	core.RequireNoError(t, medium.Write("group/key", "value"))
	file, err := medium.Open("group/key")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, file)
	core.RequireNoError(t, file.Close())
}

func TestAX7_Medium_Open_Bad(t *core.T) {
	medium := newAX7Medium(t)
	file, err := medium.Open("group")
	core.AssertError(t, err)
	core.AssertNil(t, file)
}

func TestAX7_Medium_Open_Ugly(t *core.T) {
	medium := newAX7Medium(t)
	core.RequireNoError(t, medium.Write("group/empty", ""))
	file, err := medium.Open("group/empty")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, file)
}

func TestAX7_Medium_Create_Good(t *core.T) {
	medium := newAX7Medium(t)
	writer, err := medium.Create("group/key")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("value"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestAX7_Medium_Create_Bad(t *core.T) {
	medium := newAX7Medium(t)
	writer, err := medium.Create("group")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestAX7_Medium_Create_Ugly(t *core.T) {
	medium := newAX7Medium(t)
	writer, err := medium.Create("/group/key")
	core.RequireNoError(t, err)
	core.AssertNoError(t, writer.Close())
	core.AssertTrue(t, medium.Exists("group/key"))
}

func TestAX7_Medium_Append_Good(t *core.T) {
	medium := newAX7Medium(t)
	core.RequireNoError(t, medium.Write("group/key", "a"))
	writer, err := medium.Append("group/key")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("b"))
	core.RequireNoError(t, writeErr)
	core.RequireNoError(t, writer.Close())
}

func TestAX7_Medium_Append_Bad(t *core.T) {
	medium := newAX7Medium(t)
	writer, err := medium.Append("group")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestAX7_Medium_Append_Ugly(t *core.T) {
	medium := newAX7Medium(t)
	writer, err := medium.Append("group/new")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("created"))
	core.RequireNoError(t, writeErr)
	core.RequireNoError(t, writer.Close())
}

func TestAX7_Medium_ReadStream_Good(t *core.T) {
	medium := newAX7Medium(t)
	core.RequireNoError(t, medium.Write("group/key", "value"))
	reader, err := medium.ReadStream("group/key")
	core.RequireNoError(t, err)
	defer reader.Close()
	data, readErr := goio.ReadAll(reader)
	core.AssertNoError(t, readErr)
	core.AssertEqual(t, "value", string(data))
}

func TestAX7_Medium_ReadStream_Bad(t *core.T) {
	medium := newAX7Medium(t)
	reader, err := medium.ReadStream("group")
	core.AssertError(t, err)
	core.AssertNil(t, reader)
}

func TestAX7_Medium_ReadStream_Ugly(t *core.T) {
	medium := newAX7Medium(t)
	core.RequireNoError(t, medium.Write("group/empty", ""))
	reader, err := medium.ReadStream("group/empty")
	core.RequireNoError(t, err)
	core.AssertNoError(t, reader.Close())
}

func TestAX7_Medium_WriteStream_Good(t *core.T) {
	medium := newAX7Medium(t)
	writer, err := medium.WriteStream("group/key")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("value"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestAX7_Medium_WriteStream_Bad(t *core.T) {
	medium := newAX7Medium(t)
	writer, err := medium.WriteStream("group")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestAX7_Medium_WriteStream_Ugly(t *core.T) {
	medium := newAX7Medium(t)
	writer, err := medium.WriteStream("/group/key")
	core.RequireNoError(t, err)
	core.AssertNoError(t, writer.Close())
	core.AssertTrue(t, medium.Exists("group/key"))
}

func TestAX7_Medium_Exists_Good(t *core.T) {
	medium := newAX7Medium(t)
	core.RequireNoError(t, medium.Write("group/key", "value"))
	got := medium.Exists("group/key")
	core.AssertTrue(t, got)
}

func TestAX7_Medium_Exists_Bad(t *core.T) {
	medium := newAX7Medium(t)
	got := medium.Exists("")
	core.AssertFalse(t, got)
}

func TestAX7_Medium_Exists_Ugly(t *core.T) {
	medium := newAX7Medium(t)
	core.RequireNoError(t, medium.Write("group/key", "value"))
	got := medium.Exists("group")
	core.AssertTrue(t, got)
}

func TestAX7_Medium_IsDir_Good(t *core.T) {
	medium := newAX7Medium(t)
	core.RequireNoError(t, medium.Write("group/key", "value"))
	got := medium.IsDir("group")
	core.AssertTrue(t, got)
}

func TestAX7_Medium_IsDir_Bad(t *core.T) {
	medium := newAX7Medium(t)
	got := medium.IsDir("")
	core.AssertFalse(t, got)
}

func TestAX7_Medium_IsDir_Ugly(t *core.T) {
	medium := newAX7Medium(t)
	core.RequireNoError(t, medium.Write("group/key", "value"))
	got := medium.IsDir("group/key")
	core.AssertFalse(t, got)
}

func TestAX7_ValueFileInfo_Name_Good(t *core.T) {
	info := &keyValueFileInfo{name: "key"}
	got := info.Name()
	core.AssertEqual(t, "key", got)
}

func TestAX7_ValueFileInfo_Name_Bad(t *core.T) {
	info := &keyValueFileInfo{}
	got := info.Name()
	core.AssertEqual(t, "", got)
}

func TestAX7_ValueFileInfo_Name_Ugly(t *core.T) {
	info := &keyValueFileInfo{name: "."}
	got := info.Name()
	core.AssertEqual(t, ".", got)
}

func TestAX7_ValueFileInfo_Size_Good(t *core.T) {
	info := &keyValueFileInfo{size: 5}
	got := info.Size()
	core.AssertEqual(t, int64(5), got)
}

func TestAX7_ValueFileInfo_Size_Bad(t *core.T) {
	info := &keyValueFileInfo{size: -1}
	got := info.Size()
	core.AssertEqual(t, int64(-1), got)
}

func TestAX7_ValueFileInfo_Size_Ugly(t *core.T) {
	info := &keyValueFileInfo{}
	got := info.Size()
	core.AssertEqual(t, int64(0), got)
}

func TestAX7_ValueFileInfo_Mode_Good(t *core.T) {
	info := &keyValueFileInfo{name: "key"}
	got := info.Mode()
	core.AssertEqual(t, fs.FileMode(0644), got)
}

func TestAX7_ValueFileInfo_Mode_Bad(t *core.T) {
	info := &keyValueFileInfo{isDir: true}
	got := info.Mode()
	core.AssertTrue(t, got.IsDir())
}

func TestAX7_ValueFileInfo_Mode_Ugly(t *core.T) {
	info := &keyValueFileInfo{}
	got := info.Mode()
	core.AssertEqual(t, fs.FileMode(0644), got)
}

func TestAX7_ValueFileInfo_ModTime_Good(t *core.T) {
	info := &keyValueFileInfo{name: "key"}
	got := info.ModTime()
	core.AssertTrue(t, got.IsZero())
}

func TestAX7_ValueFileInfo_ModTime_Bad(t *core.T) {
	info := &keyValueFileInfo{}
	got := info.ModTime()
	core.AssertTrue(t, got.IsZero())
}

func TestAX7_ValueFileInfo_ModTime_Ugly(t *core.T) {
	info := &keyValueFileInfo{isDir: true}
	got := info.ModTime()
	core.AssertTrue(t, got.IsZero())
}

func TestAX7_ValueFileInfo_IsDir_Good(t *core.T) {
	info := &keyValueFileInfo{isDir: true}
	got := info.IsDir()
	core.AssertTrue(t, got)
}

func TestAX7_ValueFileInfo_IsDir_Bad(t *core.T) {
	info := &keyValueFileInfo{}
	got := info.IsDir()
	core.AssertFalse(t, got)
}

func TestAX7_ValueFileInfo_IsDir_Ugly(t *core.T) {
	info := &keyValueFileInfo{name: "group", isDir: false}
	got := info.IsDir()
	core.AssertFalse(t, got)
}

func TestAX7_ValueFileInfo_Sys_Good(t *core.T) {
	info := &keyValueFileInfo{name: "key"}
	got := info.Sys()
	core.AssertNil(t, got)
}

func TestAX7_ValueFileInfo_Sys_Bad(t *core.T) {
	info := &keyValueFileInfo{}
	got := info.Sys()
	core.AssertNil(t, got)
}

func TestAX7_ValueFileInfo_Sys_Ugly(t *core.T) {
	info := &keyValueFileInfo{isDir: true}
	got := info.Sys()
	core.AssertNil(t, got)
}

func TestAX7_ValueDirEntry_Name_Good(t *core.T) {
	entry := &keyValueDirEntry{name: "key"}
	got := entry.Name()
	core.AssertEqual(t, "key", got)
}

func TestAX7_ValueDirEntry_Name_Bad(t *core.T) {
	entry := &keyValueDirEntry{}
	got := entry.Name()
	core.AssertEqual(t, "", got)
}

func TestAX7_ValueDirEntry_Name_Ugly(t *core.T) {
	entry := &keyValueDirEntry{name: "."}
	got := entry.Name()
	core.AssertEqual(t, ".", got)
}

func TestAX7_ValueDirEntry_IsDir_Good(t *core.T) {
	entry := &keyValueDirEntry{isDir: true}
	got := entry.IsDir()
	core.AssertTrue(t, got)
}

func TestAX7_ValueDirEntry_IsDir_Bad(t *core.T) {
	entry := &keyValueDirEntry{}
	got := entry.IsDir()
	core.AssertFalse(t, got)
}

func TestAX7_ValueDirEntry_IsDir_Ugly(t *core.T) {
	entry := &keyValueDirEntry{name: "key", isDir: false}
	got := entry.IsDir()
	core.AssertFalse(t, got)
}

func TestAX7_ValueDirEntry_Type_Good(t *core.T) {
	entry := &keyValueDirEntry{isDir: true}
	got := entry.Type()
	core.AssertTrue(t, got.IsDir())
}

func TestAX7_ValueDirEntry_Type_Bad(t *core.T) {
	entry := &keyValueDirEntry{}
	got := entry.Type()
	core.AssertEqual(t, fs.FileMode(0), got)
}

func TestAX7_ValueDirEntry_Type_Ugly(t *core.T) {
	entry := &keyValueDirEntry{name: "key", isDir: false}
	got := entry.Type()
	core.AssertEqual(t, fs.FileMode(0), got)
}

func TestAX7_ValueDirEntry_Info_Good(t *core.T) {
	entry := &keyValueDirEntry{name: "key", size: 5}
	info, err := entry.Info()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "key", info.Name())
}

func TestAX7_ValueDirEntry_Info_Bad(t *core.T) {
	entry := &keyValueDirEntry{}
	info, err := entry.Info()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "", info.Name())
}

func TestAX7_ValueDirEntry_Info_Ugly(t *core.T) {
	entry := &keyValueDirEntry{name: "group", isDir: true}
	info, err := entry.Info()
	core.AssertNoError(t, err)
	core.AssertTrue(t, info.IsDir())
}

func TestAX7_ValueFile_Stat_Good(t *core.T) {
	file := &keyValueFile{name: "key", content: []byte("value")}
	info, err := file.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "key", info.Name())
}

func TestAX7_ValueFile_Stat_Bad(t *core.T) {
	file := &keyValueFile{}
	info, err := file.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "", info.Name())
}

func TestAX7_ValueFile_Stat_Ugly(t *core.T) {
	file := &keyValueFile{name: "empty"}
	info, err := file.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, int64(0), info.Size())
}

func TestAX7_ValueFile_Read_Good(t *core.T) {
	file := &keyValueFile{content: []byte("value")}
	buffer := make([]byte, 5)
	count, err := file.Read(buffer)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "value", string(buffer[:count]))
}

func TestAX7_ValueFile_Read_Bad(t *core.T) {
	file := &keyValueFile{content: []byte("x"), offset: 1}
	buffer := make([]byte, 1)
	count, err := file.Read(buffer)
	core.AssertErrorIs(t, err, goio.EOF)
	core.AssertEqual(t, 0, count)
}

func TestAX7_ValueFile_Read_Ugly(t *core.T) {
	file := &keyValueFile{content: []byte("value")}
	buffer := make([]byte, 2)
	count, err := file.Read(buffer)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "va", string(buffer[:count]))
}

func TestAX7_ValueFile_Close_Good(t *core.T) {
	file := &keyValueFile{name: "key"}
	err := file.Close()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "key", file.name)
}

func TestAX7_ValueFile_Close_Bad(t *core.T) {
	file := &keyValueFile{}
	err := file.Close()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "", file.name)
}

func TestAX7_ValueFile_Close_Ugly(t *core.T) {
	file := &keyValueFile{offset: 99}
	err := file.Close()
	core.AssertNoError(t, err)
	core.AssertEqual(t, int64(99), file.offset)
}

func TestAX7_ValueWriteCloser_Write_Good(t *core.T) {
	writer := &keyValueWriteCloser{}
	count, err := writer.Write([]byte("value"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, len("value"), count)
}

func TestAX7_ValueWriteCloser_Write_Bad(t *core.T) {
	writer := &keyValueWriteCloser{}
	count, err := writer.Write(nil)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, count)
}

func TestAX7_ValueWriteCloser_Write_Ugly(t *core.T) {
	writer := &keyValueWriteCloser{data: []byte("a")}
	count, err := writer.Write([]byte("b"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, 1, count)
}

func TestAX7_ValueWriteCloser_Close_Good(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	writer := &keyValueWriteCloser{keyValueStore: keyValueStore, group: "group", key: "key", data: []byte("value")}
	err := writer.Close()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "value", mustGetAX7(t, keyValueStore, "group", "key"))
}

func TestAX7_ValueWriteCloser_Close_Bad(t *core.T) {
	writer := &keyValueWriteCloser{}
	core.AssertPanics(t, func() { _ = writer.Close() })
	core.AssertNil(t, writer.keyValueStore)
}

func TestAX7_ValueWriteCloser_Close_Ugly(t *core.T) {
	keyValueStore := newKeyValueStore(t)
	writer := &keyValueWriteCloser{keyValueStore: keyValueStore, group: "", key: "", data: []byte("value")}
	err := writer.Close()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "value", mustGetAX7(t, keyValueStore, "", ""))
}

func mustCountAX7(t *core.T, keyValueStore *KeyValueStore, group string) int {
	t.Helper()

	count, err := keyValueStore.Count(group)
	core.RequireNoError(t, err)
	return count
}

func mustGetAX7(t *core.T, keyValueStore *KeyValueStore, group, key string) string {
	t.Helper()

	value, err := keyValueStore.Get(group, key)
	core.RequireNoError(t, err)
	return value
}
