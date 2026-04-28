package sqlite

import (
	goio "io"
	"io/fs"

	core "dappco.re/go"
)

func newSqliteMedium(t *core.T) *Medium {
	t.Helper()
	sqliteMedium, err := New(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	t.Cleanup(func() { sqliteMedium.Close() })
	return sqliteMedium
}

func TestSqlite_New_Good(t *core.T) {
	sqliteMedium, err := New(Options{Path: ":memory:"})
	core.RequireNoError(t, err)
	defer sqliteMedium.Close()
	core.AssertEqual(t, "files", sqliteMedium.table)
}

func TestSqlite_New_Options_Good(t *core.T) {
	sqliteMedium, err := New(Options{Path: ":memory:", Table: "custom"})
	core.RequireNoError(t, err)
	defer sqliteMedium.Close()
	core.AssertEqual(t, "custom", sqliteMedium.table)
}

func TestSqlite_New_EmptyPath_Bad(t *core.T) {
	_, err := New(Options{})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "database path is required")
}

func TestSqlite_ReadWrite_Good(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	err := sqliteMedium.Write("hello.txt", "world")
	core.RequireNoError(t, err)

	content, err := sqliteMedium.Read("hello.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "world", content)
}

func TestSqlite_ReadWrite_Overwrite_Good(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	core.RequireNoError(t, sqliteMedium.Write("file.txt", "first"))
	core.RequireNoError(t, sqliteMedium.Write("file.txt", "second"))

	content, err := sqliteMedium.Read("file.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "second", content)
}

func TestSqlite_ReadWrite_NestedPath_Good(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	err := sqliteMedium.Write("a/b/c.txt", "nested")
	core.RequireNoError(t, err)

	content, err := sqliteMedium.Read("a/b/c.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "nested", content)
}

func TestSqlite_Read_NotFound_Bad(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	_, err := sqliteMedium.Read("nonexistent.txt")
	core.AssertError(t, err)
}

func TestSqlite_Read_EmptyPath_Bad(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	_, err := sqliteMedium.Read("")
	core.AssertError(t, err)
}

func TestSqlite_Write_EmptyPath_Bad(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	err := sqliteMedium.Write("", "content")
	core.AssertError(t, err)
}

func TestSqlite_Read_IsDirectory_Bad(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	core.RequireNoError(t, sqliteMedium.EnsureDir("mydir"))
	_, err := sqliteMedium.Read("mydir")
	core.AssertError(t, err)
}

func TestSqlite_EnsureDir_Good(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	err := sqliteMedium.EnsureDir("mydir")
	core.RequireNoError(t, err)
	core.AssertTrue(t, sqliteMedium.IsDir("mydir"))
}

func TestSqlite_EnsureDir_EmptyPath_Good(t *core.T) {
	sqliteMedium := newSqliteMedium(t)
	err := sqliteMedium.EnsureDir("")
	core.AssertNoError(t, err)
}

func TestSqlite_EnsureDir_Idempotent_Good(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	core.RequireNoError(t, sqliteMedium.EnsureDir("mydir"))
	core.RequireNoError(t, sqliteMedium.EnsureDir("mydir"))
	core.AssertTrue(t, sqliteMedium.IsDir("mydir"))
}

func TestSqlite_IsFile_Good(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	core.RequireNoError(t, sqliteMedium.Write("file.txt", "content"))
	core.RequireNoError(t, sqliteMedium.EnsureDir("mydir"))

	core.AssertTrue(t, sqliteMedium.IsFile("file.txt"))
	core.AssertFalse(t, sqliteMedium.IsFile("mydir"))
	core.AssertFalse(t, sqliteMedium.IsFile("nonexistent"))
	core.AssertFalse(t, sqliteMedium.IsFile(""))
}

func TestSqlite_Delete_Good(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	core.RequireNoError(t, sqliteMedium.Write("to-delete.txt", "content"))
	core.AssertTrue(t, sqliteMedium.Exists("to-delete.txt"))

	err := sqliteMedium.Delete("to-delete.txt")
	core.RequireNoError(t, err)
	core.AssertFalse(t, sqliteMedium.Exists("to-delete.txt"))
}

func TestSqlite_Delete_EmptyDir_Good(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	core.RequireNoError(t, sqliteMedium.EnsureDir("emptydir"))
	core.AssertTrue(t, sqliteMedium.IsDir("emptydir"))

	err := sqliteMedium.Delete("emptydir")
	core.RequireNoError(t, err)
	core.AssertFalse(t, sqliteMedium.IsDir("emptydir"))
}

func TestSqlite_Delete_NotFound_Bad(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	err := sqliteMedium.Delete("nonexistent")
	core.AssertError(t, err)
}

func TestSqlite_Delete_EmptyPath_Bad(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	err := sqliteMedium.Delete("")
	core.AssertError(t, err)
}

func TestSqlite_Delete_NotEmpty_Bad(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	core.RequireNoError(t, sqliteMedium.EnsureDir("mydir"))
	core.RequireNoError(t, sqliteMedium.Write("mydir/file.txt", "content"))

	err := sqliteMedium.Delete("mydir")
	core.AssertError(t, err)
}

func TestSqlite_DeleteAll_Good(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	core.RequireNoError(t, sqliteMedium.Write("dir/file1.txt", "a"))
	core.RequireNoError(t, sqliteMedium.Write("dir/sub/file2.txt", "b"))
	core.RequireNoError(t, sqliteMedium.Write("other.txt", "c"))

	err := sqliteMedium.DeleteAll("dir")
	core.RequireNoError(t, err)

	core.AssertFalse(t, sqliteMedium.Exists("dir/file1.txt"))
	core.AssertFalse(t, sqliteMedium.Exists("dir/sub/file2.txt"))
	core.AssertTrue(t, sqliteMedium.Exists("other.txt"))
}

func TestSqlite_DeleteAll_SingleFile_Good(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	core.RequireNoError(t, sqliteMedium.Write("file.txt", "content"))

	err := sqliteMedium.DeleteAll("file.txt")
	core.RequireNoError(t, err)
	core.AssertFalse(t, sqliteMedium.Exists("file.txt"))
}

func TestSqlite_DeleteAll_NotFound_Bad(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	err := sqliteMedium.DeleteAll("nonexistent")
	core.AssertError(t, err)
}

func TestSqlite_DeleteAll_EmptyPath_Bad(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	err := sqliteMedium.DeleteAll("")
	core.AssertError(t, err)
}

func TestSqlite_Rename_Good(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	core.RequireNoError(t, sqliteMedium.Write("old.txt", "content"))

	err := sqliteMedium.Rename("old.txt", "new.txt")
	core.RequireNoError(t, err)

	core.AssertFalse(t, sqliteMedium.Exists("old.txt"))
	core.AssertTrue(t, sqliteMedium.IsFile("new.txt"))

	content, err := sqliteMedium.Read("new.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "content", content)
}

func TestSqlite_Rename_Directory_Good(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	core.RequireNoError(t, sqliteMedium.EnsureDir("olddir"))
	core.RequireNoError(t, sqliteMedium.Write("olddir/file.txt", "content"))

	err := sqliteMedium.Rename("olddir", "newdir")
	core.RequireNoError(t, err)

	core.AssertFalse(t, sqliteMedium.Exists("olddir"))
	core.AssertFalse(t, sqliteMedium.Exists("olddir/file.txt"))
	core.AssertTrue(t, sqliteMedium.IsDir("newdir"))
	core.AssertTrue(t, sqliteMedium.IsFile("newdir/file.txt"))

	content, err := sqliteMedium.Read("newdir/file.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "content", content)
}

func TestSqlite_Rename_SourceNotFound_Bad(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	err := sqliteMedium.Rename("nonexistent", "new")
	core.AssertError(t, err)
}

func TestSqlite_Rename_EmptyPath_Bad(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	err := sqliteMedium.Rename("", "new")
	core.AssertError(t, err)

	err = sqliteMedium.Rename("old", "")
	core.AssertError(t, err)
}

func TestSqlite_List_Good(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	core.RequireNoError(t, sqliteMedium.Write("dir/file1.txt", "a"))
	core.RequireNoError(t, sqliteMedium.Write("dir/file2.txt", "b"))
	core.RequireNoError(t, sqliteMedium.Write("dir/sub/file3.txt", "c"))

	entries, err := sqliteMedium.List("dir")
	core.RequireNoError(t, err)

	names := make(map[string]bool)
	for _, entry := range entries {
		names[entry.Name()] = true
	}

	core.AssertTrue(t, names["file1.txt"])
	core.AssertTrue(t, names["file2.txt"])
	core.AssertTrue(t, names["sub"])
	core.AssertLen(t, entries, 3)
}

func TestSqlite_List_Root_Good(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	core.RequireNoError(t, sqliteMedium.Write("root.txt", "content"))
	core.RequireNoError(t, sqliteMedium.Write("dir/nested.txt", "nested"))

	entries, err := sqliteMedium.List("")
	core.RequireNoError(t, err)

	names := make(map[string]bool)
	for _, entry := range entries {
		names[entry.Name()] = true
	}

	core.AssertTrue(t, names["root.txt"])
	core.AssertTrue(t, names["dir"])
}

func TestSqlite_List_DirectoryEntry_Good(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	core.RequireNoError(t, sqliteMedium.Write("dir/sub/file.txt", "content"))

	entries, err := sqliteMedium.List("dir")
	core.RequireNoError(t, err)

	core.AssertLen(t, entries, 1)
	core.AssertEqual(t, "sub", entries[0].Name())
	core.AssertTrue(t, entries[0].IsDir())

	info, err := entries[0].Info()
	core.RequireNoError(t, err)
	core.AssertTrue(t, info.IsDir())
}

func TestSqlite_Stat_Good(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	core.RequireNoError(t, sqliteMedium.Write("file.txt", "hello world"))

	info, err := sqliteMedium.Stat("file.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "file.txt", info.Name())
	core.AssertEqual(t, int64(11), info.Size())
	core.AssertFalse(t, info.IsDir())
}

func TestSqlite_Stat_Directory_Good(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	core.RequireNoError(t, sqliteMedium.EnsureDir("mydir"))

	info, err := sqliteMedium.Stat("mydir")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "mydir", info.Name())
	core.AssertTrue(t, info.IsDir())
}

func TestSqlite_Stat_NotFound_Bad(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	_, err := sqliteMedium.Stat("nonexistent")
	core.AssertError(t, err)
}

func TestSqlite_Stat_EmptyPath_Bad(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	_, err := sqliteMedium.Stat("")
	core.AssertError(t, err)
}

func TestSqlite_Open_Good(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	core.RequireNoError(t, sqliteMedium.Write("file.txt", "open me"))

	file, err := sqliteMedium.Open("file.txt")
	core.RequireNoError(t, err)
	defer file.Close()

	data, err := goio.ReadAll(file.(goio.Reader))
	core.RequireNoError(t, err)
	core.AssertEqual(t, "open me", string(data))

	stat, err := file.Stat()
	core.RequireNoError(t, err)
	core.AssertEqual(t, "file.txt", stat.Name())
}

func TestSqlite_Open_NotFound_Bad(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	_, err := sqliteMedium.Open("nonexistent.txt")
	core.AssertError(t, err)
}

func TestSqlite_Open_IsDirectory_Bad(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	core.RequireNoError(t, sqliteMedium.EnsureDir("mydir"))
	_, err := sqliteMedium.Open("mydir")
	core.AssertError(t, err)
}

func TestSqlite_Create_Good(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	writer, err := sqliteMedium.Create("new.txt")
	core.RequireNoError(t, err)

	bytesWritten, err := writer.Write([]byte("created"))
	core.RequireNoError(t, err)
	core.AssertEqual(t, 7, bytesWritten)

	err = writer.Close()
	core.RequireNoError(t, err)

	content, err := sqliteMedium.Read("new.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "created", content)
}

func TestSqlite_Create_Overwrite_Good(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	core.RequireNoError(t, sqliteMedium.Write("file.txt", "old content"))

	writer, err := sqliteMedium.Create("file.txt")
	core.RequireNoError(t, err)
	_, err = writer.Write([]byte("new"))
	core.RequireNoError(t, err)
	core.RequireNoError(t, writer.Close())

	content, err := sqliteMedium.Read("file.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "new", content)
}

func TestSqlite_Create_EmptyPath_Bad(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	_, err := sqliteMedium.Create("")
	core.AssertError(t, err)
}

func TestSqlite_Append_Good(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	core.RequireNoError(t, sqliteMedium.Write("append.txt", "hello"))

	writer, err := sqliteMedium.Append("append.txt")
	core.RequireNoError(t, err)

	_, err = writer.Write([]byte(" world"))
	core.RequireNoError(t, err)
	core.RequireNoError(t, writer.Close())

	content, err := sqliteMedium.Read("append.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "hello world", content)
}

func TestSqlite_Append_NewFile_Good(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	writer, err := sqliteMedium.Append("new.txt")
	core.RequireNoError(t, err)

	_, err = writer.Write([]byte("fresh"))
	core.RequireNoError(t, err)
	core.RequireNoError(t, writer.Close())

	content, err := sqliteMedium.Read("new.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "fresh", content)
}

func TestSqlite_Append_EmptyPath_Bad(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	_, err := sqliteMedium.Append("")
	core.AssertError(t, err)
}

func TestSqlite_ReadStream_Good(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	core.RequireNoError(t, sqliteMedium.Write("stream.txt", "streaming content"))

	reader, err := sqliteMedium.ReadStream("stream.txt")
	core.RequireNoError(t, err)
	defer reader.Close()

	data, err := goio.ReadAll(reader)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "streaming content", string(data))
}

func TestSqlite_ReadStream_NotFound_Bad(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	_, err := sqliteMedium.ReadStream("nonexistent.txt")
	core.AssertError(t, err)
}

func TestSqlite_ReadStream_IsDirectory_Bad(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	core.RequireNoError(t, sqliteMedium.EnsureDir("mydir"))
	_, err := sqliteMedium.ReadStream("mydir")
	core.AssertError(t, err)
}

func TestSqlite_WriteStream_Good(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	writer, err := sqliteMedium.WriteStream("output.txt")
	core.RequireNoError(t, err)

	_, err = goio.Copy(writer, core.NewReader("piped data"))
	core.RequireNoError(t, err)
	core.RequireNoError(t, writer.Close())

	content, err := sqliteMedium.Read("output.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "piped data", content)
}

func TestSqlite_Exists_Good(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	core.AssertFalse(t, sqliteMedium.Exists("nonexistent"))

	core.RequireNoError(t, sqliteMedium.Write("file.txt", "content"))
	core.AssertTrue(t, sqliteMedium.Exists("file.txt"))

	core.RequireNoError(t, sqliteMedium.EnsureDir("mydir"))
	core.AssertTrue(t, sqliteMedium.Exists("mydir"))
}

func TestSqlite_Exists_EmptyPath_Good(t *core.T) {
	sqliteMedium := newSqliteMedium(t)
	core.AssertTrue(t, sqliteMedium.Exists(""))
	core.AssertFalse(t, sqliteMedium.IsFile(""))
	core.AssertFalse(t, sqliteMedium.IsDir(""))
}

func TestSqlite_IsDir_Good(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	core.RequireNoError(t, sqliteMedium.Write("file.txt", "content"))
	core.RequireNoError(t, sqliteMedium.EnsureDir("mydir"))

	core.AssertTrue(t, sqliteMedium.IsDir("mydir"))
	core.AssertFalse(t, sqliteMedium.IsDir("file.txt"))
	core.AssertFalse(t, sqliteMedium.IsDir("nonexistent"))
	core.AssertFalse(t, sqliteMedium.IsDir(""))
}

func TestSqlite_NormaliseEntryPath_Good(t *core.T) {
	core.AssertEqual(t, "file.txt", normaliseEntryPath("file.txt"))
	core.AssertEqual(t, "dir/file.txt", normaliseEntryPath("dir/file.txt"))
	core.AssertEqual(t, "file.txt", normaliseEntryPath("/file.txt"))
	core.AssertEqual(t, "file.txt", normaliseEntryPath("../file.txt"))
	core.AssertEqual(t, "file.txt", normaliseEntryPath("dir/../file.txt"))
	core.AssertEqual(t, "", normaliseEntryPath(""))
	core.AssertEqual(t, "", normaliseEntryPath("."))
	core.AssertEqual(t, "", normaliseEntryPath("/"))
}

func TestSqlite_InterfaceCompliance_Good(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	var _ interface {
		Read(string) (string, error)
		Write(string, string) error
		EnsureDir(string) error
		IsFile(string) bool
		Delete(string) error
		DeleteAll(string) error
		Rename(string, string) error
		List(string) ([]fs.DirEntry, error)
		Stat(string) (fs.FileInfo, error)
		Open(string) (fs.File, error)
		Create(string) (goio.WriteCloser, error)
		Append(string) (goio.WriteCloser, error)
		ReadStream(string) (goio.ReadCloser, error)
		WriteStream(string) (goio.WriteCloser, error)
		Exists(string) bool
		IsDir(string) bool
	} = sqliteMedium
}

func TestSqlite_CustomTable_Good(t *core.T) {
	sqliteMedium, err := New(Options{Path: ":memory:", Table: "my_files"})
	core.RequireNoError(t, err)
	defer sqliteMedium.Close()

	core.RequireNoError(t, sqliteMedium.Write("file.txt", "content"))

	content, err := sqliteMedium.Read("file.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "content", content)
}
