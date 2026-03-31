package io

import (
	goio "io"
	"io/fs"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryMedium_NewMemoryMedium_Good(t *testing.T) {
	medium := NewMemoryMedium()
	assert.NotNil(t, medium)
	assert.NotNil(t, medium.files)
	assert.NotNil(t, medium.dirs)
	assert.Empty(t, medium.files)
	assert.Empty(t, medium.dirs)
}

func TestMemoryMedium_NewFileInfo_Good(t *testing.T) {
	info := NewFileInfo("app.yaml", 8, 0644, time.Unix(0, 0), false)

	assert.Equal(t, "app.yaml", info.Name())
	assert.Equal(t, int64(8), info.Size())
	assert.Equal(t, fs.FileMode(0644), info.Mode())
	assert.True(t, info.ModTime().Equal(time.Unix(0, 0)))
	assert.False(t, info.IsDir())
	assert.Nil(t, info.Sys())
}

func TestMemoryMedium_NewDirEntry_Good(t *testing.T) {
	info := NewFileInfo("app.yaml", 8, 0644, time.Unix(0, 0), false)
	entry := NewDirEntry("app.yaml", false, 0644, info)

	assert.Equal(t, "app.yaml", entry.Name())
	assert.False(t, entry.IsDir())
	assert.Equal(t, fs.FileMode(0), entry.Type())

	entryInfo, err := entry.Info()
	require.NoError(t, err)
	assert.Equal(t, "app.yaml", entryInfo.Name())
	assert.Equal(t, int64(8), entryInfo.Size())
}

func TestMemoryMedium_Read_Good(t *testing.T) {
	m := NewMemoryMedium()
	m.files["test.txt"] = "hello world"
	content, err := m.Read("test.txt")
	assert.NoError(t, err)
	assert.Equal(t, "hello world", content)
}

func TestMemoryMedium_Read_Bad(t *testing.T) {
	m := NewMemoryMedium()
	_, err := m.Read("nonexistent.txt")
	assert.Error(t, err)
}

func TestMemoryMedium_Write_Good(t *testing.T) {
	m := NewMemoryMedium()
	err := m.Write("test.txt", "content")
	assert.NoError(t, err)
	assert.Equal(t, "content", m.files["test.txt"])

	err = m.Write("test.txt", "new content")
	assert.NoError(t, err)
	assert.Equal(t, "new content", m.files["test.txt"])
}

func TestMemoryMedium_WriteMode_Good(t *testing.T) {
	m := NewMemoryMedium()

	err := m.WriteMode("secure.txt", "secret", 0600)
	require.NoError(t, err)

	content, err := m.Read("secure.txt")
	require.NoError(t, err)
	assert.Equal(t, "secret", content)
}

func TestMemoryMedium_EnsureDir_Good(t *testing.T) {
	m := NewMemoryMedium()
	err := m.EnsureDir("/path/to/dir")
	assert.NoError(t, err)
	assert.True(t, m.dirs["/path/to/dir"])
}

func TestMemoryMedium_IsFile_Good(t *testing.T) {
	m := NewMemoryMedium()
	m.files["exists.txt"] = "content"

	assert.True(t, m.IsFile("exists.txt"))
	assert.False(t, m.IsFile("nonexistent.txt"))
}

func TestMemoryMedium_FileGet_Good(t *testing.T) {
	m := NewMemoryMedium()
	m.files["test.txt"] = "content"
	content, err := m.FileGet("test.txt")
	assert.NoError(t, err)
	assert.Equal(t, "content", content)
}

func TestMemoryMedium_FileSet_Good(t *testing.T) {
	m := NewMemoryMedium()
	err := m.FileSet("test.txt", "content")
	assert.NoError(t, err)
	assert.Equal(t, "content", m.files["test.txt"])
}

func TestMemoryMedium_Delete_Good(t *testing.T) {
	m := NewMemoryMedium()
	m.files["test.txt"] = "content"

	err := m.Delete("test.txt")
	assert.NoError(t, err)
	assert.False(t, m.IsFile("test.txt"))
}

func TestMemoryMedium_Delete_NotFound_Bad(t *testing.T) {
	m := NewMemoryMedium()
	err := m.Delete("nonexistent.txt")
	assert.Error(t, err)
}

func TestMemoryMedium_Delete_DirNotEmpty_Bad(t *testing.T) {
	m := NewMemoryMedium()
	m.dirs["mydir"] = true
	m.files["mydir/file.txt"] = "content"

	err := m.Delete("mydir")
	assert.Error(t, err)
}

func TestMemoryMedium_DeleteAll_Good(t *testing.T) {
	m := NewMemoryMedium()
	m.dirs["mydir"] = true
	m.dirs["mydir/subdir"] = true
	m.files["mydir/file.txt"] = "content"
	m.files["mydir/subdir/nested.txt"] = "nested"

	err := m.DeleteAll("mydir")
	assert.NoError(t, err)
	assert.Empty(t, m.dirs)
	assert.Empty(t, m.files)
}

func TestMemoryMedium_Rename_Good(t *testing.T) {
	m := NewMemoryMedium()
	m.files["old.txt"] = "content"

	err := m.Rename("old.txt", "new.txt")
	assert.NoError(t, err)
	assert.False(t, m.IsFile("old.txt"))
	assert.True(t, m.IsFile("new.txt"))
	assert.Equal(t, "content", m.files["new.txt"])
}

func TestMemoryMedium_Rename_Dir_Good(t *testing.T) {
	m := NewMemoryMedium()
	m.dirs["olddir"] = true
	m.files["olddir/file.txt"] = "content"

	err := m.Rename("olddir", "newdir")
	assert.NoError(t, err)
	assert.False(t, m.dirs["olddir"])
	assert.True(t, m.dirs["newdir"])
	assert.Equal(t, "content", m.files["newdir/file.txt"])
}

func TestMemoryMedium_List_Good(t *testing.T) {
	m := NewMemoryMedium()
	m.dirs["mydir"] = true
	m.files["mydir/file1.txt"] = "content1"
	m.files["mydir/file2.txt"] = "content2"
	m.dirs["mydir/subdir"] = true

	entries, err := m.List("mydir")
	assert.NoError(t, err)
	assert.Len(t, entries, 3)

	names := make(map[string]bool)
	for _, e := range entries {
		names[e.Name()] = true
	}
	assert.True(t, names["file1.txt"])
	assert.True(t, names["file2.txt"])
	assert.True(t, names["subdir"])
}

func TestMemoryMedium_Stat_Good(t *testing.T) {
	m := NewMemoryMedium()
	m.files["test.txt"] = "hello world"

	info, err := m.Stat("test.txt")
	assert.NoError(t, err)
	assert.Equal(t, "test.txt", info.Name())
	assert.Equal(t, int64(11), info.Size())
	assert.False(t, info.IsDir())
}

func TestMemoryMedium_Stat_Dir_Good(t *testing.T) {
	m := NewMemoryMedium()
	m.dirs["mydir"] = true

	info, err := m.Stat("mydir")
	assert.NoError(t, err)
	assert.Equal(t, "mydir", info.Name())
	assert.True(t, info.IsDir())
}

func TestMemoryMedium_Exists_Good(t *testing.T) {
	m := NewMemoryMedium()
	m.files["file.txt"] = "content"
	m.dirs["mydir"] = true

	assert.True(t, m.Exists("file.txt"))
	assert.True(t, m.Exists("mydir"))
	assert.False(t, m.Exists("nonexistent"))
}

func TestMemoryMedium_IsDir_Good(t *testing.T) {
	m := NewMemoryMedium()
	m.files["file.txt"] = "content"
	m.dirs["mydir"] = true

	assert.False(t, m.IsDir("file.txt"))
	assert.True(t, m.IsDir("mydir"))
	assert.False(t, m.IsDir("nonexistent"))
}

func TestMemoryMedium_StreamAndFSHelpers_Good(t *testing.T) {
	m := NewMemoryMedium()
	require.NoError(t, m.EnsureDir("dir"))
	require.NoError(t, m.Write("dir/file.txt", "alpha"))

	file, err := m.Open("dir/file.txt")
	require.NoError(t, err)

	info, err := file.Stat()
	require.NoError(t, err)
	assert.Equal(t, "file.txt", info.Name())
	assert.Equal(t, int64(5), info.Size())
	assert.Equal(t, fs.FileMode(0), info.Mode())
	assert.True(t, info.ModTime().IsZero())
	assert.False(t, info.IsDir())
	assert.Nil(t, info.Sys())

	data, err := goio.ReadAll(file)
	require.NoError(t, err)
	assert.Equal(t, "alpha", string(data))
	require.NoError(t, file.Close())

	entries, err := m.List("dir")
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "file.txt", entries[0].Name())
	assert.False(t, entries[0].IsDir())
	assert.Equal(t, fs.FileMode(0), entries[0].Type())

	entryInfo, err := entries[0].Info()
	require.NoError(t, err)
	assert.Equal(t, "file.txt", entryInfo.Name())
	assert.Equal(t, int64(5), entryInfo.Size())

	writer, err := m.Create("created.txt")
	require.NoError(t, err)
	_, err = writer.Write([]byte("created"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	appendWriter, err := m.Append("created.txt")
	require.NoError(t, err)
	_, err = appendWriter.Write([]byte(" later"))
	require.NoError(t, err)
	require.NoError(t, appendWriter.Close())

	reader, err := m.ReadStream("created.txt")
	require.NoError(t, err)
	streamed, err := goio.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, "created later", string(streamed))
	require.NoError(t, reader.Close())

	writeStream, err := m.WriteStream("streamed.txt")
	require.NoError(t, err)
	_, err = writeStream.Write([]byte("stream output"))
	require.NoError(t, err)
	require.NoError(t, writeStream.Close())

	assert.Equal(t, "stream output", m.files["streamed.txt"])
}

func TestIO_Read_Good(t *testing.T) {
	m := NewMemoryMedium()
	m.files["test.txt"] = "hello"
	content, err := Read(m, "test.txt")
	assert.NoError(t, err)
	assert.Equal(t, "hello", content)
}

func TestIO_Write_Good(t *testing.T) {
	m := NewMemoryMedium()
	err := Write(m, "test.txt", "hello")
	assert.NoError(t, err)
	assert.Equal(t, "hello", m.files["test.txt"])
}

func TestIO_EnsureDir_Good(t *testing.T) {
	m := NewMemoryMedium()
	err := EnsureDir(m, "/my/dir")
	assert.NoError(t, err)
	assert.True(t, m.dirs["/my/dir"])
}

func TestIO_IsFile_Good(t *testing.T) {
	m := NewMemoryMedium()
	m.files["exists.txt"] = "content"

	assert.True(t, IsFile(m, "exists.txt"))
	assert.False(t, IsFile(m, "nonexistent.txt"))
}

func TestIO_NewSandboxed_Good(t *testing.T) {
	root := t.TempDir()

	m, err := NewSandboxed(root)
	require.NoError(t, err)

	require.NoError(t, m.Write("config/app.yaml", "port: 8080"))

	content, err := m.Read("config/app.yaml")
	require.NoError(t, err)
	assert.Equal(t, "port: 8080", content)
	assert.True(t, m.IsDir("config"))
}

func TestIO_ReadWriteStream_Good(t *testing.T) {
	m := NewMemoryMedium()

	writer, err := WriteStream(m, "logs/run.txt")
	require.NoError(t, err)
	_, err = writer.Write([]byte("started"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	reader, err := ReadStream(m, "logs/run.txt")
	require.NoError(t, err)
	data, err := goio.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, "started", string(data))
	require.NoError(t, reader.Close())
}

func TestIO_Copy_Good(t *testing.T) {
	source := NewMemoryMedium()
	dest := NewMemoryMedium()
	source.files["test.txt"] = "hello"
	err := Copy(source, "test.txt", dest, "test.txt")
	assert.NoError(t, err)
	assert.Equal(t, "hello", dest.files["test.txt"])

	source.files["original.txt"] = "content"
	err = Copy(source, "original.txt", dest, "copied.txt")
	assert.NoError(t, err)
	assert.Equal(t, "content", dest.files["copied.txt"])
}

func TestIO_Copy_Bad(t *testing.T) {
	source := NewMemoryMedium()
	dest := NewMemoryMedium()
	err := Copy(source, "nonexistent.txt", dest, "dest.txt")
	assert.Error(t, err)
}

func TestIO_LocalGlobal_Good(t *testing.T) {
	assert.NotNil(t, Local, "io.Local should be initialised")

	var m = Local
	assert.NotNil(t, m)
}
