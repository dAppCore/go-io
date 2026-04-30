package sqlite

import (
	core "dappco.re/go"
	goio "io"
	"io/fs"
)

const (
	sqliteMemoryPath     = ":memory:"
	sqliteFilePath       = "file.txt"
	sqliteMissingPath    = "nonexistent.txt"
	sqliteDeletePath     = "to-delete.txt"
	sqliteDirFileOnePath = "dir/file1.txt"
	sqliteOldPath        = "old.txt"
	sqliteNewPath        = "new.txt"
	sqliteAppendPath     = "append.txt"
)

func newSqliteMedium(t *core.T) *Medium {
	t.Helper()
	sqliteMedium, err := New(Options{Path: sqliteMemoryPath})
	core.RequireNoError(t, err)
	t.Cleanup(func() { _ = sqliteMedium.Close() })
	return sqliteMedium
}

func TestSqlite_New_Good(t *core.T) {
	sqliteMedium, err := New(Options{Path: sqliteMemoryPath})
	core.RequireNoError(t, err)
	defer func() { _ = sqliteMedium.Close() }()
	core.AssertEqual(t, "files", sqliteMedium.table)
}

func TestSqlite_New_Options_Good(t *core.T) {
	sqliteMedium, err := New(Options{Path: sqliteMemoryPath, Table: "custom"})
	core.RequireNoError(t, err)
	defer func() { _ = sqliteMedium.Close() }()
	core.AssertEqual(t, "custom", sqliteMedium.table)
}

func TestSqlite_New_EmptyPath_Bad(t *core.T) {
	_, err := New(Options{})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "database path is required")
}

func TestSqlite_ReadWriteGood(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	err := sqliteMedium.Write("hello.txt", "world")
	core.RequireNoError(t, err)

	content, err := sqliteMedium.Read("hello.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "world", content)
}

func TestSqlite_ReadWrite_OverwriteGood(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	core.RequireNoError(t, sqliteMedium.Write(sqliteFilePath, "first"))
	core.RequireNoError(t, sqliteMedium.Write(sqliteFilePath, "second"))

	content, err := sqliteMedium.Read(sqliteFilePath)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "second", content)
}

func TestSqlite_ReadWrite_NestedPathGood(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	err := sqliteMedium.Write("a/b/c.txt", "nested")
	core.RequireNoError(t, err)

	content, err := sqliteMedium.Read("a/b/c.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "nested", content)
}

func TestSqlite_Read_NotFound_Bad(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	_, err := sqliteMedium.Read(sqliteMissingPath)
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

	core.RequireNoError(t, sqliteMedium.Write(sqliteFilePath, "content"))
	core.RequireNoError(t, sqliteMedium.EnsureDir("mydir"))

	core.AssertTrue(t, sqliteMedium.IsFile(sqliteFilePath))
	core.AssertFalse(t, sqliteMedium.IsFile("mydir"))
	core.AssertFalse(t, sqliteMedium.IsFile("nonexistent"))
	core.AssertFalse(t, sqliteMedium.IsFile(""))
}

func TestSqlite_Delete_Good(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	core.RequireNoError(t, sqliteMedium.Write(sqliteDeletePath, "content"))
	core.AssertTrue(t, sqliteMedium.Exists(sqliteDeletePath))

	err := sqliteMedium.Delete(sqliteDeletePath)
	core.RequireNoError(t, err)
	core.AssertFalse(t, sqliteMedium.Exists(sqliteDeletePath))
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

	core.RequireNoError(t, sqliteMedium.Write(sqliteDirFileOnePath, "a"))
	core.RequireNoError(t, sqliteMedium.Write("dir/sub/file2.txt", "b"))
	core.RequireNoError(t, sqliteMedium.Write("other.txt", "c"))

	err := sqliteMedium.DeleteAll("dir")
	core.RequireNoError(t, err)

	core.AssertFalse(t, sqliteMedium.Exists(sqliteDirFileOnePath))
	core.AssertFalse(t, sqliteMedium.Exists("dir/sub/file2.txt"))
	core.AssertTrue(t, sqliteMedium.Exists("other.txt"))
}

func TestSqlite_DeleteAll_SingleFile_Good(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	core.RequireNoError(t, sqliteMedium.Write(sqliteFilePath, "content"))

	err := sqliteMedium.DeleteAll(sqliteFilePath)
	core.RequireNoError(t, err)
	core.AssertFalse(t, sqliteMedium.Exists(sqliteFilePath))
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

	core.RequireNoError(t, sqliteMedium.Write(sqliteOldPath, "content"))

	err := sqliteMedium.Rename(sqliteOldPath, sqliteNewPath)
	core.RequireNoError(t, err)

	core.AssertFalse(t, sqliteMedium.Exists(sqliteOldPath))
	core.AssertTrue(t, sqliteMedium.IsFile(sqliteNewPath))

	content, err := sqliteMedium.Read(sqliteNewPath)
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

	core.RequireNoError(t, sqliteMedium.Write(sqliteDirFileOnePath, "a"))
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

	core.RequireNoError(t, sqliteMedium.Write(sqliteFilePath, "hello world"))

	info, err := sqliteMedium.Stat(sqliteFilePath)
	core.RequireNoError(t, err)
	core.AssertEqual(t, sqliteFilePath, info.Name())
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

	core.RequireNoError(t, sqliteMedium.Write(sqliteFilePath, "open me"))

	file, err := sqliteMedium.Open(sqliteFilePath)
	core.RequireNoError(t, err)
	defer func() { _ = file.Close() }()

	data, err := goio.ReadAll(file.(goio.Reader))
	core.RequireNoError(t, err)
	core.AssertEqual(t, "open me", string(data))

	stat, err := file.Stat()
	core.RequireNoError(t, err)
	core.AssertEqual(t, sqliteFilePath, stat.Name())
}

func TestSqlite_Open_NotFound_Bad(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	_, err := sqliteMedium.Open(sqliteMissingPath)
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

	writer, err := sqliteMedium.Create(sqliteNewPath)
	core.RequireNoError(t, err)

	bytesWritten, err := writer.Write([]byte("created"))
	core.RequireNoError(t, err)
	core.AssertEqual(t, 7, bytesWritten)

	err = writer.Close()
	core.RequireNoError(t, err)

	content, err := sqliteMedium.Read(sqliteNewPath)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "created", content)
}

func TestSqlite_Create_Overwrite_Good(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	core.RequireNoError(t, sqliteMedium.Write(sqliteFilePath, "old content"))

	writer, err := sqliteMedium.Create(sqliteFilePath)
	core.RequireNoError(t, err)
	_, err = writer.Write([]byte("new"))
	core.RequireNoError(t, err)
	core.RequireNoError(t, writer.Close())

	content, err := sqliteMedium.Read(sqliteFilePath)
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

	core.RequireNoError(t, sqliteMedium.Write(sqliteAppendPath, "hello"))

	writer, err := sqliteMedium.Append(sqliteAppendPath)
	core.RequireNoError(t, err)

	_, err = writer.Write([]byte(" world"))
	core.RequireNoError(t, err)
	core.RequireNoError(t, writer.Close())

	content, err := sqliteMedium.Read(sqliteAppendPath)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "hello world", content)
}

func TestSqlite_Append_NewFile_Good(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	writer, err := sqliteMedium.Append(sqliteNewPath)
	core.RequireNoError(t, err)

	_, err = writer.Write([]byte("fresh"))
	core.RequireNoError(t, err)
	core.RequireNoError(t, writer.Close())

	content, err := sqliteMedium.Read(sqliteNewPath)
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
	defer func() { _ = reader.Close() }()

	data, err := goio.ReadAll(reader)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "streaming content", string(data))
}

func TestSqlite_ReadStream_NotFound_Bad(t *core.T) {
	sqliteMedium := newSqliteMedium(t)

	_, err := sqliteMedium.ReadStream(sqliteMissingPath)
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

	core.RequireNoError(t, sqliteMedium.Write(sqliteFilePath, "content"))
	core.AssertTrue(t, sqliteMedium.Exists(sqliteFilePath))

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

	core.RequireNoError(t, sqliteMedium.Write(sqliteFilePath, "content"))
	core.RequireNoError(t, sqliteMedium.EnsureDir("mydir"))

	core.AssertTrue(t, sqliteMedium.IsDir("mydir"))
	core.AssertFalse(t, sqliteMedium.IsDir(sqliteFilePath))
	core.AssertFalse(t, sqliteMedium.IsDir("nonexistent"))
	core.AssertFalse(t, sqliteMedium.IsDir(""))
}

func TestSqlite_NormaliseEntryPathGood(t *core.T) {
	core.AssertEqual(t, sqliteFilePath, normaliseEntryPath(sqliteFilePath))
	core.AssertEqual(t, "dir/file.txt", normaliseEntryPath("dir/file.txt"))
	core.AssertEqual(t, sqliteFilePath, normaliseEntryPath("/file.txt"))
	core.AssertEqual(t, sqliteFilePath, normaliseEntryPath("../file.txt"))
	core.AssertEqual(t, sqliteFilePath, normaliseEntryPath("dir/../file.txt"))
	core.AssertEqual(t, "", normaliseEntryPath(""))
	core.AssertEqual(t, "", normaliseEntryPath("."))
	core.AssertEqual(t, "", normaliseEntryPath("/"))
}

func TestSqlite_InterfaceComplianceGood(t *core.T) {
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

func TestSqlite_CustomTableGood(t *core.T) {
	sqliteMedium, err := New(Options{Path: sqliteMemoryPath, Table: "my_files"})
	core.RequireNoError(t, err)
	defer func() { _ = sqliteMedium.Close() }()

	core.RequireNoError(t, sqliteMedium.Write(sqliteFilePath, "content"))

	content, err := sqliteMedium.Read(sqliteFilePath)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "content", content)
}

func TestSqlite_Medium_Read_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.Write("read.txt", "payload"))
	got, err := medium.Read("read.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", got)
}

func TestSqlite_Medium_Read_Bad(t *core.T) {
	medium := newSqliteMedium(t)
	got, err := medium.Read("missing.txt")
	core.AssertError(t, err)
	core.AssertEqual(t, "", got)
}

func TestSqlite_Medium_Read_Ugly(t *core.T) {
	medium := newSqliteMedium(t)
	got, err := medium.Read("")
	core.AssertError(t, err)
	core.AssertEqual(t, "", got)
}

func TestSqlite_Medium_Write_Good(t *core.T) {
	medium := newSqliteMedium(t)
	err := medium.Write("write.txt", "payload")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("write.txt"))
}

func TestSqlite_Medium_Write_Bad(t *core.T) {
	medium := newSqliteMedium(t)
	err := medium.Write("", "payload")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.IsFile(""))
}

func TestSqlite_Medium_Write_Ugly(t *core.T) {
	medium := newSqliteMedium(t)
	err := medium.Write("nested/write.txt", "payload")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("nested/write.txt"))
}

func TestSqlite_Medium_WriteMode_Good(t *core.T) {
	medium := newSqliteMedium(t)
	err := medium.WriteMode("mode.txt", "payload", 0600)
	info, statErr := medium.Stat("mode.txt")
	core.AssertNoError(t, err)
	core.AssertNoError(t, statErr)
	core.AssertEqual(t, fs.FileMode(0600), info.Mode().Perm())
}

func TestSqlite_Medium_WriteMode_Bad(t *core.T) {
	medium := newSqliteMedium(t)
	err := medium.WriteMode("", "payload", 0600)
	core.AssertError(t, err)
	core.AssertFalse(t, medium.IsFile(""))
}

func TestSqlite_Medium_WriteMode_Ugly(t *core.T) {
	medium := newSqliteMedium(t)
	err := medium.WriteMode("zero-mode.txt", "payload", 0)
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("zero-mode.txt"))
}

func TestSqlite_Medium_EnsureDir_Good(t *core.T) {
	medium := newSqliteMedium(t)
	err := medium.EnsureDir("dir")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir("dir"))
}

func TestSqlite_Medium_EnsureDir_Bad(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.Write("file", "payload"))
	err := medium.EnsureDir("file")
	core.AssertNoError(t, err)
}

func TestSqlite_Medium_EnsureDir_Ugly(t *core.T) {
	medium := newSqliteMedium(t)
	err := medium.EnsureDir("a/b/c")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir("a/b/c"))
}

func TestSqlite_Medium_IsFile_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.Write(sqliteFilePath, "payload"))
	got := medium.IsFile(sqliteFilePath)
	core.AssertTrue(t, got)
}

func TestSqlite_Medium_IsFile_Bad(t *core.T) {
	medium := newSqliteMedium(t)
	got := medium.IsFile("missing.txt")
	core.AssertFalse(t, got)
}

func TestSqlite_Medium_IsFile_Ugly(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.IsFile("dir")
	core.AssertFalse(t, got)
}

func TestSqlite_Medium_Delete_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.Write("delete.txt", "payload"))
	err := medium.Delete("delete.txt")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("delete.txt"))
}

func TestSqlite_Medium_Delete_Bad(t *core.T) {
	medium := newSqliteMedium(t)
	err := medium.Delete("missing.txt")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("missing.txt"))
}

func TestSqlite_Medium_Delete_Ugly(t *core.T) {
	medium := newSqliteMedium(t)
	err := medium.Delete("")
	core.AssertError(t, err)
	core.AssertTrue(t, medium.Exists(""))
}

func TestSqlite_Medium_DeleteAll_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.Write("tree/file.txt", "payload"))
	err := medium.DeleteAll("tree")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("tree/file.txt"))
}

func TestSqlite_Medium_DeleteAll_Bad(t *core.T) {
	medium := newSqliteMedium(t)
	err := medium.DeleteAll("missing")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("missing"))
}

func TestSqlite_Medium_DeleteAll_Ugly(t *core.T) {
	medium := newSqliteMedium(t)
	err := medium.DeleteAll("")
	core.AssertError(t, err)
	core.AssertTrue(t, medium.Exists(""))
}

func TestSqlite_Medium_Rename_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.Write(sqliteOldPath, "payload"))
	err := medium.Rename(sqliteOldPath, sqliteNewPath)
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile(sqliteNewPath))
}

func TestSqlite_Medium_Rename_Bad(t *core.T) {
	medium := newSqliteMedium(t)
	err := medium.Rename("missing.txt", sqliteNewPath)
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists(sqliteNewPath))
}

func TestSqlite_Medium_Rename_Ugly(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.Write(sqliteOldPath, "payload"))
	err := medium.Rename(sqliteOldPath, "nested/new.txt")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("nested/new.txt"))
}

func TestSqlite_Medium_List_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.Write("dir/a.txt", "payload"))
	entries, err := medium.List("dir")
	core.AssertNoError(t, err)
	core.AssertLen(t, entries, 1)
}

func TestSqlite_Medium_List_Bad(t *core.T) {
	medium := newSqliteMedium(t)
	entries, err := medium.List("missing")
	core.AssertNoError(t, err)
	core.AssertNil(t, entries)
}

func TestSqlite_Medium_List_Ugly(t *core.T) {
	medium := newSqliteMedium(t)
	entries, err := medium.List("")
	core.AssertNoError(t, err)
	core.AssertNil(t, entries)
}

func TestSqlite_Medium_Stat_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.Write("stat.txt", "payload"))
	info, err := medium.Stat("stat.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "stat.txt", info.Name())
}

func TestSqlite_Medium_Stat_Bad(t *core.T) {
	medium := newSqliteMedium(t)
	info, err := medium.Stat("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, info)
}

func TestSqlite_Medium_Stat_Ugly(t *core.T) {
	medium := newSqliteMedium(t)
	info, err := medium.Stat("")
	core.AssertError(t, err)
	core.AssertNil(t, info)
}

func TestSqlite_Medium_Open_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.Write("open.txt", "payload"))
	file, err := medium.Open("open.txt")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, file)
	core.RequireNoError(t, file.Close())
}

func TestSqlite_Medium_Open_Bad(t *core.T) {
	medium := newSqliteMedium(t)
	file, err := medium.Open("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, file)
}

func TestSqlite_Medium_Open_Ugly(t *core.T) {
	medium := newSqliteMedium(t)
	file, err := medium.Open("")
	core.AssertError(t, err)
	core.AssertNil(t, file)
}

func TestSqlite_Medium_Create_Good(t *core.T) {
	medium := newSqliteMedium(t)
	writer, err := medium.Create("create.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestSqlite_Medium_Create_Bad(t *core.T) {
	medium := newSqliteMedium(t)
	writer, err := medium.Create("")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestSqlite_Medium_Create_Ugly(t *core.T) {
	medium := newSqliteMedium(t)
	writer, err := medium.Create("nested/create.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestSqlite_Medium_Append_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.Write(sqliteAppendPath, "a"))
	writer, err := medium.Append(sqliteAppendPath)
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("b"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestSqlite_Medium_Append_Bad(t *core.T) {
	medium := newSqliteMedium(t)
	writer, err := medium.Append("")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestSqlite_Medium_Append_Ugly(t *core.T) {
	medium := newSqliteMedium(t)
	writer, err := medium.Append(sqliteNewPath)
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("new"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestSqlite_Medium_ReadStream_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.Write("stream.txt", "payload"))
	reader, err := medium.ReadStream("stream.txt")
	core.RequireNoError(t, err)
	data, readErr := goio.ReadAll(reader)
	core.AssertNoError(t, readErr)
	core.AssertEqual(t, "payload", string(data))
}

func TestSqlite_Medium_ReadStream_Bad(t *core.T) {
	medium := newSqliteMedium(t)
	reader, err := medium.ReadStream("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, reader)
}

func TestSqlite_Medium_ReadStream_Ugly(t *core.T) {
	medium := newSqliteMedium(t)
	reader, err := medium.ReadStream("")
	core.AssertError(t, err)
	core.AssertNil(t, reader)
}

func TestSqlite_Medium_WriteStream_Good(t *core.T) {
	medium := newSqliteMedium(t)
	writer, err := medium.WriteStream("stream.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestSqlite_Medium_WriteStream_Bad(t *core.T) {
	medium := newSqliteMedium(t)
	writer, err := medium.WriteStream("")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestSqlite_Medium_WriteStream_Ugly(t *core.T) {
	medium := newSqliteMedium(t)
	writer, err := medium.WriteStream("nested/stream.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestSqlite_Medium_Exists_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.Write("exists.txt", "payload"))
	got := medium.Exists("exists.txt")
	core.AssertTrue(t, got)
}

func TestSqlite_Medium_Exists_Bad(t *core.T) {
	medium := newSqliteMedium(t)
	got := medium.Exists("missing.txt")
	core.AssertFalse(t, got)
}

func TestSqlite_Medium_Exists_Ugly(t *core.T) {
	medium := newSqliteMedium(t)
	got := medium.Exists("")
	core.AssertTrue(t, got)
}

func TestSqlite_Medium_IsDir_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.IsDir("dir")
	core.AssertTrue(t, got)
}

func TestSqlite_Medium_IsDir_Bad(t *core.T) {
	medium := newSqliteMedium(t)
	got := medium.IsDir("missing")
	core.AssertFalse(t, got)
}

func TestSqlite_Medium_IsDir_Ugly(t *core.T) {
	medium := newSqliteMedium(t)
	got := medium.IsDir("")
	core.AssertFalse(t, got)
}

func TestSqlite_New_Bad(t *core.T) {
	medium, err := New(Options{})
	core.AssertError(t, err)
	core.AssertNil(t, medium)
}

func TestSqlite_New_Ugly(t *core.T) {
	medium, err := New(Options{Path: sqliteMemoryPath, Table: "ax7_files"})
	core.RequireNoError(t, err)
	defer func() { _ = medium.Close() }()
	core.AssertEqual(t, "ax7_files", medium.table)
}

func TestSqlite_Medium_Close_Good(t *core.T) {
	medium, err := New(Options{Path: sqliteMemoryPath})
	core.RequireNoError(t, err)
	err = medium.Close()
	core.AssertNoError(t, err)
}

func TestSqlite_Medium_Close_Bad(t *core.T) {
	medium := &Medium{}
	err := medium.Close()
	core.AssertNoError(t, err)
}

func TestSqlite_Medium_Close_Ugly(t *core.T) {
	medium, err := New(Options{Path: sqliteMemoryPath})
	core.RequireNoError(t, err)
	core.AssertNoError(t, medium.Close())
	core.AssertNoError(t, medium.Close())
}

func TestSqlite_Info_Name_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.Write("info.txt", "abc"))
	info, err := medium.Stat("info.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "info.txt", info.Name())
}

func TestSqlite_Info_Name_Bad(t *core.T) {
	info := &fileInfo{}
	got := info.Name()
	core.AssertEqual(t, "", got)
}

func TestSqlite_Info_Name_Ugly(t *core.T) {
	info := &fileInfo{name: ""}
	got := info.Name()
	core.AssertEqual(t, "", got)
}

func TestSqlite_Info_Size_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.Write("size.txt", "abcd"))
	info, err := medium.Stat("size.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, int64(4), info.Size())
}

func TestSqlite_Info_Size_Bad(t *core.T) {
	info := &fileInfo{}
	got := info.Size()
	core.AssertEqual(t, int64(0), got)
}

func TestSqlite_Info_Size_Ugly(t *core.T) {
	info := &fileInfo{size: -1}
	got := info.Size()
	core.AssertEqual(t, int64(-1), got)
}

func TestSqlite_Info_Mode_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.WriteMode("mode.txt", "abc", 0600))
	info, err := medium.Stat("mode.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, fs.FileMode(0600), info.Mode().Perm())
}

func TestSqlite_Info_Mode_Bad(t *core.T) {
	info := &fileInfo{}
	got := info.Mode()
	core.AssertEqual(t, fs.FileMode(0), got)
}

func TestSqlite_Info_Mode_Ugly(t *core.T) {
	info := &fileInfo{mode: fs.ModeDir | 0755}
	got := info.Mode()
	core.AssertTrue(t, got.IsDir())
}

func TestSqlite_Info_ModTime_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.Write("time.txt", "abc"))
	info, err := medium.Stat("time.txt")
	core.RequireNoError(t, err)
	core.AssertFalse(t, info.ModTime().IsZero())
}

func TestSqlite_Info_ModTime_Bad(t *core.T) {
	info := &fileInfo{}
	got := info.ModTime()
	core.AssertTrue(t, got.IsZero())
}

func TestSqlite_Info_ModTime_Ugly(t *core.T) {
	stamp := core.Now()
	info := &fileInfo{modTime: stamp}
	core.AssertEqual(t, stamp, info.ModTime())
}

func TestSqlite_Info_IsDir_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.EnsureDir("dir"))
	info, err := medium.Stat("dir")
	core.RequireNoError(t, err)
	core.AssertTrue(t, info.IsDir())
}

func TestSqlite_Info_IsDir_Bad(t *core.T) {
	info := &fileInfo{}
	got := info.IsDir()
	core.AssertFalse(t, got)
}

func TestSqlite_Info_IsDir_Ugly(t *core.T) {
	info := &fileInfo{isDir: true}
	got := info.IsDir()
	core.AssertTrue(t, got)
}

func TestSqlite_Info_Sys_Good(t *core.T) {
	info := &fileInfo{name: "sys"}
	got := info.Sys()
	core.AssertNil(t, got)
}

func TestSqlite_Info_Sys_Bad(t *core.T) {
	info := &fileInfo{}
	got := info.Sys()
	core.AssertNil(t, got)
}

func TestSqlite_Info_Sys_Ugly(t *core.T) {
	var info *fileInfo
	got := info.Sys()
	core.AssertNil(t, got)
}

func TestSqlite_Entry_Name_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.Write("dir/entry.txt", "abc"))
	entries, err := medium.List("dir")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "entry.txt", entries[0].Name())
}

func TestSqlite_Entry_Name_Bad(t *core.T) {
	entry := &dirEntry{}
	got := entry.Name()
	core.AssertEqual(t, "", got)
}

func TestSqlite_Entry_Name_Ugly(t *core.T) {
	entry := &dirEntry{name: ""}
	got := entry.Name()
	core.AssertEqual(t, "", got)
}

func TestSqlite_Entry_IsDir_Good(t *core.T) {
	entry := &dirEntry{name: "dir", isDir: true}
	got := entry.IsDir()
	core.AssertTrue(t, got)
}

func TestSqlite_Entry_IsDir_Bad(t *core.T) {
	entry := &dirEntry{name: "file"}
	got := entry.IsDir()
	core.AssertFalse(t, got)
}

func TestSqlite_Entry_IsDir_Ugly(t *core.T) {
	entry := &dirEntry{}
	got := entry.IsDir()
	core.AssertFalse(t, got)
}

func TestSqlite_Entry_Type_Good(t *core.T) {
	entry := &dirEntry{name: "dir", isDir: true, mode: fs.ModeDir | 0755}
	got := entry.Type()
	core.AssertTrue(t, got.IsDir())
}

func TestSqlite_Entry_Type_Bad(t *core.T) {
	entry := &dirEntry{name: "file", mode: 0644}
	got := entry.Type()
	core.AssertEqual(t, fs.FileMode(0), got)
}

func TestSqlite_Entry_Type_Ugly(t *core.T) {
	entry := &dirEntry{}
	got := entry.Type()
	core.AssertEqual(t, fs.FileMode(0), got)
}

func TestSqlite_Entry_Info_Good(t *core.T) {
	entry := &dirEntry{name: "file", info: &fileInfo{name: "file"}}
	info, err := entry.Info()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "file", info.Name())
}

func TestSqlite_Entry_Info_Bad(t *core.T) {
	entry := &dirEntry{}
	info, err := entry.Info()
	core.AssertNoError(t, err)
	core.AssertNil(t, info)
}

func TestSqlite_Entry_Info_Ugly(t *core.T) {
	entry := &dirEntry{info: &fileInfo{name: ""}}
	info, err := entry.Info()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "", info.Name())
}

func TestSqlite_File_Read_Good(t *core.T) {
	medium := newSqliteMedium(t)
	core.RequireNoError(t, medium.Write(sqliteFilePath, "abc"))
	file, err := medium.Open(sqliteFilePath)
	core.RequireNoError(t, err)
	buf := make([]byte, 3)
	n, readErr := file.Read(buf)
	core.AssertNoError(t, readErr)
	core.AssertEqual(t, 3, n)
}

func TestSqlite_File_Read_Bad(t *core.T) {
	file := &sqliteFile{}
	buf := make([]byte, 1)
	n, err := file.Read(buf)
	core.AssertErrorIs(t, err, goio.EOF)
	core.AssertEqual(t, 0, n)
}

func TestSqlite_File_Read_Ugly(t *core.T) {
	file := &sqliteFile{content: []byte("abc"), offset: 2}
	buf := make([]byte, 4)
	n, err := file.Read(buf)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 1, n)
}

func TestSqlite_File_Stat_Good(t *core.T) {
	file := &sqliteFile{name: sqliteFilePath, content: []byte("abc"), mode: 0644}
	info, err := file.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, sqliteFilePath, info.Name())
}

func TestSqlite_File_Stat_Bad(t *core.T) {
	file := &sqliteFile{}
	info, err := file.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "", info.Name())
}

func TestSqlite_File_Stat_Ugly(t *core.T) {
	file := &sqliteFile{name: "empty", content: nil}
	info, err := file.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, int64(0), info.Size())
}

func TestSqlite_File_Close_Good(t *core.T) {
	file := &sqliteFile{name: sqliteFilePath}
	err := file.Close()
	core.AssertNoError(t, err)
}

func TestSqlite_File_Close_Bad(t *core.T) {
	file := &sqliteFile{}
	err := file.Close()
	core.AssertNoError(t, err)
}

func TestSqlite_File_Close_Ugly(t *core.T) {
	file := &sqliteFile{}
	core.AssertNoError(t, file.Close())
	core.AssertNoError(t, file.Close())
}

func TestSqlite_WriteCloser_Write_Good(t *core.T) {
	medium := newSqliteMedium(t)
	writer, err := medium.Create("writer.txt")
	core.RequireNoError(t, err)
	n, writeErr := writer.Write([]byte("abc"))
	core.AssertNoError(t, writeErr)
	core.AssertEqual(t, 3, n)
}

func TestSqlite_WriteCloser_Write_Bad(t *core.T) {
	writer := &sqliteWriteCloser{}
	n, err := writer.Write(nil)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, n)
}

func TestSqlite_WriteCloser_Write_Ugly(t *core.T) {
	writer := &sqliteWriteCloser{}
	n, err := writer.Write([]byte{})
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, n)
}

func TestSqlite_WriteCloser_Close_Good(t *core.T) {
	medium := newSqliteMedium(t)
	writer, err := medium.Create("writer-close.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("abc"))
	core.RequireNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestSqlite_WriteCloser_Close_Bad(t *core.T) {
	writer := &sqliteWriteCloser{}
	core.AssertPanics(t, func() {
		_ = writer.Close()
	})
}

func TestSqlite_WriteCloser_Close_Ugly(t *core.T) {
	medium := newSqliteMedium(t)
	writer, err := medium.Create("empty.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("x"))
	core.RequireNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}
