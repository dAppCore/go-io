package sftp

import (
	"context"
	goio "io"
	"io/fs"

	core "dappco.re/go"
)

func TestAX7_New_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.AssertNotNil(t, medium)
	core.AssertEqual(t, "/", medium.root)
}

func TestAX7_New_Bad(t *core.T) {
	medium, err := New(Options{})
	core.AssertError(t, err)
	core.AssertNil(t, medium)
}

func TestAX7_New_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	err := medium.EnsureDir("rooted")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir("rooted"))
}

func TestAX7_Medium_Close_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	err := medium.Close()
	core.AssertNoError(t, err)
	core.AssertNotNil(t, medium.client)
}

func TestAX7_Medium_Close_Bad(t *core.T) {
	medium := &Medium{}
	err := medium.Close()
	core.AssertNoError(t, err)
	core.AssertNil(t, medium.client)
}

func TestAX7_Medium_Close_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.Close())
	err := medium.Close()
	core.AssertNoError(t, err)
}

func TestAX7_Medium_Read_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.Write("read.txt", "payload"))
	got, err := medium.Read("read.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", got)
}

func TestAX7_Medium_Read_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)
	got, err := medium.Read("")
	core.AssertError(t, err)
	core.AssertEqual(t, "", got)
}

func TestAX7_Medium_Read_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.Write("safe/file.txt", "payload"))
	got, err := medium.Read("/safe/../safe/file.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", got)
}

func TestAX7_Medium_Write_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	err := medium.Write("write.txt", "payload")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("write.txt"))
}

func TestAX7_Medium_Write_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)
	err := medium.Write("", "payload")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.IsFile(""))
}

func TestAX7_Medium_Write_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	err := medium.Write("nested/write.txt", "")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("nested/write.txt"))
}

func TestAX7_Medium_WriteMode_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	err := medium.WriteMode("mode.txt", "payload", 0600)
	info, statErr := medium.Stat("mode.txt")
	core.AssertNoError(t, err)
	core.AssertNoError(t, statErr)
	core.AssertEqual(t, fs.FileMode(0644), info.Mode().Perm())
}

func TestAX7_Medium_WriteMode_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)
	err := medium.WriteMode("", "payload", 0600)
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists(""))
}

func TestAX7_Medium_WriteMode_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	err := medium.WriteMode("zero-mode.txt", "payload", 0)
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("zero-mode.txt"))
}

func TestAX7_Medium_EnsureDir_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	err := medium.EnsureDir("dir")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir("dir"))
}

func TestAX7_Medium_EnsureDir_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)
	err := medium.EnsureDir("")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.IsDir(""))
}

func TestAX7_Medium_EnsureDir_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	err := medium.EnsureDir("a/b/c")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir("a/b/c"))
}

func TestAX7_Medium_IsFile_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.Write("file.txt", "payload"))
	got := medium.IsFile("file.txt")
	core.AssertTrue(t, got)
}

func TestAX7_Medium_IsFile_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)
	got := medium.IsFile("")
	core.AssertFalse(t, got)
}

func TestAX7_Medium_IsFile_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.IsFile("dir")
	core.AssertFalse(t, got)
}

func TestAX7_Medium_Delete_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.Write("delete.txt", "payload"))
	err := medium.Delete("delete.txt")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("delete.txt"))
}

func TestAX7_Medium_Delete_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)
	err := medium.Delete("")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists(""))
}

func TestAX7_Medium_Delete_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	err := medium.Delete("missing.txt")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("missing.txt"))
}

func TestAX7_Medium_DeleteAll_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.Write("tree/file.txt", "payload"))
	err := medium.DeleteAll("tree")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("tree/file.txt"))
}

func TestAX7_Medium_DeleteAll_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)
	err := medium.DeleteAll("")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists(""))
}

func TestAX7_Medium_DeleteAll_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	err := medium.DeleteAll("missing")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("missing"))
}

func TestAX7_Medium_Rename_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.Write("old.txt", "payload"))
	err := medium.Rename("old.txt", "new.txt")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("new.txt"))
}

func TestAX7_Medium_Rename_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)
	err := medium.Rename("", "new.txt")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("new.txt"))
}

func TestAX7_Medium_Rename_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.Write("old.txt", "payload"))
	err := medium.Rename("old.txt", "nested/new.txt")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("nested/new.txt"))
}

func TestAX7_Medium_List_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.Write("dir/a.txt", "payload"))
	entries, err := medium.List("dir")
	core.AssertNoError(t, err)
	core.AssertLen(t, entries, 1)
}

func TestAX7_Medium_List_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)
	entries, err := medium.List("missing")
	core.AssertError(t, err)
	core.AssertNil(t, entries)
}

func TestAX7_Medium_List_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	entries, err := medium.List("")
	core.AssertNoError(t, err)
	core.AssertEmpty(t, entries)
}

func TestAX7_Medium_Stat_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.Write("stat.txt", "payload"))
	info, err := medium.Stat("stat.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "stat.txt", info.Name())
}

func TestAX7_Medium_Stat_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)
	info, err := medium.Stat("")
	core.AssertError(t, err)
	core.AssertNil(t, info)
}

func TestAX7_Medium_Stat_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.EnsureDir("dir"))
	info, err := medium.Stat("dir")
	core.AssertNoError(t, err)
	core.AssertTrue(t, info.IsDir())
}

func TestAX7_Medium_Open_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.Write("open.txt", "payload"))
	file, err := medium.Open("open.txt")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, file)
	core.RequireNoError(t, file.Close())
}

func TestAX7_Medium_Open_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)
	file, err := medium.Open("")
	core.AssertError(t, err)
	core.AssertNil(t, file)
}

func TestAX7_Medium_Open_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	file, err := medium.Open("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, file)
}

func TestAX7_Medium_Create_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	writer, err := medium.Create("create.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestAX7_Medium_Create_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)
	writer, err := medium.Create("")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestAX7_Medium_Create_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	writer, err := medium.Create("nested/create.txt")
	core.RequireNoError(t, err)
	core.AssertNoError(t, writer.Close())
	core.AssertTrue(t, medium.Exists("nested/create.txt"))
}

func TestAX7_Medium_Append_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.Write("append.txt", "a"))
	writer, err := medium.Append("append.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("b"))
	core.RequireNoError(t, writeErr)
	core.RequireNoError(t, writer.Close())
}

func TestAX7_Medium_Append_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)
	writer, err := medium.Append("")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestAX7_Medium_Append_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	writer, err := medium.Append("new.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("created"))
	core.RequireNoError(t, writeErr)
	core.RequireNoError(t, writer.Close())
}

func TestAX7_Medium_ReadStream_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.Write("stream.txt", "payload"))
	reader, err := medium.ReadStream("stream.txt")
	core.RequireNoError(t, err)
	defer reader.Close()
	data, readErr := goio.ReadAll(reader)
	core.AssertNoError(t, readErr)
	core.AssertEqual(t, "payload", string(data))
}

func TestAX7_Medium_ReadStream_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)
	reader, err := medium.ReadStream("")
	core.AssertError(t, err)
	core.AssertNil(t, reader)
}

func TestAX7_Medium_ReadStream_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	reader, err := medium.ReadStream("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, reader)
}

func TestAX7_Medium_WriteStream_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	writer, err := medium.WriteStream("stream.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestAX7_Medium_WriteStream_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)
	writer, err := medium.WriteStream("")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestAX7_Medium_WriteStream_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	writer, err := medium.WriteStream("nested/stream.txt")
	core.RequireNoError(t, err)
	core.AssertNoError(t, writer.Close())
	core.AssertTrue(t, medium.Exists("nested/stream.txt"))
}

func TestAX7_Medium_Exists_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.Write("exists.txt", "payload"))
	got := medium.Exists("exists.txt")
	core.AssertTrue(t, got)
}

func TestAX7_Medium_Exists_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)
	got := medium.Exists("")
	core.AssertFalse(t, got)
}

func TestAX7_Medium_Exists_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.Exists("dir")
	core.AssertTrue(t, got)
}

func TestAX7_Medium_IsDir_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.IsDir("dir")
	core.AssertTrue(t, got)
}

func TestAX7_Medium_IsDir_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)
	got := medium.IsDir("")
	core.AssertFalse(t, got)
}

func TestAX7_Medium_IsDir_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.Write("file.txt", "payload"))
	got := medium.IsDir("file.txt")
	core.AssertFalse(t, got)
}

func TestAX7_RegisterFactory_Good(t *core.T) {
	result := RegisterFactory("ax7-sftp-good", New)
	core.AssertTrue(t, result.OK)
	factory, ok := FactoryFor("ax7-sftp-good")
	core.AssertTrue(t, ok)
	core.AssertNotNil(t, factory)
}

func TestAX7_RegisterFactory_Bad(t *core.T) {
	result := RegisterFactory("ax7-sftp-bad", nil)
	core.AssertTrue(t, result.OK)
	factory, ok := FactoryFor("ax7-sftp-bad")
	core.AssertTrue(t, ok)
	core.AssertNil(t, factory)
}

func TestAX7_RegisterFactory_Ugly(t *core.T) {
	result := RegisterFactory("ax7-sftp-ugly", New)
	core.AssertTrue(t, result.OK)
	result = RegisterFactory("ax7-sftp-ugly", New)
	core.AssertTrue(t, result.OK)
}

func TestAX7_FactoryFor_Good(t *core.T) {
	RegisterFactory("ax7-sftp-factory", New)
	factory, ok := FactoryFor("ax7-sftp-factory")
	core.AssertTrue(t, ok)
	core.AssertNotNil(t, factory)
}

func TestAX7_FactoryFor_Bad(t *core.T) {
	factory, ok := FactoryFor("missing-sftp-factory")
	core.AssertFalse(t, ok)
	core.AssertNil(t, factory)
}

func TestAX7_FactoryFor_Ugly(t *core.T) {
	factory, ok := FactoryFor("")
	core.AssertFalse(t, ok)
	core.AssertNil(t, factory)
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
	result := c.Action(ActionRead).Run(context.Background(), core.NewOptions())
	core.AssertFalse(t, result.OK)
}
