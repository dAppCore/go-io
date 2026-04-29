package webdav

import (
	core "dappco.re/go"
	xwebdav "golang.org/x/net/webdav"
	goio "io"
	"io/fs"
	"net/http/httptest"
	"time"
)

func newWebDAVTestMedium(t *core.T) *Medium {
	t.Helper()

	handler := &xwebdav.Handler{
		FileSystem: xwebdav.NewMemFS(),
		LockSystem: xwebdav.NewMemLS(),
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	medium, err := New(Options{BaseURL: server.URL})
	core.RequireNoError(t, err)
	return medium
}

func TestWebDAVMedium_Read_Good(t *core.T) {
	medium := newWebDAVTestMedium(t)

	core.RequireNoError(t, medium.Write("notes/read.txt", "hello webdav"))

	content, err := medium.Read("notes/read.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "hello webdav", content)
}

func TestWebDAVMedium_Read_Bad(t *core.T) {
	medium := newWebDAVTestMedium(t)

	_, err := medium.Read("missing.txt")

	core.AssertError(t, err)
}

func TestWebDAVMedium_Read_Ugly(t *core.T) {
	medium := newWebDAVTestMedium(t)

	core.RequireNoError(t, medium.Write("safe/file.txt", "normalised"))

	content, err := medium.Read("/safe/../safe/./file.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "normalised", content)
}

func TestWebDAVMedium_Write_Good(t *core.T) {
	medium := newWebDAVTestMedium(t)

	err := medium.Write("nested/path/file.txt", "content")
	core.RequireNoError(t, err)

	core.AssertTrue(t, medium.IsFile("nested/path/file.txt"))
	content, err := medium.Read("nested/path/file.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "content", content)
}

func TestWebDAVMedium_Write_Bad(t *core.T) {
	medium := newWebDAVTestMedium(t)

	err := medium.Write("", "content")

	core.AssertError(t, err)
}

func TestWebDAVMedium_Write_Ugly(t *core.T) {
	medium := newWebDAVTestMedium(t)

	core.RequireNoError(t, medium.Write("../escaped.txt", "contained"))

	content, err := medium.Read("escaped.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "contained", content)
}

func TestWebDAVMedium_List_Good(t *core.T) {
	medium := newWebDAVTestMedium(t)

	core.RequireNoError(t, medium.Write("dir/b.txt", "b"))
	core.RequireNoError(t, medium.Write("dir/a.txt", "a"))
	core.RequireNoError(t, medium.EnsureDir("dir/sub"))

	entries, err := medium.List("dir")
	core.RequireNoError(t, err)

	core.AssertLen(t, entries, 3)
	core.AssertEqual(t, "a.txt", entries[0].Name())
	core.AssertEqual(t, "b.txt", entries[1].Name())
	core.AssertEqual(t, "sub", entries[2].Name())
	core.AssertTrue(t, entries[2].IsDir())
}

func TestWebDAVMedium_List_Bad(t *core.T) {
	medium := newWebDAVTestMedium(t)

	_, err := medium.List("missing")

	core.AssertError(t, err)
}

func TestWebDAVMedium_List_Ugly(t *core.T) {
	medium := newWebDAVTestMedium(t)

	core.RequireNoError(t, medium.Write("dir/file.txt", "content"))

	entries, err := medium.List("//dir/../dir/.")
	core.RequireNoError(t, err)

	core.AssertLen(t, entries, 1)
	core.AssertEqual(t, "file.txt", entries[0].Name())
}

func TestWebdav_New_Good(t *core.T) {
	handler := &xwebdav.Handler{
		FileSystem: xwebdav.NewMemFS(),
		LockSystem: xwebdav.NewMemLS(),
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	medium, err := New(Options{BaseURL: server.URL})
	core.RequireNoError(t, err)

	core.AssertNotNil(t, medium)
	core.AssertNotNil(t, medium.client)
}

func TestWebdav_New_Bad(t *core.T) {
	medium, err := New(Options{})
	core.AssertError(t, err)
	core.AssertNil(t, medium)
}

func TestWebdav_New_Ugly(t *core.T) {
	medium, err := New(Options{BaseURL: "http://example.test/root/", Header: map[string][]string{"X-Test": {"1"}}})
	core.AssertNoError(t, err)
	core.AssertEqual(t, "example.test", medium.baseURL.Host)
}

func TestWebdav_Medium_Read_Good(t *core.T) {
	medium := newWebDAVTestMedium(t)
	core.RequireNoError(t, medium.Write("read.txt", "payload"))
	got, err := medium.Read("read.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", got)
}

func TestWebdav_Medium_Read_Bad(t *core.T) {
	medium := newWebDAVTestMedium(t)
	got, err := medium.Read("")
	core.AssertError(t, err)
	core.AssertEqual(t, "", got)
}

func TestWebdav_Medium_Read_Ugly(t *core.T) {
	medium := newWebDAVTestMedium(t)
	core.RequireNoError(t, medium.Write("safe/file.txt", "payload"))
	got, err := medium.Read("/safe/../safe/file.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", got)
}

func TestWebdav_Medium_Write_Good(t *core.T) {
	medium := newWebDAVTestMedium(t)
	err := medium.Write("write.txt", "payload")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("write.txt"))
}

func TestWebdav_Medium_Write_Bad(t *core.T) {
	medium := newWebDAVTestMedium(t)
	err := medium.Write("", "payload")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.IsFile(""))
}

func TestWebdav_Medium_Write_Ugly(t *core.T) {
	medium := newWebDAVTestMedium(t)
	err := medium.Write("nested/write.txt", "")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("nested/write.txt"))
}

func TestWebdav_Medium_WriteMode_Good(t *core.T) {
	medium := newWebDAVTestMedium(t)
	err := medium.WriteMode("mode.txt", "payload", 0600)
	info, statErr := medium.Stat("mode.txt")
	core.AssertNoError(t, err)
	core.AssertNoError(t, statErr)
	core.AssertEqual(t, fs.FileMode(0644), info.Mode().Perm())
}

func TestWebdav_Medium_WriteMode_Bad(t *core.T) {
	medium := newWebDAVTestMedium(t)
	err := medium.WriteMode("", "payload", 0600)
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists(""))
}

func TestWebdav_Medium_WriteMode_Ugly(t *core.T) {
	medium := newWebDAVTestMedium(t)
	err := medium.WriteMode("zero-mode.txt", "payload", 0)
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("zero-mode.txt"))
}

func TestWebdav_Medium_EnsureDir_Good(t *core.T) {
	medium := newWebDAVTestMedium(t)
	err := medium.EnsureDir("dir")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir("dir"))
}

func TestWebdav_Medium_EnsureDir_Bad(t *core.T) {
	medium := newWebDAVTestMedium(t)
	err := medium.EnsureDir("")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.IsDir(""))
}

func TestWebdav_Medium_EnsureDir_Ugly(t *core.T) {
	medium := newWebDAVTestMedium(t)
	err := medium.EnsureDir("a/b/c")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir("a/b/c"))
}

func TestWebdav_Medium_IsFile_Good(t *core.T) {
	medium := newWebDAVTestMedium(t)
	core.RequireNoError(t, medium.Write("file.txt", "payload"))
	got := medium.IsFile("file.txt")
	core.AssertTrue(t, got)
}

func TestWebdav_Medium_IsFile_Bad(t *core.T) {
	medium := newWebDAVTestMedium(t)
	got := medium.IsFile("")
	core.AssertFalse(t, got)
}

func TestWebdav_Medium_IsFile_Ugly(t *core.T) {
	medium := newWebDAVTestMedium(t)
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.IsFile("dir")
	core.AssertFalse(t, got)
}

func TestWebdav_Medium_Delete_Good(t *core.T) {
	medium := newWebDAVTestMedium(t)
	core.RequireNoError(t, medium.Write("delete.txt", "payload"))
	err := medium.Delete("delete.txt")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("delete.txt"))
}

func TestWebdav_Medium_Delete_Bad(t *core.T) {
	medium := newWebDAVTestMedium(t)
	err := medium.Delete("")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists(""))
}

func TestWebdav_Medium_Delete_Ugly(t *core.T) {
	medium := newWebDAVTestMedium(t)
	err := medium.Delete("missing.txt")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("missing.txt"))
}

func TestWebdav_Medium_DeleteAll_Good(t *core.T) {
	medium := newWebDAVTestMedium(t)
	core.RequireNoError(t, medium.Write("tree/file.txt", "payload"))
	err := medium.DeleteAll("tree")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("tree/file.txt"))
}

func TestWebdav_Medium_DeleteAll_Bad(t *core.T) {
	medium := newWebDAVTestMedium(t)
	err := medium.DeleteAll("")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists(""))
}

func TestWebdav_Medium_DeleteAll_Ugly(t *core.T) {
	medium := newWebDAVTestMedium(t)
	err := medium.DeleteAll("missing")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("missing"))
}

func TestWebdav_Medium_Rename_Good(t *core.T) {
	medium := newWebDAVTestMedium(t)
	core.RequireNoError(t, medium.Write("old.txt", "payload"))
	err := medium.Rename("old.txt", "new.txt")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("new.txt"))
}

func TestWebdav_Medium_Rename_Bad(t *core.T) {
	medium := newWebDAVTestMedium(t)
	err := medium.Rename("", "new.txt")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("new.txt"))
}

func TestWebdav_Medium_Rename_Ugly(t *core.T) {
	medium := newWebDAVTestMedium(t)
	core.RequireNoError(t, medium.Write("old.txt", "payload"))
	err := medium.Rename("old.txt", "nested/new.txt")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("nested/new.txt"))
}

func TestWebdav_Medium_List_Good(t *core.T) {
	medium := newWebDAVTestMedium(t)
	core.RequireNoError(t, medium.Write("dir/a.txt", "payload"))
	entries, err := medium.List("dir")
	core.AssertNoError(t, err)
	core.AssertLen(t, entries, 1)
}

func TestWebdav_Medium_List_Bad(t *core.T) {
	medium := newWebDAVTestMedium(t)
	entries, err := medium.List("missing")
	core.AssertError(t, err)
	core.AssertNil(t, entries)
}

func TestWebdav_Medium_List_Ugly(t *core.T) {
	medium := newWebDAVTestMedium(t)
	entries, err := medium.List("")
	core.AssertNoError(t, err)
	core.AssertEmpty(t, entries)
}

func TestWebdav_Medium_Stat_Good(t *core.T) {
	medium := newWebDAVTestMedium(t)
	core.RequireNoError(t, medium.Write("stat.txt", "payload"))
	info, err := medium.Stat("stat.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "stat.txt", info.Name())
}

func TestWebdav_Medium_Stat_Bad(t *core.T) {
	medium := newWebDAVTestMedium(t)
	info, err := medium.Stat("")
	core.AssertError(t, err)
	core.AssertNil(t, info)
}

func TestWebdav_Medium_Stat_Ugly(t *core.T) {
	medium := newWebDAVTestMedium(t)
	core.RequireNoError(t, medium.EnsureDir("dir"))
	info, err := medium.Stat("dir")
	core.AssertNoError(t, err)
	core.AssertTrue(t, info.IsDir())
}

func TestWebdav_Medium_Open_Good(t *core.T) {
	medium := newWebDAVTestMedium(t)
	core.RequireNoError(t, medium.Write("open.txt", "payload"))
	file, err := medium.Open("open.txt")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, file)
	core.RequireNoError(t, file.Close())
}

func TestWebdav_Medium_Open_Bad(t *core.T) {
	medium := newWebDAVTestMedium(t)
	file, err := medium.Open("")
	core.AssertError(t, err)
	core.AssertNil(t, file)
}

func TestWebdav_Medium_Open_Ugly(t *core.T) {
	medium := newWebDAVTestMedium(t)
	file, err := medium.Open("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, file)
}

func TestWebdav_Medium_Create_Good(t *core.T) {
	medium := newWebDAVTestMedium(t)
	writer, err := medium.Create("create.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestWebdav_Medium_Create_Bad(t *core.T) {
	medium := newWebDAVTestMedium(t)
	writer, err := medium.Create("")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestWebdav_Medium_Create_Ugly(t *core.T) {
	medium := newWebDAVTestMedium(t)
	writer, err := medium.Create("nested/create.txt")
	core.RequireNoError(t, err)
	core.AssertNoError(t, writer.Close())
	core.AssertTrue(t, medium.Exists("nested/create.txt"))
}

func TestWebdav_Medium_Append_Good(t *core.T) {
	medium := newWebDAVTestMedium(t)
	core.RequireNoError(t, medium.Write("append.txt", "a"))
	writer, err := medium.Append("append.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("b"))
	core.RequireNoError(t, writeErr)
	core.RequireNoError(t, writer.Close())
}

func TestWebdav_Medium_Append_Bad(t *core.T) {
	medium := newWebDAVTestMedium(t)
	writer, err := medium.Append("")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestWebdav_Medium_Append_Ugly(t *core.T) {
	medium := newWebDAVTestMedium(t)
	writer, err := medium.Append("new.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("created"))
	core.RequireNoError(t, writeErr)
	core.RequireNoError(t, writer.Close())
}

func TestWebdav_Medium_ReadStream_Good(t *core.T) {
	medium := newWebDAVTestMedium(t)
	core.RequireNoError(t, medium.Write("stream.txt", "payload"))
	reader, err := medium.ReadStream("stream.txt")
	core.RequireNoError(t, err)
	defer reader.Close()
	data, readErr := goio.ReadAll(reader)
	core.AssertNoError(t, readErr)
	core.AssertEqual(t, "payload", string(data))
}

func TestWebdav_Medium_ReadStream_Bad(t *core.T) {
	medium := newWebDAVTestMedium(t)
	reader, err := medium.ReadStream("")
	core.AssertError(t, err)
	core.AssertNil(t, reader)
}

func TestWebdav_Medium_ReadStream_Ugly(t *core.T) {
	medium := newWebDAVTestMedium(t)
	reader, err := medium.ReadStream("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, reader)
}

func TestWebdav_Medium_WriteStream_Good(t *core.T) {
	medium := newWebDAVTestMedium(t)
	writer, err := medium.WriteStream("stream.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestWebdav_Medium_WriteStream_Bad(t *core.T) {
	medium := newWebDAVTestMedium(t)
	writer, err := medium.WriteStream("")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestWebdav_Medium_WriteStream_Ugly(t *core.T) {
	medium := newWebDAVTestMedium(t)
	writer, err := medium.WriteStream("nested/stream.txt")
	core.RequireNoError(t, err)
	core.AssertNoError(t, writer.Close())
	core.AssertTrue(t, medium.Exists("nested/stream.txt"))
}

func TestWebdav_Medium_Exists_Good(t *core.T) {
	medium := newWebDAVTestMedium(t)
	core.RequireNoError(t, medium.Write("exists.txt", "payload"))
	got := medium.Exists("exists.txt")
	core.AssertTrue(t, got)
}

func TestWebdav_Medium_Exists_Bad(t *core.T) {
	medium := newWebDAVTestMedium(t)
	got := medium.Exists("")
	core.AssertFalse(t, got)
}

func TestWebdav_Medium_Exists_Ugly(t *core.T) {
	medium := newWebDAVTestMedium(t)
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.Exists("dir")
	core.AssertTrue(t, got)
}

func TestWebdav_Medium_IsDir_Good(t *core.T) {
	medium := newWebDAVTestMedium(t)
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.IsDir("dir")
	core.AssertTrue(t, got)
}

func TestWebdav_Medium_IsDir_Bad(t *core.T) {
	medium := newWebDAVTestMedium(t)
	got := medium.IsDir("")
	core.AssertFalse(t, got)
}

func TestWebdav_Medium_IsDir_Ugly(t *core.T) {
	medium := newWebDAVTestMedium(t)
	core.RequireNoError(t, medium.Write("file.txt", "payload"))
	got := medium.IsDir("file.txt")
	core.AssertFalse(t, got)
}

func TestWebdav_File_Stat_Good(t *core.T) {
	file := &webdavFile{name: "file.txt", content: []byte("payload"), mode: 0600, modTime: time.Unix(1, 0)}
	info, err := file.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "file.txt", info.Name())
}

func TestWebdav_File_Stat_Bad(t *core.T) {
	file := &webdavFile{}
	info, err := file.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "", info.Name())
}

func TestWebdav_File_Stat_Ugly(t *core.T) {
	file := &webdavFile{name: "empty.txt"}
	info, err := file.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, int64(0), info.Size())
}

func TestWebdav_File_Read_Good(t *core.T) {
	file := &webdavFile{content: []byte("payload")}
	buffer := make([]byte, 7)
	count, err := file.Read(buffer)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", string(buffer[:count]))
}

func TestWebdav_File_Read_Bad(t *core.T) {
	file := &webdavFile{content: []byte("x"), offset: 1}
	buffer := make([]byte, 1)
	count, err := file.Read(buffer)
	core.AssertErrorIs(t, err, goio.EOF)
	core.AssertEqual(t, 0, count)
}

func TestWebdav_File_Read_Ugly(t *core.T) {
	file := &webdavFile{content: []byte("payload")}
	buffer := make([]byte, 3)
	count, err := file.Read(buffer)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "pay", string(buffer[:count]))
}

func TestWebdav_File_Close_Good(t *core.T) {
	file := &webdavFile{name: "file.txt"}
	err := file.Close()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "file.txt", file.name)
}

func TestWebdav_File_Close_Bad(t *core.T) {
	file := &webdavFile{}
	err := file.Close()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "", file.name)
}

func TestWebdav_File_Close_Ugly(t *core.T) {
	file := &webdavFile{offset: 99}
	err := file.Close()
	core.AssertNoError(t, err)
	core.AssertEqual(t, int64(99), file.offset)
}

func TestWebdav_WriteCloser_Write_Good(t *core.T) {
	writer := &webdavWriteCloser{}
	count, err := writer.Write([]byte("payload"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, len("payload"), count)
}

func TestWebdav_WriteCloser_Write_Bad(t *core.T) {
	writer := &webdavWriteCloser{}
	count, err := writer.Write(nil)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, count)
}

func TestWebdav_WriteCloser_Write_Ugly(t *core.T) {
	writer := &webdavWriteCloser{data: []byte("a")}
	count, err := writer.Write([]byte("b"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, 1, count)
}

func TestWebdav_WriteCloser_Close_Good(t *core.T) {
	medium := newWebDAVTestMedium(t)
	writer := &webdavWriteCloser{medium: medium, path: "close.txt", data: []byte("payload")}
	err := writer.Close()
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("close.txt"))
}

func TestWebdav_WriteCloser_Close_Bad(t *core.T) {
	writer := &webdavWriteCloser{}
	err := writer.Close()
	core.AssertError(t, err)
	core.AssertNil(t, writer.medium)
}

func TestWebdav_WriteCloser_Close_Ugly(t *core.T) {
	medium := newWebDAVTestMedium(t)
	writer := &webdavWriteCloser{medium: medium, path: "empty.txt"}
	err := writer.Close()
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("empty.txt"))
}
