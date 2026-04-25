package sftp

import (
	"net"
	"testing"
	"time"

	pkgsftp "github.com/pkg/sftp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSFTPTestMedium(t *testing.T) *Medium {
	t.Helper()

	serverConn, clientConn := net.Pipe()
	server := pkgsftp.NewRequestServer(serverConn, pkgsftp.InMemHandler())
	done := make(chan error, 1)
	go func() {
		done <- server.Serve()
	}()

	client, err := pkgsftp.NewClientPipe(clientConn, clientConn)
	require.NoError(t, err)

	medium, err := New(Options{Client: client})
	require.NoError(t, err)

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

func TestSFTPMedium_Read_Good(t *testing.T) {
	medium := newSFTPTestMedium(t)

	require.NoError(t, medium.Write("notes/read.txt", "hello sftp"))

	content, err := medium.Read("notes/read.txt")
	require.NoError(t, err)
	assert.Equal(t, "hello sftp", content)
}

func TestSFTPMedium_Read_Bad(t *testing.T) {
	medium := newSFTPTestMedium(t)

	_, err := medium.Read("missing.txt")

	assert.Error(t, err)
}

func TestSFTPMedium_Read_Ugly(t *testing.T) {
	medium := newSFTPTestMedium(t)

	require.NoError(t, medium.Write("safe/file.txt", "normalised"))

	content, err := medium.Read("/safe/../safe/./file.txt")
	require.NoError(t, err)
	assert.Equal(t, "normalised", content)
}

func TestSFTPMedium_Write_Good(t *testing.T) {
	medium := newSFTPTestMedium(t)

	err := medium.Write("nested/path/file.txt", "content")
	require.NoError(t, err)

	assert.True(t, medium.IsFile("nested/path/file.txt"))
	content, err := medium.Read("nested/path/file.txt")
	require.NoError(t, err)
	assert.Equal(t, "content", content)
}

func TestSFTPMedium_Write_Bad(t *testing.T) {
	medium := newSFTPTestMedium(t)

	err := medium.Write("", "content")

	assert.Error(t, err)
}

func TestSFTPMedium_Write_Ugly(t *testing.T) {
	medium := newSFTPTestMedium(t)

	require.NoError(t, medium.Write("../escaped.txt", "contained"))

	content, err := medium.Read("escaped.txt")
	require.NoError(t, err)
	assert.Equal(t, "contained", content)
}

func TestSFTPMedium_List_Good(t *testing.T) {
	medium := newSFTPTestMedium(t)

	require.NoError(t, medium.Write("dir/b.txt", "b"))
	require.NoError(t, medium.Write("dir/a.txt", "a"))
	require.NoError(t, medium.EnsureDir("dir/sub"))

	entries, err := medium.List("dir")
	require.NoError(t, err)

	require.Len(t, entries, 3)
	assert.Equal(t, "a.txt", entries[0].Name())
	assert.Equal(t, "b.txt", entries[1].Name())
	assert.Equal(t, "sub", entries[2].Name())
	assert.True(t, entries[2].IsDir())
}

func TestSFTPMedium_List_Bad(t *testing.T) {
	medium := newSFTPTestMedium(t)

	_, err := medium.List("missing")

	assert.Error(t, err)
}

func TestSFTPMedium_List_Ugly(t *testing.T) {
	medium := newSFTPTestMedium(t)

	require.NoError(t, medium.Write("dir/file.txt", "content"))

	entries, err := medium.List("//dir/../dir/.")
	require.NoError(t, err)

	require.Len(t, entries, 1)
	assert.Equal(t, "file.txt", entries[0].Name())
}
