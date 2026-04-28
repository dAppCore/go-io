package datanode

import (
	goio "io"
	"io/fs"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
)

var _ coreio.Medium = (*Medium)(nil)

func TestDataNode_ReadWrite_Good(t *core.T) {
	dataNodeMedium := New()

	err := dataNodeMedium.Write("hello.txt", "world")
	core.RequireNoError(t, err)

	got, err := dataNodeMedium.Read("hello.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "world", got)
}

func TestDataNode_ReadWrite_Bad(t *core.T) {
	dataNodeMedium := New()

	_, err := dataNodeMedium.Read("missing.txt")
	core.AssertError(t, err)

	err = dataNodeMedium.Write("", "content")
	core.AssertError(t, err)
}

func TestDataNode_NestedPaths_Good(t *core.T) {
	dataNodeMedium := New()

	core.RequireNoError(t, dataNodeMedium.Write("a/b/c/deep.txt", "deep"))

	got, err := dataNodeMedium.Read("a/b/c/deep.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "deep", got)

	core.AssertTrue(t, dataNodeMedium.IsDir("a"))
	core.AssertTrue(t, dataNodeMedium.IsDir("a/b"))
	core.AssertTrue(t, dataNodeMedium.IsDir("a/b/c"))
}

func TestDataNode_LeadingSlash_Good(t *core.T) {
	dataNodeMedium := New()

	core.RequireNoError(t, dataNodeMedium.Write("/leading/file.txt", "stripped"))
	got, err := dataNodeMedium.Read("leading/file.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "stripped", got)

	got, err = dataNodeMedium.Read("/leading/file.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "stripped", got)
}

func TestDataNode_IsFile_Good(t *core.T) {
	dataNodeMedium := New()

	core.RequireNoError(t, dataNodeMedium.Write("file.go", "package main"))

	core.AssertTrue(t, dataNodeMedium.IsFile("file.go"))
	core.AssertFalse(t, dataNodeMedium.IsFile("missing.go"))
	core.AssertFalse(t, dataNodeMedium.IsFile(""))
}

func TestDataNode_EnsureDir_Good(t *core.T) {
	dataNodeMedium := New()

	core.RequireNoError(t, dataNodeMedium.EnsureDir("foo/bar/baz"))

	core.AssertTrue(t, dataNodeMedium.IsDir("foo"))
	core.AssertTrue(t, dataNodeMedium.IsDir("foo/bar"))
	core.AssertTrue(t, dataNodeMedium.IsDir("foo/bar/baz"))
	core.AssertTrue(t, dataNodeMedium.Exists("foo/bar/baz"))
}

func TestDataNode_Delete_Good(t *core.T) {
	dataNodeMedium := New()

	core.RequireNoError(t, dataNodeMedium.Write("delete-me.txt", "bye"))
	core.AssertTrue(t, dataNodeMedium.Exists("delete-me.txt"))

	core.RequireNoError(t, dataNodeMedium.Delete("delete-me.txt"))
	core.AssertFalse(t, dataNodeMedium.Exists("delete-me.txt"))
}

func TestDataNode_Delete_Bad(t *core.T) {
	dataNodeMedium := New()

	core.AssertError(t, dataNodeMedium.Delete("ghost.txt"))

	core.RequireNoError(t, dataNodeMedium.Write("dir/file.txt", "content"))
	core.AssertError(t, dataNodeMedium.Delete("dir"))
}

func TestDataNode_Delete_DirectoryInspectionFailure_Bad(t *core.T) {
	dataNodeMedium := New()
	core.RequireNoError(t, dataNodeMedium.Write("dir/file.txt", "content"))

	original := dataNodeWalkDir
	dataNodeWalkDir = func(_ fs.FS, _ string, _ fs.WalkDirFunc) error {
		return core.NewError("walk failed")
	}
	t.Cleanup(func() {
		dataNodeWalkDir = original
	})

	err := dataNodeMedium.Delete("dir")
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "failed to inspect directory")
}

func TestDataNode_DeleteAll_Good(t *core.T) {
	dataNodeMedium := New()

	core.RequireNoError(t, dataNodeMedium.Write("tree/a.txt", "a"))
	core.RequireNoError(t, dataNodeMedium.Write("tree/sub/b.txt", "b"))
	core.RequireNoError(t, dataNodeMedium.Write("keep.txt", "keep"))

	core.RequireNoError(t, dataNodeMedium.DeleteAll("tree"))

	core.AssertFalse(t, dataNodeMedium.Exists("tree/a.txt"))
	core.AssertFalse(t, dataNodeMedium.Exists("tree/sub/b.txt"))
	core.AssertTrue(t, dataNodeMedium.Exists("keep.txt"))
}

func TestDataNode_DeleteAll_WalkFailure_Bad(t *core.T) {
	dataNodeMedium := New()
	core.RequireNoError(t, dataNodeMedium.Write("tree/a.txt", "a"))

	original := dataNodeWalkDir
	dataNodeWalkDir = func(_ fs.FS, _ string, _ fs.WalkDirFunc) error {
		return core.NewError("walk failed")
	}
	t.Cleanup(func() {
		dataNodeWalkDir = original
	})

	err := dataNodeMedium.DeleteAll("tree")
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "failed to inspect tree")
}

func TestDataNode_Delete_RemoveFailure_Bad(t *core.T) {
	dataNodeMedium := New()
	core.RequireNoError(t, dataNodeMedium.Write("keep.txt", "keep"))
	core.RequireNoError(t, dataNodeMedium.Write("bad.txt", "bad"))

	original := dataNodeReadAll
	dataNodeReadAll = func(_ goio.Reader) ([]byte, error) {
		return nil, core.NewError("read failed")
	}
	t.Cleanup(func() {
		dataNodeReadAll = original
	})

	err := dataNodeMedium.Delete("bad.txt")
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "failed to delete file")
}

func TestDataNode_Rename_Good(t *core.T) {
	dataNodeMedium := New()

	core.RequireNoError(t, dataNodeMedium.Write("old.txt", "content"))
	core.RequireNoError(t, dataNodeMedium.Rename("old.txt", "new.txt"))

	core.AssertFalse(t, dataNodeMedium.Exists("old.txt"))
	got, err := dataNodeMedium.Read("new.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "content", got)
}

func TestDataNode_RenameDir_Good(t *core.T) {
	dataNodeMedium := New()

	core.RequireNoError(t, dataNodeMedium.Write("src/a.go", "package a"))
	core.RequireNoError(t, dataNodeMedium.Write("src/sub/b.go", "package b"))

	core.RequireNoError(t, dataNodeMedium.Rename("src", "destination"))

	core.AssertFalse(t, dataNodeMedium.Exists("src/a.go"))

	got, err := dataNodeMedium.Read("destination/a.go")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "package a", got)

	got, err = dataNodeMedium.Read("destination/sub/b.go")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "package b", got)
}

func TestDataNode_RenameDir_ReadFailure_Bad(t *core.T) {
	dataNodeMedium := New()
	core.RequireNoError(t, dataNodeMedium.Write("src/a.go", "package a"))

	original := dataNodeReadAll
	dataNodeReadAll = func(_ goio.Reader) ([]byte, error) {
		return nil, core.NewError("read failed")
	}
	t.Cleanup(func() {
		dataNodeReadAll = original
	})

	err := dataNodeMedium.Rename("src", "destination")
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "failed to read source file")
}

func TestDataNode_List_Good(t *core.T) {
	dataNodeMedium := New()

	core.RequireNoError(t, dataNodeMedium.Write("root.txt", "r"))
	core.RequireNoError(t, dataNodeMedium.Write("pkg/a.go", "a"))
	core.RequireNoError(t, dataNodeMedium.Write("pkg/b.go", "b"))
	core.RequireNoError(t, dataNodeMedium.Write("pkg/sub/c.go", "c"))

	entries, err := dataNodeMedium.List("")
	core.RequireNoError(t, err)

	names := make([]string, len(entries))
	for index, entry := range entries {
		names[index] = entry.Name()
	}
	core.AssertContains(t, names, "root.txt")
	core.AssertContains(t, names, "pkg")

	entries, err = dataNodeMedium.List("pkg")
	core.RequireNoError(t, err)
	names = make([]string, len(entries))
	for index, entry := range entries {
		names[index] = entry.Name()
	}
	core.AssertContains(t, names, "a.go")
	core.AssertContains(t, names, "b.go")
	core.AssertContains(t, names, "sub")
}

func TestDataNode_Stat_Good(t *core.T) {
	dataNodeMedium := New()

	core.RequireNoError(t, dataNodeMedium.Write("stat.txt", "hello"))

	info, err := dataNodeMedium.Stat("stat.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, int64(5), info.Size())
	core.AssertFalse(t, info.IsDir())

	info, err = dataNodeMedium.Stat("")
	core.RequireNoError(t, err)
	core.AssertTrue(t, info.IsDir())
}

func TestDataNode_Open_Good(t *core.T) {
	dataNodeMedium := New()

	core.RequireNoError(t, dataNodeMedium.Write("open.txt", "opened"))

	file, err := dataNodeMedium.Open("open.txt")
	core.RequireNoError(t, err)
	defer file.Close()

	data, err := goio.ReadAll(file)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "opened", string(data))
}

func TestDataNode_CreateAppend_Good(t *core.T) {
	dataNodeMedium := New()

	writer, err := dataNodeMedium.Create("new.txt")
	core.RequireNoError(t, err)
	_, _ = writer.Write([]byte("hello"))
	core.RequireNoError(t, writer.Close())

	got, err := dataNodeMedium.Read("new.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "hello", got)

	writer, err = dataNodeMedium.Append("new.txt")
	core.RequireNoError(t, err)
	_, _ = writer.Write([]byte(" world"))
	core.RequireNoError(t, writer.Close())

	got, err = dataNodeMedium.Read("new.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "hello world", got)
}

func TestDataNode_Append_ReadFailure_Bad(t *core.T) {
	dataNodeMedium := New()
	core.RequireNoError(t, dataNodeMedium.Write("new.txt", "hello"))

	original := dataNodeReadAll
	dataNodeReadAll = func(_ goio.Reader) ([]byte, error) {
		return nil, core.NewError("read failed")
	}
	t.Cleanup(func() {
		dataNodeReadAll = original
	})

	_, err := dataNodeMedium.Append("new.txt")
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "failed to read existing content")
}

func TestDataNode_Streams_Good(t *core.T) {
	dataNodeMedium := New()

	writeStream, err := dataNodeMedium.WriteStream("stream.txt")
	core.RequireNoError(t, err)
	_, _ = writeStream.Write([]byte("streamed"))
	core.RequireNoError(t, writeStream.Close())

	readStream, err := dataNodeMedium.ReadStream("stream.txt")
	core.RequireNoError(t, err)
	data, err := goio.ReadAll(readStream)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "streamed", string(data))
	core.RequireNoError(t, readStream.Close())
}

func TestDataNode_SnapshotRestore_Good(t *core.T) {
	dataNodeMedium := New()

	core.RequireNoError(t, dataNodeMedium.Write("a.txt", "alpha"))
	core.RequireNoError(t, dataNodeMedium.Write("b/c.txt", "charlie"))

	snapshotData, err := dataNodeMedium.Snapshot()
	core.RequireNoError(t, err)
	core.AssertNotEmpty(t, snapshotData)

	restoredNode, err := FromTar(snapshotData)
	core.RequireNoError(t, err)

	got, err := restoredNode.Read("a.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "alpha", got)

	got, err = restoredNode.Read("b/c.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "charlie", got)
}

func TestDataNode_Restore_Good(t *core.T) {
	dataNodeMedium := New()

	core.RequireNoError(t, dataNodeMedium.Write("original.txt", "before"))

	snapshotData, err := dataNodeMedium.Snapshot()
	core.RequireNoError(t, err)

	core.RequireNoError(t, dataNodeMedium.Write("original.txt", "after"))
	core.RequireNoError(t, dataNodeMedium.Write("extra.txt", "extra"))

	core.RequireNoError(t, dataNodeMedium.Restore(snapshotData))

	got, err := dataNodeMedium.Read("original.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "before", got)

	core.AssertFalse(t, dataNodeMedium.Exists("extra.txt"))
}

func TestDataNode_DataNode_Good(t *core.T) {
	dataNodeMedium := New()

	core.RequireNoError(t, dataNodeMedium.Write("test.txt", "borg"))

	dataNode := dataNodeMedium.DataNode()
	core.AssertNotNil(t, dataNode)

	file, err := dataNode.Open("test.txt")
	core.RequireNoError(t, err)
	defer file.Close()

	data, err := goio.ReadAll(file)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "borg", string(data))
}

func TestDataNode_Overwrite_Good(t *core.T) {
	dataNodeMedium := New()

	core.RequireNoError(t, dataNodeMedium.Write("file.txt", "v1"))
	core.RequireNoError(t, dataNodeMedium.Write("file.txt", "v2"))

	got, err := dataNodeMedium.Read("file.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "v2", got)
}

func TestDataNode_Exists_Good(t *core.T) {
	dataNodeMedium := New()

	core.AssertTrue(t, dataNodeMedium.Exists(""))
	core.AssertFalse(t, dataNodeMedium.Exists("x"))

	core.RequireNoError(t, dataNodeMedium.Write("x", "y"))
	core.AssertTrue(t, dataNodeMedium.Exists("x"))
}

func TestDataNode_ReadExistingFile_Good(t *core.T) {
	dataNodeMedium := New()

	core.RequireNoError(t, dataNodeMedium.Write("file.txt", "content"))
	got, err := dataNodeMedium.Read("file.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "content", got)
}
