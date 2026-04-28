package pwa

import (
	goio "io"
	"io/fs"

	core "dappco.re/go"
	borgdatanode "forge.lthn.ai/Snider/Borg/pkg/datanode"
)

func ax7PWAMedium() *Medium {
	dataNode := borgdatanode.New()
	dataNode.AddData("page", []byte("payload"))
	dataNode.AddData("pages/item.txt", []byte("item"))
	return &Medium{url: "https://example.test", dataNode: dataNode}
}

func assertAX7PWANotImplemented(t *core.T, err error) {
	t.Helper()
	core.AssertError(t, err)
}

func TestAX7_New_Good(t *core.T) {
	medium, err := New(Options{URL: "https://example.test"})
	core.AssertNoError(t, err)
	core.AssertEqual(t, "https://example.test", medium.url)
}

func TestAX7_New_Bad(t *core.T) {
	medium, err := New(Options{})
	core.AssertNoError(t, err)
	core.AssertEqual(t, "", medium.url)
}

func TestAX7_New_Ugly(t *core.T) {
	medium, err := New(Options{URL: "  "})
	core.AssertNoError(t, err)
	core.AssertEqual(t, "  ", medium.url)
}

func TestAX7_RegisterFactory_Good(t *core.T) {
	result := RegisterFactory("ax7-pwa", New)
	factory, ok := FactoryFor("ax7-pwa")
	core.AssertTrue(t, result.OK)
	core.AssertTrue(t, ok)
	core.AssertNotNil(t, factory)
}

func TestAX7_RegisterFactory_Bad(t *core.T) {
	result := RegisterFactory("ax7-pwa-nil", nil)
	factory, ok := FactoryFor("ax7-pwa-nil")
	core.AssertTrue(t, result.OK)
	core.AssertTrue(t, ok)
	core.AssertNil(t, factory)
}

func TestAX7_RegisterFactory_Ugly(t *core.T) {
	result := RegisterFactory("", New)
	factory, ok := FactoryFor("")
	core.AssertTrue(t, result.OK)
	core.AssertTrue(t, ok)
	core.AssertNotNil(t, factory)
}

func TestAX7_FactoryFor_Good(t *core.T) {
	factory, ok := FactoryFor(Scheme)
	core.AssertTrue(t, ok)
	core.AssertNotNil(t, factory)
}

func TestAX7_FactoryFor_Bad(t *core.T) {
	factory, ok := FactoryFor("missing-pwa")
	core.AssertFalse(t, ok)
	core.AssertNil(t, factory)
}

func TestAX7_FactoryFor_Ugly(t *core.T) {
	RegisterFactory("ax7-pwa-empty", New)
	factory, ok := FactoryFor("ax7-pwa-empty")
	core.AssertTrue(t, ok)
	core.AssertNotNil(t, factory)
}

func TestAX7_RegisterActions_Good(t *core.T) {
	c := core.New()
	RegisterActions(c)
	core.AssertTrue(t, c.Action(ActionRead).Exists())
	core.AssertTrue(t, c.Action(ActionWrite).Exists())
}

func TestAX7_RegisterActions_Bad(t *core.T) {
	core.AssertNotPanics(t, func() { RegisterActions(nil) })
	c := core.New()
	core.AssertFalse(t, c.Action(ActionRead).Exists())
}

func TestAX7_RegisterActions_Ugly(t *core.T) {
	c := core.New()
	RegisterActions(c)
	core.AssertNotPanics(t, func() { RegisterActions(c) })
}

func TestAX7_Medium_Read_Good(t *core.T) {
	medium := ax7PWAMedium()
	got, err := medium.Read("page")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", got)
}

func TestAX7_Medium_Read_Bad(t *core.T) {
	medium := ax7PWAMedium()
	got, err := medium.Read("")
	assertAX7PWANotImplemented(t, err)
	core.AssertEqual(t, "", got)
}

func TestAX7_Medium_Read_Ugly(t *core.T) {
	medium := ax7PWAMedium()
	got, err := medium.Read("../page")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", got)
}

func TestAX7_Medium_Write_Good(t *core.T) {
	medium := ax7PWAMedium()
	err := medium.Write("page", "content")
	assertAX7PWANotImplemented(t, err)
	core.AssertTrue(t, medium.Exists("page"))
}

func TestAX7_Medium_Write_Bad(t *core.T) {
	medium := ax7PWAMedium()
	err := medium.Write("", "content")
	assertAX7PWANotImplemented(t, err)
	core.AssertFalse(t, medium.IsFile(""))
}

func TestAX7_Medium_Write_Ugly(t *core.T) {
	medium := ax7PWAMedium()
	err := medium.Write("../page", "content")
	assertAX7PWANotImplemented(t, err)
	core.AssertTrue(t, medium.Exists("../page"))
}

func TestAX7_Medium_WriteMode_Good(t *core.T) {
	medium := ax7PWAMedium()
	err := medium.WriteMode("page", "content", 0600)
	assertAX7PWANotImplemented(t, err)
	core.AssertTrue(t, medium.Exists("page"))
}

func TestAX7_Medium_WriteMode_Bad(t *core.T) {
	medium := ax7PWAMedium()
	err := medium.WriteMode("", "content", fs.FileMode(0))
	assertAX7PWANotImplemented(t, err)
	core.AssertFalse(t, medium.IsFile(""))
}

func TestAX7_Medium_WriteMode_Ugly(t *core.T) {
	medium := ax7PWAMedium()
	err := medium.WriteMode("page", "content", fs.ModeDir)
	assertAX7PWANotImplemented(t, err)
	core.AssertTrue(t, medium.Exists("page"))
}

func TestAX7_Medium_EnsureDir_Good(t *core.T) {
	medium := ax7PWAMedium()
	err := medium.EnsureDir("pages")
	assertAX7PWANotImplemented(t, err)
	core.AssertTrue(t, medium.IsDir("pages"))
}

func TestAX7_Medium_EnsureDir_Bad(t *core.T) {
	medium := ax7PWAMedium()
	err := medium.EnsureDir("")
	assertAX7PWANotImplemented(t, err)
	core.AssertTrue(t, medium.IsDir(""))
}

func TestAX7_Medium_EnsureDir_Ugly(t *core.T) {
	medium := ax7PWAMedium()
	err := medium.EnsureDir("../pages")
	assertAX7PWANotImplemented(t, err)
	core.AssertTrue(t, medium.Exists("../pages"))
}

func TestAX7_Medium_IsFile_Good(t *core.T) {
	medium := ax7PWAMedium()
	got := medium.IsFile("page")
	core.AssertTrue(t, got)
}

func TestAX7_Medium_IsFile_Bad(t *core.T) {
	medium := ax7PWAMedium()
	got := medium.IsFile("")
	core.AssertFalse(t, got)
}

func TestAX7_Medium_IsFile_Ugly(t *core.T) {
	medium := ax7PWAMedium()
	got := medium.IsFile("../page")
	core.AssertTrue(t, got)
}

func TestAX7_Medium_Delete_Good(t *core.T) {
	medium := ax7PWAMedium()
	err := medium.Delete("page")
	assertAX7PWANotImplemented(t, err)
	core.AssertTrue(t, medium.Exists("page"))
}

func TestAX7_Medium_Delete_Bad(t *core.T) {
	medium := ax7PWAMedium()
	err := medium.Delete("")
	assertAX7PWANotImplemented(t, err)
	core.AssertTrue(t, medium.Exists(""))
}

func TestAX7_Medium_Delete_Ugly(t *core.T) {
	medium := ax7PWAMedium()
	err := medium.Delete("../page")
	assertAX7PWANotImplemented(t, err)
	core.AssertTrue(t, medium.Exists("../page"))
}

func TestAX7_Medium_DeleteAll_Good(t *core.T) {
	medium := ax7PWAMedium()
	err := medium.DeleteAll("pages")
	assertAX7PWANotImplemented(t, err)
	core.AssertTrue(t, medium.Exists("pages"))
}

func TestAX7_Medium_DeleteAll_Bad(t *core.T) {
	medium := ax7PWAMedium()
	err := medium.DeleteAll("")
	assertAX7PWANotImplemented(t, err)
	core.AssertTrue(t, medium.Exists(""))
}

func TestAX7_Medium_DeleteAll_Ugly(t *core.T) {
	medium := ax7PWAMedium()
	err := medium.DeleteAll("../pages")
	assertAX7PWANotImplemented(t, err)
	core.AssertTrue(t, medium.Exists("../pages"))
}

func TestAX7_Medium_Rename_Good(t *core.T) {
	medium := ax7PWAMedium()
	err := medium.Rename("old", "new")
	assertAX7PWANotImplemented(t, err)
	core.AssertFalse(t, medium.Exists("new"))
}

func TestAX7_Medium_Rename_Bad(t *core.T) {
	medium := ax7PWAMedium()
	err := medium.Rename("", "new")
	assertAX7PWANotImplemented(t, err)
	core.AssertFalse(t, medium.Exists("new"))
}

func TestAX7_Medium_Rename_Ugly(t *core.T) {
	medium := ax7PWAMedium()
	err := medium.Rename("old", "../new")
	assertAX7PWANotImplemented(t, err)
	core.AssertFalse(t, medium.Exists("../new"))
}

func TestAX7_Medium_List_Good(t *core.T) {
	medium := ax7PWAMedium()
	entries, err := medium.List("pages")
	core.AssertNoError(t, err)
	core.AssertLen(t, entries, 1)
}

func TestAX7_Medium_List_Bad(t *core.T) {
	medium := ax7PWAMedium()
	entries, err := medium.List("")
	core.AssertNoError(t, err)
	core.AssertNotEmpty(t, entries)
}

func TestAX7_Medium_List_Ugly(t *core.T) {
	medium := ax7PWAMedium()
	entries, err := medium.List("../pages")
	core.AssertNoError(t, err)
	core.AssertLen(t, entries, 1)
}

func TestAX7_Medium_Stat_Good(t *core.T) {
	medium := ax7PWAMedium()
	info, err := medium.Stat("page")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "page", info.Name())
}

func TestAX7_Medium_Stat_Bad(t *core.T) {
	medium := ax7PWAMedium()
	info, err := medium.Stat("")
	core.AssertNoError(t, err)
	core.AssertTrue(t, info.IsDir())
}

func TestAX7_Medium_Stat_Ugly(t *core.T) {
	medium := ax7PWAMedium()
	info, err := medium.Stat("../page")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "page", info.Name())
}

func TestAX7_Medium_Open_Good(t *core.T) {
	medium := ax7PWAMedium()
	file, err := medium.Open("page")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, file)
	core.RequireNoError(t, file.Close())
}

func TestAX7_Medium_Open_Bad(t *core.T) {
	medium := ax7PWAMedium()
	file, err := medium.Open("")
	assertAX7PWANotImplemented(t, err)
	core.AssertNil(t, file)
}

func TestAX7_Medium_Open_Ugly(t *core.T) {
	medium := ax7PWAMedium()
	file, err := medium.Open("../page")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, file)
	core.RequireNoError(t, file.Close())
}

func TestAX7_Medium_Create_Good(t *core.T) {
	medium := ax7PWAMedium()
	writer, err := medium.Create("page")
	assertAX7PWANotImplemented(t, err)
	core.AssertNil(t, writer)
}

func TestAX7_Medium_Create_Bad(t *core.T) {
	medium := ax7PWAMedium()
	writer, err := medium.Create("")
	assertAX7PWANotImplemented(t, err)
	core.AssertNil(t, writer)
}

func TestAX7_Medium_Create_Ugly(t *core.T) {
	medium := ax7PWAMedium()
	writer, err := medium.Create("../page")
	assertAX7PWANotImplemented(t, err)
	core.AssertNil(t, writer)
}

func TestAX7_Medium_Append_Good(t *core.T) {
	medium := ax7PWAMedium()
	writer, err := medium.Append("page")
	assertAX7PWANotImplemented(t, err)
	core.AssertNil(t, writer)
}

func TestAX7_Medium_Append_Bad(t *core.T) {
	medium := ax7PWAMedium()
	writer, err := medium.Append("")
	assertAX7PWANotImplemented(t, err)
	core.AssertNil(t, writer)
}

func TestAX7_Medium_Append_Ugly(t *core.T) {
	medium := ax7PWAMedium()
	writer, err := medium.Append("../page")
	assertAX7PWANotImplemented(t, err)
	core.AssertNil(t, writer)
}

func TestAX7_Medium_ReadStream_Good(t *core.T) {
	medium := ax7PWAMedium()
	reader, err := medium.ReadStream("page")
	core.RequireNoError(t, err)
	data, readErr := goio.ReadAll(reader)
	core.AssertNoError(t, readErr)
	core.AssertEqual(t, "payload", string(data))
	core.RequireNoError(t, reader.Close())
}

func TestAX7_Medium_ReadStream_Bad(t *core.T) {
	medium := ax7PWAMedium()
	reader, err := medium.ReadStream("")
	assertAX7PWANotImplemented(t, err)
	core.AssertNil(t, reader)
}

func TestAX7_Medium_ReadStream_Ugly(t *core.T) {
	medium := ax7PWAMedium()
	reader, err := medium.ReadStream("../page")
	core.RequireNoError(t, err)
	data, readErr := goio.ReadAll(reader)
	core.AssertNoError(t, readErr)
	core.AssertEqual(t, "payload", string(data))
	core.RequireNoError(t, reader.Close())
}

func TestAX7_Medium_WriteStream_Good(t *core.T) {
	medium := ax7PWAMedium()
	writer, err := medium.WriteStream("page")
	assertAX7PWANotImplemented(t, err)
	core.AssertNil(t, writer)
}

func TestAX7_Medium_WriteStream_Bad(t *core.T) {
	medium := ax7PWAMedium()
	writer, err := medium.WriteStream("")
	assertAX7PWANotImplemented(t, err)
	core.AssertNil(t, writer)
}

func TestAX7_Medium_WriteStream_Ugly(t *core.T) {
	medium := ax7PWAMedium()
	writer, err := medium.WriteStream("../page")
	assertAX7PWANotImplemented(t, err)
	core.AssertNil(t, writer)
}

func TestAX7_Medium_Exists_Good(t *core.T) {
	medium := ax7PWAMedium()
	got := medium.Exists("page")
	core.AssertTrue(t, got)
}

func TestAX7_Medium_Exists_Bad(t *core.T) {
	medium := ax7PWAMedium()
	got := medium.Exists("")
	core.AssertTrue(t, got)
}

func TestAX7_Medium_Exists_Ugly(t *core.T) {
	medium := ax7PWAMedium()
	got := medium.Exists("../page")
	core.AssertTrue(t, got)
}

func TestAX7_Medium_IsDir_Good(t *core.T) {
	medium := ax7PWAMedium()
	got := medium.IsDir("pages")
	core.AssertTrue(t, got)
}

func TestAX7_Medium_IsDir_Bad(t *core.T) {
	medium := ax7PWAMedium()
	got := medium.IsDir("")
	core.AssertTrue(t, got)
}

func TestAX7_Medium_IsDir_Ugly(t *core.T) {
	medium := ax7PWAMedium()
	got := medium.IsDir("../pages")
	core.AssertTrue(t, got)
}
