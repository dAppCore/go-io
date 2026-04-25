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
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newGitHubTestMedium(t *testing.T, handler http.Handler) *Medium {
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
	require.NoError(t, err)
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

func TestGitHubMedium_Read_Good(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/Snider/demo/contents/docs/read.txt", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "main", r.URL.Query().Get("ref"))
		_, _ = fmt.Fprint(w, githubFileJSON("docs/read.txt", "hello github"))
	})
	medium := newGitHubTestMedium(t, mux)

	content, err := medium.Read("docs/read.txt")

	require.NoError(t, err)
	assert.Equal(t, "hello github", content)
}

func TestGitHubMedium_Read_Bad(t *testing.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())

	_, err := medium.Read("missing.txt")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, fs.ErrNotExist))
}

func TestGitHubMedium_Read_Ugly(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/Snider/demo/contents/safe/file.txt", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, githubFileJSON("safe/file.txt", "normalised"))
	})
	medium := newGitHubTestMedium(t, mux)

	content, err := medium.Read("//safe/../safe/./file.txt")

	require.NoError(t, err)
	assert.Equal(t, "normalised", content)
}

func TestGitHubMedium_List_Good(t *testing.T) {
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

	require.NoError(t, err)
	require.Len(t, entries, 4)
	assert.Equal(t, "dir/a.txt", entries[0].Name())
	assert.Equal(t, "dir/b.txt", entries[1].Name())
	assert.Equal(t, "dir/sub", entries[2].Name())
	assert.True(t, entries[2].IsDir())
	assert.Equal(t, "dir/sub/c.txt", entries[3].Name())
}

func TestGitHubMedium_List_Bad(t *testing.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())

	_, err := medium.List("missing")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, fs.ErrNotExist))
}

func TestGitHubMedium_List_Ugly(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/Snider/demo/contents/dir", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, `[%s]`, githubFileJSON("dir/file.txt", "content"))
	})
	medium := newGitHubTestMedium(t, mux)

	entries, err := medium.List("//dir/../dir/.")

	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "dir/file.txt", entries[0].Name())
}

func TestGitHubMedium_Write_Good(t *testing.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())

	err := medium.Write("notes/write.txt", "content")

	assert.ErrorIs(t, err, ErrReadOnly)
}

func TestGitHubMedium_Write_Bad(t *testing.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())

	err := medium.Write("", "content")

	assert.ErrorIs(t, err, ErrReadOnly)
}

func TestGitHubMedium_Write_Ugly(t *testing.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())

	err := medium.Write("../escaped.txt", "content")

	assert.ErrorIs(t, err, ErrReadOnly)
}

func TestGitHubMedium_Actions_Register(t *testing.T) {
	_, ok := FactoryFor(Scheme)
	require.True(t, ok)

	c := core.New()
	RegisterActions(c)

	assert.True(t, c.Action(ActionRead).Exists())
	assert.True(t, c.Action(ActionList).Exists())
	assert.True(t, c.Action(ActionClone).Exists())

	result := c.Action(ActionRead).Run(context.Background(), core.NewOptions(
		core.Option{Key: "owner", Value: "Snider"},
		core.Option{Key: "repo", Value: "demo"},
		core.Option{Key: "path", Value: ""},
	))
	assert.False(t, result.OK)
}
