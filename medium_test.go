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
	memoryMedium := NewMemoryMedium()
	assert.NotNil(t, memoryMedium)
	assert.NotNil(t, memoryMedium.fileContents)
	assert.NotNil(t, memoryMedium.directories)
	assert.Empty(t, memoryMedium.fileContents)
	assert.Empty(t, memoryMedium.directories)
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
	memoryMedium := NewMemoryMedium()
	memoryMedium.fileContents["test.txt"] = "hello world"
	content, err := memoryMedium.Read("test.txt")
	assert.NoError(t, err)
	assert.Equal(t, "hello world", content)
}

func TestMemoryMedium_Read_Bad(t *testing.T) {
	memoryMedium := NewMemoryMedium()
	_, err := memoryMedium.Read("nonexistent.txt")
	assert.Error(t, err)
}

func TestMemoryMedium_Write_Good(t *testing.T) {
	memoryMedium := NewMemoryMedium()
	err := memoryMedium.Write("test.txt", "content")
	assert.NoError(t, err)
	assert.Equal(t, "content", memoryMedium.fileContents["test.txt"])

	err = memoryMedium.Write("test.txt", "new content")
	assert.NoError(t, err)
	assert.Equal(t, "new content", memoryMedium.fileContents["test.txt"])
}

func TestMemoryMedium_WriteMode_Good(t *testing.T) {
	memoryMedium := NewMemoryMedium()

	err := memoryMedium.WriteMode("secure.txt", "secret", 0600)
	require.NoError(t, err)

	content, err := memoryMedium.Read("secure.txt")
	require.NoError(t, err)
	assert.Equal(t, "secret", content)

	info, err := memoryMedium.Stat("secure.txt")
	require.NoError(t, err)
	assert.Equal(t, fs.FileMode(0600), info.Mode())

	file, err := memoryMedium.Open("secure.txt")
	require.NoError(t, err)
	fileInfo, err := file.Stat()
	require.NoError(t, err)
	assert.Equal(t, fs.FileMode(0600), fileInfo.Mode())
}

func TestMemoryMedium_EnsureDir_Good(t *testing.T) {
	memoryMedium := NewMemoryMedium()
	err := memoryMedium.EnsureDir("/path/to/dir")
	assert.NoError(t, err)
	assert.True(t, memoryMedium.directories["/path/to/dir"])
}

func TestMemoryMedium_EnsureDir_CreatesParents_Good(t *testing.T) {
	memoryMedium := NewMemoryMedium()

	require.NoError(t, memoryMedium.EnsureDir("alpha/beta/gamma"))

	assert.True(t, memoryMedium.IsDir("alpha"))
	assert.True(t, memoryMedium.IsDir("alpha/beta"))
	assert.True(t, memoryMedium.IsDir("alpha/beta/gamma"))
}

func TestMemoryMedium_IsFile_Good(t *testing.T) {
	memoryMedium := NewMemoryMedium()
	memoryMedium.fileContents["exists.txt"] = "content"

	assert.True(t, memoryMedium.IsFile("exists.txt"))
	assert.False(t, memoryMedium.IsFile("nonexistent.txt"))
}

func TestMemoryMedium_Write_CreatesParentDirectories_Good(t *testing.T) {
	memoryMedium := NewMemoryMedium()

	require.NoError(t, memoryMedium.Write("nested/path/file.txt", "content"))

	assert.True(t, memoryMedium.Exists("nested"))
	assert.True(t, memoryMedium.IsDir("nested"))
	assert.True(t, memoryMedium.Exists("nested/path"))
	assert.True(t, memoryMedium.IsDir("nested/path"))
}

func TestMemoryMedium_Delete_Good(t *testing.T) {
	memoryMedium := NewMemoryMedium()
	memoryMedium.fileContents["test.txt"] = "content"

	err := memoryMedium.Delete("test.txt")
	assert.NoError(t, err)
	assert.False(t, memoryMedium.IsFile("test.txt"))
}

func TestMemoryMedium_Delete_NotFound_Bad(t *testing.T) {
	memoryMedium := NewMemoryMedium()
	err := memoryMedium.Delete("nonexistent.txt")
	assert.Error(t, err)
}

func TestMemoryMedium_Delete_DirNotEmpty_Bad(t *testing.T) {
	memoryMedium := NewMemoryMedium()
	memoryMedium.directories["mydir"] = true
	memoryMedium.fileContents["mydir/file.txt"] = "content"

	err := memoryMedium.Delete("mydir")
	assert.Error(t, err)
}

func TestMemoryMedium_Delete_InferredDirNotEmpty_Bad(t *testing.T) {
	memoryMedium := NewMemoryMedium()

	require.NoError(t, memoryMedium.Write("mydir/file.txt", "content"))

	err := memoryMedium.Delete("mydir")
	assert.Error(t, err)
}

func TestMemoryMedium_DeleteAll_Good(t *testing.T) {
	memoryMedium := NewMemoryMedium()
	memoryMedium.directories["mydir"] = true
	memoryMedium.directories["mydir/subdir"] = true
	memoryMedium.fileContents["mydir/file.txt"] = "content"
	memoryMedium.fileContents["mydir/subdir/nested.txt"] = "nested"

	err := memoryMedium.DeleteAll("mydir")
	assert.NoError(t, err)
	assert.Empty(t, memoryMedium.directories)
	assert.Empty(t, memoryMedium.fileContents)
}

func TestMemoryMedium_Rename_Good(t *testing.T) {
	memoryMedium := NewMemoryMedium()
	memoryMedium.fileContents["old.txt"] = "content"

	err := memoryMedium.Rename("old.txt", "new.txt")
	assert.NoError(t, err)
	assert.False(t, memoryMedium.IsFile("old.txt"))
	assert.True(t, memoryMedium.IsFile("new.txt"))
	assert.Equal(t, "content", memoryMedium.fileContents["new.txt"])
}

func TestMemoryMedium_Rename_Dir_Good(t *testing.T) {
	memoryMedium := NewMemoryMedium()
	memoryMedium.directories["olddir"] = true
	memoryMedium.fileContents["olddir/file.txt"] = "content"

	err := memoryMedium.Rename("olddir", "newdir")
	assert.NoError(t, err)
	assert.False(t, memoryMedium.directories["olddir"])
	assert.True(t, memoryMedium.directories["newdir"])
	assert.Equal(t, "content", memoryMedium.fileContents["newdir/file.txt"])
}

func TestMemoryMedium_Rename_InferredDir_Good(t *testing.T) {
	memoryMedium := NewMemoryMedium()
	require.NoError(t, memoryMedium.Write("olddir/file.txt", "content"))

	require.NoError(t, memoryMedium.Rename("olddir", "newdir"))

	assert.False(t, memoryMedium.Exists("olddir"))
	assert.True(t, memoryMedium.Exists("newdir"))
	assert.True(t, memoryMedium.IsDir("newdir"))
	assert.Equal(t, "content", memoryMedium.fileContents["newdir/file.txt"])
}

func TestMemoryMedium_List_Good(t *testing.T) {
	memoryMedium := NewMemoryMedium()
	memoryMedium.directories["mydir"] = true
	memoryMedium.fileContents["mydir/file1.txt"] = "content1"
	memoryMedium.fileContents["mydir/file2.txt"] = "content2"
	memoryMedium.directories["mydir/subdir"] = true

	entries, err := memoryMedium.List("mydir")
	assert.NoError(t, err)
	assert.Len(t, entries, 3)
	assert.Equal(t, "file1.txt", entries[0].Name())
	assert.Equal(t, "file2.txt", entries[1].Name())
	assert.Equal(t, "subdir", entries[2].Name())

	names := make(map[string]bool)
	for _, entry := range entries {
		names[entry.Name()] = true
	}
	assert.True(t, names["file1.txt"])
	assert.True(t, names["file2.txt"])
	assert.True(t, names["subdir"])
}

func TestMemoryMedium_Stat_Good(t *testing.T) {
	memoryMedium := NewMemoryMedium()
	memoryMedium.fileContents["test.txt"] = "hello world"

	info, err := memoryMedium.Stat("test.txt")
	assert.NoError(t, err)
	assert.Equal(t, "test.txt", info.Name())
	assert.Equal(t, int64(11), info.Size())
	assert.False(t, info.IsDir())
}

func TestMemoryMedium_Stat_Dir_Good(t *testing.T) {
	memoryMedium := NewMemoryMedium()
	memoryMedium.directories["mydir"] = true

	info, err := memoryMedium.Stat("mydir")
	assert.NoError(t, err)
	assert.Equal(t, "mydir", info.Name())
	assert.True(t, info.IsDir())
}

func TestMemoryMedium_Exists_Good(t *testing.T) {
	memoryMedium := NewMemoryMedium()
	memoryMedium.fileContents["file.txt"] = "content"
	memoryMedium.directories["mydir"] = true

	assert.True(t, memoryMedium.Exists("file.txt"))
	assert.True(t, memoryMedium.Exists("mydir"))
	assert.False(t, memoryMedium.Exists("nonexistent"))
}

func TestMemoryMedium_IsDir_Good(t *testing.T) {
	memoryMedium := NewMemoryMedium()
	memoryMedium.fileContents["file.txt"] = "content"
	memoryMedium.directories["mydir"] = true

	assert.False(t, memoryMedium.IsDir("file.txt"))
	assert.True(t, memoryMedium.IsDir("mydir"))
	assert.False(t, memoryMedium.IsDir("nonexistent"))
}

func TestMemoryMedium_StreamAndFSHelpers_Good(t *testing.T) {
	memoryMedium := NewMemoryMedium()
	require.NoError(t, memoryMedium.EnsureDir("dir"))
	require.NoError(t, memoryMedium.Write("dir/file.txt", "alpha"))

	statInfo, err := memoryMedium.Stat("dir/file.txt")
	require.NoError(t, err)

	file, err := memoryMedium.Open("dir/file.txt")
	require.NoError(t, err)

	info, err := file.Stat()
	require.NoError(t, err)
	assert.Equal(t, "file.txt", info.Name())
	assert.Equal(t, int64(5), info.Size())
	assert.Equal(t, fs.FileMode(0644), info.Mode())
	assert.Equal(t, statInfo.ModTime(), info.ModTime())
	assert.False(t, info.IsDir())
	assert.Nil(t, info.Sys())

	data, err := goio.ReadAll(file)
	require.NoError(t, err)
	assert.Equal(t, "alpha", string(data))
	require.NoError(t, file.Close())

	entries, err := memoryMedium.List("dir")
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "file.txt", entries[0].Name())
	assert.False(t, entries[0].IsDir())
	assert.Equal(t, fs.FileMode(0), entries[0].Type())

	entryInfo, err := entries[0].Info()
	require.NoError(t, err)
	assert.Equal(t, "file.txt", entryInfo.Name())
	assert.Equal(t, int64(5), entryInfo.Size())
	assert.Equal(t, fs.FileMode(0644), entryInfo.Mode())
	assert.Equal(t, statInfo.ModTime(), entryInfo.ModTime())

	writer, err := memoryMedium.Create("created.txt")
	require.NoError(t, err)
	_, err = writer.Write([]byte("created"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	appendWriter, err := memoryMedium.Append("created.txt")
	require.NoError(t, err)
	_, err = appendWriter.Write([]byte(" later"))
	require.NoError(t, err)
	require.NoError(t, appendWriter.Close())

	reader, err := memoryMedium.ReadStream("created.txt")
	require.NoError(t, err)
	streamed, err := goio.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, "created later", string(streamed))
	require.NoError(t, reader.Close())

	writeStream, err := memoryMedium.WriteStream("streamed.txt")
	require.NoError(t, err)
	_, err = writeStream.Write([]byte("stream output"))
	require.NoError(t, err)
	require.NoError(t, writeStream.Close())

	assert.Equal(t, "stream output", memoryMedium.fileContents["streamed.txt"])
	statInfo, err = memoryMedium.Stat("streamed.txt")
	require.NoError(t, err)
	assert.Equal(t, fs.FileMode(0644), statInfo.Mode())
	assert.False(t, statInfo.ModTime().IsZero())
}

func TestIO_Read_Good(t *testing.T) {
	memoryMedium := NewMemoryMedium()
	memoryMedium.fileContents["test.txt"] = "hello"
	content, err := Read(memoryMedium, "test.txt")
	assert.NoError(t, err)
	assert.Equal(t, "hello", content)
}

func TestIO_Write_Good(t *testing.T) {
	memoryMedium := NewMemoryMedium()
	err := Write(memoryMedium, "test.txt", "hello")
	assert.NoError(t, err)
	assert.Equal(t, "hello", memoryMedium.fileContents["test.txt"])
}

func TestIO_EnsureDir_Good(t *testing.T) {
	memoryMedium := NewMemoryMedium()
	err := EnsureDir(memoryMedium, "/my/dir")
	assert.NoError(t, err)
	assert.True(t, memoryMedium.directories["/my/dir"])
}

func TestIO_IsFile_Good(t *testing.T) {
	memoryMedium := NewMemoryMedium()
	memoryMedium.fileContents["exists.txt"] = "content"

	assert.True(t, IsFile(memoryMedium, "exists.txt"))
	assert.False(t, IsFile(memoryMedium, "nonexistent.txt"))
}

func TestIO_NewSandboxed_Good(t *testing.T) {
	root := t.TempDir()

	memoryMedium, err := NewSandboxed(root)
	require.NoError(t, err)

	require.NoError(t, memoryMedium.Write("config/app.yaml", "port: 8080"))

	content, err := memoryMedium.Read("config/app.yaml")
	require.NoError(t, err)
	assert.Equal(t, "port: 8080", content)
	assert.True(t, memoryMedium.IsDir("config"))
}

func TestIO_ReadWriteStream_Good(t *testing.T) {
	memoryMedium := NewMemoryMedium()

	writer, err := WriteStream(memoryMedium, "logs/run.txt")
	require.NoError(t, err)
	_, err = writer.Write([]byte("started"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	reader, err := ReadStream(memoryMedium, "logs/run.txt")
	require.NoError(t, err)
	data, err := goio.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, "started", string(data))
	require.NoError(t, reader.Close())
}

func TestIO_Copy_Good(t *testing.T) {
	source := NewMemoryMedium()
	dest := NewMemoryMedium()
	source.fileContents["test.txt"] = "hello"
	err := Copy(source, "test.txt", dest, "test.txt")
	assert.NoError(t, err)
	assert.Equal(t, "hello", dest.fileContents["test.txt"])

	source.fileContents["original.txt"] = "content"
	err = Copy(source, "original.txt", dest, "copied.txt")
	assert.NoError(t, err)
	assert.Equal(t, "content", dest.fileContents["copied.txt"])
}

func TestIO_Copy_Bad(t *testing.T) {
	source := NewMemoryMedium()
	dest := NewMemoryMedium()
	err := Copy(source, "nonexistent.txt", dest, "dest.txt")
	assert.Error(t, err)
}

func TestIO_LocalGlobal_Good(t *testing.T) {
	assert.NotNil(t, Local, "io.Local should be initialised")

	var localMedium = Local
	assert.NotNil(t, localMedium)
}
