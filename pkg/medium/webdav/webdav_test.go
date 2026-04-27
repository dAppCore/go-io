package webdav

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	xwebdav "golang.org/x/net/webdav"
)

func newWebDAVTestMedium(t *testing.T) *Medium {
	t.Helper()

	handler := &xwebdav.Handler{
		FileSystem: xwebdav.NewMemFS(),
		LockSystem: xwebdav.NewMemLS(),
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	medium, err := New(Options{BaseURL: server.URL})
	require.NoError(t, err)
	return medium
}

func TestWebDAVMedium_Read_Good(t *testing.T) {
	medium := newWebDAVTestMedium(t)

	require.NoError(t, medium.Write("notes/read.txt", "hello webdav"))

	content, err := medium.Read("notes/read.txt")
	require.NoError(t, err)
	assert.Equal(t, "hello webdav", content)
}

func TestWebDAVMedium_Read_Bad(t *testing.T) {
	medium := newWebDAVTestMedium(t)

	_, err := medium.Read("missing.txt")

	assert.Error(t, err)
}

func TestWebDAVMedium_Read_Ugly(t *testing.T) {
	medium := newWebDAVTestMedium(t)

	require.NoError(t, medium.Write("safe/file.txt", "normalised"))

	content, err := medium.Read("/safe/../safe/./file.txt")
	require.NoError(t, err)
	assert.Equal(t, "normalised", content)
}

func TestWebDAVMedium_Write_Good(t *testing.T) {
	medium := newWebDAVTestMedium(t)

	err := medium.Write("nested/path/file.txt", "content")
	require.NoError(t, err)

	assert.True(t, medium.IsFile("nested/path/file.txt"))
	content, err := medium.Read("nested/path/file.txt")
	require.NoError(t, err)
	assert.Equal(t, "content", content)
}

func TestWebDAVMedium_Write_Bad(t *testing.T) {
	medium := newWebDAVTestMedium(t)

	err := medium.Write("", "content")

	assert.Error(t, err)
}

func TestWebDAVMedium_Write_Ugly(t *testing.T) {
	medium := newWebDAVTestMedium(t)

	require.NoError(t, medium.Write("../escaped.txt", "contained"))

	content, err := medium.Read("escaped.txt")
	require.NoError(t, err)
	assert.Equal(t, "contained", content)
}

func TestWebDAVMedium_List_Good(t *testing.T) {
	medium := newWebDAVTestMedium(t)

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

func TestWebDAVMedium_List_Bad(t *testing.T) {
	medium := newWebDAVTestMedium(t)

	_, err := medium.List("missing")

	assert.Error(t, err)
}

func TestWebDAVMedium_List_Ugly(t *testing.T) {
	medium := newWebDAVTestMedium(t)

	require.NoError(t, medium.Write("dir/file.txt", "content"))

	entries, err := medium.List("//dir/../dir/.")
	require.NoError(t, err)

	require.Len(t, entries, 1)
	assert.Equal(t, "file.txt", entries[0].Name())
}
