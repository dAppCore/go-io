package datanode

import (
	"io"
	"io/fs"
	"testing"

	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ coreio.Medium = (*Medium)(nil)

func TestClient_ReadWrite_Good(t *testing.T) {
	m := New()

	err := m.Write("hello.txt", "world")
	require.NoError(t, err)

	got, err := m.Read("hello.txt")
	require.NoError(t, err)
	assert.Equal(t, "world", got)
}

func TestClient_ReadWrite_Bad(t *testing.T) {
	m := New()

	_, err := m.Read("missing.txt")
	assert.Error(t, err)

	err = m.Write("", "content")
	assert.Error(t, err)
}

func TestClient_NestedPaths_Good(t *testing.T) {
	m := New()

	require.NoError(t, m.Write("a/b/c/deep.txt", "deep"))

	got, err := m.Read("a/b/c/deep.txt")
	require.NoError(t, err)
	assert.Equal(t, "deep", got)

	assert.True(t, m.IsDir("a"))
	assert.True(t, m.IsDir("a/b"))
	assert.True(t, m.IsDir("a/b/c"))
}

func TestClient_LeadingSlash_Good(t *testing.T) {
	m := New()

	require.NoError(t, m.Write("/leading/file.txt", "stripped"))
	got, err := m.Read("leading/file.txt")
	require.NoError(t, err)
	assert.Equal(t, "stripped", got)

	got, err = m.Read("/leading/file.txt")
	require.NoError(t, err)
	assert.Equal(t, "stripped", got)
}

func TestClient_IsFile_Good(t *testing.T) {
	m := New()

	require.NoError(t, m.Write("file.go", "package main"))

	assert.True(t, m.IsFile("file.go"))
	assert.False(t, m.IsFile("missing.go"))
	assert.False(t, m.IsFile(""))
}

func TestClient_EnsureDir_Good(t *testing.T) {
	m := New()

	require.NoError(t, m.EnsureDir("foo/bar/baz"))

	assert.True(t, m.IsDir("foo"))
	assert.True(t, m.IsDir("foo/bar"))
	assert.True(t, m.IsDir("foo/bar/baz"))
	assert.True(t, m.Exists("foo/bar/baz"))
}

func TestClient_Delete_Good(t *testing.T) {
	m := New()

	require.NoError(t, m.Write("delete-me.txt", "bye"))
	assert.True(t, m.Exists("delete-me.txt"))

	require.NoError(t, m.Delete("delete-me.txt"))
	assert.False(t, m.Exists("delete-me.txt"))
}

func TestClient_Delete_Bad(t *testing.T) {
	medium := New()

	assert.Error(t, medium.Delete("ghost.txt"))

	require.NoError(t, medium.Write("dir/file.txt", "content"))
	assert.Error(t, medium.Delete("dir"))
}

func TestClient_Delete_DirectoryInspectionFailure_Bad(t *testing.T) {
	m := New()
	require.NoError(t, m.Write("dir/file.txt", "content"))

	original := dataNodeWalkDir
	dataNodeWalkDir = func(_ fs.FS, _ string, _ fs.WalkDirFunc) error {
		return core.NewError("walk failed")
	}
	t.Cleanup(func() {
		dataNodeWalkDir = original
	})

	err := m.Delete("dir")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to inspect directory")
}

func TestClient_DeleteAll_Good(t *testing.T) {
	m := New()

	require.NoError(t, m.Write("tree/a.txt", "a"))
	require.NoError(t, m.Write("tree/sub/b.txt", "b"))
	require.NoError(t, m.Write("keep.txt", "keep"))

	require.NoError(t, m.DeleteAll("tree"))

	assert.False(t, m.Exists("tree/a.txt"))
	assert.False(t, m.Exists("tree/sub/b.txt"))
	assert.True(t, m.Exists("keep.txt"))
}

func TestClient_DeleteAll_WalkFailure_Bad(t *testing.T) {
	m := New()
	require.NoError(t, m.Write("tree/a.txt", "a"))

	original := dataNodeWalkDir
	dataNodeWalkDir = func(_ fs.FS, _ string, _ fs.WalkDirFunc) error {
		return core.NewError("walk failed")
	}
	t.Cleanup(func() {
		dataNodeWalkDir = original
	})

	err := m.DeleteAll("tree")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to inspect tree")
}

func TestClient_Delete_RemoveFailure_Bad(t *testing.T) {
	m := New()
	require.NoError(t, m.Write("keep.txt", "keep"))
	require.NoError(t, m.Write("bad.txt", "bad"))

	original := dataNodeReadAll
	dataNodeReadAll = func(_ io.Reader) ([]byte, error) {
		return nil, core.NewError("read failed")
	}
	t.Cleanup(func() {
		dataNodeReadAll = original
	})

	err := m.Delete("bad.txt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete file")
}

func TestClient_Rename_Good(t *testing.T) {
	m := New()

	require.NoError(t, m.Write("old.txt", "content"))
	require.NoError(t, m.Rename("old.txt", "new.txt"))

	assert.False(t, m.Exists("old.txt"))
	got, err := m.Read("new.txt")
	require.NoError(t, err)
	assert.Equal(t, "content", got)
}

func TestClient_RenameDir_Good(t *testing.T) {
	m := New()

	require.NoError(t, m.Write("src/a.go", "package a"))
	require.NoError(t, m.Write("src/sub/b.go", "package b"))

	require.NoError(t, m.Rename("src", "dst"))

	assert.False(t, m.Exists("src/a.go"))

	got, err := m.Read("dst/a.go")
	require.NoError(t, err)
	assert.Equal(t, "package a", got)

	got, err = m.Read("dst/sub/b.go")
	require.NoError(t, err)
	assert.Equal(t, "package b", got)
}

func TestClient_RenameDir_ReadFailure_Bad(t *testing.T) {
	m := New()
	require.NoError(t, m.Write("src/a.go", "package a"))

	original := dataNodeReadAll
	dataNodeReadAll = func(_ io.Reader) ([]byte, error) {
		return nil, core.NewError("read failed")
	}
	t.Cleanup(func() {
		dataNodeReadAll = original
	})

	err := m.Rename("src", "dst")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read source file")
}

func TestClient_List_Good(t *testing.T) {
	m := New()

	require.NoError(t, m.Write("root.txt", "r"))
	require.NoError(t, m.Write("pkg/a.go", "a"))
	require.NoError(t, m.Write("pkg/b.go", "b"))
	require.NoError(t, m.Write("pkg/sub/c.go", "c"))

	entries, err := m.List("")
	require.NoError(t, err)

	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name()
	}
	assert.Contains(t, names, "root.txt")
	assert.Contains(t, names, "pkg")

	entries, err = m.List("pkg")
	require.NoError(t, err)
	names = make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name()
	}
	assert.Contains(t, names, "a.go")
	assert.Contains(t, names, "b.go")
	assert.Contains(t, names, "sub")
}

func TestClient_Stat_Good(t *testing.T) {
	m := New()

	require.NoError(t, m.Write("stat.txt", "hello"))

	info, err := m.Stat("stat.txt")
	require.NoError(t, err)
	assert.Equal(t, int64(5), info.Size())
	assert.False(t, info.IsDir())

	info, err = m.Stat("")
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestClient_Open_Good(t *testing.T) {
	m := New()

	require.NoError(t, m.Write("open.txt", "opened"))

	f, err := m.Open("open.txt")
	require.NoError(t, err)
	defer f.Close()

	data, err := io.ReadAll(f)
	require.NoError(t, err)
	assert.Equal(t, "opened", string(data))
}

func TestClient_CreateAppend_Good(t *testing.T) {
	m := New()

	w, err := m.Create("new.txt")
	require.NoError(t, err)
	w.Write([]byte("hello"))
	w.Close()

	got, err := m.Read("new.txt")
	require.NoError(t, err)
	assert.Equal(t, "hello", got)

	w, err = m.Append("new.txt")
	require.NoError(t, err)
	w.Write([]byte(" world"))
	w.Close()

	got, err = m.Read("new.txt")
	require.NoError(t, err)
	assert.Equal(t, "hello world", got)
}

func TestClient_Append_ReadFailure_Bad(t *testing.T) {
	m := New()
	require.NoError(t, m.Write("new.txt", "hello"))

	original := dataNodeReadAll
	dataNodeReadAll = func(_ io.Reader) ([]byte, error) {
		return nil, core.NewError("read failed")
	}
	t.Cleanup(func() {
		dataNodeReadAll = original
	})

	_, err := m.Append("new.txt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read existing content")
}

func TestClient_Streams_Good(t *testing.T) {
	m := New()

	ws, err := m.WriteStream("stream.txt")
	require.NoError(t, err)
	ws.Write([]byte("streamed"))
	ws.Close()

	rs, err := m.ReadStream("stream.txt")
	require.NoError(t, err)
	data, err := io.ReadAll(rs)
	require.NoError(t, err)
	assert.Equal(t, "streamed", string(data))
	rs.Close()
}

func TestClient_FileGetFileSet_Good(t *testing.T) {
	m := New()

	require.NoError(t, m.FileSet("alias.txt", "via set"))

	got, err := m.FileGet("alias.txt")
	require.NoError(t, err)
	assert.Equal(t, "via set", got)
}

func TestClient_SnapshotRestore_Good(t *testing.T) {
	m := New()

	require.NoError(t, m.Write("a.txt", "alpha"))
	require.NoError(t, m.Write("b/c.txt", "charlie"))

	snap, err := m.Snapshot()
	require.NoError(t, err)
	assert.NotEmpty(t, snap)

	m2, err := FromTar(snap)
	require.NoError(t, err)

	got, err := m2.Read("a.txt")
	require.NoError(t, err)
	assert.Equal(t, "alpha", got)

	got, err = m2.Read("b/c.txt")
	require.NoError(t, err)
	assert.Equal(t, "charlie", got)
}

func TestClient_Restore_Good(t *testing.T) {
	m := New()

	require.NoError(t, m.Write("original.txt", "before"))

	snap, err := m.Snapshot()
	require.NoError(t, err)

	require.NoError(t, m.Write("original.txt", "after"))
	require.NoError(t, m.Write("extra.txt", "extra"))

	require.NoError(t, m.Restore(snap))

	got, err := m.Read("original.txt")
	require.NoError(t, err)
	assert.Equal(t, "before", got)

	assert.False(t, m.Exists("extra.txt"))
}

func TestClient_DataNode_Good(t *testing.T) {
	m := New()

	require.NoError(t, m.Write("test.txt", "borg"))

	dn := m.DataNode()
	assert.NotNil(t, dn)

	f, err := dn.Open("test.txt")
	require.NoError(t, err)
	defer f.Close()

	data, err := io.ReadAll(f)
	require.NoError(t, err)
	assert.Equal(t, "borg", string(data))
}

func TestClient_Overwrite_Good(t *testing.T) {
	m := New()

	require.NoError(t, m.Write("file.txt", "v1"))
	require.NoError(t, m.Write("file.txt", "v2"))

	got, err := m.Read("file.txt")
	require.NoError(t, err)
	assert.Equal(t, "v2", got)
}

func TestClient_Exists_Good(t *testing.T) {
	m := New()

	assert.True(t, m.Exists(""))
	assert.False(t, m.Exists("x"))

	require.NoError(t, m.Write("x", "y"))
	assert.True(t, m.Exists("x"))
}

func TestClient_ReadExistingFile_Good(t *testing.T) {
	m := New()

	require.NoError(t, m.Write("file.txt", "content"))
	got, err := m.Read("file.txt")
	require.NoError(t, err)
	assert.Equal(t, "content", got)
}
