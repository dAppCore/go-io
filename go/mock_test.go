package io

import (
	core "dappco.re/go"
	goio "io"
	"io/fs"
	"time"
)

func TestMock_NewMockMedium_Good(t *core.T) {
	medium := NewMockMedium()
	core.AssertNotNil(t, medium)
	core.AssertNotNil(t, medium.Files)
}

func TestMock_NewMockMedium_Bad(t *core.T) {
	first := NewMockMedium()
	second := NewMockMedium()
	core.RequireNoError(t, first.Write("only-first.txt", "payload"))
	core.AssertFalse(t, second.Exists("only-first.txt"))
}

func TestMock_NewMockMedium_Ugly(t *core.T) {
	medium := NewMockMedium()
	core.RequireNoError(t, medium.EnsureDir("a/b/c"))
	core.AssertTrue(t, medium.IsDir("a/b/c"))
}

func TestMock_MockMedium_Read_Good(t *core.T) {
	medium := NewMockMedium()
	core.RequireNoError(t, medium.Write("read.txt", "payload"))
	got, err := medium.Read("read.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", got)
}

func TestMock_MockMedium_Read_Bad(t *core.T) {
	medium := NewMockMedium()
	got, err := medium.Read("missing.txt")
	core.AssertErrorIs(t, err, fs.ErrNotExist)
	core.AssertEqual(t, "", got)
}

func TestMock_MockMedium_Read_Ugly(t *core.T) {
	medium := NewMockMedium()
	core.RequireNoError(t, medium.Write("empty.txt", ""))
	got, err := medium.Read("empty.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "", got)
}

func TestMock_MockMedium_Write_Good(t *core.T) {
	medium := NewMockMedium()
	err := medium.Write("write.txt", "payload")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("write.txt"))
}

func TestMock_MockMedium_Write_Bad(t *core.T) {
	medium := NewMockMedium()
	err := medium.Write("", "payload")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile(""))
}

func TestMock_MockMedium_Write_Ugly(t *core.T) {
	medium := NewMockMedium()
	err := medium.Write("nested/write.txt", "payload")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("nested/write.txt"))
}

func TestMock_MockMedium_WriteMode_Good(t *core.T) {
	medium := NewMockMedium()
	err := medium.WriteMode("mode.txt", "payload", 0600)
	info, statErr := medium.Stat("mode.txt")
	core.AssertNoError(t, err)
	core.AssertNoError(t, statErr)
	core.AssertEqual(t, fs.FileMode(0600), info.Mode())
}

func TestMock_MockMedium_WriteMode_Bad(t *core.T) {
	medium := NewMockMedium()
	err := medium.WriteMode("", "payload", 0)
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile(""))
}

func TestMock_MockMedium_WriteMode_Ugly(t *core.T) {
	medium := NewMockMedium()
	err := medium.WriteMode("zero-mode.txt", "payload", 0)
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("zero-mode.txt"))
}

func TestMock_MockMedium_EnsureDir_Good(t *core.T) {
	medium := NewMockMedium()
	err := medium.EnsureDir("dir")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir("dir"))
}

func TestMock_MockMedium_EnsureDir_Bad(t *core.T) {
	medium := NewMockMedium()
	err := medium.EnsureDir("")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir(""))
}

func TestMock_MockMedium_EnsureDir_Ugly(t *core.T) {
	medium := NewMockMedium()
	err := medium.EnsureDir("a/b/c")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir("a/b/c"))
}

func TestMock_MockMedium_IsFile_Good(t *core.T) {
	medium := NewMockMedium()
	core.RequireNoError(t, medium.Write("file.txt", "payload"))
	got := medium.IsFile("file.txt")
	core.AssertTrue(t, got)
}

func TestMock_MockMedium_IsFile_Bad(t *core.T) {
	medium := NewMockMedium()
	got := medium.IsFile("missing.txt")
	core.AssertFalse(t, got)
}

func TestMock_MockMedium_IsFile_Ugly(t *core.T) {
	medium := NewMockMedium()
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.IsFile("dir")
	core.AssertFalse(t, got)
}

func TestMock_MockMedium_Delete_Good(t *core.T) {
	medium := NewMockMedium()
	core.RequireNoError(t, medium.Write("delete.txt", "payload"))
	err := medium.Delete("delete.txt")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("delete.txt"))
}

func TestMock_MockMedium_Delete_Bad(t *core.T) {
	medium := NewMockMedium()
	err := medium.Delete("missing.txt")
	core.AssertErrorIs(t, err, fs.ErrNotExist)
	core.AssertFalse(t, medium.Exists("missing.txt"))
}

func TestMock_MockMedium_Delete_Ugly(t *core.T) {
	medium := NewMockMedium()
	core.RequireNoError(t, medium.EnsureDir("empty"))
	err := medium.Delete("empty")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("empty"))
}

func TestMock_MockMedium_DeleteAll_Good(t *core.T) {
	medium := NewMockMedium()
	core.RequireNoError(t, medium.Write("tree/file.txt", "payload"))
	err := medium.DeleteAll("tree")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("tree/file.txt"))
}

func TestMock_MockMedium_DeleteAll_Bad(t *core.T) {
	medium := NewMockMedium()
	err := medium.DeleteAll("missing")
	core.AssertErrorIs(t, err, fs.ErrNotExist)
	core.AssertFalse(t, medium.Exists("missing"))
}

func TestMock_MockMedium_DeleteAll_Ugly(t *core.T) {
	medium := NewMockMedium()
	err := medium.DeleteAll("")
	core.AssertErrorIs(t, err, fs.ErrNotExist)
	core.AssertEmpty(t, medium.Files)
}

func TestMock_MockMedium_Rename_Good(t *core.T) {
	medium := NewMockMedium()
	core.RequireNoError(t, medium.Write("old.txt", "payload"))
	err := medium.Rename("old.txt", "new.txt")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("new.txt"))
}

func TestMock_MockMedium_Rename_Bad(t *core.T) {
	medium := NewMockMedium()
	err := medium.Rename("missing.txt", "new.txt")
	core.AssertErrorIs(t, err, fs.ErrNotExist)
	core.AssertFalse(t, medium.Exists("new.txt"))
}

func TestMock_MockMedium_Rename_Ugly(t *core.T) {
	medium := NewMockMedium()
	core.RequireNoError(t, medium.WriteMode("old.txt", "payload", 0600))
	err := medium.Rename("old.txt", "new.txt")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("new.txt"))
}

func TestMock_MockMedium_List_Good(t *core.T) {
	medium := NewMockMedium()
	core.RequireNoError(t, medium.Write("dir/a.txt", "payload"))
	entries, err := medium.List("dir")
	core.AssertNoError(t, err)
	core.AssertLen(t, entries, 1)
}

func TestMock_MockMedium_List_Bad(t *core.T) {
	medium := NewMockMedium()
	entries, err := medium.List("missing")
	core.AssertNoError(t, err)
	core.AssertEmpty(t, entries)
}

func TestMock_MockMedium_List_Ugly(t *core.T) {
	medium := NewMockMedium()
	entries, err := medium.List("")
	core.AssertNoError(t, err)
	core.AssertEmpty(t, entries)
}

func TestMock_MockMedium_Stat_Good(t *core.T) {
	medium := NewMockMedium()
	core.RequireNoError(t, medium.Write("stat.txt", "payload"))
	info, err := medium.Stat("stat.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "stat.txt", info.Name())
}

func TestMock_MockMedium_Stat_Bad(t *core.T) {
	medium := NewMockMedium()
	info, err := medium.Stat("missing.txt")
	core.AssertErrorIs(t, err, fs.ErrNotExist)
	core.AssertNil(t, info)
}

func TestMock_MockMedium_Stat_Ugly(t *core.T) {
	medium := NewMockMedium()
	core.RequireNoError(t, medium.EnsureDir("dir"))
	info, err := medium.Stat("dir")
	core.AssertNoError(t, err)
	core.AssertTrue(t, info.IsDir())
}

func TestMock_MockMedium_Open_Good(t *core.T) {
	medium := NewMockMedium()
	core.RequireNoError(t, medium.Write("open.txt", "payload"))
	file, err := medium.Open("open.txt")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, file)
	core.RequireNoError(t, file.Close())
}

func TestMock_MockMedium_Open_Bad(t *core.T) {
	medium := NewMockMedium()
	file, err := medium.Open("missing.txt")
	core.AssertErrorIs(t, err, fs.ErrNotExist)
	core.AssertNil(t, file)
}

func TestMock_MockMedium_Open_Ugly(t *core.T) {
	medium := NewMockMedium()
	core.RequireNoError(t, medium.WriteMode("open.txt", "", 0))
	file, err := medium.Open("open.txt")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, file)
}

func TestMock_MockMedium_Create_Good(t *core.T) {
	medium := NewMockMedium()
	writer, err := medium.Create("create.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestMock_MockMedium_Create_Bad(t *core.T) {
	medium := NewMockMedium()
	writer, err := medium.Create("")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, writer)
}

func TestMock_MockMedium_Create_Ugly(t *core.T) {
	medium := NewMockMedium()
	writer, err := medium.Create("nested/create.txt")
	core.RequireNoError(t, err)
	core.AssertNoError(t, writer.Close())
	core.AssertTrue(t, medium.Exists("nested/create.txt"))
}

func TestMock_MockMedium_Append_Good(t *core.T) {
	medium := NewMockMedium()
	core.RequireNoError(t, medium.Write("append.txt", "a"))
	writer, err := medium.Append("append.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("b"))
	core.RequireNoError(t, writeErr)
	core.RequireNoError(t, writer.Close())
}

func TestMock_MockMedium_Append_Bad(t *core.T) {
	medium := NewMockMedium()
	writer, err := medium.Append("")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, writer)
}

func TestMock_MockMedium_Append_Ugly(t *core.T) {
	medium := NewMockMedium()
	writer, err := medium.Append("new.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("created"))
	core.RequireNoError(t, writeErr)
	core.RequireNoError(t, writer.Close())
}

func TestMock_MockMedium_ReadStream_Good(t *core.T) {
	medium := NewMockMedium()
	core.RequireNoError(t, medium.Write("stream.txt", "payload"))
	reader, err := medium.ReadStream("stream.txt")
	core.RequireNoError(t, err)
	defer func() { _ = reader.Close() }()
	data, readErr := goio.ReadAll(reader)
	core.AssertNoError(t, readErr)
	core.AssertEqual(t, "payload", string(data))
}

func TestMock_MockMedium_ReadStream_Bad(t *core.T) {
	medium := NewMockMedium()
	reader, err := medium.ReadStream("missing.txt")
	core.AssertErrorIs(t, err, fs.ErrNotExist)
	core.AssertNil(t, reader)
}

func TestMock_MockMedium_ReadStream_Ugly(t *core.T) {
	medium := NewMockMedium()
	core.RequireNoError(t, medium.Write("empty.txt", ""))
	reader, err := medium.ReadStream("empty.txt")
	core.RequireNoError(t, err)
	core.AssertNoError(t, reader.Close())
}

func TestMock_MockMedium_WriteStream_Good(t *core.T) {
	medium := NewMockMedium()
	writer, err := medium.WriteStream("stream.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestMock_MockMedium_WriteStream_Bad(t *core.T) {
	medium := NewMockMedium()
	writer, err := medium.WriteStream("")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, writer)
}

func TestMock_MockMedium_WriteStream_Ugly(t *core.T) {
	medium := NewMockMedium()
	writer, err := medium.WriteStream("nested/stream.txt")
	core.RequireNoError(t, err)
	core.AssertNoError(t, writer.Close())
	core.AssertTrue(t, medium.Exists("nested/stream.txt"))
}

func TestMock_MockMedium_Exists_Good(t *core.T) {
	medium := NewMockMedium()
	core.RequireNoError(t, medium.Write("exists.txt", "payload"))
	got := medium.Exists("exists.txt")
	core.AssertTrue(t, got)
}

func TestMock_MockMedium_Exists_Bad(t *core.T) {
	medium := NewMockMedium()
	got := medium.Exists("missing.txt")
	core.AssertFalse(t, got)
}

func TestMock_MockMedium_Exists_Ugly(t *core.T) {
	medium := NewMockMedium()
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.Exists("dir")
	core.AssertTrue(t, got)
}

func TestMock_MockMedium_IsDir_Good(t *core.T) {
	medium := NewMockMedium()
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.IsDir("dir")
	core.AssertTrue(t, got)
}

func TestMock_MockMedium_IsDir_Bad(t *core.T) {
	medium := NewMockMedium()
	got := medium.IsDir("missing")
	core.AssertFalse(t, got)
}

func TestMock_MockMedium_IsDir_Ugly(t *core.T) {
	medium := NewMockMedium()
	core.RequireNoError(t, medium.Write("file.txt", "payload"))
	got := medium.IsDir("file.txt")
	core.AssertFalse(t, got)
}

func TestMock_MockFile_Stat_Good(t *core.T) {
	file := &MockFile{info: NewFileInfo("file.txt", 7, 0644, time.Time{}, false)}
	info, err := file.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "file.txt", info.Name())
}

func TestMock_MockFile_Stat_Bad(t *core.T) {
	file := &MockFile{}
	info, err := file.Stat()
	core.AssertNoError(t, err)
	core.AssertNil(t, info)
}

func TestMock_MockFile_Stat_Ugly(t *core.T) {
	file := &MockFile{info: NewFileInfo("dir", 0, fs.ModeDir, time.Time{}, true)}
	info, err := file.Stat()
	core.AssertNoError(t, err)
	core.AssertTrue(t, info.IsDir())
}

func TestMock_MockFile_Read_Good(t *core.T) {
	file := &MockFile{reader: core.NewReader("payload")}
	buffer := make([]byte, 7)
	count, err := file.Read(buffer)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 7, count)
	core.AssertEqual(t, "payload", string(buffer))
}

func TestMock_MockFile_Read_Bad(t *core.T) {
	file := &MockFile{reader: core.NewReader("")}
	buffer := make([]byte, 1)
	count, err := file.Read(buffer)
	core.AssertEqual(t, 0, count)
	core.AssertErrorIs(t, err, goio.EOF)
}

func TestMock_MockFile_Read_Ugly(t *core.T) {
	file := &MockFile{reader: core.NewReader("ab")}
	buffer := make([]byte, 4)
	count, err := file.Read(buffer)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 2, count)
	core.AssertEqual(t, "ab", string(buffer[:count]))
}

func TestMock_MockFile_Close_Good(t *core.T) {
	file := &MockFile{info: NewFileInfo("file.txt", 0, 0644, time.Time{}, false)}
	err := file.Close()
	core.AssertNoError(t, err)
	core.AssertNotNil(t, file.info)
}

func TestMock_MockFile_Close_Bad(t *core.T) {
	file := &MockFile{}
	err := file.Close()
	core.AssertNoError(t, err)
	core.AssertNil(t, file.info)
}

func TestMock_MockFile_Close_Ugly(t *core.T) {
	file := &MockFile{}
	err := file.Close()
	core.AssertNoError(t, err)
	core.AssertEqual(t, (*MockFile)(file), file)
}

func TestMock_MockWriteCloser_Write_Good(t *core.T) {
	writer := &MockWriteCloser{}
	count, err := writer.Write([]byte("payload"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, len("payload"), count)
}

func TestMock_MockWriteCloser_Write_Bad(t *core.T) {
	writer := &MockWriteCloser{}
	count, err := writer.Write(nil)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, count)
}

func TestMock_MockWriteCloser_Write_Ugly(t *core.T) {
	writer := &MockWriteCloser{}
	count, err := writer.Write([]byte(""))
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, count)
}

func TestMock_MockWriteCloser_Close_Good(t *core.T) {
	medium := NewMockMedium()
	writer := &MockWriteCloser{medium: medium, path: "close.txt"}
	_, writeErr := writer.Write([]byte("payload"))
	core.RequireNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
	core.AssertTrue(t, medium.Exists("close.txt"))
}

func TestMock_MockWriteCloser_Close_Bad(t *core.T) {
	writer := &MockWriteCloser{}
	core.AssertPanics(t, func() { _ = writer.Close() })
	core.AssertNil(t, writer.medium)
}

func TestMock_MockWriteCloser_Close_Ugly(t *core.T) {
	medium := NewMockMedium()
	writer := &MockWriteCloser{medium: medium, path: "empty.txt"}
	err := writer.Close()
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("empty.txt"))
}
