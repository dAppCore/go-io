package sqlite

import (
	goio "io"
	"io/fs"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSqliteMedium(t *testing.T) *Medium {
	t.Helper()
	sqliteMedium, err := New(Options{Path: ":memory:"})
	require.NoError(t, err)
	t.Cleanup(func() { sqliteMedium.Close() })
	return sqliteMedium
}

func TestSqlite_New_Good(t *testing.T) {
	sqliteMedium, err := New(Options{Path: ":memory:"})
	require.NoError(t, err)
	defer sqliteMedium.Close()
	assert.Equal(t, "files", sqliteMedium.table)
}

func TestSqlite_New_Options_Good(t *testing.T) {
	sqliteMedium, err := New(Options{Path: ":memory:", Table: "custom"})
	require.NoError(t, err)
	defer sqliteMedium.Close()
	assert.Equal(t, "custom", sqliteMedium.table)
}

func TestSqlite_New_EmptyPath_Bad(t *testing.T) {
	_, err := New(Options{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database path is required")
}

func TestSqlite_ReadWrite_Good(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	err := sqliteMedium.Write("hello.txt", "world")
	require.NoError(t, err)

	content, err := sqliteMedium.Read("hello.txt")
	require.NoError(t, err)
	assert.Equal(t, "world", content)
}

func TestSqlite_ReadWrite_Overwrite_Good(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	require.NoError(t, sqliteMedium.Write("file.txt", "first"))
	require.NoError(t, sqliteMedium.Write("file.txt", "second"))

	content, err := sqliteMedium.Read("file.txt")
	require.NoError(t, err)
	assert.Equal(t, "second", content)
}

func TestSqlite_ReadWrite_NestedPath_Good(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	err := sqliteMedium.Write("a/b/c.txt", "nested")
	require.NoError(t, err)

	content, err := sqliteMedium.Read("a/b/c.txt")
	require.NoError(t, err)
	assert.Equal(t, "nested", content)
}

func TestSqlite_Read_NotFound_Bad(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	_, err := sqliteMedium.Read("nonexistent.txt")
	assert.Error(t, err)
}

func TestSqlite_Read_EmptyPath_Bad(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	_, err := sqliteMedium.Read("")
	assert.Error(t, err)
}

func TestSqlite_Write_EmptyPath_Bad(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	err := sqliteMedium.Write("", "content")
	assert.Error(t, err)
}

func TestSqlite_Read_IsDirectory_Bad(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	require.NoError(t, sqliteMedium.EnsureDir("mydir"))
	_, err := sqliteMedium.Read("mydir")
	assert.Error(t, err)
}

func TestSqlite_EnsureDir_Good(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	err := sqliteMedium.EnsureDir("mydir")
	require.NoError(t, err)
	assert.True(t, sqliteMedium.IsDir("mydir"))
}

func TestSqlite_EnsureDir_EmptyPath_Good(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)
	err := sqliteMedium.EnsureDir("")
	assert.NoError(t, err)
}

func TestSqlite_EnsureDir_Idempotent_Good(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	require.NoError(t, sqliteMedium.EnsureDir("mydir"))
	require.NoError(t, sqliteMedium.EnsureDir("mydir"))
	assert.True(t, sqliteMedium.IsDir("mydir"))
}

func TestSqlite_IsFile_Good(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	require.NoError(t, sqliteMedium.Write("file.txt", "content"))
	require.NoError(t, sqliteMedium.EnsureDir("mydir"))

	assert.True(t, sqliteMedium.IsFile("file.txt"))
	assert.False(t, sqliteMedium.IsFile("mydir"))
	assert.False(t, sqliteMedium.IsFile("nonexistent"))
	assert.False(t, sqliteMedium.IsFile(""))
}

func TestSqlite_Delete_Good(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	require.NoError(t, sqliteMedium.Write("to-delete.txt", "content"))
	assert.True(t, sqliteMedium.Exists("to-delete.txt"))

	err := sqliteMedium.Delete("to-delete.txt")
	require.NoError(t, err)
	assert.False(t, sqliteMedium.Exists("to-delete.txt"))
}

func TestSqlite_Delete_EmptyDir_Good(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	require.NoError(t, sqliteMedium.EnsureDir("emptydir"))
	assert.True(t, sqliteMedium.IsDir("emptydir"))

	err := sqliteMedium.Delete("emptydir")
	require.NoError(t, err)
	assert.False(t, sqliteMedium.IsDir("emptydir"))
}

func TestSqlite_Delete_NotFound_Bad(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	err := sqliteMedium.Delete("nonexistent")
	assert.Error(t, err)
}

func TestSqlite_Delete_EmptyPath_Bad(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	err := sqliteMedium.Delete("")
	assert.Error(t, err)
}

func TestSqlite_Delete_NotEmpty_Bad(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	require.NoError(t, sqliteMedium.EnsureDir("mydir"))
	require.NoError(t, sqliteMedium.Write("mydir/file.txt", "content"))

	err := sqliteMedium.Delete("mydir")
	assert.Error(t, err)
}

func TestSqlite_DeleteAll_Good(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	require.NoError(t, sqliteMedium.Write("dir/file1.txt", "a"))
	require.NoError(t, sqliteMedium.Write("dir/sub/file2.txt", "b"))
	require.NoError(t, sqliteMedium.Write("other.txt", "c"))

	err := sqliteMedium.DeleteAll("dir")
	require.NoError(t, err)

	assert.False(t, sqliteMedium.Exists("dir/file1.txt"))
	assert.False(t, sqliteMedium.Exists("dir/sub/file2.txt"))
	assert.True(t, sqliteMedium.Exists("other.txt"))
}

func TestSqlite_DeleteAll_SingleFile_Good(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	require.NoError(t, sqliteMedium.Write("file.txt", "content"))

	err := sqliteMedium.DeleteAll("file.txt")
	require.NoError(t, err)
	assert.False(t, sqliteMedium.Exists("file.txt"))
}

func TestSqlite_DeleteAll_NotFound_Bad(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	err := sqliteMedium.DeleteAll("nonexistent")
	assert.Error(t, err)
}

func TestSqlite_DeleteAll_EmptyPath_Bad(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	err := sqliteMedium.DeleteAll("")
	assert.Error(t, err)
}

func TestSqlite_Rename_Good(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	require.NoError(t, sqliteMedium.Write("old.txt", "content"))

	err := sqliteMedium.Rename("old.txt", "new.txt")
	require.NoError(t, err)

	assert.False(t, sqliteMedium.Exists("old.txt"))
	assert.True(t, sqliteMedium.IsFile("new.txt"))

	content, err := sqliteMedium.Read("new.txt")
	require.NoError(t, err)
	assert.Equal(t, "content", content)
}

func TestSqlite_Rename_Directory_Good(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	require.NoError(t, sqliteMedium.EnsureDir("olddir"))
	require.NoError(t, sqliteMedium.Write("olddir/file.txt", "content"))

	err := sqliteMedium.Rename("olddir", "newdir")
	require.NoError(t, err)

	assert.False(t, sqliteMedium.Exists("olddir"))
	assert.False(t, sqliteMedium.Exists("olddir/file.txt"))
	assert.True(t, sqliteMedium.IsDir("newdir"))
	assert.True(t, sqliteMedium.IsFile("newdir/file.txt"))

	content, err := sqliteMedium.Read("newdir/file.txt")
	require.NoError(t, err)
	assert.Equal(t, "content", content)
}

func TestSqlite_Rename_SourceNotFound_Bad(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	err := sqliteMedium.Rename("nonexistent", "new")
	assert.Error(t, err)
}

func TestSqlite_Rename_EmptyPath_Bad(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	err := sqliteMedium.Rename("", "new")
	assert.Error(t, err)

	err = sqliteMedium.Rename("old", "")
	assert.Error(t, err)
}

func TestSqlite_List_Good(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	require.NoError(t, sqliteMedium.Write("dir/file1.txt", "a"))
	require.NoError(t, sqliteMedium.Write("dir/file2.txt", "b"))
	require.NoError(t, sqliteMedium.Write("dir/sub/file3.txt", "c"))

	entries, err := sqliteMedium.List("dir")
	require.NoError(t, err)

	names := make(map[string]bool)
	for _, entry := range entries {
		names[entry.Name()] = true
	}

	assert.True(t, names["file1.txt"])
	assert.True(t, names["file2.txt"])
	assert.True(t, names["sub"])
	assert.Len(t, entries, 3)
}

func TestSqlite_List_Root_Good(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	require.NoError(t, sqliteMedium.Write("root.txt", "content"))
	require.NoError(t, sqliteMedium.Write("dir/nested.txt", "nested"))

	entries, err := sqliteMedium.List("")
	require.NoError(t, err)

	names := make(map[string]bool)
	for _, entry := range entries {
		names[entry.Name()] = true
	}

	assert.True(t, names["root.txt"])
	assert.True(t, names["dir"])
}

func TestSqlite_List_DirectoryEntry_Good(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	require.NoError(t, sqliteMedium.Write("dir/sub/file.txt", "content"))

	entries, err := sqliteMedium.List("dir")
	require.NoError(t, err)

	require.Len(t, entries, 1)
	assert.Equal(t, "sub", entries[0].Name())
	assert.True(t, entries[0].IsDir())

	info, err := entries[0].Info()
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestSqlite_Stat_Good(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	require.NoError(t, sqliteMedium.Write("file.txt", "hello world"))

	info, err := sqliteMedium.Stat("file.txt")
	require.NoError(t, err)
	assert.Equal(t, "file.txt", info.Name())
	assert.Equal(t, int64(11), info.Size())
	assert.False(t, info.IsDir())
}

func TestSqlite_Stat_Directory_Good(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	require.NoError(t, sqliteMedium.EnsureDir("mydir"))

	info, err := sqliteMedium.Stat("mydir")
	require.NoError(t, err)
	assert.Equal(t, "mydir", info.Name())
	assert.True(t, info.IsDir())
}

func TestSqlite_Stat_NotFound_Bad(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	_, err := sqliteMedium.Stat("nonexistent")
	assert.Error(t, err)
}

func TestSqlite_Stat_EmptyPath_Bad(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	_, err := sqliteMedium.Stat("")
	assert.Error(t, err)
}

func TestSqlite_Open_Good(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	require.NoError(t, sqliteMedium.Write("file.txt", "open me"))

	file, err := sqliteMedium.Open("file.txt")
	require.NoError(t, err)
	defer file.Close()

	data, err := goio.ReadAll(file.(goio.Reader))
	require.NoError(t, err)
	assert.Equal(t, "open me", string(data))

	stat, err := file.Stat()
	require.NoError(t, err)
	assert.Equal(t, "file.txt", stat.Name())
}

func TestSqlite_Open_NotFound_Bad(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	_, err := sqliteMedium.Open("nonexistent.txt")
	assert.Error(t, err)
}

func TestSqlite_Open_IsDirectory_Bad(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	require.NoError(t, sqliteMedium.EnsureDir("mydir"))
	_, err := sqliteMedium.Open("mydir")
	assert.Error(t, err)
}

func TestSqlite_Create_Good(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	writer, err := sqliteMedium.Create("new.txt")
	require.NoError(t, err)

	bytesWritten, err := writer.Write([]byte("created"))
	require.NoError(t, err)
	assert.Equal(t, 7, bytesWritten)

	err = writer.Close()
	require.NoError(t, err)

	content, err := sqliteMedium.Read("new.txt")
	require.NoError(t, err)
	assert.Equal(t, "created", content)
}

func TestSqlite_Create_Overwrite_Good(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	require.NoError(t, sqliteMedium.Write("file.txt", "old content"))

	writer, err := sqliteMedium.Create("file.txt")
	require.NoError(t, err)
	_, err = writer.Write([]byte("new"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	content, err := sqliteMedium.Read("file.txt")
	require.NoError(t, err)
	assert.Equal(t, "new", content)
}

func TestSqlite_Create_EmptyPath_Bad(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	_, err := sqliteMedium.Create("")
	assert.Error(t, err)
}

func TestSqlite_Append_Good(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	require.NoError(t, sqliteMedium.Write("append.txt", "hello"))

	writer, err := sqliteMedium.Append("append.txt")
	require.NoError(t, err)

	_, err = writer.Write([]byte(" world"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	content, err := sqliteMedium.Read("append.txt")
	require.NoError(t, err)
	assert.Equal(t, "hello world", content)
}

func TestSqlite_Append_NewFile_Good(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	writer, err := sqliteMedium.Append("new.txt")
	require.NoError(t, err)

	_, err = writer.Write([]byte("fresh"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	content, err := sqliteMedium.Read("new.txt")
	require.NoError(t, err)
	assert.Equal(t, "fresh", content)
}

func TestSqlite_Append_EmptyPath_Bad(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	_, err := sqliteMedium.Append("")
	assert.Error(t, err)
}

func TestSqlite_ReadStream_Good(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	require.NoError(t, sqliteMedium.Write("stream.txt", "streaming content"))

	reader, err := sqliteMedium.ReadStream("stream.txt")
	require.NoError(t, err)
	defer reader.Close()

	data, err := goio.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, "streaming content", string(data))
}

func TestSqlite_ReadStream_NotFound_Bad(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	_, err := sqliteMedium.ReadStream("nonexistent.txt")
	assert.Error(t, err)
}

func TestSqlite_ReadStream_IsDirectory_Bad(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	require.NoError(t, sqliteMedium.EnsureDir("mydir"))
	_, err := sqliteMedium.ReadStream("mydir")
	assert.Error(t, err)
}

func TestSqlite_WriteStream_Good(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	writer, err := sqliteMedium.WriteStream("output.txt")
	require.NoError(t, err)

	_, err = goio.Copy(writer, core.NewReader("piped data"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	content, err := sqliteMedium.Read("output.txt")
	require.NoError(t, err)
	assert.Equal(t, "piped data", content)
}

func TestSqlite_Exists_Good(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	assert.False(t, sqliteMedium.Exists("nonexistent"))

	require.NoError(t, sqliteMedium.Write("file.txt", "content"))
	assert.True(t, sqliteMedium.Exists("file.txt"))

	require.NoError(t, sqliteMedium.EnsureDir("mydir"))
	assert.True(t, sqliteMedium.Exists("mydir"))
}

func TestSqlite_Exists_EmptyPath_Good(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)
	assert.True(t, sqliteMedium.Exists(""))
}

func TestSqlite_IsDir_Good(t *testing.T) {
	sqliteMedium := newSqliteMedium(t)

	require.NoError(t, sqliteMedium.Write("file.txt", "content"))
	require.NoError(t, sqliteMedium.EnsureDir("mydir"))

	assert.True(t, sqliteMedium.IsDir("mydir"))
	assert.False(t, sqliteMedium.IsDir("file.txt"))
	assert.False(t, sqliteMedium.IsDir("nonexistent"))
	assert.False(t, sqliteMedium.IsDir(""))
}

func TestSqlite_NormaliseEntryPath_Good(t *testing.T) {
	assert.Equal(t, "file.txt", normaliseEntryPath("file.txt"))
	assert.Equal(t, "dir/file.txt", normaliseEntryPath("dir/file.txt"))
	assert.Equal(t, "file.txt", normaliseEntryPath("/file.txt"))
	assert.Equal(t, "file.txt", normaliseEntryPath("../file.txt"))
	assert.Equal(t, "file.txt", normaliseEntryPath("dir/../file.txt"))
	assert.Equal(t, "", normaliseEntryPath(""))
	assert.Equal(t, "", normaliseEntryPath("."))
	assert.Equal(t, "", normaliseEntryPath("/"))
}

func TestSqlite_InterfaceCompliance_Good(t *testing.T) {
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

func TestSqlite_CustomTable_Good(t *testing.T) {
	sqliteMedium, err := New(Options{Path: ":memory:", Table: "my_files"})
	require.NoError(t, err)
	defer sqliteMedium.Close()

	require.NoError(t, sqliteMedium.Write("file.txt", "content"))

	content, err := sqliteMedium.Read("file.txt")
	require.NoError(t, err)
	assert.Equal(t, "content", content)
}
