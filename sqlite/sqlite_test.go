package sqlite

import (
	goio "io"
	"io/fs"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestMedium(t *testing.T) *Medium {
	t.Helper()
	m, err := New(Options{Path: ":memory:"})
	require.NoError(t, err)
	t.Cleanup(func() { m.Close() })
	return m
}

// --- Constructor Tests ---

func TestSqlite_New_Good(t *testing.T) {
	m, err := New(Options{Path: ":memory:"})
	require.NoError(t, err)
	defer m.Close()
	assert.Equal(t, "files", m.table)
}

func TestSqlite_New_Options_Good(t *testing.T) {
	m, err := New(Options{Path: ":memory:", Table: "custom"})
	require.NoError(t, err)
	defer m.Close()
	assert.Equal(t, "custom", m.table)
}

func TestSqlite_New_EmptyPath_Bad(t *testing.T) {
	_, err := New(Options{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database path is required")
}

// --- Read/Write Tests ---

func TestSqlite_ReadWrite_Good(t *testing.T) {
	m := newTestMedium(t)

	err := m.Write("hello.txt", "world")
	require.NoError(t, err)

	content, err := m.Read("hello.txt")
	require.NoError(t, err)
	assert.Equal(t, "world", content)
}

func TestSqlite_ReadWrite_Overwrite_Good(t *testing.T) {
	m := newTestMedium(t)

	require.NoError(t, m.Write("file.txt", "first"))
	require.NoError(t, m.Write("file.txt", "second"))

	content, err := m.Read("file.txt")
	require.NoError(t, err)
	assert.Equal(t, "second", content)
}

func TestSqlite_ReadWrite_NestedPath_Good(t *testing.T) {
	m := newTestMedium(t)

	err := m.Write("a/b/c.txt", "nested")
	require.NoError(t, err)

	content, err := m.Read("a/b/c.txt")
	require.NoError(t, err)
	assert.Equal(t, "nested", content)
}

func TestSqlite_Read_NotFound_Bad(t *testing.T) {
	m := newTestMedium(t)

	_, err := m.Read("nonexistent.txt")
	assert.Error(t, err)
}

func TestSqlite_Read_EmptyPath_Bad(t *testing.T) {
	m := newTestMedium(t)

	_, err := m.Read("")
	assert.Error(t, err)
}

func TestSqlite_Write_EmptyPath_Bad(t *testing.T) {
	m := newTestMedium(t)

	err := m.Write("", "content")
	assert.Error(t, err)
}

func TestSqlite_Read_IsDirectory_Bad(t *testing.T) {
	m := newTestMedium(t)

	require.NoError(t, m.EnsureDir("mydir"))
	_, err := m.Read("mydir")
	assert.Error(t, err)
}

// --- EnsureDir Tests ---

func TestSqlite_EnsureDir_Good(t *testing.T) {
	m := newTestMedium(t)

	err := m.EnsureDir("mydir")
	require.NoError(t, err)
	assert.True(t, m.IsDir("mydir"))
}

func TestSqlite_EnsureDir_EmptyPath_Good(t *testing.T) {
	m := newTestMedium(t)
	// Root always exists, no-op
	err := m.EnsureDir("")
	assert.NoError(t, err)
}

func TestSqlite_EnsureDir_Idempotent_Good(t *testing.T) {
	m := newTestMedium(t)

	require.NoError(t, m.EnsureDir("mydir"))
	require.NoError(t, m.EnsureDir("mydir"))
	assert.True(t, m.IsDir("mydir"))
}

// --- IsFile Tests ---

func TestSqlite_IsFile_Good(t *testing.T) {
	m := newTestMedium(t)

	require.NoError(t, m.Write("file.txt", "content"))
	require.NoError(t, m.EnsureDir("mydir"))

	assert.True(t, m.IsFile("file.txt"))
	assert.False(t, m.IsFile("mydir"))
	assert.False(t, m.IsFile("nonexistent"))
	assert.False(t, m.IsFile(""))
}

// --- FileGet/FileSet Tests ---

func TestSqlite_FileGetFileSet_Good(t *testing.T) {
	m := newTestMedium(t)

	err := m.FileSet("key.txt", "value")
	require.NoError(t, err)

	val, err := m.FileGet("key.txt")
	require.NoError(t, err)
	assert.Equal(t, "value", val)
}

// --- Delete Tests ---

func TestSqlite_Delete_Good(t *testing.T) {
	m := newTestMedium(t)

	require.NoError(t, m.Write("to-delete.txt", "content"))
	assert.True(t, m.Exists("to-delete.txt"))

	err := m.Delete("to-delete.txt")
	require.NoError(t, err)
	assert.False(t, m.Exists("to-delete.txt"))
}

func TestSqlite_Delete_EmptyDir_Good(t *testing.T) {
	m := newTestMedium(t)

	require.NoError(t, m.EnsureDir("emptydir"))
	assert.True(t, m.IsDir("emptydir"))

	err := m.Delete("emptydir")
	require.NoError(t, err)
	assert.False(t, m.IsDir("emptydir"))
}

func TestSqlite_Delete_NotFound_Bad(t *testing.T) {
	m := newTestMedium(t)

	err := m.Delete("nonexistent")
	assert.Error(t, err)
}

func TestSqlite_Delete_EmptyPath_Bad(t *testing.T) {
	m := newTestMedium(t)

	err := m.Delete("")
	assert.Error(t, err)
}

func TestSqlite_Delete_NotEmpty_Bad(t *testing.T) {
	m := newTestMedium(t)

	require.NoError(t, m.EnsureDir("mydir"))
	require.NoError(t, m.Write("mydir/file.txt", "content"))

	err := m.Delete("mydir")
	assert.Error(t, err)
}

// --- DeleteAll Tests ---

func TestSqlite_DeleteAll_Good(t *testing.T) {
	m := newTestMedium(t)

	require.NoError(t, m.Write("dir/file1.txt", "a"))
	require.NoError(t, m.Write("dir/sub/file2.txt", "b"))
	require.NoError(t, m.Write("other.txt", "c"))

	err := m.DeleteAll("dir")
	require.NoError(t, err)

	assert.False(t, m.Exists("dir/file1.txt"))
	assert.False(t, m.Exists("dir/sub/file2.txt"))
	assert.True(t, m.Exists("other.txt"))
}

func TestSqlite_DeleteAll_SingleFile_Good(t *testing.T) {
	m := newTestMedium(t)

	require.NoError(t, m.Write("file.txt", "content"))

	err := m.DeleteAll("file.txt")
	require.NoError(t, err)
	assert.False(t, m.Exists("file.txt"))
}

func TestSqlite_DeleteAll_NotFound_Bad(t *testing.T) {
	m := newTestMedium(t)

	err := m.DeleteAll("nonexistent")
	assert.Error(t, err)
}

func TestSqlite_DeleteAll_EmptyPath_Bad(t *testing.T) {
	m := newTestMedium(t)

	err := m.DeleteAll("")
	assert.Error(t, err)
}

// --- Rename Tests ---

func TestSqlite_Rename_Good(t *testing.T) {
	m := newTestMedium(t)

	require.NoError(t, m.Write("old.txt", "content"))

	err := m.Rename("old.txt", "new.txt")
	require.NoError(t, err)

	assert.False(t, m.Exists("old.txt"))
	assert.True(t, m.IsFile("new.txt"))

	content, err := m.Read("new.txt")
	require.NoError(t, err)
	assert.Equal(t, "content", content)
}

func TestSqlite_Rename_Directory_Good(t *testing.T) {
	m := newTestMedium(t)

	require.NoError(t, m.EnsureDir("olddir"))
	require.NoError(t, m.Write("olddir/file.txt", "content"))

	err := m.Rename("olddir", "newdir")
	require.NoError(t, err)

	assert.False(t, m.Exists("olddir"))
	assert.False(t, m.Exists("olddir/file.txt"))
	assert.True(t, m.IsDir("newdir"))
	assert.True(t, m.IsFile("newdir/file.txt"))

	content, err := m.Read("newdir/file.txt")
	require.NoError(t, err)
	assert.Equal(t, "content", content)
}

func TestSqlite_Rename_SourceNotFound_Bad(t *testing.T) {
	m := newTestMedium(t)

	err := m.Rename("nonexistent", "new")
	assert.Error(t, err)
}

func TestSqlite_Rename_EmptyPath_Bad(t *testing.T) {
	m := newTestMedium(t)

	err := m.Rename("", "new")
	assert.Error(t, err)

	err = m.Rename("old", "")
	assert.Error(t, err)
}

// --- List Tests ---

func TestSqlite_List_Good(t *testing.T) {
	m := newTestMedium(t)

	require.NoError(t, m.Write("dir/file1.txt", "a"))
	require.NoError(t, m.Write("dir/file2.txt", "b"))
	require.NoError(t, m.Write("dir/sub/file3.txt", "c"))

	entries, err := m.List("dir")
	require.NoError(t, err)

	names := make(map[string]bool)
	for _, e := range entries {
		names[e.Name()] = true
	}

	assert.True(t, names["file1.txt"])
	assert.True(t, names["file2.txt"])
	assert.True(t, names["sub"])
	assert.Len(t, entries, 3)
}

func TestSqlite_List_Root_Good(t *testing.T) {
	m := newTestMedium(t)

	require.NoError(t, m.Write("root.txt", "content"))
	require.NoError(t, m.Write("dir/nested.txt", "nested"))

	entries, err := m.List("")
	require.NoError(t, err)

	names := make(map[string]bool)
	for _, e := range entries {
		names[e.Name()] = true
	}

	assert.True(t, names["root.txt"])
	assert.True(t, names["dir"])
}

func TestSqlite_List_DirectoryEntry_Good(t *testing.T) {
	m := newTestMedium(t)

	require.NoError(t, m.Write("dir/sub/file.txt", "content"))

	entries, err := m.List("dir")
	require.NoError(t, err)

	require.Len(t, entries, 1)
	assert.Equal(t, "sub", entries[0].Name())
	assert.True(t, entries[0].IsDir())

	info, err := entries[0].Info()
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

// --- Stat Tests ---

func TestSqlite_Stat_Good(t *testing.T) {
	m := newTestMedium(t)

	require.NoError(t, m.Write("file.txt", "hello world"))

	info, err := m.Stat("file.txt")
	require.NoError(t, err)
	assert.Equal(t, "file.txt", info.Name())
	assert.Equal(t, int64(11), info.Size())
	assert.False(t, info.IsDir())
}

func TestSqlite_Stat_Directory_Good(t *testing.T) {
	m := newTestMedium(t)

	require.NoError(t, m.EnsureDir("mydir"))

	info, err := m.Stat("mydir")
	require.NoError(t, err)
	assert.Equal(t, "mydir", info.Name())
	assert.True(t, info.IsDir())
}

func TestSqlite_Stat_NotFound_Bad(t *testing.T) {
	m := newTestMedium(t)

	_, err := m.Stat("nonexistent")
	assert.Error(t, err)
}

func TestSqlite_Stat_EmptyPath_Bad(t *testing.T) {
	m := newTestMedium(t)

	_, err := m.Stat("")
	assert.Error(t, err)
}

// --- Open Tests ---

func TestSqlite_Open_Good(t *testing.T) {
	m := newTestMedium(t)

	require.NoError(t, m.Write("file.txt", "open me"))

	f, err := m.Open("file.txt")
	require.NoError(t, err)
	defer f.Close()

	data, err := goio.ReadAll(f.(goio.Reader))
	require.NoError(t, err)
	assert.Equal(t, "open me", string(data))

	stat, err := f.Stat()
	require.NoError(t, err)
	assert.Equal(t, "file.txt", stat.Name())
}

func TestSqlite_Open_NotFound_Bad(t *testing.T) {
	m := newTestMedium(t)

	_, err := m.Open("nonexistent.txt")
	assert.Error(t, err)
}

func TestSqlite_Open_IsDirectory_Bad(t *testing.T) {
	m := newTestMedium(t)

	require.NoError(t, m.EnsureDir("mydir"))
	_, err := m.Open("mydir")
	assert.Error(t, err)
}

// --- Create Tests ---

func TestSqlite_Create_Good(t *testing.T) {
	m := newTestMedium(t)

	w, err := m.Create("new.txt")
	require.NoError(t, err)

	n, err := w.Write([]byte("created"))
	require.NoError(t, err)
	assert.Equal(t, 7, n)

	err = w.Close()
	require.NoError(t, err)

	content, err := m.Read("new.txt")
	require.NoError(t, err)
	assert.Equal(t, "created", content)
}

func TestSqlite_Create_Overwrite_Good(t *testing.T) {
	m := newTestMedium(t)

	require.NoError(t, m.Write("file.txt", "old content"))

	w, err := m.Create("file.txt")
	require.NoError(t, err)
	_, err = w.Write([]byte("new"))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	content, err := m.Read("file.txt")
	require.NoError(t, err)
	assert.Equal(t, "new", content)
}

func TestSqlite_Create_EmptyPath_Bad(t *testing.T) {
	m := newTestMedium(t)

	_, err := m.Create("")
	assert.Error(t, err)
}

// --- Append Tests ---

func TestSqlite_Append_Good(t *testing.T) {
	m := newTestMedium(t)

	require.NoError(t, m.Write("append.txt", "hello"))

	w, err := m.Append("append.txt")
	require.NoError(t, err)

	_, err = w.Write([]byte(" world"))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	content, err := m.Read("append.txt")
	require.NoError(t, err)
	assert.Equal(t, "hello world", content)
}

func TestSqlite_Append_NewFile_Good(t *testing.T) {
	m := newTestMedium(t)

	w, err := m.Append("new.txt")
	require.NoError(t, err)

	_, err = w.Write([]byte("fresh"))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	content, err := m.Read("new.txt")
	require.NoError(t, err)
	assert.Equal(t, "fresh", content)
}

func TestSqlite_Append_EmptyPath_Bad(t *testing.T) {
	m := newTestMedium(t)

	_, err := m.Append("")
	assert.Error(t, err)
}

// --- ReadStream Tests ---

func TestSqlite_ReadStream_Good(t *testing.T) {
	m := newTestMedium(t)

	require.NoError(t, m.Write("stream.txt", "streaming content"))

	reader, err := m.ReadStream("stream.txt")
	require.NoError(t, err)
	defer reader.Close()

	data, err := goio.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, "streaming content", string(data))
}

func TestSqlite_ReadStream_NotFound_Bad(t *testing.T) {
	m := newTestMedium(t)

	_, err := m.ReadStream("nonexistent.txt")
	assert.Error(t, err)
}

func TestSqlite_ReadStream_IsDirectory_Bad(t *testing.T) {
	m := newTestMedium(t)

	require.NoError(t, m.EnsureDir("mydir"))
	_, err := m.ReadStream("mydir")
	assert.Error(t, err)
}

// --- WriteStream Tests ---

func TestSqlite_WriteStream_Good(t *testing.T) {
	m := newTestMedium(t)

	writer, err := m.WriteStream("output.txt")
	require.NoError(t, err)

	_, err = goio.Copy(writer, core.NewReader("piped data"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	content, err := m.Read("output.txt")
	require.NoError(t, err)
	assert.Equal(t, "piped data", content)
}

// --- Exists Tests ---

func TestSqlite_Exists_Good(t *testing.T) {
	m := newTestMedium(t)

	assert.False(t, m.Exists("nonexistent"))

	require.NoError(t, m.Write("file.txt", "content"))
	assert.True(t, m.Exists("file.txt"))

	require.NoError(t, m.EnsureDir("mydir"))
	assert.True(t, m.Exists("mydir"))
}

func TestSqlite_Exists_EmptyPath_Good(t *testing.T) {
	m := newTestMedium(t)
	// Root always exists
	assert.True(t, m.Exists(""))
}

// --- IsDir Tests ---

func TestSqlite_IsDir_Good(t *testing.T) {
	m := newTestMedium(t)

	require.NoError(t, m.Write("file.txt", "content"))
	require.NoError(t, m.EnsureDir("mydir"))

	assert.True(t, m.IsDir("mydir"))
	assert.False(t, m.IsDir("file.txt"))
	assert.False(t, m.IsDir("nonexistent"))
	assert.False(t, m.IsDir(""))
}

// --- normaliseEntryPath Tests ---

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

// --- Interface Compliance ---

func TestSqlite_InterfaceCompliance_Ugly(t *testing.T) {
	m := newTestMedium(t)

	// Verify all methods exist by asserting the interface shape.
	var _ interface {
		Read(string) (string, error)
		Write(string, string) error
		EnsureDir(string) error
		IsFile(string) bool
		FileGet(string) (string, error)
		FileSet(string, string) error
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
	} = m
}

// --- Custom Table ---

func TestSqlite_CustomTable_Good(t *testing.T) {
	m, err := New(Options{Path: ":memory:", Table: "my_files"})
	require.NoError(t, err)
	defer m.Close()

	require.NoError(t, m.Write("file.txt", "content"))

	content, err := m.Read("file.txt")
	require.NoError(t, err)
	assert.Equal(t, "content", content)
}
