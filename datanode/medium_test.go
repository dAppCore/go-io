package datanode

import (
	goio "io"
	"io/fs"
	"testing"

	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ coreio.Medium = (*Medium)(nil)

func TestDataNode_ReadWrite_Good(t *testing.T) {
	dataNodeMedium := New()

	err := dataNodeMedium.Write("hello.txt", "world")
	require.NoError(t, err)

	got, err := dataNodeMedium.Read("hello.txt")
	require.NoError(t, err)
	assert.Equal(t, "world", got)
}

func TestDataNode_ReadWrite_Bad(t *testing.T) {
	dataNodeMedium := New()

	_, err := dataNodeMedium.Read("missing.txt")
	assert.Error(t, err)

	err = dataNodeMedium.Write("", "content")
	assert.Error(t, err)
}

func TestDataNode_NestedPaths_Good(t *testing.T) {
	dataNodeMedium := New()

	require.NoError(t, dataNodeMedium.Write("a/b/c/deep.txt", "deep"))

	got, err := dataNodeMedium.Read("a/b/c/deep.txt")
	require.NoError(t, err)
	assert.Equal(t, "deep", got)

	assert.True(t, dataNodeMedium.IsDir("a"))
	assert.True(t, dataNodeMedium.IsDir("a/b"))
	assert.True(t, dataNodeMedium.IsDir("a/b/c"))
}

func TestDataNode_LeadingSlash_Good(t *testing.T) {
	dataNodeMedium := New()

	require.NoError(t, dataNodeMedium.Write("/leading/file.txt", "stripped"))
	got, err := dataNodeMedium.Read("leading/file.txt")
	require.NoError(t, err)
	assert.Equal(t, "stripped", got)

	got, err = dataNodeMedium.Read("/leading/file.txt")
	require.NoError(t, err)
	assert.Equal(t, "stripped", got)
}

func TestDataNode_IsFile_Good(t *testing.T) {
	dataNodeMedium := New()

	require.NoError(t, dataNodeMedium.Write("file.go", "package main"))

	assert.True(t, dataNodeMedium.IsFile("file.go"))
	assert.False(t, dataNodeMedium.IsFile("missing.go"))
	assert.False(t, dataNodeMedium.IsFile(""))
}

func TestDataNode_EnsureDir_Good(t *testing.T) {
	dataNodeMedium := New()

	require.NoError(t, dataNodeMedium.EnsureDir("foo/bar/baz"))

	assert.True(t, dataNodeMedium.IsDir("foo"))
	assert.True(t, dataNodeMedium.IsDir("foo/bar"))
	assert.True(t, dataNodeMedium.IsDir("foo/bar/baz"))
	assert.True(t, dataNodeMedium.Exists("foo/bar/baz"))
}

func TestDataNode_Delete_Good(t *testing.T) {
	dataNodeMedium := New()

	require.NoError(t, dataNodeMedium.Write("delete-me.txt", "bye"))
	assert.True(t, dataNodeMedium.Exists("delete-me.txt"))

	require.NoError(t, dataNodeMedium.Delete("delete-me.txt"))
	assert.False(t, dataNodeMedium.Exists("delete-me.txt"))
}

func TestDataNode_Delete_Bad(t *testing.T) {
	dataNodeMedium := New()

	assert.Error(t, dataNodeMedium.Delete("ghost.txt"))

	require.NoError(t, dataNodeMedium.Write("dir/file.txt", "content"))
	assert.Error(t, dataNodeMedium.Delete("dir"))
}

func TestDataNode_Delete_DirectoryInspectionFailure_Bad(t *testing.T) {
	dataNodeMedium := New()
	require.NoError(t, dataNodeMedium.Write("dir/file.txt", "content"))

	original := dataNodeWalkDir
	dataNodeWalkDir = func(_ fs.FS, _ string, _ fs.WalkDirFunc) error {
		return core.NewError("walk failed")
	}
	t.Cleanup(func() {
		dataNodeWalkDir = original
	})

	err := dataNodeMedium.Delete("dir")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to inspect directory")
}

func TestDataNode_DeleteAll_Good(t *testing.T) {
	dataNodeMedium := New()

	require.NoError(t, dataNodeMedium.Write("tree/a.txt", "a"))
	require.NoError(t, dataNodeMedium.Write("tree/sub/b.txt", "b"))
	require.NoError(t, dataNodeMedium.Write("keep.txt", "keep"))

	require.NoError(t, dataNodeMedium.DeleteAll("tree"))

	assert.False(t, dataNodeMedium.Exists("tree/a.txt"))
	assert.False(t, dataNodeMedium.Exists("tree/sub/b.txt"))
	assert.True(t, dataNodeMedium.Exists("keep.txt"))
}

func TestDataNode_DeleteAll_WalkFailure_Bad(t *testing.T) {
	dataNodeMedium := New()
	require.NoError(t, dataNodeMedium.Write("tree/a.txt", "a"))

	original := dataNodeWalkDir
	dataNodeWalkDir = func(_ fs.FS, _ string, _ fs.WalkDirFunc) error {
		return core.NewError("walk failed")
	}
	t.Cleanup(func() {
		dataNodeWalkDir = original
	})

	err := dataNodeMedium.DeleteAll("tree")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to inspect tree")
}

func TestDataNode_Delete_RemoveFailure_Bad(t *testing.T) {
	dataNodeMedium := New()
	require.NoError(t, dataNodeMedium.Write("keep.txt", "keep"))
	require.NoError(t, dataNodeMedium.Write("bad.txt", "bad"))

	original := dataNodeReadAll
	dataNodeReadAll = func(_ goio.Reader) ([]byte, error) {
		return nil, core.NewError("read failed")
	}
	t.Cleanup(func() {
		dataNodeReadAll = original
	})

	err := dataNodeMedium.Delete("bad.txt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete file")
}

func TestDataNode_Rename_Good(t *testing.T) {
	dataNodeMedium := New()

	require.NoError(t, dataNodeMedium.Write("old.txt", "content"))
	require.NoError(t, dataNodeMedium.Rename("old.txt", "new.txt"))

	assert.False(t, dataNodeMedium.Exists("old.txt"))
	got, err := dataNodeMedium.Read("new.txt")
	require.NoError(t, err)
	assert.Equal(t, "content", got)
}

func TestDataNode_RenameDir_Good(t *testing.T) {
	dataNodeMedium := New()

	require.NoError(t, dataNodeMedium.Write("src/a.go", "package a"))
	require.NoError(t, dataNodeMedium.Write("src/sub/b.go", "package b"))

	require.NoError(t, dataNodeMedium.Rename("src", "destination"))

	assert.False(t, dataNodeMedium.Exists("src/a.go"))

	got, err := dataNodeMedium.Read("destination/a.go")
	require.NoError(t, err)
	assert.Equal(t, "package a", got)

	got, err = dataNodeMedium.Read("destination/sub/b.go")
	require.NoError(t, err)
	assert.Equal(t, "package b", got)
}

func TestDataNode_RenameDir_ReadFailure_Bad(t *testing.T) {
	dataNodeMedium := New()
	require.NoError(t, dataNodeMedium.Write("src/a.go", "package a"))

	original := dataNodeReadAll
	dataNodeReadAll = func(_ goio.Reader) ([]byte, error) {
		return nil, core.NewError("read failed")
	}
	t.Cleanup(func() {
		dataNodeReadAll = original
	})

	err := dataNodeMedium.Rename("src", "destination")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read source file")
}

func TestDataNode_List_Good(t *testing.T) {
	dataNodeMedium := New()

	require.NoError(t, dataNodeMedium.Write("root.txt", "r"))
	require.NoError(t, dataNodeMedium.Write("pkg/a.go", "a"))
	require.NoError(t, dataNodeMedium.Write("pkg/b.go", "b"))
	require.NoError(t, dataNodeMedium.Write("pkg/sub/c.go", "c"))

	entries, err := dataNodeMedium.List("")
	require.NoError(t, err)

	names := make([]string, len(entries))
	for index, entry := range entries {
		names[index] = entry.Name()
	}
	assert.Contains(t, names, "root.txt")
	assert.Contains(t, names, "pkg")

	entries, err = dataNodeMedium.List("pkg")
	require.NoError(t, err)
	names = make([]string, len(entries))
	for index, entry := range entries {
		names[index] = entry.Name()
	}
	assert.Contains(t, names, "a.go")
	assert.Contains(t, names, "b.go")
	assert.Contains(t, names, "sub")
}

func TestDataNode_Stat_Good(t *testing.T) {
	dataNodeMedium := New()

	require.NoError(t, dataNodeMedium.Write("stat.txt", "hello"))

	info, err := dataNodeMedium.Stat("stat.txt")
	require.NoError(t, err)
	assert.Equal(t, int64(5), info.Size())
	assert.False(t, info.IsDir())

	info, err = dataNodeMedium.Stat("")
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestDataNode_Open_Good(t *testing.T) {
	dataNodeMedium := New()

	require.NoError(t, dataNodeMedium.Write("open.txt", "opened"))

	file, err := dataNodeMedium.Open("open.txt")
	require.NoError(t, err)
	defer file.Close()

	data, err := goio.ReadAll(file)
	require.NoError(t, err)
	assert.Equal(t, "opened", string(data))
}

func TestDataNode_CreateAppend_Good(t *testing.T) {
	dataNodeMedium := New()

	writer, err := dataNodeMedium.Create("new.txt")
	require.NoError(t, err)
	_, _ = writer.Write([]byte("hello"))
	require.NoError(t, writer.Close())

	got, err := dataNodeMedium.Read("new.txt")
	require.NoError(t, err)
	assert.Equal(t, "hello", got)

	writer, err = dataNodeMedium.Append("new.txt")
	require.NoError(t, err)
	_, _ = writer.Write([]byte(" world"))
	require.NoError(t, writer.Close())

	got, err = dataNodeMedium.Read("new.txt")
	require.NoError(t, err)
	assert.Equal(t, "hello world", got)
}

func TestDataNode_Append_ReadFailure_Bad(t *testing.T) {
	dataNodeMedium := New()
	require.NoError(t, dataNodeMedium.Write("new.txt", "hello"))

	original := dataNodeReadAll
	dataNodeReadAll = func(_ goio.Reader) ([]byte, error) {
		return nil, core.NewError("read failed")
	}
	t.Cleanup(func() {
		dataNodeReadAll = original
	})

	_, err := dataNodeMedium.Append("new.txt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read existing content")
}

func TestDataNode_Streams_Good(t *testing.T) {
	dataNodeMedium := New()

	writeStream, err := dataNodeMedium.WriteStream("stream.txt")
	require.NoError(t, err)
	_, _ = writeStream.Write([]byte("streamed"))
	require.NoError(t, writeStream.Close())

	readStream, err := dataNodeMedium.ReadStream("stream.txt")
	require.NoError(t, err)
	data, err := goio.ReadAll(readStream)
	require.NoError(t, err)
	assert.Equal(t, "streamed", string(data))
	require.NoError(t, readStream.Close())
}

func TestDataNode_SnapshotRestore_Good(t *testing.T) {
	dataNodeMedium := New()

	require.NoError(t, dataNodeMedium.Write("a.txt", "alpha"))
	require.NoError(t, dataNodeMedium.Write("b/c.txt", "charlie"))

	snapshotData, err := dataNodeMedium.Snapshot()
	require.NoError(t, err)
	assert.NotEmpty(t, snapshotData)

	restoredNode, err := FromTar(snapshotData)
	require.NoError(t, err)

	got, err := restoredNode.Read("a.txt")
	require.NoError(t, err)
	assert.Equal(t, "alpha", got)

	got, err = restoredNode.Read("b/c.txt")
	require.NoError(t, err)
	assert.Equal(t, "charlie", got)
}

func TestDataNode_Restore_Good(t *testing.T) {
	dataNodeMedium := New()

	require.NoError(t, dataNodeMedium.Write("original.txt", "before"))

	snapshotData, err := dataNodeMedium.Snapshot()
	require.NoError(t, err)

	require.NoError(t, dataNodeMedium.Write("original.txt", "after"))
	require.NoError(t, dataNodeMedium.Write("extra.txt", "extra"))

	require.NoError(t, dataNodeMedium.Restore(snapshotData))

	got, err := dataNodeMedium.Read("original.txt")
	require.NoError(t, err)
	assert.Equal(t, "before", got)

	assert.False(t, dataNodeMedium.Exists("extra.txt"))
}

func TestDataNode_DataNode_Good(t *testing.T) {
	dataNodeMedium := New()

	require.NoError(t, dataNodeMedium.Write("test.txt", "borg"))

	dataNode := dataNodeMedium.DataNode()
	assert.NotNil(t, dataNode)

	file, err := dataNode.Open("test.txt")
	require.NoError(t, err)
	defer file.Close()

	data, err := goio.ReadAll(file)
	require.NoError(t, err)
	assert.Equal(t, "borg", string(data))
}

func TestDataNode_Overwrite_Good(t *testing.T) {
	dataNodeMedium := New()

	require.NoError(t, dataNodeMedium.Write("file.txt", "v1"))
	require.NoError(t, dataNodeMedium.Write("file.txt", "v2"))

	got, err := dataNodeMedium.Read("file.txt")
	require.NoError(t, err)
	assert.Equal(t, "v2", got)
}

func TestDataNode_Exists_Good(t *testing.T) {
	dataNodeMedium := New()

	assert.True(t, dataNodeMedium.Exists(""))
	assert.False(t, dataNodeMedium.Exists("x"))

	require.NoError(t, dataNodeMedium.Write("x", "y"))
	assert.True(t, dataNodeMedium.Exists("x"))
}

func TestDataNode_ReadExistingFile_Good(t *testing.T) {
	dataNodeMedium := New()

	require.NoError(t, dataNodeMedium.Write("file.txt", "content"))
	got, err := dataNodeMedium.Read("file.txt")
	require.NoError(t, err)
	assert.Equal(t, "content", got)
}
