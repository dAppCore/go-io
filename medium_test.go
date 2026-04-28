package io

import (
	core "dappco.re/go"
	goio "io"
	"io/fs"
	"time"
)

func TestMemoryMedium_NewMemoryMedium_Good(t *core.T) {
	memoryMedium := NewMemoryMedium()
	core.AssertNotNil(t, memoryMedium)
	core.AssertNotNil(t, memoryMedium.fileContents)
	core.AssertNotNil(t, memoryMedium.directories)
	core.AssertEmpty(t, memoryMedium.fileContents)
	core.AssertEmpty(t, memoryMedium.directories)
}

func TestMemoryMedium_NewFileInfo_Good(t *core.T) {
	info := NewFileInfo("app.yaml", 8, 0644, time.Unix(0, 0), false)

	core.AssertEqual(t, "app.yaml", info.Name())
	core.AssertEqual(t, int64(8), info.Size())
	core.AssertEqual(t, fs.FileMode(0644), info.Mode())
	core.AssertTrue(t, info.ModTime().Equal(time.Unix(0, 0)))
	core.AssertFalse(t, info.IsDir())
	core.AssertNil(t, info.Sys())
}

func TestMemoryMedium_NewDirEntry_Good(t *core.T) {
	info := NewFileInfo("app.yaml", 8, 0644, time.Unix(0, 0), false)
	entry := NewDirEntry("app.yaml", false, 0644, info)

	core.AssertEqual(t, "app.yaml", entry.Name())
	core.AssertFalse(t, entry.IsDir())
	core.AssertEqual(t, fs.FileMode(0), entry.Type())

	entryInfo, err := entry.Info()
	core.RequireNoError(t, err)
	core.AssertEqual(t, "app.yaml", entryInfo.Name())
	core.AssertEqual(t, int64(8), entryInfo.Size())
}

func TestMemoryMedium_Read_Good(t *core.T) {
	memoryMedium := NewMemoryMedium()
	memoryMedium.fileContents["test.txt"] = "hello world"
	content, err := memoryMedium.Read("test.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "hello world", content)
}

func TestMemoryMedium_Read_Bad(t *core.T) {
	memoryMedium := NewMemoryMedium()
	_, err := memoryMedium.Read("nonexistent.txt")
	core.AssertError(t, err)
}

func TestMemoryMedium_Write_Good(t *core.T) {
	memoryMedium := NewMemoryMedium()
	err := memoryMedium.Write("test.txt", "content")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "content", memoryMedium.fileContents["test.txt"])

	err = memoryMedium.Write("test.txt", "new content")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "new content", memoryMedium.fileContents["test.txt"])
}

func TestMemoryMedium_WriteMode_Good(t *core.T) {
	memoryMedium := NewMemoryMedium()

	err := memoryMedium.WriteMode("secure.txt", "secret", 0600)
	core.RequireNoError(t, err)

	content, err := memoryMedium.Read("secure.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "secret", content)

	info, err := memoryMedium.Stat("secure.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, fs.FileMode(0600), info.Mode())

	file, err := memoryMedium.Open("secure.txt")
	core.RequireNoError(t, err)
	fileInfo, err := file.Stat()
	core.RequireNoError(t, err)
	core.AssertEqual(t, fs.FileMode(0600), fileInfo.Mode())
}

func TestMemoryMedium_EnsureDir_Good(t *core.T) {
	memoryMedium := NewMemoryMedium()
	err := memoryMedium.EnsureDir("/path/to/dir")
	core.AssertNoError(t, err)
	core.AssertTrue(t, memoryMedium.directories["/path/to/dir"])
}

func TestMemoryMedium_EnsureDir_CreatesParents_Good(t *core.T) {
	memoryMedium := NewMemoryMedium()

	core.RequireNoError(t, memoryMedium.EnsureDir("alpha/beta/gamma"))

	core.AssertTrue(t, memoryMedium.IsDir("alpha"))
	core.AssertTrue(t, memoryMedium.IsDir("alpha/beta"))
	core.AssertTrue(t, memoryMedium.IsDir("alpha/beta/gamma"))
}

func TestMemoryMedium_IsFile_Good(t *core.T) {
	memoryMedium := NewMemoryMedium()
	memoryMedium.fileContents["exists.txt"] = "content"

	core.AssertTrue(t, memoryMedium.IsFile("exists.txt"))
	core.AssertFalse(t, memoryMedium.IsFile("nonexistent.txt"))
}

func TestMemoryMedium_Write_CreatesParentDirectories_Good(t *core.T) {
	memoryMedium := NewMemoryMedium()

	core.RequireNoError(t, memoryMedium.Write("nested/path/file.txt", "content"))

	core.AssertTrue(t, memoryMedium.Exists("nested"))
	core.AssertTrue(t, memoryMedium.IsDir("nested"))
	core.AssertTrue(t, memoryMedium.Exists("nested/path"))
	core.AssertTrue(t, memoryMedium.IsDir("nested/path"))
}

func TestMemoryMedium_Delete_Good(t *core.T) {
	memoryMedium := NewMemoryMedium()
	memoryMedium.fileContents["test.txt"] = "content"

	err := memoryMedium.Delete("test.txt")
	core.AssertNoError(t, err)
	core.AssertFalse(t, memoryMedium.IsFile("test.txt"))
}

func TestMemoryMedium_Delete_NotFound_Bad(t *core.T) {
	memoryMedium := NewMemoryMedium()
	err := memoryMedium.Delete("nonexistent.txt")
	core.AssertError(t, err)
}

func TestMemoryMedium_Delete_DirNotEmpty_Bad(t *core.T) {
	memoryMedium := NewMemoryMedium()
	memoryMedium.directories["mydir"] = true
	memoryMedium.fileContents["mydir/file.txt"] = "content"

	err := memoryMedium.Delete("mydir")
	core.AssertError(t, err)
}

func TestMemoryMedium_Delete_InferredDirNotEmpty_Bad(t *core.T) {
	memoryMedium := NewMemoryMedium()

	core.RequireNoError(t, memoryMedium.Write("mydir/file.txt", "content"))

	err := memoryMedium.Delete("mydir")
	core.AssertError(t, err)
}

func TestMemoryMedium_DeleteAll_Good(t *core.T) {
	memoryMedium := NewMemoryMedium()
	memoryMedium.directories["mydir"] = true
	memoryMedium.directories["mydir/subdir"] = true
	memoryMedium.fileContents["mydir/file.txt"] = "content"
	memoryMedium.fileContents["mydir/subdir/nested.txt"] = "nested"

	err := memoryMedium.DeleteAll("mydir")
	core.AssertNoError(t, err)
	core.AssertEmpty(t, memoryMedium.directories)
	core.AssertEmpty(t, memoryMedium.fileContents)
}

func TestMemoryMedium_Rename_Good(t *core.T) {
	memoryMedium := NewMemoryMedium()
	memoryMedium.fileContents["old.txt"] = "content"

	err := memoryMedium.Rename("old.txt", "new.txt")
	core.AssertNoError(t, err)
	core.AssertFalse(t, memoryMedium.IsFile("old.txt"))
	core.AssertTrue(t, memoryMedium.IsFile("new.txt"))
	core.AssertEqual(t, "content", memoryMedium.fileContents["new.txt"])
}

func TestMemoryMedium_Rename_Dir_Good(t *core.T) {
	memoryMedium := NewMemoryMedium()
	memoryMedium.directories["olddir"] = true
	memoryMedium.fileContents["olddir/file.txt"] = "content"

	err := memoryMedium.Rename("olddir", "newdir")
	core.AssertNoError(t, err)
	core.AssertFalse(t, memoryMedium.directories["olddir"])
	core.AssertTrue(t, memoryMedium.directories["newdir"])
	core.AssertEqual(t, "content", memoryMedium.fileContents["newdir/file.txt"])
}

func TestMemoryMedium_Rename_InferredDir_Good(t *core.T) {
	memoryMedium := NewMemoryMedium()
	core.RequireNoError(t, memoryMedium.Write("olddir/file.txt", "content"))

	core.RequireNoError(t, memoryMedium.Rename("olddir", "newdir"))

	core.AssertFalse(t, memoryMedium.Exists("olddir"))
	core.AssertTrue(t, memoryMedium.Exists("newdir"))
	core.AssertTrue(t, memoryMedium.IsDir("newdir"))
	core.AssertEqual(t, "content", memoryMedium.fileContents["newdir/file.txt"])
}

func TestMemoryMedium_List_Good(t *core.T) {
	memoryMedium := NewMemoryMedium()
	memoryMedium.directories["mydir"] = true
	memoryMedium.fileContents["mydir/file1.txt"] = "content1"
	memoryMedium.fileContents["mydir/file2.txt"] = "content2"
	memoryMedium.directories["mydir/subdir"] = true

	entries, err := memoryMedium.List("mydir")
	core.AssertNoError(t, err)
	core.AssertLen(t, entries, 3)
	core.AssertEqual(t, "file1.txt", entries[0].Name())
	core.AssertEqual(t, "file2.txt", entries[1].Name())
	core.AssertEqual(t, "subdir", entries[2].Name())

	names := make(map[string]bool)
	for _, entry := range entries {
		names[entry.Name()] = true
	}
	core.AssertTrue(t, names["file1.txt"])
	core.AssertTrue(t, names["file2.txt"])
	core.AssertTrue(t, names["subdir"])
}

func TestMemoryMedium_Stat_Good(t *core.T) {
	memoryMedium := NewMemoryMedium()
	memoryMedium.fileContents["test.txt"] = "hello world"

	info, err := memoryMedium.Stat("test.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "test.txt", info.Name())
	core.AssertEqual(t, int64(11), info.Size())
	core.AssertFalse(t, info.IsDir())
}

func TestMemoryMedium_Stat_Dir_Good(t *core.T) {
	memoryMedium := NewMemoryMedium()
	memoryMedium.directories["mydir"] = true

	info, err := memoryMedium.Stat("mydir")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "mydir", info.Name())
	core.AssertTrue(t, info.IsDir())
}

func TestMemoryMedium_Exists_Good(t *core.T) {
	memoryMedium := NewMemoryMedium()
	memoryMedium.fileContents["file.txt"] = "content"
	memoryMedium.directories["mydir"] = true

	core.AssertTrue(t, memoryMedium.Exists("file.txt"))
	core.AssertTrue(t, memoryMedium.Exists("mydir"))
	core.AssertFalse(t, memoryMedium.Exists("nonexistent"))
}

func TestMemoryMedium_IsDir_Good(t *core.T) {
	memoryMedium := NewMemoryMedium()
	memoryMedium.fileContents["file.txt"] = "content"
	memoryMedium.directories["mydir"] = true

	core.AssertFalse(t, memoryMedium.IsDir("file.txt"))
	core.AssertTrue(t, memoryMedium.IsDir("mydir"))
	core.AssertFalse(t, memoryMedium.IsDir("nonexistent"))
}

func TestMemoryMedium_StreamAndFSHelpers_Good(t *core.T) {
	memoryMedium := NewMemoryMedium()
	core.RequireNoError(t, memoryMedium.EnsureDir("dir"))
	core.RequireNoError(t, memoryMedium.Write("dir/file.txt", "alpha"))

	statInfo, err := memoryMedium.Stat("dir/file.txt")
	core.RequireNoError(t, err)

	file, err := memoryMedium.Open("dir/file.txt")
	core.RequireNoError(t, err)

	info, err := file.Stat()
	core.RequireNoError(t, err)
	core.AssertEqual(t, "file.txt", info.Name())
	core.AssertEqual(t, int64(5), info.Size())
	core.AssertEqual(t, fs.FileMode(0644), info.Mode())
	core.AssertEqual(t, statInfo.ModTime(), info.ModTime())
	core.AssertFalse(t, info.IsDir())
	core.AssertNil(t, info.Sys())

	data, err := goio.ReadAll(file)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "alpha", string(data))
	core.RequireNoError(t, file.Close())

	entries, err := memoryMedium.List("dir")
	core.RequireNoError(t, err)
	core.AssertLen(t, entries, 1)
	core.AssertEqual(t, "file.txt", entries[0].Name())
	core.AssertFalse(t, entries[0].IsDir())
	core.AssertEqual(t, fs.FileMode(0), entries[0].Type())

	entryInfo, err := entries[0].Info()
	core.RequireNoError(t, err)
	core.AssertEqual(t, "file.txt", entryInfo.Name())
	core.AssertEqual(t, int64(5), entryInfo.Size())
	core.AssertEqual(t, fs.FileMode(0644), entryInfo.Mode())
	core.AssertEqual(t, statInfo.ModTime(), entryInfo.ModTime())

	writer, err := memoryMedium.Create("created.txt")
	core.RequireNoError(t, err)
	_, err = writer.Write([]byte("created"))
	core.RequireNoError(t, err)
	core.RequireNoError(t, writer.Close())

	appendWriter, err := memoryMedium.Append("created.txt")
	core.RequireNoError(t, err)
	_, err = appendWriter.Write([]byte(" later"))
	core.RequireNoError(t, err)
	core.RequireNoError(t, appendWriter.Close())

	reader, err := memoryMedium.ReadStream("created.txt")
	core.RequireNoError(t, err)
	streamed, err := goio.ReadAll(reader)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "created later", string(streamed))
	core.RequireNoError(t, reader.Close())

	writeStream, err := memoryMedium.WriteStream("streamed.txt")
	core.RequireNoError(t, err)
	_, err = writeStream.Write([]byte("stream output"))
	core.RequireNoError(t, err)
	core.RequireNoError(t, writeStream.Close())

	core.AssertEqual(t, "stream output", memoryMedium.fileContents["streamed.txt"])
	statInfo, err = memoryMedium.Stat("streamed.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, fs.FileMode(0644), statInfo.Mode())
	core.AssertFalse(t, statInfo.ModTime().IsZero())
}

func TestIO_Read_Good(t *core.T) {
	memoryMedium := NewMemoryMedium()
	memoryMedium.fileContents["test.txt"] = "hello"
	content, err := Read(memoryMedium, "test.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "hello", content)
}

func TestIO_Write_Good(t *core.T) {
	memoryMedium := NewMemoryMedium()
	err := Write(memoryMedium, "test.txt", "hello")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "hello", memoryMedium.fileContents["test.txt"])
}

func TestIO_EnsureDir_Good(t *core.T) {
	memoryMedium := NewMemoryMedium()
	err := EnsureDir(memoryMedium, "/my/dir")
	core.AssertNoError(t, err)
	core.AssertTrue(t, memoryMedium.directories["/my/dir"])
}

func TestIO_IsFile_Good(t *core.T) {
	memoryMedium := NewMemoryMedium()
	memoryMedium.fileContents["exists.txt"] = "content"

	core.AssertTrue(t, IsFile(memoryMedium, "exists.txt"))
	core.AssertFalse(t, IsFile(memoryMedium, "nonexistent.txt"))
}

func TestIO_NewSandboxed_Good(t *core.T) {
	root := t.TempDir()

	memoryMedium, err := NewSandboxed(root)
	core.RequireNoError(t, err)

	core.RequireNoError(t, memoryMedium.Write("config/app.yaml", "port: 8080"))

	content, err := memoryMedium.Read("config/app.yaml")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "port: 8080", content)
	core.AssertTrue(t, memoryMedium.IsDir("config"))
}

func TestIO_ReadWriteStream_Good(t *core.T) {
	memoryMedium := NewMemoryMedium()

	writer, err := WriteStream(memoryMedium, "logs/run.txt")
	core.RequireNoError(t, err)
	_, err = writer.Write([]byte("started"))
	core.RequireNoError(t, err)
	core.RequireNoError(t, writer.Close())

	reader, err := ReadStream(memoryMedium, "logs/run.txt")
	core.RequireNoError(t, err)
	data, err := goio.ReadAll(reader)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "started", string(data))
	core.RequireNoError(t, reader.Close())
}

func TestIO_Copy_Good(t *core.T) {
	source := NewMemoryMedium()
	dest := NewMemoryMedium()
	source.fileContents["test.txt"] = "hello"
	err := Copy(source, "test.txt", dest, "test.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "hello", dest.fileContents["test.txt"])

	source.fileContents["original.txt"] = "content"
	err = Copy(source, "original.txt", dest, "copied.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "content", dest.fileContents["copied.txt"])
}

func TestIO_Copy_Bad(t *core.T) {
	source := NewMemoryMedium()
	dest := NewMemoryMedium()
	err := Copy(source, "nonexistent.txt", dest, "dest.txt")
	core.AssertError(t, err)
}

func TestIO_LocalGlobal_Good(t *core.T) {
	core.AssertNotNil(t, Local, "io.Local should be initialised")

	var localMedium = Local
	core.AssertNotNil(t, localMedium)
}
