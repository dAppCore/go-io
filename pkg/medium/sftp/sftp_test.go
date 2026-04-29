package sftp

import (
	core "dappco.re/go"
	pkgsftp "github.com/pkg/sftp"
	goio "io"
	"io/fs"
	"net"
	"time"
)

func newSFTPTestMedium(t *core.T) *Medium {
	t.Helper()

	serverConn, clientConn := net.Pipe()
	server := pkgsftp.NewRequestServer(serverConn, pkgsftp.InMemHandler())
	done := make(chan error, 1)
	go func() {
		done <- server.Serve()
	}()

	client, err := pkgsftp.NewClientPipe(clientConn, clientConn)
	core.RequireNoError(t, err)

	medium, err := New(Options{Client: client})
	core.RequireNoError(t, err)

	t.Cleanup(func() {
		_ = client.Close()
		_ = clientConn.Close()
		_ = serverConn.Close()
		select {
		case <-done:
		case <-time.After(time.Second):
		}
	})

	return medium
}

func TestSFTPMedium_Read_Good(t *core.T) {
	medium := newSFTPTestMedium(t)

	core.RequireNoError(t, medium.Write("notes/read.txt", "hello sftp"))

	content, err := medium.Read("notes/read.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "hello sftp", content)
}

func TestSFTPMedium_Read_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)

	_, err := medium.Read("missing.txt")

	core.AssertError(t, err)
}

func TestSFTPMedium_Read_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)

	core.RequireNoError(t, medium.Write("safe/file.txt", "normalised"))

	content, err := medium.Read("/safe/../safe/./file.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "normalised", content)
}

func TestSFTPMedium_Write_Good(t *core.T) {
	medium := newSFTPTestMedium(t)

	err := medium.Write("nested/path/file.txt", "content")
	core.RequireNoError(t, err)

	core.AssertTrue(t, medium.IsFile("nested/path/file.txt"))
	content, err := medium.Read("nested/path/file.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "content", content)
}

func TestSFTPMedium_Write_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)

	err := medium.Write("", "content")

	core.AssertError(t, err)
}

func TestSFTPMedium_Write_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)

	core.RequireNoError(t, medium.Write("../escaped.txt", "contained"))

	content, err := medium.Read("escaped.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "contained", content)
}

func TestSFTPMedium_List_Good(t *core.T) {
	medium := newSFTPTestMedium(t)

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

func TestSFTPMedium_List_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)

	_, err := medium.List("missing")

	core.AssertError(t, err)
}

func TestSFTPMedium_List_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)

	core.RequireNoError(t, medium.Write("dir/file.txt", "content"))

	entries, err := medium.List("//dir/../dir/.")
	core.RequireNoError(t, err)

	core.AssertLen(t, entries, 1)
	core.AssertEqual(t, "file.txt", entries[0].Name())
}

func TestSftp_New_Good(t *core.T) {
	base := newSFTPTestMedium(t)
	medium, err := New(Options{Client: base.client})
	core.RequireNoError(t, err)

	core.AssertNotNil(t, medium)
	core.AssertEqual(t, "/", medium.root)
}

func TestSftp_New_Bad(t *core.T) {
	medium, err := New(Options{})
	core.AssertError(t, err)
	core.AssertNil(t, medium)
}

func TestSftp_New_Ugly(t *core.T) {
	base := newSFTPTestMedium(t)
	medium, err := New(Options{Client: base.client, Root: "/srv/app/"})
	core.RequireNoError(t, err)

	err = medium.EnsureDir("rooted")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir("rooted"))
}

func TestSftp_Medium_Close_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	err := medium.Close()
	core.AssertNoError(t, err)
	core.AssertNotNil(t, medium.client)
}

func TestSftp_Medium_Close_Bad(t *core.T) {
	medium := &Medium{}
	err := medium.Close()
	core.AssertNoError(t, err)
	core.AssertNil(t, medium.client)
}

func TestSftp_Medium_Close_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.Close())
	err := medium.Close()
	core.AssertNoError(t, err)
}

func TestSftp_Medium_Read_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.Write("read.txt", "payload"))
	got, err := medium.Read("read.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", got)
}

func TestSftp_Medium_Read_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)
	got, err := medium.Read("")
	core.AssertError(t, err)
	core.AssertEqual(t, "", got)
}

func TestSftp_Medium_Read_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.Write("safe/file.txt", "payload"))
	got, err := medium.Read("/safe/../safe/file.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", got)
}

func TestSftp_Medium_Write_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	err := medium.Write("write.txt", "payload")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("write.txt"))
}

func TestSftp_Medium_Write_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)
	err := medium.Write("", "payload")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.IsFile(""))
}

func TestSftp_Medium_Write_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	err := medium.Write("nested/write.txt", "")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("nested/write.txt"))
}

func TestSftp_Medium_WriteMode_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	err := medium.WriteMode("mode.txt", "payload", 0600)
	info, statErr := medium.Stat("mode.txt")
	core.AssertNoError(t, err)
	core.AssertNoError(t, statErr)
	core.AssertEqual(t, fs.FileMode(0644), info.Mode().Perm())
}

func TestSftp_Medium_WriteMode_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)
	err := medium.WriteMode("", "payload", 0600)
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists(""))
}

func TestSftp_Medium_WriteMode_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	err := medium.WriteMode("zero-mode.txt", "payload", 0)
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("zero-mode.txt"))
}

func TestSftp_Medium_EnsureDir_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	err := medium.EnsureDir("dir")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir("dir"))
}

func TestSftp_Medium_EnsureDir_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)
	err := medium.EnsureDir("")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.IsDir(""))
}

func TestSftp_Medium_EnsureDir_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	err := medium.EnsureDir("a/b/c")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir("a/b/c"))
}

func TestSftp_Medium_IsFile_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.Write("file.txt", "payload"))
	got := medium.IsFile("file.txt")
	core.AssertTrue(t, got)
}

func TestSftp_Medium_IsFile_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)
	got := medium.IsFile("")
	core.AssertFalse(t, got)
}

func TestSftp_Medium_IsFile_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.IsFile("dir")
	core.AssertFalse(t, got)
}

func TestSftp_Medium_Delete_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.Write("delete.txt", "payload"))
	err := medium.Delete("delete.txt")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("delete.txt"))
}

func TestSftp_Medium_Delete_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)
	err := medium.Delete("")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists(""))
}

func TestSftp_Medium_Delete_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	err := medium.Delete("missing.txt")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("missing.txt"))
}

func TestSftp_Medium_DeleteAll_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.Write("tree/file.txt", "payload"))
	err := medium.DeleteAll("tree")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("tree/file.txt"))
}

func TestSftp_Medium_DeleteAll_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)
	err := medium.DeleteAll("")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists(""))
}

func TestSftp_Medium_DeleteAll_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	err := medium.DeleteAll("missing")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("missing"))
}

func TestSftp_Medium_Rename_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.Write("old.txt", "payload"))
	err := medium.Rename("old.txt", "new.txt")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("new.txt"))
}

func TestSftp_Medium_Rename_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)
	err := medium.Rename("", "new.txt")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("new.txt"))
}

func TestSftp_Medium_Rename_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.Write("old.txt", "payload"))
	err := medium.Rename("old.txt", "nested/new.txt")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("nested/new.txt"))
}

func TestSftp_Medium_List_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.Write("dir/a.txt", "payload"))
	entries, err := medium.List("dir")
	core.AssertNoError(t, err)
	core.AssertLen(t, entries, 1)
}

func TestSftp_Medium_List_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)
	entries, err := medium.List("missing")
	core.AssertError(t, err)
	core.AssertNil(t, entries)
}

func TestSftp_Medium_List_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	entries, err := medium.List("")
	core.AssertNoError(t, err)
	core.AssertEmpty(t, entries)
}

func TestSftp_Medium_Stat_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.Write("stat.txt", "payload"))
	info, err := medium.Stat("stat.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "stat.txt", info.Name())
}

func TestSftp_Medium_Stat_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)
	info, err := medium.Stat("")
	core.AssertError(t, err)
	core.AssertNil(t, info)
}

func TestSftp_Medium_Stat_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.EnsureDir("dir"))
	info, err := medium.Stat("dir")
	core.AssertNoError(t, err)
	core.AssertTrue(t, info.IsDir())
}

func TestSftp_Medium_Open_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.Write("open.txt", "payload"))
	file, err := medium.Open("open.txt")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, file)
	core.RequireNoError(t, file.Close())
}

func TestSftp_Medium_Open_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)
	file, err := medium.Open("")
	core.AssertError(t, err)
	core.AssertNil(t, file)
}

func TestSftp_Medium_Open_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	file, err := medium.Open("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, file)
}

func TestSftp_Medium_Create_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	writer, err := medium.Create("create.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestSftp_Medium_Create_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)
	writer, err := medium.Create("")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestSftp_Medium_Create_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	writer, err := medium.Create("nested/create.txt")
	core.RequireNoError(t, err)
	core.AssertNoError(t, writer.Close())
	core.AssertTrue(t, medium.Exists("nested/create.txt"))
}

func TestSftp_Medium_Append_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.Write("append.txt", "a"))
	writer, err := medium.Append("append.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("b"))
	core.RequireNoError(t, writeErr)
	core.RequireNoError(t, writer.Close())
}

func TestSftp_Medium_Append_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)
	writer, err := medium.Append("")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestSftp_Medium_Append_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	writer, err := medium.Append("new.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("created"))
	core.RequireNoError(t, writeErr)
	core.RequireNoError(t, writer.Close())
}

func TestSftp_Medium_ReadStream_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.Write("stream.txt", "payload"))
	reader, err := medium.ReadStream("stream.txt")
	core.RequireNoError(t, err)
	defer reader.Close()
	data, readErr := goio.ReadAll(reader)
	core.AssertNoError(t, readErr)
	core.AssertEqual(t, "payload", string(data))
}

func TestSftp_Medium_ReadStream_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)
	reader, err := medium.ReadStream("")
	core.AssertError(t, err)
	core.AssertNil(t, reader)
}

func TestSftp_Medium_ReadStream_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	reader, err := medium.ReadStream("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, reader)
}

func TestSftp_Medium_WriteStream_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	writer, err := medium.WriteStream("stream.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestSftp_Medium_WriteStream_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)
	writer, err := medium.WriteStream("")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestSftp_Medium_WriteStream_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	writer, err := medium.WriteStream("nested/stream.txt")
	core.RequireNoError(t, err)
	core.AssertNoError(t, writer.Close())
	core.AssertTrue(t, medium.Exists("nested/stream.txt"))
}

func TestSftp_Medium_Exists_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.Write("exists.txt", "payload"))
	got := medium.Exists("exists.txt")
	core.AssertTrue(t, got)
}

func TestSftp_Medium_Exists_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)
	got := medium.Exists("")
	core.AssertFalse(t, got)
}

func TestSftp_Medium_Exists_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.Exists("dir")
	core.AssertTrue(t, got)
}

func TestSftp_Medium_IsDir_Good(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.IsDir("dir")
	core.AssertTrue(t, got)
}

func TestSftp_Medium_IsDir_Bad(t *core.T) {
	medium := newSFTPTestMedium(t)
	got := medium.IsDir("")
	core.AssertFalse(t, got)
}

func TestSftp_Medium_IsDir_Ugly(t *core.T) {
	medium := newSFTPTestMedium(t)
	core.RequireNoError(t, medium.Write("file.txt", "payload"))
	got := medium.IsDir("file.txt")
	core.AssertFalse(t, got)
}
