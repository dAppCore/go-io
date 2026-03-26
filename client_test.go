package io

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- MockMedium Tests ---

func TestClient_NewMockMedium_Good(t *testing.T) {
	m := NewMockMedium()
	assert.NotNil(t, m)
	assert.NotNil(t, m.Files)
	assert.NotNil(t, m.Dirs)
	assert.Empty(t, m.Files)
	assert.Empty(t, m.Dirs)
}

func TestClient_MockMedium_Read_Good(t *testing.T) {
	m := NewMockMedium()
	m.Files["test.txt"] = "hello world"
	content, err := m.Read("test.txt")
	assert.NoError(t, err)
	assert.Equal(t, "hello world", content)
}

func TestClient_MockMedium_Read_Bad(t *testing.T) {
	m := NewMockMedium()
	_, err := m.Read("nonexistent.txt")
	assert.Error(t, err)
}

func TestClient_MockMedium_Write_Good(t *testing.T) {
	m := NewMockMedium()
	err := m.Write("test.txt", "content")
	assert.NoError(t, err)
	assert.Equal(t, "content", m.Files["test.txt"])

	// Overwrite existing file
	err = m.Write("test.txt", "new content")
	assert.NoError(t, err)
	assert.Equal(t, "new content", m.Files["test.txt"])
}

func TestClient_MockMedium_EnsureDir_Good(t *testing.T) {
	m := NewMockMedium()
	err := m.EnsureDir("/path/to/dir")
	assert.NoError(t, err)
	assert.True(t, m.Dirs["/path/to/dir"])
}

func TestClient_MockMedium_IsFile_Good(t *testing.T) {
	m := NewMockMedium()
	m.Files["exists.txt"] = "content"

	assert.True(t, m.IsFile("exists.txt"))
	assert.False(t, m.IsFile("nonexistent.txt"))
}

func TestClient_MockMedium_FileGet_Good(t *testing.T) {
	m := NewMockMedium()
	m.Files["test.txt"] = "content"
	content, err := m.FileGet("test.txt")
	assert.NoError(t, err)
	assert.Equal(t, "content", content)
}

func TestClient_MockMedium_FileSet_Good(t *testing.T) {
	m := NewMockMedium()
	err := m.FileSet("test.txt", "content")
	assert.NoError(t, err)
	assert.Equal(t, "content", m.Files["test.txt"])
}

func TestClient_MockMedium_Delete_Good(t *testing.T) {
	m := NewMockMedium()
	m.Files["test.txt"] = "content"

	err := m.Delete("test.txt")
	assert.NoError(t, err)
	assert.False(t, m.IsFile("test.txt"))
}

func TestClient_MockMedium_Delete_NotFound_Bad(t *testing.T) {
	m := NewMockMedium()
	err := m.Delete("nonexistent.txt")
	assert.Error(t, err)
}

func TestClient_MockMedium_Delete_DirNotEmpty_Bad(t *testing.T) {
	m := NewMockMedium()
	m.Dirs["mydir"] = true
	m.Files["mydir/file.txt"] = "content"

	err := m.Delete("mydir")
	assert.Error(t, err)
}

func TestClient_MockMedium_DeleteAll_Good(t *testing.T) {
	m := NewMockMedium()
	m.Dirs["mydir"] = true
	m.Dirs["mydir/subdir"] = true
	m.Files["mydir/file.txt"] = "content"
	m.Files["mydir/subdir/nested.txt"] = "nested"

	err := m.DeleteAll("mydir")
	assert.NoError(t, err)
	assert.Empty(t, m.Dirs)
	assert.Empty(t, m.Files)
}

func TestClient_MockMedium_Rename_Good(t *testing.T) {
	m := NewMockMedium()
	m.Files["old.txt"] = "content"

	err := m.Rename("old.txt", "new.txt")
	assert.NoError(t, err)
	assert.False(t, m.IsFile("old.txt"))
	assert.True(t, m.IsFile("new.txt"))
	assert.Equal(t, "content", m.Files["new.txt"])
}

func TestClient_MockMedium_Rename_Dir_Good(t *testing.T) {
	m := NewMockMedium()
	m.Dirs["olddir"] = true
	m.Files["olddir/file.txt"] = "content"

	err := m.Rename("olddir", "newdir")
	assert.NoError(t, err)
	assert.False(t, m.Dirs["olddir"])
	assert.True(t, m.Dirs["newdir"])
	assert.Equal(t, "content", m.Files["newdir/file.txt"])
}

func TestClient_MockMedium_List_Good(t *testing.T) {
	m := NewMockMedium()
	m.Dirs["mydir"] = true
	m.Files["mydir/file1.txt"] = "content1"
	m.Files["mydir/file2.txt"] = "content2"
	m.Dirs["mydir/subdir"] = true

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

func TestClient_MockMedium_Stat_Good(t *testing.T) {
	m := NewMockMedium()
	m.Files["test.txt"] = "hello world"

	info, err := m.Stat("test.txt")
	assert.NoError(t, err)
	assert.Equal(t, "test.txt", info.Name())
	assert.Equal(t, int64(11), info.Size())
	assert.False(t, info.IsDir())
}

func TestClient_MockMedium_Stat_Dir_Good(t *testing.T) {
	m := NewMockMedium()
	m.Dirs["mydir"] = true

	info, err := m.Stat("mydir")
	assert.NoError(t, err)
	assert.Equal(t, "mydir", info.Name())
	assert.True(t, info.IsDir())
}

func TestClient_MockMedium_Exists_Good(t *testing.T) {
	m := NewMockMedium()
	m.Files["file.txt"] = "content"
	m.Dirs["mydir"] = true

	assert.True(t, m.Exists("file.txt"))
	assert.True(t, m.Exists("mydir"))
	assert.False(t, m.Exists("nonexistent"))
}

func TestClient_MockMedium_IsDir_Good(t *testing.T) {
	m := NewMockMedium()
	m.Files["file.txt"] = "content"
	m.Dirs["mydir"] = true

	assert.False(t, m.IsDir("file.txt"))
	assert.True(t, m.IsDir("mydir"))
	assert.False(t, m.IsDir("nonexistent"))
}

// --- Wrapper Function Tests ---

func TestClient_Read_Good(t *testing.T) {
	m := NewMockMedium()
	m.Files["test.txt"] = "hello"
	content, err := Read(m, "test.txt")
	assert.NoError(t, err)
	assert.Equal(t, "hello", content)
}

func TestClient_Write_Good(t *testing.T) {
	m := NewMockMedium()
	err := Write(m, "test.txt", "hello")
	assert.NoError(t, err)
	assert.Equal(t, "hello", m.Files["test.txt"])
}

func TestClient_EnsureDir_Good(t *testing.T) {
	m := NewMockMedium()
	err := EnsureDir(m, "/my/dir")
	assert.NoError(t, err)
	assert.True(t, m.Dirs["/my/dir"])
}

func TestClient_IsFile_Good(t *testing.T) {
	m := NewMockMedium()
	m.Files["exists.txt"] = "content"

	assert.True(t, IsFile(m, "exists.txt"))
	assert.False(t, IsFile(m, "nonexistent.txt"))
}

func TestClient_Copy_Good(t *testing.T) {
	source := NewMockMedium()
	dest := NewMockMedium()
	source.Files["test.txt"] = "hello"
	err := Copy(source, "test.txt", dest, "test.txt")
	assert.NoError(t, err)
	assert.Equal(t, "hello", dest.Files["test.txt"])

	// Copy to different path
	source.Files["original.txt"] = "content"
	err = Copy(source, "original.txt", dest, "copied.txt")
	assert.NoError(t, err)
	assert.Equal(t, "content", dest.Files["copied.txt"])
}

func TestClient_Copy_Bad(t *testing.T) {
	source := NewMockMedium()
	dest := NewMockMedium()
	err := Copy(source, "nonexistent.txt", dest, "dest.txt")
	assert.Error(t, err)
}

// --- Local Global Tests ---

func TestClient_LocalGlobal_Good(t *testing.T) {
	// io.Local should be initialised by init()
	assert.NotNil(t, Local, "io.Local should be initialised")

	// Should be able to use it as a Medium
	var m = Local
	assert.NotNil(t, m)
}
