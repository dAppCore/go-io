package sqlite

import (
	goio "io"
	"io/fs"

	core "dappco.re/go"
)

func TestAX7_Medium_Read_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.Write("read.txt", "payload"))
	got, err := medium.Read("read.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", got)
}

func TestAX7_Medium_Read_Bad(t *core.T) {
	medium := newSqliteMedium(t)
	got, err := medium.Read("missing.txt")
	core.AssertError(t, err)
	core.AssertEqual(t, "", got)
}

func TestAX7_Medium_Read_Ugly(t *core.T) {
	medium := newSqliteMedium(t)
	got, err := medium.Read("")
	core.AssertError(t, err)
	core.AssertEqual(t, "", got)
}

func TestAX7_Medium_Write_Good(t *core.T) {
	medium := newSqliteMedium(t)
	err := medium.Write("write.txt", "payload")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("write.txt"))
}

func TestAX7_Medium_Write_Bad(t *core.T) {
	medium := newSqliteMedium(t)
	err := medium.Write("", "payload")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.IsFile(""))
}

func TestAX7_Medium_Write_Ugly(t *core.T) {
	medium := newSqliteMedium(t)
	err := medium.Write("nested/write.txt", "payload")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("nested/write.txt"))
}

func TestAX7_Medium_WriteMode_Good(t *core.T) {
	medium := newSqliteMedium(t)
	err := medium.WriteMode("mode.txt", "payload", 0600)
	info, statErr := medium.Stat("mode.txt")
	core.AssertNoError(t, err)
	core.AssertNoError(t, statErr)
	core.AssertEqual(t, fs.FileMode(0600), info.Mode().Perm())
}

func TestAX7_Medium_WriteMode_Bad(t *core.T) {
	medium := newSqliteMedium(t)
	err := medium.WriteMode("", "payload", 0600)
	core.AssertError(t, err)
	core.AssertFalse(t, medium.IsFile(""))
}

func TestAX7_Medium_WriteMode_Ugly(t *core.T) {
	medium := newSqliteMedium(t)
	err := medium.WriteMode("zero-mode.txt", "payload", 0)
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("zero-mode.txt"))
}

func TestAX7_Medium_EnsureDir_Good(t *core.T) {
	medium := newSqliteMedium(t)
	err := medium.EnsureDir("dir")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir("dir"))
}

func TestAX7_Medium_EnsureDir_Bad(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.Write("file", "payload"))
	err := medium.EnsureDir("file")
	core.AssertNoError(t, err)
}

func TestAX7_Medium_EnsureDir_Ugly(t *core.T) {
	medium := newSqliteMedium(t)
	err := medium.EnsureDir("a/b/c")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir("a/b/c"))
}

func TestAX7_Medium_IsFile_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.Write("file.txt", "payload"))
	got := medium.IsFile("file.txt")
	core.AssertTrue(t, got)
}

func TestAX7_Medium_IsFile_Bad(t *core.T) {
	medium := newSqliteMedium(t)
	got := medium.IsFile("missing.txt")
	core.AssertFalse(t, got)
}

func TestAX7_Medium_IsFile_Ugly(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.IsFile("dir")
	core.AssertFalse(t, got)
}

func TestAX7_Medium_Delete_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.Write("delete.txt", "payload"))
	err := medium.Delete("delete.txt")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("delete.txt"))
}

func TestAX7_Medium_Delete_Bad(t *core.T) {
	medium := newSqliteMedium(t)
	err := medium.Delete("missing.txt")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("missing.txt"))
}

func TestAX7_Medium_Delete_Ugly(t *core.T) {
	medium := newSqliteMedium(t)
	err := medium.Delete("")
	core.AssertError(t, err)
	core.AssertTrue(t, medium.Exists(""))
}

func TestAX7_Medium_DeleteAll_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.Write("tree/file.txt", "payload"))
	err := medium.DeleteAll("tree")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("tree/file.txt"))
}

func TestAX7_Medium_DeleteAll_Bad(t *core.T) {
	medium := newSqliteMedium(t)
	err := medium.DeleteAll("missing")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("missing"))
}

func TestAX7_Medium_DeleteAll_Ugly(t *core.T) {
	medium := newSqliteMedium(t)
	err := medium.DeleteAll("")
	core.AssertError(t, err)
	core.AssertTrue(t, medium.Exists(""))
}

func TestAX7_Medium_Rename_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.Write("old.txt", "payload"))
	err := medium.Rename("old.txt", "new.txt")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("new.txt"))
}

func TestAX7_Medium_Rename_Bad(t *core.T) {
	medium := newSqliteMedium(t)
	err := medium.Rename("missing.txt", "new.txt")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("new.txt"))
}

func TestAX7_Medium_Rename_Ugly(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.Write("old.txt", "payload"))
	err := medium.Rename("old.txt", "nested/new.txt")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("nested/new.txt"))
}

func TestAX7_Medium_List_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.Write("dir/a.txt", "payload"))
	entries, err := medium.List("dir")
	core.AssertNoError(t, err)
	core.AssertLen(t, entries, 1)
}

func TestAX7_Medium_List_Bad(t *core.T) {
	medium := newSqliteMedium(t)
	entries, err := medium.List("missing")
	core.AssertNoError(t, err)
	core.AssertNil(t, entries)
}

func TestAX7_Medium_List_Ugly(t *core.T) {
	medium := newSqliteMedium(t)
	entries, err := medium.List("")
	core.AssertNoError(t, err)
	core.AssertNil(t, entries)
}

func TestAX7_Medium_Stat_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.Write("stat.txt", "payload"))
	info, err := medium.Stat("stat.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "stat.txt", info.Name())
}

func TestAX7_Medium_Stat_Bad(t *core.T) {
	medium := newSqliteMedium(t)
	info, err := medium.Stat("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, info)
}

func TestAX7_Medium_Stat_Ugly(t *core.T) {
	medium := newSqliteMedium(t)
	info, err := medium.Stat("")
	core.AssertError(t, err)
	core.AssertNil(t, info)
}

func TestAX7_Medium_Open_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.Write("open.txt", "payload"))
	file, err := medium.Open("open.txt")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, file)
	core.RequireNoError(t, file.Close())
}

func TestAX7_Medium_Open_Bad(t *core.T) {
	medium := newSqliteMedium(t)
	file, err := medium.Open("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, file)
}

func TestAX7_Medium_Open_Ugly(t *core.T) {
	medium := newSqliteMedium(t)
	file, err := medium.Open("")
	core.AssertError(t, err)
	core.AssertNil(t, file)
}

func TestAX7_Medium_Create_Good(t *core.T) {
	medium := newSqliteMedium(t)
	writer, err := medium.Create("create.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestAX7_Medium_Create_Bad(t *core.T) {
	medium := newSqliteMedium(t)
	writer, err := medium.Create("")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestAX7_Medium_Create_Ugly(t *core.T) {
	medium := newSqliteMedium(t)
	writer, err := medium.Create("nested/create.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestAX7_Medium_Append_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.Write("append.txt", "a"))
	writer, err := medium.Append("append.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("b"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestAX7_Medium_Append_Bad(t *core.T) {
	medium := newSqliteMedium(t)
	writer, err := medium.Append("")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestAX7_Medium_Append_Ugly(t *core.T) {
	medium := newSqliteMedium(t)
	writer, err := medium.Append("new.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("new"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestAX7_Medium_ReadStream_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.Write("stream.txt", "payload"))
	reader, err := medium.ReadStream("stream.txt")
	core.RequireNoError(t, err)
	data, readErr := goio.ReadAll(reader)
	core.AssertNoError(t, readErr)
	core.AssertEqual(t, "payload", string(data))
}

func TestAX7_Medium_ReadStream_Bad(t *core.T) {
	medium := newSqliteMedium(t)
	reader, err := medium.ReadStream("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, reader)
}

func TestAX7_Medium_ReadStream_Ugly(t *core.T) {
	medium := newSqliteMedium(t)
	reader, err := medium.ReadStream("")
	core.AssertError(t, err)
	core.AssertNil(t, reader)
}

func TestAX7_Medium_WriteStream_Good(t *core.T) {
	medium := newSqliteMedium(t)
	writer, err := medium.WriteStream("stream.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestAX7_Medium_WriteStream_Bad(t *core.T) {
	medium := newSqliteMedium(t)
	writer, err := medium.WriteStream("")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestAX7_Medium_WriteStream_Ugly(t *core.T) {
	medium := newSqliteMedium(t)
	writer, err := medium.WriteStream("nested/stream.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestAX7_Medium_Exists_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.Write("exists.txt", "payload"))
	got := medium.Exists("exists.txt")
	core.AssertTrue(t, got)
}

func TestAX7_Medium_Exists_Bad(t *core.T) {
	medium := newSqliteMedium(t)
	got := medium.Exists("missing.txt")
	core.AssertFalse(t, got)
}

func TestAX7_Medium_Exists_Ugly(t *core.T) {
	medium := newSqliteMedium(t)
	got := medium.Exists("")
	core.AssertTrue(t, got)
}

func TestAX7_Medium_IsDir_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.IsDir("dir")
	core.AssertTrue(t, got)
}

func TestAX7_Medium_IsDir_Bad(t *core.T) {
	medium := newSqliteMedium(t)
	got := medium.IsDir("missing")
	core.AssertFalse(t, got)
}

func TestAX7_Medium_IsDir_Ugly(t *core.T) {
	medium := newSqliteMedium(t)
	got := medium.IsDir("")
	core.AssertFalse(t, got)
}

func TestAX7_New_Good(t *core.T) {
	medium, err := New(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	defer medium.Close()
	core.AssertEqual(t, "files", medium.table)
}

func TestAX7_New_Bad(t *core.T) {
	medium, err := New(Options{})
	core.AssertError(t, err)
	core.AssertNil(t, medium)
}

func TestAX7_New_Ugly(t *core.T) {
	medium, err := New(Options{Path: ":memory:", Table: "ax7_files"})
	core.RequireNoError(t, err)
	defer medium.Close()
	core.AssertEqual(t, "ax7_files", medium.table)
}

func TestAX7_Medium_Close_Good(t *core.T) {
	medium, err := New(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	err = medium.Close()
	core.AssertNoError(t, err)
}

func TestAX7_Medium_Close_Bad(t *core.T) {
	medium := &Medium{}
	err := medium.Close()
	core.AssertNoError(t, err)
}

func TestAX7_Medium_Close_Ugly(t *core.T) {
	medium, err := New(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	core.AssertNoError(t, medium.Close())
	core.AssertNoError(t, medium.Close())
}

func TestAX7_Info_Name_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.Write("info.txt", "abc"))
	info, err := medium.Stat("info.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "info.txt", info.Name())
}

func TestAX7_Info_Name_Bad(t *core.T) {
	info := &fileInfo{}
	got := info.Name()
	core.AssertEqual(t, "", got)
}

func TestAX7_Info_Name_Ugly(t *core.T) {
	info := &fileInfo{name: ""}
	got := info.Name()
	core.AssertEqual(t, "", got)
}

func TestAX7_Info_Size_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.Write("size.txt", "abcd"))
	info, err := medium.Stat("size.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, int64(4), info.Size())
}

func TestAX7_Info_Size_Bad(t *core.T) {
	info := &fileInfo{}
	got := info.Size()
	core.AssertEqual(t, int64(0), got)
}

func TestAX7_Info_Size_Ugly(t *core.T) {
	info := &fileInfo{size: -1}
	got := info.Size()
	core.AssertEqual(t, int64(-1), got)
}

func TestAX7_Info_Mode_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.WriteMode("mode.txt", "abc", 0600))
	info, err := medium.Stat("mode.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, fs.FileMode(0600), info.Mode().Perm())
}

func TestAX7_Info_Mode_Bad(t *core.T) {
	info := &fileInfo{}
	got := info.Mode()
	core.AssertEqual(t, fs.FileMode(0), got)
}

func TestAX7_Info_Mode_Ugly(t *core.T) {
	info := &fileInfo{mode: fs.ModeDir | 0755}
	got := info.Mode()
	core.AssertTrue(t, got.IsDir())
}

func TestAX7_Info_ModTime_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.Write("time.txt", "abc"))
	info, err := medium.Stat("time.txt")
	core.RequireNoError(t, err)
	core.AssertFalse(t, info.ModTime().IsZero())
}

func TestAX7_Info_ModTime_Bad(t *core.T) {
	info := &fileInfo{}
	got := info.ModTime()
	core.AssertTrue(t, got.IsZero())
}

func TestAX7_Info_ModTime_Ugly(t *core.T) {
	stamp := core.Now()
	info := &fileInfo{modTime: stamp}
	core.AssertEqual(t, stamp, info.ModTime())
}

func TestAX7_Info_IsDir_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.EnsureDir("dir"))
	info, err := medium.Stat("dir")
	core.RequireNoError(t, err)
	core.AssertTrue(t, info.IsDir())
}

func TestAX7_Info_IsDir_Bad(t *core.T) {
	info := &fileInfo{}
	got := info.IsDir()
	core.AssertFalse(t, got)
}

func TestAX7_Info_IsDir_Ugly(t *core.T) {
	info := &fileInfo{isDir: true}
	got := info.IsDir()
	core.AssertTrue(t, got)
}

func TestAX7_Info_Sys_Good(t *core.T) {
	info := &fileInfo{name: "sys"}
	got := info.Sys()
	core.AssertNil(t, got)
}

func TestAX7_Info_Sys_Bad(t *core.T) {
	info := &fileInfo{}
	got := info.Sys()
	core.AssertNil(t, got)
}

func TestAX7_Info_Sys_Ugly(t *core.T) {
	var info *fileInfo = &fileInfo{}
	got := info.Sys()
	core.AssertNil(t, got)
}

func TestAX7_Entry_Name_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.Write("dir/entry.txt", "abc"))
	entries, err := medium.List("dir")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "entry.txt", entries[0].Name())
}

func TestAX7_Entry_Name_Bad(t *core.T) {
	entry := &dirEntry{}
	got := entry.Name()
	core.AssertEqual(t, "", got)
}

func TestAX7_Entry_Name_Ugly(t *core.T) {
	entry := &dirEntry{name: ""}
	got := entry.Name()
	core.AssertEqual(t, "", got)
}

func TestAX7_Entry_IsDir_Good(t *core.T) {
	entry := &dirEntry{name: "dir", isDir: true}
	got := entry.IsDir()
	core.AssertTrue(t, got)
}

func TestAX7_Entry_IsDir_Bad(t *core.T) {
	entry := &dirEntry{name: "file"}
	got := entry.IsDir()
	core.AssertFalse(t, got)
}

func TestAX7_Entry_IsDir_Ugly(t *core.T) {
	entry := &dirEntry{}
	got := entry.IsDir()
	core.AssertFalse(t, got)
}

func TestAX7_Entry_Type_Good(t *core.T) {
	entry := &dirEntry{name: "dir", isDir: true, mode: fs.ModeDir | 0755}
	got := entry.Type()
	core.AssertTrue(t, got.IsDir())
}

func TestAX7_Entry_Type_Bad(t *core.T) {
	entry := &dirEntry{name: "file", mode: 0644}
	got := entry.Type()
	core.AssertEqual(t, fs.FileMode(0), got)
}

func TestAX7_Entry_Type_Ugly(t *core.T) {
	entry := &dirEntry{}
	got := entry.Type()
	core.AssertEqual(t, fs.FileMode(0), got)
}

func TestAX7_Entry_Info_Good(t *core.T) {
	entry := &dirEntry{name: "file", info: &fileInfo{name: "file"}}
	info, err := entry.Info()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "file", info.Name())
}

func TestAX7_Entry_Info_Bad(t *core.T) {
	entry := &dirEntry{}
	info, err := entry.Info()
	core.AssertNoError(t, err)
	core.AssertNil(t, info)
}

func TestAX7_Entry_Info_Ugly(t *core.T) {
	entry := &dirEntry{info: &fileInfo{name: ""}}
	info, err := entry.Info()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "", info.Name())
}

func TestAX7_File_Read_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.Write("file.txt", "abc"))
	file, err := medium.Open("file.txt")
	core.RequireNoError(t, err)
	buf := make([]byte, 3)
	n, readErr := file.Read(buf)
	core.AssertNoError(t, readErr)
	core.AssertEqual(t, 3, n)
}

func TestAX7_File_Read_Bad(t *core.T) {
	file := &sqliteFile{}
	buf := make([]byte, 1)
	n, err := file.Read(buf)
	core.AssertErrorIs(t, err, goio.EOF)
	core.AssertEqual(t, 0, n)
}

func TestAX7_File_Read_Ugly(t *core.T) {
	file := &sqliteFile{content: []byte("abc"), offset: 2}
	buf := make([]byte, 4)
	n, err := file.Read(buf)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 1, n)
}

func TestAX7_File_Stat_Good(t *core.T) {
	file := &sqliteFile{name: "file.txt", content: []byte("abc"), mode: 0644}
	info, err := file.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "file.txt", info.Name())
}

func TestAX7_File_Stat_Bad(t *core.T) {
	file := &sqliteFile{}
	info, err := file.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "", info.Name())
}

func TestAX7_File_Stat_Ugly(t *core.T) {
	file := &sqliteFile{name: "empty", content: nil}
	info, err := file.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, int64(0), info.Size())
}

func TestAX7_File_Close_Good(t *core.T) {
	file := &sqliteFile{name: "file.txt"}
	err := file.Close()
	core.AssertNoError(t, err)
}

func TestAX7_File_Close_Bad(t *core.T) {
	file := &sqliteFile{}
	err := file.Close()
	core.AssertNoError(t, err)
}

func TestAX7_File_Close_Ugly(t *core.T) {
	file := &sqliteFile{}
	core.AssertNoError(t, file.Close())
	core.AssertNoError(t, file.Close())
}

func TestAX7_WriteCloser_Write_Good(t *core.T) {
	medium := newSqliteMedium(t)
	writer, err := medium.Create("writer.txt")
	core.RequireNoError(t, err)
	n, writeErr := writer.Write([]byte("abc"))
	core.AssertNoError(t, writeErr)
	core.AssertEqual(t, 3, n)
}

func TestAX7_WriteCloser_Write_Bad(t *core.T) {
	writer := &sqliteWriteCloser{}
	n, err := writer.Write(nil)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, n)
}

func TestAX7_WriteCloser_Write_Ugly(t *core.T) {
	writer := &sqliteWriteCloser{}
	n, err := writer.Write([]byte{})
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, n)
}

func TestAX7_WriteCloser_Close_Good(t *core.T) {
	medium := newSqliteMedium(t)
	writer, err := medium.Create("writer-close.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("abc"))
	core.RequireNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestAX7_WriteCloser_Close_Bad(t *core.T) {
	writer := &sqliteWriteCloser{}
	core.AssertPanics(t, func() {
		_ = writer.Close()
	})
}

func TestAX7_WriteCloser_Close_Ugly(t *core.T) {
	medium := newSqliteMedium(t)
	writer, err := medium.Create("empty.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("x"))
	core.RequireNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}
