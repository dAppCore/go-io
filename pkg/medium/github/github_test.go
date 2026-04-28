package github

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	pathpkg "path"

	core "dappco.re/go"
)

func newGitHubTestMedium(t *core.T, handler http.Handler) *Medium {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	medium, err := New(Options{
		HTTPClient: server.Client(),
		Owner:      "Snider",
		Repo:       "demo",
		Ref:        "main",
		TokenFile:  pathpkg.Join(t.TempDir(), "missing-token"),
		BaseURL:    server.URL + "/",
	})
	core.RequireNoError(t, err)
	return medium
}

func githubFileJSON(filePath, content string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(content))
	return fmt.Sprintf(
		`{"type":"file","name":%q,"path":%q,"encoding":"base64","content":%q,"size":%d}`,
		pathpkg.Base(filePath),
		filePath,
		encoded,
		len(content),
	)
}

func githubDirJSON(filePath string) string {
	return fmt.Sprintf(
		`{"type":"dir","name":%q,"path":%q,"size":0}`,
		pathpkg.Base(filePath),
		filePath,
	)
}

func TestGitHubMedium_Read_Good(t *core.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/Snider/demo/contents/docs/read.txt", func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "main", r.URL.Query().Get("ref"))
		_, _ = fmt.Fprint(w, githubFileJSON("docs/read.txt", "hello github"))
	})
	medium := newGitHubTestMedium(t, mux)

	content, err := medium.Read("docs/read.txt")

	core.RequireNoError(t, err)
	core.AssertEqual(t, "hello github", content)
}

func TestGitHubMedium_Read_Bad(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())

	_, err := medium.Read("missing.txt")

	core.AssertError(t, err)
	core.AssertTrue(t, errors.Is(err, fs.ErrNotExist))
}

func TestGitHubMedium_Read_Ugly(t *core.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/Snider/demo/contents/safe/file.txt", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, githubFileJSON("safe/file.txt", "normalised"))
	})
	medium := newGitHubTestMedium(t, mux)

	content, err := medium.Read("//safe/../safe/./file.txt")

	core.RequireNoError(t, err)
	core.AssertEqual(t, "normalised", content)
}

func TestGitHubMedium_List_Good(t *core.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/Snider/demo/contents/dir", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, `[%s,%s,%s]`,
			githubFileJSON("dir/b.txt", "b"),
			githubFileJSON("dir/a.txt", "a"),
			githubDirJSON("dir/sub"),
		)
	})
	mux.HandleFunc("/repos/Snider/demo/contents/dir/sub", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, `[%s]`, githubFileJSON("dir/sub/c.txt", "c"))
	})
	medium := newGitHubTestMedium(t, mux)

	entries, err := medium.List("dir")

	core.RequireNoError(t, err)
	core.AssertLen(t, entries, 4)
	core.AssertEqual(t, "dir/a.txt", entries[0].Name())
	core.AssertEqual(t, "dir/b.txt", entries[1].Name())
	core.AssertEqual(t, "dir/sub", entries[2].Name())
	core.AssertTrue(t, entries[2].IsDir())
	core.AssertEqual(t, "dir/sub/c.txt", entries[3].Name())
}

func TestGitHubMedium_List_Bad(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())

	_, err := medium.List("missing")

	core.AssertError(t, err)
	core.AssertTrue(t, errors.Is(err, fs.ErrNotExist))
}

func TestGitHubMedium_List_Ugly(t *core.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/Snider/demo/contents/dir", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, `[%s]`, githubFileJSON("dir/file.txt", "content"))
	})
	medium := newGitHubTestMedium(t, mux)

	entries, err := medium.List("//dir/../dir/.")

	core.RequireNoError(t, err)
	core.AssertLen(t, entries, 1)
	core.AssertEqual(t, "dir/file.txt", entries[0].Name())
}

func TestGitHubMedium_Write_Good(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())

	err := medium.Write("notes/write.txt", "content")

	core.AssertErrorIs(t, err, ErrReadOnly)
}

func TestGitHubMedium_Write_Bad(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())

	err := medium.Write("", "content")

	core.AssertErrorIs(t, err, ErrReadOnly)
}

func TestGitHubMedium_Write_Ugly(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())

	err := medium.Write("../escaped.txt", "content")

	core.AssertErrorIs(t, err, ErrReadOnly)
}

func TestGitHubMedium_Actions_Register(t *core.T) {
	_, ok := FactoryFor(Scheme)
	core.RequireTrue(t, ok)

	c := core.New()
	RegisterActions(c)

	core.AssertTrue(t, c.Action(ActionRead).Exists())
	core.AssertTrue(t, c.Action(ActionList).Exists())
	core.AssertTrue(t, c.Action(ActionClone).Exists())

	result := c.Action(ActionRead).Run(context.Background(), core.NewOptions(
		core.Option{Key: "owner", Value: "Snider"},
		core.Option{Key: "repo", Value: "demo"},
		core.Option{Key: "path", Value: ""},
	))
	core.AssertFalse(t, result.OK)
}
