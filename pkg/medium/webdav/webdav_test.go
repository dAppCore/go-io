package webdav

import (
	core "dappco.re/go"
	xwebdav "golang.org/x/net/webdav"
	"net/http/httptest"
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
