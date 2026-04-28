package sftp

import (
	core "dappco.re/go"
	"net"
	"time"

	pkgsftp "github.com/pkg/sftp"
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
