package datanode

import (
	core "dappco.re/go"
	coreio "dappco.re/go/io"
	goio "io"
	"io/fs"
	"time"
)

var _ coreio.Medium = (*Medium)(nil)

const (
	dataNodeNestedPath   = "foo/bar/baz"
	dataNodeDeletePath   = "delete-me.txt"
	dataNodeTreeAPath    = "tree/a.txt"
	dataNodeKeepPath     = "keep.txt"
	dataNodeReadFailed   = "read failed"
	dataNodeOldPath      = "old.txt"
	dataNodeNewPath      = "new.txt"
	dataNodePackageA     = "package a"
	dataNodeSourcePath   = "src/a.go"
	dataNodeOriginalPath = "original.txt"
	dataNodeFilePath     = "file.txt"
)

func TestDataNode_ReadWriteGood(t *core.T) {
	dataNodeMedium := New()

	err := dataNodeMedium.Write("hello.txt", "world")
	core.RequireNoError(t, err)

	got, err := dataNodeMedium.Read("hello.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "world", got)
}

func TestDataNode_ReadWriteBad(t *core.T) {
	dataNodeMedium := New()

	_, err := dataNodeMedium.Read("missing.txt")
	core.AssertError(t, err)

	err = dataNodeMedium.Write("", "content")
	core.AssertError(t, err)
}

func TestDataNode_NestedPathsGood(t *core.T) {
	dataNodeMedium := New()

	core.RequireNoError(t, dataNodeMedium.Write("a/b/c/deep.txt", "deep"))

	got, err := dataNodeMedium.Read("a/b/c/deep.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "deep", got)

	core.AssertTrue(t, dataNodeMedium.IsDir("a"))
	core.AssertTrue(t, dataNodeMedium.IsDir("a/b"))
	core.AssertTrue(t, dataNodeMedium.IsDir("a/b/c"))
}

func TestDataNode_LeadingSlashGood(t *core.T) {
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

	core.RequireNoError(t, dataNodeMedium.EnsureDir(dataNodeNestedPath))

	core.AssertTrue(t, dataNodeMedium.IsDir("foo"))
	core.AssertTrue(t, dataNodeMedium.IsDir("foo/bar"))
	core.AssertTrue(t, dataNodeMedium.IsDir(dataNodeNestedPath))
	core.AssertTrue(t, dataNodeMedium.Exists(dataNodeNestedPath))
}

func TestDataNode_Delete_Good(t *core.T) {
	dataNodeMedium := New()

	core.RequireNoError(t, dataNodeMedium.Write(dataNodeDeletePath, "bye"))
	core.AssertTrue(t, dataNodeMedium.Exists(dataNodeDeletePath))

	core.RequireNoError(t, dataNodeMedium.Delete(dataNodeDeletePath))
	core.AssertFalse(t, dataNodeMedium.Exists(dataNodeDeletePath))
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

	core.RequireNoError(t, dataNodeMedium.Write(dataNodeTreeAPath, "a"))
	core.RequireNoError(t, dataNodeMedium.Write("tree/sub/b.txt", "b"))
	core.RequireNoError(t, dataNodeMedium.Write(dataNodeKeepPath, "keep"))

	core.RequireNoError(t, dataNodeMedium.DeleteAll("tree"))

	core.AssertFalse(t, dataNodeMedium.Exists(dataNodeTreeAPath))
	core.AssertFalse(t, dataNodeMedium.Exists("tree/sub/b.txt"))
	core.AssertTrue(t, dataNodeMedium.Exists(dataNodeKeepPath))
}

func TestDataNode_DeleteAll_WalkFailure_Bad(t *core.T) {
	dataNodeMedium := New()
	core.RequireNoError(t, dataNodeMedium.Write(dataNodeTreeAPath, "a"))

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
	core.RequireNoError(t, dataNodeMedium.Write(dataNodeKeepPath, "keep"))
	core.RequireNoError(t, dataNodeMedium.Write("bad.txt", "bad"))

	original := dataNodeReadAll
	dataNodeReadAll = func(_ goio.Reader) ([]byte, error) {
		return nil, core.NewError(dataNodeReadFailed)
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

	core.RequireNoError(t, dataNodeMedium.Write(dataNodeOldPath, "content"))
	core.RequireNoError(t, dataNodeMedium.Rename(dataNodeOldPath, dataNodeNewPath))

	core.AssertFalse(t, dataNodeMedium.Exists(dataNodeOldPath))
	got, err := dataNodeMedium.Read(dataNodeNewPath)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "content", got)
}

func TestDataNode_RenameDirGood(t *core.T) {
	dataNodeMedium := New()

	core.RequireNoError(t, dataNodeMedium.Write(dataNodeSourcePath, dataNodePackageA))
	core.RequireNoError(t, dataNodeMedium.Write("src/sub/b.go", "package b"))

	core.RequireNoError(t, dataNodeMedium.Rename("src", "destination"))

	core.AssertFalse(t, dataNodeMedium.Exists(dataNodeSourcePath))

	got, err := dataNodeMedium.Read("destination/a.go")
	core.RequireNoError(t, err)
	core.AssertEqual(t, dataNodePackageA, got)

	got, err = dataNodeMedium.Read("destination/sub/b.go")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "package b", got)
}

func TestDataNode_RenameDir_ReadFailureBad(t *core.T) {
	dataNodeMedium := New()
	core.RequireNoError(t, dataNodeMedium.Write(dataNodeSourcePath, dataNodePackageA))

	original := dataNodeReadAll
	dataNodeReadAll = func(_ goio.Reader) ([]byte, error) {
		return nil, core.NewError(dataNodeReadFailed)
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
	defer func() { _ = file.Close() }()

	data, err := goio.ReadAll(file)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "opened", string(data))
}

func TestDataNode_CreateAppendGood(t *core.T) {
	dataNodeMedium := New()

	writer, err := dataNodeMedium.Create(dataNodeNewPath)
	core.RequireNoError(t, err)
	_, _ = writer.Write([]byte("hello"))
	core.RequireNoError(t, writer.Close())

	got, err := dataNodeMedium.Read(dataNodeNewPath)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "hello", got)

	writer, err = dataNodeMedium.Append(dataNodeNewPath)
	core.RequireNoError(t, err)
	_, _ = writer.Write([]byte(" world"))
	core.RequireNoError(t, writer.Close())

	got, err = dataNodeMedium.Read(dataNodeNewPath)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "hello world", got)
}

func TestDataNode_Append_ReadFailure_Bad(t *core.T) {
	dataNodeMedium := New()
	core.RequireNoError(t, dataNodeMedium.Write(dataNodeNewPath, "hello"))

	original := dataNodeReadAll
	dataNodeReadAll = func(_ goio.Reader) ([]byte, error) {
		return nil, core.NewError(dataNodeReadFailed)
	}
	t.Cleanup(func() {
		dataNodeReadAll = original
	})

	_, err := dataNodeMedium.Append(dataNodeNewPath)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "failed to read existing content")
}

func TestDataNode_StreamsGood(t *core.T) {
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

func TestDataNode_SnapshotRestoreGood(t *core.T) {
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

	core.RequireNoError(t, dataNodeMedium.Write(dataNodeOriginalPath, "before"))

	snapshotData, err := dataNodeMedium.Snapshot()
	core.RequireNoError(t, err)

	core.RequireNoError(t, dataNodeMedium.Write(dataNodeOriginalPath, "after"))
	core.RequireNoError(t, dataNodeMedium.Write("extra.txt", "extra"))

	core.RequireNoError(t, dataNodeMedium.Restore(snapshotData))

	got, err := dataNodeMedium.Read(dataNodeOriginalPath)
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
	defer func() { _ = file.Close() }()

	data, err := goio.ReadAll(file)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "borg", string(data))
}

func TestDataNode_OverwriteGood(t *core.T) {
	dataNodeMedium := New()

	core.RequireNoError(t, dataNodeMedium.Write(dataNodeFilePath, "v1"))
	core.RequireNoError(t, dataNodeMedium.Write(dataNodeFilePath, "v2"))

	got, err := dataNodeMedium.Read(dataNodeFilePath)
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

func TestDataNode_ReadExistingFileGood(t *core.T) {
	dataNodeMedium := New()

	core.RequireNoError(t, dataNodeMedium.Write(dataNodeFilePath, "content"))
	got, err := dataNodeMedium.Read(dataNodeFilePath)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "content", got)
}

func TestMedium_New_Good(t *core.T) {
	medium := New()
	core.AssertNotNil(t, medium)
	core.AssertTrue(t, medium.Exists(""))
}

func TestMedium_New_Bad(t *core.T) {
	first := New()
	second := New()
	core.RequireNoError(t, first.Write("only-first.txt", "x"))
	core.AssertFalse(t, second.Exists("only-first.txt"))
}

func TestMedium_New_Ugly(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.EnsureDir("a/b/c"))
	core.AssertTrue(t, medium.IsDir("a/b/c"))
}

func TestMedium_FromTar_Good(t *core.T) {
	source := New()
	core.RequireNoError(t, source.Write(dataNodeFilePath, "payload"))
	snapshot, err := source.Snapshot()
	core.RequireNoError(t, err)

	restored, err := FromTar(snapshot)
	core.AssertNoError(t, err)
	core.AssertTrue(t, restored.IsFile(dataNodeFilePath))
}

func TestMedium_FromTar_Bad(t *core.T) {
	restored, err := FromTar([]byte("not a tar archive"))
	core.AssertError(t, err)
	core.AssertNil(t, restored)
}

func TestMedium_FromTar_Ugly(t *core.T) {
	source := New()
	snapshot, err := source.Snapshot()
	core.RequireNoError(t, err)
	restored, err := FromTar(snapshot)
	core.AssertNoError(t, err)
	core.AssertTrue(t, restored.Exists(""))
}

func TestMedium_Medium_Snapshot_Good(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.Write("snap.txt", "payload"))
	snapshot, err := medium.Snapshot()
	core.AssertNoError(t, err)
	core.AssertNotEmpty(t, snapshot)
}

func TestMedium_Medium_Snapshot_Bad(t *core.T) {
	var medium *Medium
	core.AssertPanics(t, func() { _, _ = medium.Snapshot() })
	core.AssertNil(t, medium)
}

func TestMedium_Medium_Snapshot_Ugly(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.Write("nested/path/file.txt", "payload"))
	snapshot, err := medium.Snapshot()
	core.AssertNoError(t, err)
	core.AssertContains(t, string(snapshot), "nested/path/file.txt")
}

func TestMedium_Medium_Restore_Good(t *core.T) {
	source := New()
	core.RequireNoError(t, source.Write("restore.txt", "payload"))
	snapshot, err := source.Snapshot()
	core.RequireNoError(t, err)

	medium := New()
	err = medium.Restore(snapshot)
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("restore.txt"))
}

func TestMedium_Medium_Restore_Bad(t *core.T) {
	medium := New()
	err := medium.Restore([]byte("not tar"))
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("not tar"))
}

func TestMedium_Medium_Restore_Ugly(t *core.T) {
	source := New()
	core.RequireNoError(t, source.Write("a/b/c.txt", "payload"))
	snapshot, err := source.Snapshot()
	core.RequireNoError(t, err)
	medium := New()
	core.RequireNoError(t, medium.Restore(snapshot))
	core.AssertTrue(t, medium.IsDir("a/b"))
}

func TestMedium_Medium_DataNode_Good(t *core.T) {
	medium := New()
	dataNode := medium.DataNode()
	core.AssertNotNil(t, dataNode)
	core.AssertSame(t, dataNode, medium.DataNode())
}

func TestMedium_Medium_DataNode_Bad(t *core.T) {
	var medium *Medium
	core.AssertPanics(t, func() { _ = medium.DataNode() })
	core.AssertNil(t, medium)
}

func TestMedium_Medium_DataNode_Ugly(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.Write(dataNodeFilePath, "payload"))
	dataNode := medium.DataNode()
	core.AssertNotNil(t, dataNode)
	core.AssertTrue(t, medium.Exists(dataNodeFilePath))
}

func TestMedium_Medium_Read_Good(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.Write("read.txt", "payload"))
	got, err := medium.Read("read.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", got)
}

func TestMedium_Medium_Read_Bad(t *core.T) {
	medium := New()
	got, err := medium.Read("missing.txt")
	core.AssertError(t, err)
	core.AssertEqual(t, "", got)
}

func TestMedium_Medium_Read_Ugly(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.Write("/safe/./file.txt", "payload"))
	got, err := medium.Read("safe/file.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", got)
}

func TestMedium_Medium_Write_Good(t *core.T) {
	medium := New()
	err := medium.Write("write.txt", "payload")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("write.txt"))
}

func TestMedium_Medium_Write_Bad(t *core.T) {
	medium := New()
	err := medium.Write("", "payload")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.IsFile(""))
}

func TestMedium_Medium_Write_Ugly(t *core.T) {
	medium := New()
	err := medium.Write("/nested/path.txt", "")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("nested/path.txt"))
}

func TestMedium_Medium_WriteMode_Good(t *core.T) {
	medium := New()
	err := medium.WriteMode("mode.txt", "payload", 0600)
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("mode.txt"))
}

func TestMedium_Medium_WriteMode_Bad(t *core.T) {
	medium := New()
	err := medium.WriteMode("", "payload", 0600)
	core.AssertError(t, err)
	core.AssertFalse(t, medium.IsFile(""))
}

func TestMedium_Medium_WriteMode_Ugly(t *core.T) {
	medium := New()
	err := medium.WriteMode("zero-mode.txt", "payload", 0)
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("zero-mode.txt"))
}

func TestMedium_Medium_EnsureDir_Good(t *core.T) {
	medium := New()
	err := medium.EnsureDir("dir/sub")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir("dir/sub"))
}

func TestMedium_Medium_EnsureDir_Bad(t *core.T) {
	medium := New()
	err := medium.EnsureDir("")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists(""))
}

func TestMedium_Medium_EnsureDir_Ugly(t *core.T) {
	medium := New()
	err := medium.EnsureDir("/leading/slash")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir("leading/slash"))
}

func TestMedium_Medium_IsFile_Good(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.Write(dataNodeFilePath, "payload"))
	got := medium.IsFile(dataNodeFilePath)
	core.AssertTrue(t, got)
}

func TestMedium_Medium_IsFile_Bad(t *core.T) {
	medium := New()
	got := medium.IsFile("missing.txt")
	core.AssertFalse(t, got)
}

func TestMedium_Medium_IsFile_Ugly(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.IsFile("dir")
	core.AssertFalse(t, got)
}

func TestMedium_Medium_Delete_Good(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.Write("delete.txt", "payload"))
	err := medium.Delete("delete.txt")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("delete.txt"))
}

func TestMedium_Medium_Delete_Bad(t *core.T) {
	medium := New()
	err := medium.Delete("missing.txt")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("missing.txt"))
}

func TestMedium_Medium_Delete_Ugly(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.EnsureDir("empty"))
	err := medium.Delete("empty")
	core.AssertError(t, err)
	core.AssertTrue(t, medium.Exists("empty"))
}

func TestMedium_Medium_DeleteAll_Good(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.Write("tree/file.txt", "payload"))
	err := medium.DeleteAll("tree")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("tree/file.txt"))
}

func TestMedium_Medium_DeleteAll_Bad(t *core.T) {
	medium := New()
	err := medium.DeleteAll("missing")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("missing"))
}

func TestMedium_Medium_DeleteAll_Ugly(t *core.T) {
	medium := New()
	err := medium.DeleteAll("")
	core.AssertError(t, err)
	core.AssertTrue(t, medium.Exists(""))
}

func TestMedium_Medium_Rename_Good(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.Write(dataNodeOldPath, "payload"))
	err := medium.Rename(dataNodeOldPath, dataNodeNewPath)
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile(dataNodeNewPath))
}

func TestMedium_Medium_Rename_Bad(t *core.T) {
	medium := New()
	err := medium.Rename("missing.txt", dataNodeNewPath)
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists(dataNodeNewPath))
}

func TestMedium_Medium_Rename_Ugly(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.Write("dir/old.txt", "payload"))
	err := medium.Rename("dir", "moved")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("moved/old.txt"))
}

func TestMedium_Medium_List_Good(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.Write("dir/a.txt", "payload"))
	entries, err := medium.List("dir")
	core.AssertNoError(t, err)
	core.AssertLen(t, entries, 1)
}

func TestMedium_Medium_List_Bad(t *core.T) {
	medium := New()
	entries, err := medium.List("missing")
	core.AssertNoError(t, err)
	core.AssertEmpty(t, entries)
}

func TestMedium_Medium_List_Ugly(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.EnsureDir("empty"))
	entries, err := medium.List("empty")
	core.AssertNoError(t, err)
	core.AssertEmpty(t, entries)
}

func TestMedium_Medium_Stat_Good(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.Write("stat.txt", "payload"))
	info, err := medium.Stat("stat.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "stat.txt", info.Name())
}

func TestMedium_Medium_Stat_Bad(t *core.T) {
	medium := New()
	info, err := medium.Stat("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, info)
}

func TestMedium_Medium_Stat_Ugly(t *core.T) {
	medium := New()
	info, err := medium.Stat("")
	core.AssertNoError(t, err)
	core.AssertTrue(t, info.IsDir())
}

func TestMedium_Medium_Open_Good(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.Write("open.txt", "payload"))
	file, err := medium.Open("open.txt")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, file)
	core.RequireNoError(t, file.Close())
}

func TestMedium_Medium_Open_Bad(t *core.T) {
	medium := New()
	file, err := medium.Open("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, file)
}

func TestMedium_Medium_Open_Ugly(t *core.T) {
	medium := New()
	file, err := medium.Open("")
	core.AssertError(t, err)
	core.AssertNil(t, file)
}

func TestMedium_Medium_Create_Good(t *core.T) {
	medium := New()
	writer, err := medium.Create("create.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestMedium_Medium_Create_Bad(t *core.T) {
	medium := New()
	writer, err := medium.Create("")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestMedium_Medium_Create_Ugly(t *core.T) {
	medium := New()
	writer, err := medium.Create("/leading.txt")
	core.RequireNoError(t, err)
	core.AssertNoError(t, writer.Close())
	core.AssertTrue(t, medium.IsFile("leading.txt"))
}

func TestMedium_Medium_Append_Good(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.Write("append.txt", "a"))
	writer, err := medium.Append("append.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("b"))
	core.RequireNoError(t, writeErr)
	core.RequireNoError(t, writer.Close())
}

func TestMedium_Medium_Append_Bad(t *core.T) {
	medium := New()
	writer, err := medium.Append("")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestMedium_Medium_Append_Ugly(t *core.T) {
	medium := New()
	writer, err := medium.Append(dataNodeNewPath)
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("created"))
	core.RequireNoError(t, writeErr)
	core.RequireNoError(t, writer.Close())
}

func TestMedium_Medium_ReadStream_Good(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.Write("stream.txt", "payload"))
	reader, err := medium.ReadStream("stream.txt")
	core.RequireNoError(t, err)
	defer func() { _ = reader.Close() }()
	data, readErr := goio.ReadAll(reader)
	core.AssertNoError(t, readErr)
	core.AssertEqual(t, "payload", string(data))
}

func TestMedium_Medium_ReadStream_Bad(t *core.T) {
	medium := New()
	reader, err := medium.ReadStream("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, reader)
}

func TestMedium_Medium_ReadStream_Ugly(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.Write("/stream.txt", "payload"))
	reader, err := medium.ReadStream("stream.txt")
	core.RequireNoError(t, err)
	core.AssertNoError(t, reader.Close())
}

func TestMedium_Medium_WriteStream_Good(t *core.T) {
	medium := New()
	writer, err := medium.WriteStream("stream-write.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestMedium_Medium_WriteStream_Bad(t *core.T) {
	medium := New()
	writer, err := medium.WriteStream("")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestMedium_Medium_WriteStream_Ugly(t *core.T) {
	medium := New()
	writer, err := medium.WriteStream("/leading.txt")
	core.RequireNoError(t, err)
	core.AssertNoError(t, writer.Close())
	core.AssertTrue(t, medium.Exists("leading.txt"))
}

func TestMedium_Medium_Exists_Good(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.Write("exists.txt", "payload"))
	got := medium.Exists("exists.txt")
	core.AssertTrue(t, got)
}

func TestMedium_Medium_Exists_Bad(t *core.T) {
	medium := New()
	got := medium.Exists("missing.txt")
	core.AssertFalse(t, got)
}

func TestMedium_Medium_Exists_Ugly(t *core.T) {
	medium := New()
	got := medium.Exists("")
	core.AssertTrue(t, got)
}

func TestMedium_Medium_IsDir_Good(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.IsDir("dir")
	core.AssertTrue(t, got)
}

func TestMedium_Medium_IsDir_Bad(t *core.T) {
	medium := New()
	got := medium.IsDir("missing")
	core.AssertFalse(t, got)
}

func TestMedium_Medium_IsDir_Ugly(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.Write(dataNodeFilePath, "payload"))
	got := medium.IsDir(dataNodeFilePath)
	core.AssertFalse(t, got)
}

func TestMedium_Closer_Write_Good(t *core.T) {
	medium := New()
	writer, err := medium.Create("closer.txt")
	core.RequireNoError(t, err)
	count, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertEqual(t, len("payload"), count)
}

func TestMedium_Closer_Write_Bad(t *core.T) {
	writer := &writeCloser{}
	count, err := writer.Write(nil)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, count)
}

func TestMedium_Closer_Write_Ugly(t *core.T) {
	writer := &writeCloser{}
	count, err := writer.Write([]byte(""))
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, count)
}

func TestMedium_Closer_Close_Good(t *core.T) {
	medium := New()
	writer := &writeCloser{medium: medium, path: "close.txt", buffer: []byte("payload")}
	err := writer.Close()
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("close.txt"))
}

func TestMedium_Closer_Close_Bad(t *core.T) {
	writer := &writeCloser{}
	core.AssertPanics(t, func() { _ = writer.Close() })
	core.AssertNil(t, writer.medium)
}

func TestMedium_Closer_Close_Ugly(t *core.T) {
	medium := New()
	writer := &writeCloser{medium: medium, path: "nested/empty.txt"}
	err := writer.Close()
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("nested/empty.txt"))
}

func TestMedium_Entry_Name_Good(t *core.T) {
	entry := &dirEntry{name: "dir"}
	got := entry.Name()
	core.AssertEqual(t, "dir", got)
}

func TestMedium_Entry_Name_Bad(t *core.T) {
	entry := &dirEntry{}
	got := entry.Name()
	core.AssertEqual(t, "", got)
}

func TestMedium_Entry_Name_Ugly(t *core.T) {
	entry := &dirEntry{name: "nested/path"}
	got := entry.Name()
	core.AssertEqual(t, "nested/path", got)
}

func TestMedium_Entry_IsDir_Good(t *core.T) {
	entry := &dirEntry{name: "dir"}
	got := entry.IsDir()
	core.AssertTrue(t, got)
}

func TestMedium_Entry_IsDir_Bad(t *core.T) {
	entry := &dirEntry{}
	got := entry.IsDir()
	core.AssertTrue(t, got)
}

func TestMedium_Entry_IsDir_Ugly(t *core.T) {
	entry := &dirEntry{name: ""}
	got := entry.IsDir()
	core.AssertTrue(t, got)
}

func TestMedium_Entry_Type_Good(t *core.T) {
	entry := &dirEntry{name: "dir"}
	got := entry.Type()
	core.AssertEqual(t, fs.ModeDir, got)
}

func TestMedium_Entry_Type_Bad(t *core.T) {
	entry := &dirEntry{}
	got := entry.Type()
	core.AssertEqual(t, fs.ModeDir, got)
}

func TestMedium_Entry_Type_Ugly(t *core.T) {
	entry := &dirEntry{name: "nested"}
	got := entry.Type()
	core.AssertTrue(t, got.IsDir())
}

func TestMedium_Entry_Info_Good(t *core.T) {
	entry := &dirEntry{name: "dir"}
	info, err := entry.Info()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "dir", info.Name())
}

func TestMedium_Entry_Info_Bad(t *core.T) {
	entry := &dirEntry{}
	info, err := entry.Info()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "", info.Name())
}

func TestMedium_Entry_Info_Ugly(t *core.T) {
	entry := &dirEntry{name: "nested/path"}
	info, err := entry.Info()
	core.AssertNoError(t, err)
	core.AssertTrue(t, info.IsDir())
}

func TestMedium_Info_Name_Good(t *core.T) {
	info := &fileInfo{name: dataNodeFilePath}
	got := info.Name()
	core.AssertEqual(t, dataNodeFilePath, got)
}

func TestMedium_Info_Name_Bad(t *core.T) {
	info := &fileInfo{}
	got := info.Name()
	core.AssertEqual(t, "", got)
}

func TestMedium_Info_Name_Ugly(t *core.T) {
	info := &fileInfo{name: "."}
	got := info.Name()
	core.AssertEqual(t, ".", got)
}

func TestMedium_Info_Size_Good(t *core.T) {
	info := &fileInfo{size: 7}
	got := info.Size()
	core.AssertEqual(t, int64(7), got)
}

func TestMedium_Info_Size_Bad(t *core.T) {
	info := &fileInfo{size: -1}
	got := info.Size()
	core.AssertEqual(t, int64(-1), got)
}

func TestMedium_Info_Size_Ugly(t *core.T) {
	info := &fileInfo{}
	got := info.Size()
	core.AssertEqual(t, int64(0), got)
}

func TestMedium_Info_Mode_Good(t *core.T) {
	info := &fileInfo{mode: 0644}
	got := info.Mode()
	core.AssertEqual(t, fs.FileMode(0644), got)
}

func TestMedium_Info_Mode_Bad(t *core.T) {
	info := &fileInfo{}
	got := info.Mode()
	core.AssertEqual(t, fs.FileMode(0), got)
}

func TestMedium_Info_Mode_Ugly(t *core.T) {
	info := &fileInfo{mode: fs.ModeDir | 0755}
	got := info.Mode()
	core.AssertTrue(t, got.IsDir())
}

func TestMedium_Info_ModTime_Good(t *core.T) {
	now := time.Unix(1, 0)
	info := &fileInfo{modTime: now}
	got := info.ModTime()
	core.AssertTrue(t, got.Equal(now))
}

func TestMedium_Info_ModTime_Bad(t *core.T) {
	info := &fileInfo{}
	got := info.ModTime()
	core.AssertTrue(t, got.IsZero())
}

func TestMedium_Info_ModTime_Ugly(t *core.T) {
	now := time.Unix(0, 1)
	info := &fileInfo{modTime: now}
	got := info.ModTime()
	core.AssertEqual(t, 1, got.Nanosecond())
}

func TestMedium_Info_IsDir_Good(t *core.T) {
	info := &fileInfo{isDir: true}
	got := info.IsDir()
	core.AssertTrue(t, got)
}

func TestMedium_Info_IsDir_Bad(t *core.T) {
	info := &fileInfo{}
	got := info.IsDir()
	core.AssertFalse(t, got)
}

func TestMedium_Info_IsDir_Ugly(t *core.T) {
	info := &fileInfo{mode: fs.ModeDir | 0755, isDir: false}
	got := info.IsDir()
	core.AssertFalse(t, got)
}

func TestMedium_Info_Sys_Good(t *core.T) {
	info := &fileInfo{name: dataNodeFilePath}
	got := info.Sys()
	core.AssertNil(t, got)
}

func TestMedium_Info_Sys_Bad(t *core.T) {
	info := &fileInfo{}
	got := info.Sys()
	core.AssertNil(t, got)
}

func TestMedium_Info_Sys_Ugly(t *core.T) {
	info := &fileInfo{isDir: true}
	got := info.Sys()
	core.AssertNil(t, got)
}
