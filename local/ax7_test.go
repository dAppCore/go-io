package local

import (
	goio "io"
	"io/fs"

	core "dappco.re/go"
)

func ax7LocalMedium(t *core.T) *Medium {
	t.Helper()
	medium, err := New(t.TempDir())
	core.RequireNoError(t, err)
	return medium
}

func TestAX7_New_Good(t *core.T) {
	root := t.TempDir()
	medium, err := New(root)
	core.AssertNoError(t, err)
	core.AssertNotNil(t, medium)
}

func TestAX7_New_Bad(t *core.T) {
	medium, err := New("")
	core.AssertNoError(t, err)
	core.AssertNotEmpty(t, medium.filesystemRoot)
}

func TestAX7_New_Ugly(t *core.T) {
	medium, err := New(".")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, medium)
}

func TestAX7_Medium_Read_Good(t *core.T) {
	medium := ax7LocalMedium(t)
	core.RequireNoError(t, medium.Write("read.txt", "local"))
	got, err := medium.Read("read.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "local", got)
}

func TestAX7_Medium_Read_Bad(t *core.T) {
	medium := ax7LocalMedium(t)
	got, err := medium.Read("missing.txt")
	core.AssertError(t, err)
	core.AssertEqual(t, "", got)
}

func TestAX7_Medium_Read_Ugly(t *core.T) {
	medium := ax7LocalMedium(t)
	got, err := medium.Read("../missing.txt")
	core.AssertError(t, err)
	core.AssertEqual(t, "", got)
}

func TestAX7_Medium_Write_Good(t *core.T) {
	medium := ax7LocalMedium(t)
	err := medium.Write("write.txt", "local")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("write.txt"))
}

func TestAX7_Medium_Write_Bad(t *core.T) {
	medium := ax7LocalMedium(t)
	err := medium.Write("", "local")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.IsFile(""))
}

func TestAX7_Medium_Write_Ugly(t *core.T) {
	medium := ax7LocalMedium(t)
	err := medium.Write("nested/write.txt", "local")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("nested/write.txt"))
}

func TestAX7_Medium_WriteMode_Good(t *core.T) {
	medium := ax7LocalMedium(t)
	err := medium.WriteMode("mode.txt", "local", 0600)
	info, statErr := medium.Stat("mode.txt")
	core.AssertNoError(t, err)
	core.AssertNoError(t, statErr)
	core.AssertEqual(t, fs.FileMode(0600), info.Mode().Perm())
}

func TestAX7_Medium_WriteMode_Bad(t *core.T) {
	medium := ax7LocalMedium(t)
	err := medium.WriteMode("", "local", 0600)
	core.AssertError(t, err)
	core.AssertFalse(t, medium.IsFile(""))
}

func TestAX7_Medium_WriteMode_Ugly(t *core.T) {
	medium := ax7LocalMedium(t)
	err := medium.WriteMode("zero-mode.txt", "local", 0)
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("zero-mode.txt"))
}

func TestAX7_Medium_EnsureDir_Good(t *core.T) {
	medium := ax7LocalMedium(t)
	err := medium.EnsureDir("dir")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir("dir"))
}

func TestAX7_Medium_EnsureDir_Bad(t *core.T) {
	medium := ax7LocalMedium(t)
	core.RequireNoError(t, medium.Write("file", "local"))
	err := medium.EnsureDir("file")
	core.AssertError(t, err)
}

func TestAX7_Medium_EnsureDir_Ugly(t *core.T) {
	medium := ax7LocalMedium(t)
	err := medium.EnsureDir("a/b/c")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir("a/b/c"))
}

func TestAX7_Medium_IsFile_Good(t *core.T) {
	medium := ax7LocalMedium(t)
	core.RequireNoError(t, medium.Write("file.txt", "local"))
	got := medium.IsFile("file.txt")
	core.AssertTrue(t, got)
}

func TestAX7_Medium_IsFile_Bad(t *core.T) {
	medium := ax7LocalMedium(t)
	got := medium.IsFile("missing.txt")
	core.AssertFalse(t, got)
}

func TestAX7_Medium_IsFile_Ugly(t *core.T) {
	medium := ax7LocalMedium(t)
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.IsFile("dir")
	core.AssertFalse(t, got)
}

func TestAX7_Medium_Delete_Good(t *core.T) {
	medium := ax7LocalMedium(t)
	core.RequireNoError(t, medium.Write("delete.txt", "local"))
	err := medium.Delete("delete.txt")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("delete.txt"))
}

func TestAX7_Medium_Delete_Bad(t *core.T) {
	medium := ax7LocalMedium(t)
	err := medium.Delete("missing.txt")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("missing.txt"))
}

func TestAX7_Medium_Delete_Ugly(t *core.T) {
	medium := ax7LocalMedium(t)
	err := medium.Delete("")
	core.AssertError(t, err)
	core.AssertTrue(t, medium.Exists(""))
}

func TestAX7_Medium_DeleteAll_Good(t *core.T) {
	medium := ax7LocalMedium(t)
	core.RequireNoError(t, medium.Write("tree/file.txt", "local"))
	err := medium.DeleteAll("tree")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("tree/file.txt"))
}

func TestAX7_Medium_DeleteAll_Bad(t *core.T) {
	medium := ax7LocalMedium(t)
	err := medium.DeleteAll("missing")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("missing"))
}

func TestAX7_Medium_DeleteAll_Ugly(t *core.T) {
	medium := ax7LocalMedium(t)
	err := medium.DeleteAll("")
	core.AssertError(t, err)
	core.AssertTrue(t, medium.Exists(""))
}

func TestAX7_Medium_Rename_Good(t *core.T) {
	medium := ax7LocalMedium(t)
	core.RequireNoError(t, medium.Write("old.txt", "local"))
	err := medium.Rename("old.txt", "new.txt")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("new.txt"))
}

func TestAX7_Medium_Rename_Bad(t *core.T) {
	medium := ax7LocalMedium(t)
	err := medium.Rename("missing.txt", "new.txt")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("new.txt"))
}

func TestAX7_Medium_Rename_Ugly(t *core.T) {
	medium := ax7LocalMedium(t)
	core.RequireNoError(t, medium.Write("old.txt", "local"))
	err := medium.Rename("old.txt", "../new.txt")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("new.txt"))
}

func TestAX7_Medium_List_Good(t *core.T) {
	medium := ax7LocalMedium(t)
	core.RequireNoError(t, medium.Write("dir/a.txt", "a"))
	entries, err := medium.List("dir")
	core.AssertNoError(t, err)
	core.AssertLen(t, entries, 1)
}

func TestAX7_Medium_List_Bad(t *core.T) {
	medium := ax7LocalMedium(t)
	entries, err := medium.List("missing")
	core.AssertError(t, err)
	core.AssertNil(t, entries)
}

func TestAX7_Medium_List_Ugly(t *core.T) {
	medium := ax7LocalMedium(t)
	entries, err := medium.List("")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, entries)
}

func TestAX7_Medium_Stat_Good(t *core.T) {
	medium := ax7LocalMedium(t)
	core.RequireNoError(t, medium.Write("stat.txt", "local"))
	info, err := medium.Stat("stat.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "stat.txt", info.Name())
}

func TestAX7_Medium_Stat_Bad(t *core.T) {
	medium := ax7LocalMedium(t)
	info, err := medium.Stat("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, info)
}

func TestAX7_Medium_Stat_Ugly(t *core.T) {
	medium := ax7LocalMedium(t)
	info, err := medium.Stat("")
	core.AssertNoError(t, err)
	core.AssertTrue(t, info.IsDir())
}

func TestAX7_Medium_Open_Good(t *core.T) {
	medium := ax7LocalMedium(t)
	core.RequireNoError(t, medium.Write("open.txt", "local"))
	file, err := medium.Open("open.txt")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, file)
	core.RequireNoError(t, file.Close())
}

func TestAX7_Medium_Open_Bad(t *core.T) {
	medium := ax7LocalMedium(t)
	file, err := medium.Open("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, file)
}

func TestAX7_Medium_Open_Ugly(t *core.T) {
	medium := ax7LocalMedium(t)
	file, err := medium.Open("")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, file)
	core.RequireNoError(t, file.Close())
}

func TestAX7_Medium_Create_Good(t *core.T) {
	medium := ax7LocalMedium(t)
	writer, err := medium.Create("create.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("local"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestAX7_Medium_Create_Bad(t *core.T) {
	medium := ax7LocalMedium(t)
	writer, err := medium.Create("")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestAX7_Medium_Create_Ugly(t *core.T) {
	medium := ax7LocalMedium(t)
	writer, err := medium.Create("nested/create.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("local"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestAX7_Medium_Append_Good(t *core.T) {
	medium := ax7LocalMedium(t)
	core.RequireNoError(t, medium.Write("append.txt", "a"))
	writer, err := medium.Append("append.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("b"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestAX7_Medium_Append_Bad(t *core.T) {
	medium := ax7LocalMedium(t)
	writer, err := medium.Append("")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestAX7_Medium_Append_Ugly(t *core.T) {
	medium := ax7LocalMedium(t)
	writer, err := medium.Append("new.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("new"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestAX7_Medium_ReadStream_Good(t *core.T) {
	medium := ax7LocalMedium(t)
	core.RequireNoError(t, medium.Write("stream.txt", "local"))
	reader, err := medium.ReadStream("stream.txt")
	core.RequireNoError(t, err)
	data, readErr := goio.ReadAll(reader)
	core.AssertNoError(t, readErr)
	core.AssertEqual(t, "local", string(data))
}

func TestAX7_Medium_ReadStream_Bad(t *core.T) {
	medium := ax7LocalMedium(t)
	reader, err := medium.ReadStream("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, reader)
}

func TestAX7_Medium_ReadStream_Ugly(t *core.T) {
	medium := ax7LocalMedium(t)
	reader, err := medium.ReadStream("")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, reader)
	core.RequireNoError(t, reader.Close())
}

func TestAX7_Medium_WriteStream_Good(t *core.T) {
	medium := ax7LocalMedium(t)
	writer, err := medium.WriteStream("stream.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("local"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestAX7_Medium_WriteStream_Bad(t *core.T) {
	medium := ax7LocalMedium(t)
	writer, err := medium.WriteStream("")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestAX7_Medium_WriteStream_Ugly(t *core.T) {
	medium := ax7LocalMedium(t)
	writer, err := medium.WriteStream("nested/stream.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("local"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestAX7_Medium_Exists_Good(t *core.T) {
	medium := ax7LocalMedium(t)
	core.RequireNoError(t, medium.Write("exists.txt", "local"))
	got := medium.Exists("exists.txt")
	core.AssertTrue(t, got)
}

func TestAX7_Medium_Exists_Bad(t *core.T) {
	medium := ax7LocalMedium(t)
	got := medium.Exists("missing.txt")
	core.AssertFalse(t, got)
}

func TestAX7_Medium_Exists_Ugly(t *core.T) {
	medium := ax7LocalMedium(t)
	got := medium.Exists("")
	core.AssertTrue(t, got)
}

func TestAX7_Medium_IsDir_Good(t *core.T) {
	medium := ax7LocalMedium(t)
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.IsDir("dir")
	core.AssertTrue(t, got)
}

func TestAX7_Medium_IsDir_Bad(t *core.T) {
	medium := ax7LocalMedium(t)
	got := medium.IsDir("missing")
	core.AssertFalse(t, got)
}

func TestAX7_Medium_IsDir_Ugly(t *core.T) {
	medium := ax7LocalMedium(t)
	got := medium.IsDir("")
	core.AssertFalse(t, got)
}
