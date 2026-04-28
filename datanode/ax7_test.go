package datanode

import (
	goio "io"
	"io/fs"
	"time"

	core "dappco.re/go"
)

func TestAX7_New_Good(t *core.T) {
	medium := New()
	core.AssertNotNil(t, medium)
	core.AssertTrue(t, medium.Exists(""))
}

func TestAX7_New_Bad(t *core.T) {
	first := New()
	second := New()
	core.RequireNoError(t, first.Write("only-first.txt", "x"))
	core.AssertFalse(t, second.Exists("only-first.txt"))
}

func TestAX7_New_Ugly(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.EnsureDir("a/b/c"))
	core.AssertTrue(t, medium.IsDir("a/b/c"))
}

func TestAX7_FromTar_Good(t *core.T) {
	source := New()
	core.RequireNoError(t, source.Write("file.txt", "payload"))
	snapshot, err := source.Snapshot()
	core.RequireNoError(t, err)

	restored, err := FromTar(snapshot)
	core.AssertNoError(t, err)
	core.AssertTrue(t, restored.IsFile("file.txt"))
}

func TestAX7_FromTar_Bad(t *core.T) {
	restored, err := FromTar([]byte("not a tar archive"))
	core.AssertError(t, err)
	core.AssertNil(t, restored)
}

func TestAX7_FromTar_Ugly(t *core.T) {
	source := New()
	snapshot, err := source.Snapshot()
	core.RequireNoError(t, err)
	restored, err := FromTar(snapshot)
	core.AssertNoError(t, err)
	core.AssertTrue(t, restored.Exists(""))
}

func TestAX7_Medium_Snapshot_Good(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.Write("snap.txt", "payload"))
	snapshot, err := medium.Snapshot()
	core.AssertNoError(t, err)
	core.AssertNotEmpty(t, snapshot)
}

func TestAX7_Medium_Snapshot_Bad(t *core.T) {
	var medium *Medium
	core.AssertPanics(t, func() { _, _ = medium.Snapshot() })
	core.AssertNil(t, medium)
}

func TestAX7_Medium_Snapshot_Ugly(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.Write("nested/path/file.txt", "payload"))
	snapshot, err := medium.Snapshot()
	core.AssertNoError(t, err)
	core.AssertContains(t, string(snapshot), "nested/path/file.txt")
}

func TestAX7_Medium_Restore_Good(t *core.T) {
	source := New()
	core.RequireNoError(t, source.Write("restore.txt", "payload"))
	snapshot, err := source.Snapshot()
	core.RequireNoError(t, err)

	medium := New()
	err = medium.Restore(snapshot)
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("restore.txt"))
}

func TestAX7_Medium_Restore_Bad(t *core.T) {
	medium := New()
	err := medium.Restore([]byte("not tar"))
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("not tar"))
}

func TestAX7_Medium_Restore_Ugly(t *core.T) {
	source := New()
	core.RequireNoError(t, source.Write("a/b/c.txt", "payload"))
	snapshot, err := source.Snapshot()
	core.RequireNoError(t, err)
	medium := New()
	core.RequireNoError(t, medium.Restore(snapshot))
	core.AssertTrue(t, medium.IsDir("a/b"))
}

func TestAX7_Medium_DataNode_Good(t *core.T) {
	medium := New()
	dataNode := medium.DataNode()
	core.AssertNotNil(t, dataNode)
	core.AssertSame(t, dataNode, medium.DataNode())
}

func TestAX7_Medium_DataNode_Bad(t *core.T) {
	var medium *Medium
	core.AssertPanics(t, func() { _ = medium.DataNode() })
	core.AssertNil(t, medium)
}

func TestAX7_Medium_DataNode_Ugly(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.Write("file.txt", "payload"))
	dataNode := medium.DataNode()
	core.AssertNotNil(t, dataNode)
	core.AssertTrue(t, medium.Exists("file.txt"))
}

func TestAX7_Medium_Read_Good(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.Write("read.txt", "payload"))
	got, err := medium.Read("read.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", got)
}

func TestAX7_Medium_Read_Bad(t *core.T) {
	medium := New()
	got, err := medium.Read("missing.txt")
	core.AssertError(t, err)
	core.AssertEqual(t, "", got)
}

func TestAX7_Medium_Read_Ugly(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.Write("/safe/./file.txt", "payload"))
	got, err := medium.Read("safe/file.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", got)
}

func TestAX7_Medium_Write_Good(t *core.T) {
	medium := New()
	err := medium.Write("write.txt", "payload")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("write.txt"))
}

func TestAX7_Medium_Write_Bad(t *core.T) {
	medium := New()
	err := medium.Write("", "payload")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.IsFile(""))
}

func TestAX7_Medium_Write_Ugly(t *core.T) {
	medium := New()
	err := medium.Write("/nested/path.txt", "")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("nested/path.txt"))
}

func TestAX7_Medium_WriteMode_Good(t *core.T) {
	medium := New()
	err := medium.WriteMode("mode.txt", "payload", 0600)
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("mode.txt"))
}

func TestAX7_Medium_WriteMode_Bad(t *core.T) {
	medium := New()
	err := medium.WriteMode("", "payload", 0600)
	core.AssertError(t, err)
	core.AssertFalse(t, medium.IsFile(""))
}

func TestAX7_Medium_WriteMode_Ugly(t *core.T) {
	medium := New()
	err := medium.WriteMode("zero-mode.txt", "payload", 0)
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("zero-mode.txt"))
}

func TestAX7_Medium_EnsureDir_Good(t *core.T) {
	medium := New()
	err := medium.EnsureDir("dir/sub")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir("dir/sub"))
}

func TestAX7_Medium_EnsureDir_Bad(t *core.T) {
	medium := New()
	err := medium.EnsureDir("")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists(""))
}

func TestAX7_Medium_EnsureDir_Ugly(t *core.T) {
	medium := New()
	err := medium.EnsureDir("/leading/slash")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir("leading/slash"))
}

func TestAX7_Medium_IsFile_Good(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.Write("file.txt", "payload"))
	got := medium.IsFile("file.txt")
	core.AssertTrue(t, got)
}

func TestAX7_Medium_IsFile_Bad(t *core.T) {
	medium := New()
	got := medium.IsFile("missing.txt")
	core.AssertFalse(t, got)
}

func TestAX7_Medium_IsFile_Ugly(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.IsFile("dir")
	core.AssertFalse(t, got)
}

func TestAX7_Medium_Delete_Good(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.Write("delete.txt", "payload"))
	err := medium.Delete("delete.txt")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("delete.txt"))
}

func TestAX7_Medium_Delete_Bad(t *core.T) {
	medium := New()
	err := medium.Delete("missing.txt")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("missing.txt"))
}

func TestAX7_Medium_Delete_Ugly(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.EnsureDir("empty"))
	err := medium.Delete("empty")
	core.AssertError(t, err)
	core.AssertTrue(t, medium.Exists("empty"))
}

func TestAX7_Medium_DeleteAll_Good(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.Write("tree/file.txt", "payload"))
	err := medium.DeleteAll("tree")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("tree/file.txt"))
}

func TestAX7_Medium_DeleteAll_Bad(t *core.T) {
	medium := New()
	err := medium.DeleteAll("missing")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("missing"))
}

func TestAX7_Medium_DeleteAll_Ugly(t *core.T) {
	medium := New()
	err := medium.DeleteAll("")
	core.AssertError(t, err)
	core.AssertTrue(t, medium.Exists(""))
}

func TestAX7_Medium_Rename_Good(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.Write("old.txt", "payload"))
	err := medium.Rename("old.txt", "new.txt")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("new.txt"))
}

func TestAX7_Medium_Rename_Bad(t *core.T) {
	medium := New()
	err := medium.Rename("missing.txt", "new.txt")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("new.txt"))
}

func TestAX7_Medium_Rename_Ugly(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.Write("dir/old.txt", "payload"))
	err := medium.Rename("dir", "moved")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("moved/old.txt"))
}

func TestAX7_Medium_List_Good(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.Write("dir/a.txt", "payload"))
	entries, err := medium.List("dir")
	core.AssertNoError(t, err)
	core.AssertLen(t, entries, 1)
}

func TestAX7_Medium_List_Bad(t *core.T) {
	medium := New()
	entries, err := medium.List("missing")
	core.AssertNoError(t, err)
	core.AssertEmpty(t, entries)
}

func TestAX7_Medium_List_Ugly(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.EnsureDir("empty"))
	entries, err := medium.List("empty")
	core.AssertNoError(t, err)
	core.AssertEmpty(t, entries)
}

func TestAX7_Medium_Stat_Good(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.Write("stat.txt", "payload"))
	info, err := medium.Stat("stat.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "stat.txt", info.Name())
}

func TestAX7_Medium_Stat_Bad(t *core.T) {
	medium := New()
	info, err := medium.Stat("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, info)
}

func TestAX7_Medium_Stat_Ugly(t *core.T) {
	medium := New()
	info, err := medium.Stat("")
	core.AssertNoError(t, err)
	core.AssertTrue(t, info.IsDir())
}

func TestAX7_Medium_Open_Good(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.Write("open.txt", "payload"))
	file, err := medium.Open("open.txt")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, file)
	core.RequireNoError(t, file.Close())
}

func TestAX7_Medium_Open_Bad(t *core.T) {
	medium := New()
	file, err := medium.Open("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, file)
}

func TestAX7_Medium_Open_Ugly(t *core.T) {
	medium := New()
	file, err := medium.Open("")
	core.AssertError(t, err)
	core.AssertNil(t, file)
}

func TestAX7_Medium_Create_Good(t *core.T) {
	medium := New()
	writer, err := medium.Create("create.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestAX7_Medium_Create_Bad(t *core.T) {
	medium := New()
	writer, err := medium.Create("")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestAX7_Medium_Create_Ugly(t *core.T) {
	medium := New()
	writer, err := medium.Create("/leading.txt")
	core.RequireNoError(t, err)
	core.AssertNoError(t, writer.Close())
	core.AssertTrue(t, medium.IsFile("leading.txt"))
}

func TestAX7_Medium_Append_Good(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.Write("append.txt", "a"))
	writer, err := medium.Append("append.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("b"))
	core.RequireNoError(t, writeErr)
	core.RequireNoError(t, writer.Close())
}

func TestAX7_Medium_Append_Bad(t *core.T) {
	medium := New()
	writer, err := medium.Append("")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestAX7_Medium_Append_Ugly(t *core.T) {
	medium := New()
	writer, err := medium.Append("new.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("created"))
	core.RequireNoError(t, writeErr)
	core.RequireNoError(t, writer.Close())
}

func TestAX7_Medium_ReadStream_Good(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.Write("stream.txt", "payload"))
	reader, err := medium.ReadStream("stream.txt")
	core.RequireNoError(t, err)
	defer reader.Close()
	data, readErr := goio.ReadAll(reader)
	core.AssertNoError(t, readErr)
	core.AssertEqual(t, "payload", string(data))
}

func TestAX7_Medium_ReadStream_Bad(t *core.T) {
	medium := New()
	reader, err := medium.ReadStream("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, reader)
}

func TestAX7_Medium_ReadStream_Ugly(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.Write("/stream.txt", "payload"))
	reader, err := medium.ReadStream("stream.txt")
	core.RequireNoError(t, err)
	core.AssertNoError(t, reader.Close())
}

func TestAX7_Medium_WriteStream_Good(t *core.T) {
	medium := New()
	writer, err := medium.WriteStream("stream-write.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestAX7_Medium_WriteStream_Bad(t *core.T) {
	medium := New()
	writer, err := medium.WriteStream("")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestAX7_Medium_WriteStream_Ugly(t *core.T) {
	medium := New()
	writer, err := medium.WriteStream("/leading.txt")
	core.RequireNoError(t, err)
	core.AssertNoError(t, writer.Close())
	core.AssertTrue(t, medium.Exists("leading.txt"))
}

func TestAX7_Medium_Exists_Good(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.Write("exists.txt", "payload"))
	got := medium.Exists("exists.txt")
	core.AssertTrue(t, got)
}

func TestAX7_Medium_Exists_Bad(t *core.T) {
	medium := New()
	got := medium.Exists("missing.txt")
	core.AssertFalse(t, got)
}

func TestAX7_Medium_Exists_Ugly(t *core.T) {
	medium := New()
	got := medium.Exists("")
	core.AssertTrue(t, got)
}

func TestAX7_Medium_IsDir_Good(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.IsDir("dir")
	core.AssertTrue(t, got)
}

func TestAX7_Medium_IsDir_Bad(t *core.T) {
	medium := New()
	got := medium.IsDir("missing")
	core.AssertFalse(t, got)
}

func TestAX7_Medium_IsDir_Ugly(t *core.T) {
	medium := New()
	core.RequireNoError(t, medium.Write("file.txt", "payload"))
	got := medium.IsDir("file.txt")
	core.AssertFalse(t, got)
}

func TestAX7_Closer_Write_Good(t *core.T) {
	medium := New()
	writer, err := medium.Create("closer.txt")
	core.RequireNoError(t, err)
	count, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertEqual(t, len("payload"), count)
}

func TestAX7_Closer_Write_Bad(t *core.T) {
	writer := &writeCloser{}
	count, err := writer.Write(nil)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, count)
}

func TestAX7_Closer_Write_Ugly(t *core.T) {
	writer := &writeCloser{}
	count, err := writer.Write([]byte(""))
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, count)
}

func TestAX7_Closer_Close_Good(t *core.T) {
	medium := New()
	writer := &writeCloser{medium: medium, path: "close.txt", buffer: []byte("payload")}
	err := writer.Close()
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("close.txt"))
}

func TestAX7_Closer_Close_Bad(t *core.T) {
	writer := &writeCloser{}
	core.AssertPanics(t, func() { _ = writer.Close() })
	core.AssertNil(t, writer.medium)
}

func TestAX7_Closer_Close_Ugly(t *core.T) {
	medium := New()
	writer := &writeCloser{medium: medium, path: "nested/empty.txt"}
	err := writer.Close()
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("nested/empty.txt"))
}

func TestAX7_Entry_Name_Good(t *core.T) {
	entry := &dirEntry{name: "dir"}
	got := entry.Name()
	core.AssertEqual(t, "dir", got)
}

func TestAX7_Entry_Name_Bad(t *core.T) {
	entry := &dirEntry{}
	got := entry.Name()
	core.AssertEqual(t, "", got)
}

func TestAX7_Entry_Name_Ugly(t *core.T) {
	entry := &dirEntry{name: "nested/path"}
	got := entry.Name()
	core.AssertEqual(t, "nested/path", got)
}

func TestAX7_Entry_IsDir_Good(t *core.T) {
	entry := &dirEntry{name: "dir"}
	got := entry.IsDir()
	core.AssertTrue(t, got)
}

func TestAX7_Entry_IsDir_Bad(t *core.T) {
	entry := &dirEntry{}
	got := entry.IsDir()
	core.AssertTrue(t, got)
}

func TestAX7_Entry_IsDir_Ugly(t *core.T) {
	entry := &dirEntry{name: ""}
	got := entry.IsDir()
	core.AssertTrue(t, got)
}

func TestAX7_Entry_Type_Good(t *core.T) {
	entry := &dirEntry{name: "dir"}
	got := entry.Type()
	core.AssertEqual(t, fs.ModeDir, got)
}

func TestAX7_Entry_Type_Bad(t *core.T) {
	entry := &dirEntry{}
	got := entry.Type()
	core.AssertEqual(t, fs.ModeDir, got)
}

func TestAX7_Entry_Type_Ugly(t *core.T) {
	entry := &dirEntry{name: "nested"}
	got := entry.Type()
	core.AssertTrue(t, got.IsDir())
}

func TestAX7_Entry_Info_Good(t *core.T) {
	entry := &dirEntry{name: "dir"}
	info, err := entry.Info()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "dir", info.Name())
}

func TestAX7_Entry_Info_Bad(t *core.T) {
	entry := &dirEntry{}
	info, err := entry.Info()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "", info.Name())
}

func TestAX7_Entry_Info_Ugly(t *core.T) {
	entry := &dirEntry{name: "nested/path"}
	info, err := entry.Info()
	core.AssertNoError(t, err)
	core.AssertTrue(t, info.IsDir())
}

func TestAX7_Info_Name_Good(t *core.T) {
	info := &fileInfo{name: "file.txt"}
	got := info.Name()
	core.AssertEqual(t, "file.txt", got)
}

func TestAX7_Info_Name_Bad(t *core.T) {
	info := &fileInfo{}
	got := info.Name()
	core.AssertEqual(t, "", got)
}

func TestAX7_Info_Name_Ugly(t *core.T) {
	info := &fileInfo{name: "."}
	got := info.Name()
	core.AssertEqual(t, ".", got)
}

func TestAX7_Info_Size_Good(t *core.T) {
	info := &fileInfo{size: 7}
	got := info.Size()
	core.AssertEqual(t, int64(7), got)
}

func TestAX7_Info_Size_Bad(t *core.T) {
	info := &fileInfo{size: -1}
	got := info.Size()
	core.AssertEqual(t, int64(-1), got)
}

func TestAX7_Info_Size_Ugly(t *core.T) {
	info := &fileInfo{}
	got := info.Size()
	core.AssertEqual(t, int64(0), got)
}

func TestAX7_Info_Mode_Good(t *core.T) {
	info := &fileInfo{mode: 0644}
	got := info.Mode()
	core.AssertEqual(t, fs.FileMode(0644), got)
}

func TestAX7_Info_Mode_Bad(t *core.T) {
	info := &fileInfo{}
	got := info.Mode()
	core.AssertEqual(t, fs.FileMode(0), got)
}

func TestAX7_Info_Mode_Ugly(t *core.T) {
	info := &fileInfo{mode: fs.ModeDir | 0755}
	got := info.Mode()
	core.AssertTrue(t, got.IsDir())
}

func TestAX7_Info_ModTime_Good(t *core.T) {
	now := time.Unix(1, 0)
	info := &fileInfo{modTime: now}
	got := info.ModTime()
	core.AssertTrue(t, got.Equal(now))
}

func TestAX7_Info_ModTime_Bad(t *core.T) {
	info := &fileInfo{}
	got := info.ModTime()
	core.AssertTrue(t, got.IsZero())
}

func TestAX7_Info_ModTime_Ugly(t *core.T) {
	now := time.Unix(0, 1)
	info := &fileInfo{modTime: now}
	got := info.ModTime()
	core.AssertEqual(t, 1, got.Nanosecond())
}

func TestAX7_Info_IsDir_Good(t *core.T) {
	info := &fileInfo{isDir: true}
	got := info.IsDir()
	core.AssertTrue(t, got)
}

func TestAX7_Info_IsDir_Bad(t *core.T) {
	info := &fileInfo{}
	got := info.IsDir()
	core.AssertFalse(t, got)
}

func TestAX7_Info_IsDir_Ugly(t *core.T) {
	info := &fileInfo{mode: fs.ModeDir | 0755, isDir: false}
	got := info.IsDir()
	core.AssertFalse(t, got)
}

func TestAX7_Info_Sys_Good(t *core.T) {
	info := &fileInfo{name: "file.txt"}
	got := info.Sys()
	core.AssertNil(t, got)
}

func TestAX7_Info_Sys_Bad(t *core.T) {
	info := &fileInfo{}
	got := info.Sys()
	core.AssertNil(t, got)
}

func TestAX7_Info_Sys_Ugly(t *core.T) {
	info := &fileInfo{isDir: true}
	got := info.Sys()
	core.AssertNil(t, got)
}
