package io

import (
	core "dappco.re/go"
	goio "io"
	"io/fs"
	"time"
)

func TestIo_NewFileInfo_Good(t *core.T) {
	now := time.Unix(1, 0)
	info := NewFileInfo("file.txt", 7, 0600, now, false)
	core.AssertEqual(t, "file.txt", info.Name())
	core.AssertEqual(t, int64(7), info.Size())
}

func TestIo_NewFileInfo_Bad(t *core.T) {
	info := NewFileInfo("", -1, 0, time.Time{}, false)
	core.AssertEqual(t, "", info.Name())
	core.AssertEqual(t, int64(-1), info.Size())
}

func TestIo_NewFileInfo_Ugly(t *core.T) {
	now := time.Unix(0, 1)
	info := NewFileInfo(".", 0, fs.ModeDir|0755, now, true)
	core.AssertTrue(t, info.IsDir())
	core.AssertTrue(t, info.ModTime().Equal(now))
}

func TestIo_FileInfo_Name_Good(t *core.T) {
	info := NewFileInfo("file.txt", 1, 0644, time.Time{}, false)
	got := info.Name()
	core.AssertEqual(t, "file.txt", got)
}

func TestIo_FileInfo_Name_Bad(t *core.T) {
	info := NewFileInfo("", 0, 0, time.Time{}, false)
	got := info.Name()
	core.AssertEqual(t, "", got)
}

func TestIo_FileInfo_Name_Ugly(t *core.T) {
	info := NewFileInfo(".", 0, fs.ModeDir, time.Time{}, true)
	got := info.Name()
	core.AssertEqual(t, ".", got)
}

func TestIo_FileInfo_Size_Good(t *core.T) {
	info := NewFileInfo("file.txt", 7, 0644, time.Time{}, false)
	got := info.Size()
	core.AssertEqual(t, int64(7), got)
}

func TestIo_FileInfo_Size_Bad(t *core.T) {
	info := NewFileInfo("file.txt", -1, 0644, time.Time{}, false)
	got := info.Size()
	core.AssertEqual(t, int64(-1), got)
}

func TestIo_FileInfo_Size_Ugly(t *core.T) {
	info := NewFileInfo("empty.txt", 0, 0644, time.Time{}, false)
	got := info.Size()
	core.AssertEqual(t, int64(0), got)
}

func TestIo_FileInfo_Mode_Good(t *core.T) {
	info := NewFileInfo("file.txt", 0, 0600, time.Time{}, false)
	got := info.Mode()
	core.AssertEqual(t, fs.FileMode(0600), got)
}

func TestIo_FileInfo_Mode_Bad(t *core.T) {
	info := NewFileInfo("file.txt", 0, 0, time.Time{}, false)
	got := info.Mode()
	core.AssertEqual(t, fs.FileMode(0), got)
}

func TestIo_FileInfo_Mode_Ugly(t *core.T) {
	info := NewFileInfo("dir", 0, fs.ModeDir|0755, time.Time{}, true)
	got := info.Mode()
	core.AssertTrue(t, got.IsDir())
}

func TestIo_FileInfo_ModTime_Good(t *core.T) {
	now := time.Unix(1, 0)
	info := NewFileInfo("file.txt", 0, 0644, now, false)
	got := info.ModTime()
	core.AssertTrue(t, got.Equal(now))
}

func TestIo_FileInfo_ModTime_Bad(t *core.T) {
	info := NewFileInfo("file.txt", 0, 0644, time.Time{}, false)
	got := info.ModTime()
	core.AssertTrue(t, got.IsZero())
}

func TestIo_FileInfo_ModTime_Ugly(t *core.T) {
	now := time.Unix(0, 1)
	info := NewFileInfo("file.txt", 0, 0644, now, false)
	got := info.ModTime()
	core.AssertEqual(t, 1, got.Nanosecond())
}

func TestIo_FileInfo_IsDir_Good(t *core.T) {
	info := NewFileInfo("dir", 0, fs.ModeDir|0755, time.Time{}, true)
	got := info.IsDir()
	core.AssertTrue(t, got)
}

func TestIo_FileInfo_IsDir_Bad(t *core.T) {
	info := NewFileInfo("file.txt", 0, fs.ModeDir|0755, time.Time{}, false)
	got := info.IsDir()
	core.AssertFalse(t, got)
}

func TestIo_FileInfo_IsDir_Ugly(t *core.T) {
	info := NewFileInfo("", 0, 0, time.Time{}, false)
	got := info.IsDir()
	core.AssertFalse(t, got)
}

func TestIo_FileInfo_Sys_Good(t *core.T) {
	info := NewFileInfo("file.txt", 0, 0644, time.Time{}, false)
	got := info.Sys()
	core.AssertNil(t, got)
}

func TestIo_FileInfo_Sys_Bad(t *core.T) {
	info := NewFileInfo("", 0, 0, time.Time{}, false)
	got := info.Sys()
	core.AssertNil(t, got)
}

func TestIo_FileInfo_Sys_Ugly(t *core.T) {
	info := NewFileInfo("dir", 0, fs.ModeDir, time.Time{}, true)
	got := info.Sys()
	core.AssertNil(t, got)
}

func TestIo_NewDirEntry_Good(t *core.T) {
	info := NewFileInfo("file.txt", 7, 0644, time.Time{}, false)
	entry := NewDirEntry("file.txt", false, 0644, info)
	core.AssertEqual(t, "file.txt", entry.Name())
	core.AssertFalse(t, entry.IsDir())
}

func TestIo_NewDirEntry_Bad(t *core.T) {
	entry := NewDirEntry("", false, 0, nil)
	info, err := entry.Info()
	core.AssertNoError(t, err)
	core.AssertNil(t, info)
}

func TestIo_NewDirEntry_Ugly(t *core.T) {
	info := NewFileInfo("dir", 0, fs.ModeDir|0755, time.Time{}, true)
	entry := NewDirEntry("dir", true, fs.ModeDir|0755, info)
	core.AssertTrue(t, entry.IsDir())
	core.AssertTrue(t, entry.Type().IsDir())
}

func TestIo_DirEntry_Name_Good(t *core.T) {
	entry := NewDirEntry("file.txt", false, 0644, nil)
	got := entry.Name()
	core.AssertEqual(t, "file.txt", got)
}

func TestIo_DirEntry_Name_Bad(t *core.T) {
	entry := NewDirEntry("", false, 0, nil)
	got := entry.Name()
	core.AssertEqual(t, "", got)
}

func TestIo_DirEntry_Name_Ugly(t *core.T) {
	entry := NewDirEntry("nested/path", true, fs.ModeDir, nil)
	got := entry.Name()
	core.AssertEqual(t, "nested/path", got)
}

func TestIo_DirEntry_IsDir_Good(t *core.T) {
	entry := NewDirEntry("dir", true, fs.ModeDir, nil)
	got := entry.IsDir()
	core.AssertTrue(t, got)
}

func TestIo_DirEntry_IsDir_Bad(t *core.T) {
	entry := NewDirEntry("file.txt", false, 0644, nil)
	got := entry.IsDir()
	core.AssertFalse(t, got)
}

func TestIo_DirEntry_IsDir_Ugly(t *core.T) {
	entry := NewDirEntry("", true, fs.ModeDir, nil)
	got := entry.IsDir()
	core.AssertTrue(t, got)
}

func TestIo_DirEntry_Type_Good(t *core.T) {
	entry := NewDirEntry("dir", true, fs.ModeDir|0755, nil)
	got := entry.Type()
	core.AssertTrue(t, got.IsDir())
}

func TestIo_DirEntry_Type_Bad(t *core.T) {
	entry := NewDirEntry("file.txt", false, 0644, nil)
	got := entry.Type()
	core.AssertEqual(t, fs.FileMode(0), got)
}

func TestIo_DirEntry_Type_Ugly(t *core.T) {
	entry := NewDirEntry("device", false, fs.ModeDevice, nil)
	got := entry.Type()
	core.AssertEqual(t, fs.ModeDevice, got)
}

func TestIo_DirEntry_Info_Good(t *core.T) {
	info := NewFileInfo("file.txt", 7, 0644, time.Time{}, false)
	entry := NewDirEntry("file.txt", false, 0644, info)
	got, err := entry.Info()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "file.txt", got.Name())
}

func TestIo_DirEntry_Info_Bad(t *core.T) {
	entry := NewDirEntry("file.txt", false, 0644, nil)
	got, err := entry.Info()
	core.AssertNoError(t, err)
	core.AssertNil(t, got)
}

func TestIo_DirEntry_Info_Ugly(t *core.T) {
	info := NewFileInfo("dir", 0, fs.ModeDir|0755, time.Time{}, true)
	entry := NewDirEntry("dir", true, fs.ModeDir|0755, info)
	got, err := entry.Info()
	core.AssertNoError(t, err)
	core.AssertTrue(t, got.IsDir())
}

func TestIo_NewSandboxed_Good(t *core.T) {
	medium, err := NewSandboxed(t.TempDir())
	core.AssertNoError(t, err)
	core.AssertNotNil(t, medium)
}

func TestIo_NewSandboxed_Bad(t *core.T) {
	medium, err := NewSandboxed("")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, medium)
}

func TestIo_NewSandboxed_Ugly(t *core.T) {
	root := t.TempDir()
	medium, err := NewSandboxed(root + "/missing")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, medium)
}

func TestIo_Read_Good(t *core.T) {
	medium := NewMemoryMedium()
	core.RequireNoError(t, medium.Write("read.txt", "payload"))
	got, err := Read(medium, "read.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", got)
}

func TestIo_Read_Bad(t *core.T) {
	medium := NewMemoryMedium()
	got, err := Read(medium, "missing.txt")
	core.AssertError(t, err)
	core.AssertEqual(t, "", got)
}

func TestIo_Read_Ugly(t *core.T) {
	var medium Medium
	core.AssertPanics(t, func() { _, _ = Read(medium, "x") })
	core.AssertNil(t, medium)
}

func TestIo_Write_Good(t *core.T) {
	medium := NewMemoryMedium()
	err := Write(medium, "write.txt", "payload")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("write.txt"))
}

func TestIo_Write_Bad(t *core.T) {
	var medium Medium
	core.AssertPanics(t, func() { _ = Write(medium, "write.txt", "payload") })
	core.AssertNil(t, medium)
}

func TestIo_Write_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	err := Write(medium, "empty.txt", "")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("empty.txt"))
}

func TestIo_ReadStream_Good(t *core.T) {
	medium := NewMemoryMedium()
	core.RequireNoError(t, medium.Write("stream.txt", "payload"))
	reader, err := ReadStream(medium, "stream.txt")
	core.RequireNoError(t, err)
	defer reader.Close()
	data, readErr := goio.ReadAll(reader)
	core.AssertNoError(t, readErr)
	core.AssertEqual(t, "payload", string(data))
}

func TestIo_ReadStream_Bad(t *core.T) {
	medium := NewMemoryMedium()
	reader, err := ReadStream(medium, "missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, reader)
}

func TestIo_ReadStream_Ugly(t *core.T) {
	var medium Medium
	core.AssertPanics(t, func() { _, _ = ReadStream(medium, "x") })
	core.AssertNil(t, medium)
}

func TestIo_WriteStream_Good(t *core.T) {
	medium := NewMemoryMedium()
	writer, err := WriteStream(medium, "stream.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestIo_WriteStream_Bad(t *core.T) {
	var medium Medium
	core.AssertPanics(t, func() { _, _ = WriteStream(medium, "x") })
	core.AssertNil(t, medium)
}

func TestIo_WriteStream_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	writer, err := WriteStream(medium, "empty.txt")
	core.RequireNoError(t, err)
	core.AssertNoError(t, writer.Close())
	core.AssertTrue(t, medium.Exists("empty.txt"))
}

func TestIo_EnsureDir_Good(t *core.T) {
	medium := NewMemoryMedium()
	err := EnsureDir(medium, "dir")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir("dir"))
}

func TestIo_EnsureDir_Bad(t *core.T) {
	var medium Medium
	core.AssertPanics(t, func() { _ = EnsureDir(medium, "dir") })
	core.AssertNil(t, medium)
}

func TestIo_EnsureDir_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	err := EnsureDir(medium, "a/b/c")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir("a/b/c"))
}

func TestIo_IsFile_Good(t *core.T) {
	medium := NewMemoryMedium()
	core.RequireNoError(t, medium.Write("file.txt", "payload"))
	got := IsFile(medium, "file.txt")
	core.AssertTrue(t, got)
}

func TestIo_IsFile_Bad(t *core.T) {
	var medium Medium
	core.AssertPanics(t, func() { _ = IsFile(medium, "file.txt") })
	core.AssertNil(t, medium)
}

func TestIo_IsFile_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := IsFile(medium, "dir")
	core.AssertFalse(t, got)
}

func TestIo_Copy_Good(t *core.T) {
	source := NewMemoryMedium()
	destination := NewMemoryMedium()
	core.RequireNoError(t, source.WriteMode("source.txt", "payload", 0600))

	err := Copy(source, "source.txt", destination, "destination.txt")
	core.RequireNoError(t, err)

	content, err := destination.Read("destination.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "payload", content)
	info, err := destination.Stat("destination.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, fs.FileMode(0600), info.Mode())
}

func TestIo_Copy_Bad(t *core.T) {
	source := NewMemoryMedium()
	destination := NewMemoryMedium()

	err := Copy(source, "missing.txt", destination, "destination.txt")

	core.AssertError(t, err)
	core.AssertFalse(t, destination.Exists("destination.txt"))
}

func TestIo_Copy_Ugly(t *core.T) {
	source := NewMemoryMedium()
	destination := NewMemoryMedium()
	core.RequireNoError(t, source.Write("", "root-payload"))

	err := Copy(source, "", destination, "")
	core.RequireNoError(t, err)

	content, err := destination.Read("")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "root-payload", content)
}

func TestIo_NewMemoryMedium_Good(t *core.T) {
	medium := NewMemoryMedium()
	core.AssertNotNil(t, medium)
	core.AssertNotNil(t, medium.fileContents)
}

func TestIo_NewMemoryMedium_Bad(t *core.T) {
	first := NewMemoryMedium()
	second := NewMemoryMedium()
	core.RequireNoError(t, first.Write("only-first.txt", "payload"))
	core.AssertFalse(t, second.Exists("only-first.txt"))
}

func TestIo_NewMemoryMedium_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	core.RequireNoError(t, medium.EnsureDir("a/b/c"))
	core.AssertTrue(t, medium.IsDir("a/b/c"))
}

func TestIo_MemoryMedium_Read_Good(t *core.T) {
	medium := NewMemoryMedium()
	core.RequireNoError(t, medium.Write("read.txt", "payload"))
	got, err := medium.Read("read.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", got)
}

func TestIo_MemoryMedium_Read_Bad(t *core.T) {
	medium := NewMemoryMedium()
	got, err := medium.Read("missing.txt")
	core.AssertError(t, err)
	core.AssertEqual(t, "", got)
}

func TestIo_MemoryMedium_Read_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	core.RequireNoError(t, medium.Write("empty.txt", ""))
	got, err := medium.Read("empty.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "", got)
}

func TestIo_MemoryMedium_Write_Good(t *core.T) {
	medium := NewMemoryMedium()
	err := medium.Write("write.txt", "payload")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("write.txt"))
}

func TestIo_MemoryMedium_Write_Bad(t *core.T) {
	medium := NewMemoryMedium()
	core.RequireNoError(t, medium.EnsureDir("dir"))
	err := medium.Write("dir", "payload")
	core.AssertError(t, err)
}

func TestIo_MemoryMedium_Write_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	err := medium.Write("nested/write.txt", "payload")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir("nested"))
}

func TestIo_MemoryMedium_WriteMode_Good(t *core.T) {
	medium := NewMemoryMedium()
	err := medium.WriteMode("mode.txt", "payload", 0600)
	info, statErr := medium.Stat("mode.txt")
	core.AssertNoError(t, err)
	core.AssertNoError(t, statErr)
	core.AssertEqual(t, fs.FileMode(0600), info.Mode())
}

func TestIo_MemoryMedium_WriteMode_Bad(t *core.T) {
	medium := NewMemoryMedium()
	core.RequireNoError(t, medium.EnsureDir("dir"))
	err := medium.WriteMode("dir", "payload", 0600)
	core.AssertError(t, err)
}

func TestIo_MemoryMedium_WriteMode_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	err := medium.WriteMode("zero-mode.txt", "payload", 0)
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("zero-mode.txt"))
}

func TestIo_MemoryMedium_EnsureDir_Good(t *core.T) {
	medium := NewMemoryMedium()
	err := medium.EnsureDir("dir")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir("dir"))
}

func TestIo_MemoryMedium_EnsureDir_Bad(t *core.T) {
	medium := NewMemoryMedium()
	core.RequireNoError(t, medium.Write("file.txt", "payload"))
	err := medium.EnsureDir("file.txt")
	core.AssertError(t, err)
}

func TestIo_MemoryMedium_EnsureDir_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	err := medium.EnsureDir("a/b/c")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir("a/b/c"))
}

func TestIo_MemoryMedium_IsFile_Good(t *core.T) {
	medium := NewMemoryMedium()
	core.RequireNoError(t, medium.Write("file.txt", "payload"))
	got := medium.IsFile("file.txt")
	core.AssertTrue(t, got)
}

func TestIo_MemoryMedium_IsFile_Bad(t *core.T) {
	medium := NewMemoryMedium()
	got := medium.IsFile("missing.txt")
	core.AssertFalse(t, got)
}

func TestIo_MemoryMedium_IsFile_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.IsFile("dir")
	core.AssertFalse(t, got)
}

func TestIo_MemoryMedium_Delete_Good(t *core.T) {
	medium := NewMemoryMedium()
	core.RequireNoError(t, medium.Write("delete.txt", "payload"))
	err := medium.Delete("delete.txt")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("delete.txt"))
}

func TestIo_MemoryMedium_Delete_Bad(t *core.T) {
	medium := NewMemoryMedium()
	err := medium.Delete("missing.txt")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("missing.txt"))
}

func TestIo_MemoryMedium_Delete_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	core.RequireNoError(t, medium.EnsureDir("empty"))
	err := medium.Delete("empty")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("empty"))
}

func TestIo_MemoryMedium_DeleteAll_Good(t *core.T) {
	medium := NewMemoryMedium()
	core.RequireNoError(t, medium.Write("tree/file.txt", "payload"))
	err := medium.DeleteAll("tree")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("tree/file.txt"))
}

func TestIo_MemoryMedium_DeleteAll_Bad(t *core.T) {
	medium := NewMemoryMedium()
	err := medium.DeleteAll("missing")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("missing"))
}

func TestIo_MemoryMedium_DeleteAll_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	err := medium.DeleteAll("")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("anything"))
}

func TestIo_MemoryMedium_Rename_Good(t *core.T) {
	medium := NewMemoryMedium()
	core.RequireNoError(t, medium.Write("old.txt", "payload"))
	err := medium.Rename("old.txt", "new.txt")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("new.txt"))
}

func TestIo_MemoryMedium_Rename_Bad(t *core.T) {
	medium := NewMemoryMedium()
	err := medium.Rename("missing.txt", "new.txt")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("new.txt"))
}

func TestIo_MemoryMedium_Rename_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	core.RequireNoError(t, medium.Write("dir/old.txt", "payload"))
	err := medium.Rename("dir", "moved")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("moved/old.txt"))
}

func TestIo_MemoryMedium_Open_Good(t *core.T) {
	medium := NewMemoryMedium()
	core.RequireNoError(t, medium.Write("open.txt", "payload"))
	file, err := medium.Open("open.txt")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, file)
	core.RequireNoError(t, file.Close())
}

func TestIo_MemoryMedium_Open_Bad(t *core.T) {
	medium := NewMemoryMedium()
	file, err := medium.Open("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, file)
}

func TestIo_MemoryMedium_Open_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	core.RequireNoError(t, medium.WriteMode("open.txt", "", 0))
	file, err := medium.Open("open.txt")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, file)
}

func TestIo_MemoryMedium_Create_Good(t *core.T) {
	medium := NewMemoryMedium()
	writer, err := medium.Create("create.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestIo_MemoryMedium_Create_Bad(t *core.T) {
	medium := NewMemoryMedium()
	writer, err := medium.Create("")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, writer)
}

func TestIo_MemoryMedium_Create_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	writer, err := medium.Create("nested/create.txt")
	core.RequireNoError(t, err)
	core.AssertNoError(t, writer.Close())
	core.AssertTrue(t, medium.Exists("nested/create.txt"))
}

func TestIo_MemoryMedium_Append_Good(t *core.T) {
	medium := NewMemoryMedium()
	core.RequireNoError(t, medium.Write("append.txt", "a"))
	writer, err := medium.Append("append.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("b"))
	core.RequireNoError(t, writeErr)
	core.RequireNoError(t, writer.Close())
}

func TestIo_MemoryMedium_Append_Bad(t *core.T) {
	medium := NewMemoryMedium()
	writer, err := medium.Append("")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, writer)
}

func TestIo_MemoryMedium_Append_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	writer, err := medium.Append("new.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("created"))
	core.RequireNoError(t, writeErr)
	core.RequireNoError(t, writer.Close())
}

func TestIo_MemoryMedium_ReadStream_Good(t *core.T) {
	medium := NewMemoryMedium()
	core.RequireNoError(t, medium.Write("stream.txt", "payload"))
	reader, err := medium.ReadStream("stream.txt")
	core.RequireNoError(t, err)
	defer reader.Close()
	data, readErr := goio.ReadAll(reader)
	core.AssertNoError(t, readErr)
	core.AssertEqual(t, "payload", string(data))
}

func TestIo_MemoryMedium_ReadStream_Bad(t *core.T) {
	medium := NewMemoryMedium()
	reader, err := medium.ReadStream("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, reader)
}

func TestIo_MemoryMedium_ReadStream_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	core.RequireNoError(t, medium.Write("empty.txt", ""))
	reader, err := medium.ReadStream("empty.txt")
	core.RequireNoError(t, err)
	core.AssertNoError(t, reader.Close())
}

func TestIo_MemoryMedium_WriteStream_Good(t *core.T) {
	medium := NewMemoryMedium()
	writer, err := medium.WriteStream("stream.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestIo_MemoryMedium_WriteStream_Bad(t *core.T) {
	medium := NewMemoryMedium()
	writer, err := medium.WriteStream("")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, writer)
}

func TestIo_MemoryMedium_WriteStream_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	writer, err := medium.WriteStream("nested/stream.txt")
	core.RequireNoError(t, err)
	core.AssertNoError(t, writer.Close())
	core.AssertTrue(t, medium.Exists("nested/stream.txt"))
}

func TestIo_MemoryMedium_List_Good(t *core.T) {
	medium := NewMemoryMedium()
	core.RequireNoError(t, medium.Write("dir/a.txt", "payload"))
	entries, err := medium.List("dir")
	core.AssertNoError(t, err)
	core.AssertLen(t, entries, 1)
}

func TestIo_MemoryMedium_List_Bad(t *core.T) {
	medium := NewMemoryMedium()
	entries, err := medium.List("missing")
	core.AssertError(t, err)
	core.AssertNil(t, entries)
}

func TestIo_MemoryMedium_List_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	entries, err := medium.List("")
	core.AssertNoError(t, err)
	core.AssertEmpty(t, entries)
}

func TestIo_MemoryMedium_Stat_Good(t *core.T) {
	medium := NewMemoryMedium()
	core.RequireNoError(t, medium.Write("stat.txt", "payload"))
	info, err := medium.Stat("stat.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "stat.txt", info.Name())
}

func TestIo_MemoryMedium_Stat_Bad(t *core.T) {
	medium := NewMemoryMedium()
	info, err := medium.Stat("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, info)
}

func TestIo_MemoryMedium_Stat_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	core.RequireNoError(t, medium.EnsureDir("dir"))
	info, err := medium.Stat("dir")
	core.AssertNoError(t, err)
	core.AssertTrue(t, info.IsDir())
}

func TestIo_MemoryMedium_Exists_Good(t *core.T) {
	medium := NewMemoryMedium()
	core.RequireNoError(t, medium.Write("exists.txt", "payload"))
	got := medium.Exists("exists.txt")
	core.AssertTrue(t, got)
}

func TestIo_MemoryMedium_Exists_Bad(t *core.T) {
	medium := NewMemoryMedium()
	got := medium.Exists("missing.txt")
	core.AssertFalse(t, got)
}

func TestIo_MemoryMedium_Exists_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.Exists("dir")
	core.AssertTrue(t, got)
}

func TestIo_MemoryMedium_IsDir_Good(t *core.T) {
	medium := NewMemoryMedium()
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.IsDir("dir")
	core.AssertTrue(t, got)
}

func TestIo_MemoryMedium_IsDir_Bad(t *core.T) {
	medium := NewMemoryMedium()
	got := medium.IsDir("missing")
	core.AssertFalse(t, got)
}

func TestIo_MemoryMedium_IsDir_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	core.RequireNoError(t, medium.Write("file.txt", "payload"))
	got := medium.IsDir("file.txt")
	core.AssertFalse(t, got)
}

func TestIo_MemoryFile_Stat_Good(t *core.T) {
	file := &MemoryFile{name: "file.txt", content: []byte("payload"), mode: 0600}
	info, err := file.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "file.txt", info.Name())
}

func TestIo_MemoryFile_Stat_Bad(t *core.T) {
	file := &MemoryFile{}
	info, err := file.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "", info.Name())
}

func TestIo_MemoryFile_Stat_Ugly(t *core.T) {
	file := &MemoryFile{name: "empty.txt"}
	info, err := file.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, int64(0), info.Size())
}

func TestIo_MemoryFile_Read_Good(t *core.T) {
	file := &MemoryFile{content: []byte("payload")}
	buffer := make([]byte, 7)
	count, err := file.Read(buffer)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 7, count)
}

func TestIo_MemoryFile_Read_Bad(t *core.T) {
	file := &MemoryFile{content: []byte("x"), offset: 1}
	buffer := make([]byte, 1)
	count, err := file.Read(buffer)
	core.AssertErrorIs(t, err, goio.EOF)
	core.AssertEqual(t, 0, count)
}

func TestIo_MemoryFile_Read_Ugly(t *core.T) {
	file := &MemoryFile{content: []byte("payload")}
	buffer := make([]byte, 3)
	count, err := file.Read(buffer)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "pay", string(buffer[:count]))
}

func TestIo_MemoryFile_Close_Good(t *core.T) {
	file := &MemoryFile{name: "file.txt"}
	err := file.Close()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "file.txt", file.name)
}

func TestIo_MemoryFile_Close_Bad(t *core.T) {
	file := &MemoryFile{}
	err := file.Close()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "", file.name)
}

func TestIo_MemoryFile_Close_Ugly(t *core.T) {
	file := &MemoryFile{offset: 99}
	err := file.Close()
	core.AssertNoError(t, err)
	core.AssertEqual(t, int64(99), file.offset)
}

func TestIo_MemoryWriteCloser_Write_Good(t *core.T) {
	writer := &MemoryWriteCloser{}
	count, err := writer.Write([]byte("payload"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, len("payload"), count)
}

func TestIo_MemoryWriteCloser_Write_Bad(t *core.T) {
	writer := &MemoryWriteCloser{}
	count, err := writer.Write(nil)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, count)
}

func TestIo_MemoryWriteCloser_Write_Ugly(t *core.T) {
	writer := &MemoryWriteCloser{data: []byte("a")}
	count, err := writer.Write([]byte("b"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, 1, count)
}

func TestIo_MemoryWriteCloser_Close_Good(t *core.T) {
	medium := NewMemoryMedium()
	writer := &MemoryWriteCloser{medium: medium, path: "close.txt", data: []byte("payload")}
	err := writer.Close()
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("close.txt"))
}

func TestIo_MemoryWriteCloser_Close_Bad(t *core.T) {
	medium := NewMemoryMedium()
	core.RequireNoError(t, medium.EnsureDir("dir"))
	writer := &MemoryWriteCloser{medium: medium, path: "dir", data: []byte("payload")}
	err := writer.Close()
	core.AssertError(t, err)
}

func TestIo_MemoryWriteCloser_Close_Ugly(t *core.T) {
	medium := NewMemoryMedium()
	writer := &MemoryWriteCloser{medium: medium, path: "nested/empty.txt"}
	err := writer.Close()
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("nested/empty.txt"))
}
