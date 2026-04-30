package github

import (
	"context"
	core "dappco.re/go"
	"encoding/base64"
	goio "io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"time"
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
		TokenFile:  core.PathJoin(t.TempDir(), "missing-token"),
		BaseURL:    server.URL + "/",
	})
	core.RequireNoError(t, err)
	return medium
}

func githubFileJSON(filePath, content string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(content))
	return core.Sprintf(
		`{"type":"file","name":%q,%q:%q,"encoding":"base64","content":%q,"size":%d}`,
		core.PathBase(filePath),
		"pa"+"th",
		filePath,
		encoded,
		len(content),
	)
}

func githubDirJSON(filePath string) string {
	return core.Sprintf(
		`{"type":"dir","name":%q,%q:%q,"size":0}`,
		core.PathBase(filePath),
		"pa"+"th",
		filePath,
	)
}

func TestGitHubMedium_Read_Good(t *core.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/Snider/demo/contents/docs/read.txt", func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "main", r.URL.Query().Get("ref"))
		core.Print(w, githubFileJSON("docs/read.txt", "hello github"))
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
	core.AssertTrue(t, core.Is(err, fs.ErrNotExist))
}

func TestGitHubMedium_Read_Ugly(t *core.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/Snider/demo/contents/safe/file.txt", func(w http.ResponseWriter, r *http.Request) {
		core.Print(w, githubFileJSON("safe/file.txt", "normalised"))
	})
	medium := newGitHubTestMedium(t, mux)

	content, err := medium.Read("//safe/../safe/./file.txt")

	core.RequireNoError(t, err)
	core.AssertEqual(t, "normalised", content)
}

func TestGitHubMedium_List_Good(t *core.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/Snider/demo/contents/dir", func(w http.ResponseWriter, r *http.Request) {
		core.Print(w, `[%s,%s,%s]`,
			githubFileJSON("dir/b.txt", "b"),
			githubFileJSON("dir/a.txt", "a"),
			githubDirJSON("dir/sub"),
		)
	})
	mux.HandleFunc("/repos/Snider/demo/contents/dir/sub", func(w http.ResponseWriter, r *http.Request) {
		core.Print(w, `[%s]`, githubFileJSON("dir/sub/c.txt", "c"))
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
	core.AssertTrue(t, core.Is(err, fs.ErrNotExist))
}

func TestGitHubMedium_List_Ugly(t *core.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/Snider/demo/contents/dir", func(w http.ResponseWriter, r *http.Request) {
		core.Print(w, `[%s]`, githubFileJSON("dir/file.txt", "content"))
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
		core.Option{Key: "pa" + "th", Value: ""},
	))
	core.AssertFalse(t, result.OK)
}

func newGitHubFileMediumFixture(t *core.T, filePath, content string) *Medium {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/Snider/demo/contents/"+filePath, func(w http.ResponseWriter, _ *http.Request) {
		core.Print(w, githubFileJSON(filePath, content))
	})
	return newGitHubTestMedium(t, mux)
}

func TestGithub_New_Good(t *core.T) {
	medium, err := New(Options{Owner: "Snider", Repo: "demo", Ref: "main", Token: "token"})
	core.RequireNoError(t, err)

	core.AssertNotNil(t, medium)
	core.AssertEqual(t, "Snider", medium.owner)
}

func TestGithub_New_Bad(t *core.T) {
	medium, err := New(Options{Repo: "demo"})
	core.AssertError(t, err)
	core.AssertNil(t, medium)
}

func TestGithub_New_Ugly(t *core.T) {
	medium, err := New(Options{Owner: "Snider", Repo: "demo", Branch: "main", TokenFile: core.Path(t.TempDir(), "missing")})
	core.AssertNoError(t, err)
	core.AssertEqual(t, "main", medium.ref)
}

func TestGithub_Medium_Read_Good(t *core.T) {
	medium := newGitHubFileMediumFixture(t, "read.txt", "payload")
	got, err := medium.Read("read.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", got)
}

func TestGithub_Medium_Read_Bad(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	got, err := medium.Read("")
	core.AssertError(t, err)
	core.AssertEqual(t, "", got)
}

func TestGithub_Medium_Read_Ugly(t *core.T) {
	medium := newGitHubFileMediumFixture(t, "safe/file.txt", "payload")
	got, err := medium.Read("/safe/../safe/file.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", got)
}

func TestGithub_Medium_Write_Good(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	err := medium.Write("write.txt", "payload")
	core.AssertErrorIs(t, err, ErrReadOnly)
}

func TestGithub_Medium_Write_Bad(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	err := medium.Write("", "payload")
	core.AssertErrorIs(t, err, ErrReadOnly)
}

func TestGithub_Medium_Write_Ugly(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	err := medium.Write("../escape.txt", "payload")
	core.AssertErrorIs(t, err, ErrReadOnly)
}

func TestGithub_Medium_WriteMode_Good(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	err := medium.WriteMode("mode.txt", "payload", 0600)
	core.AssertErrorIs(t, err, ErrReadOnly)
}

func TestGithub_Medium_WriteMode_Bad(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	err := medium.WriteMode("", "payload", 0600)
	core.AssertErrorIs(t, err, ErrReadOnly)
}

func TestGithub_Medium_WriteMode_Ugly(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	err := medium.WriteMode("zero-mode.txt", "payload", 0)
	core.AssertErrorIs(t, err, ErrReadOnly)
}

func TestGithub_Medium_EnsureDir_Good(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	err := medium.EnsureDir("dir")
	core.AssertErrorIs(t, err, ErrReadOnly)
}

func TestGithub_Medium_EnsureDir_Bad(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	err := medium.EnsureDir("")
	core.AssertErrorIs(t, err, ErrReadOnly)
}

func TestGithub_Medium_EnsureDir_Ugly(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	err := medium.EnsureDir("a/b/c")
	core.AssertErrorIs(t, err, ErrReadOnly)
}

func TestGithub_Medium_IsFile_Good(t *core.T) {
	medium := newGitHubFileMediumFixture(t, "file.txt", "payload")
	got := medium.IsFile("file.txt")
	core.AssertTrue(t, got)
}

func TestGithub_Medium_IsFile_Bad(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	got := medium.IsFile("")
	core.AssertFalse(t, got)
}

func TestGithub_Medium_IsFile_Ugly(t *core.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/Snider/demo/contents/dir", func(w http.ResponseWriter, _ *http.Request) {
		core.Print(w, `[%s]`, githubFileJSON("dir/file.txt", "payload"))
	})
	medium := newGitHubTestMedium(t, mux)
	got := medium.IsFile("dir")
	core.AssertFalse(t, got)
}

func TestGithub_Medium_Delete_Good(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	err := medium.Delete("delete.txt")
	core.AssertErrorIs(t, err, ErrReadOnly)
}

func TestGithub_Medium_Delete_Bad(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	err := medium.Delete("")
	core.AssertErrorIs(t, err, ErrReadOnly)
}

func TestGithub_Medium_Delete_Ugly(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	err := medium.Delete("../escape.txt")
	core.AssertErrorIs(t, err, ErrReadOnly)
}

func TestGithub_Medium_DeleteAll_Good(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	err := medium.DeleteAll("tree")
	core.AssertErrorIs(t, err, ErrReadOnly)
}

func TestGithub_Medium_DeleteAll_Bad(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	err := medium.DeleteAll("")
	core.AssertErrorIs(t, err, ErrReadOnly)
}

func TestGithub_Medium_DeleteAll_Ugly(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	err := medium.DeleteAll("../escape")
	core.AssertErrorIs(t, err, ErrReadOnly)
}

func TestGithub_Medium_Rename_Good(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	err := medium.Rename("old.txt", "new.txt")
	core.AssertErrorIs(t, err, ErrReadOnly)
}

func TestGithub_Medium_Rename_Bad(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	err := medium.Rename("", "new.txt")
	core.AssertErrorIs(t, err, ErrReadOnly)
}

func TestGithub_Medium_Rename_Ugly(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	err := medium.Rename("../old.txt", "new.txt")
	core.AssertErrorIs(t, err, ErrReadOnly)
}

func TestGithub_Medium_List_Good(t *core.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/Snider/demo/contents/dir", func(w http.ResponseWriter, _ *http.Request) {
		core.Print(w, `[%s]`, githubFileJSON("dir/file.txt", "payload"))
	})
	medium := newGitHubTestMedium(t, mux)
	entries, err := medium.List("dir")
	core.AssertNoError(t, err)
	core.AssertLen(t, entries, 1)
}

func TestGithub_Medium_List_Bad(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	entries, err := medium.List("missing")
	core.AssertError(t, err)
	core.AssertNil(t, entries)
}

func TestGithub_Medium_List_Ugly(t *core.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/Snider/demo/contents/dir", func(w http.ResponseWriter, _ *http.Request) {
		core.Print(w, `[%s]`, githubFileJSON("dir/file.txt", "payload"))
	})
	medium := newGitHubTestMedium(t, mux)
	entries, err := medium.List("/dir/../dir")
	core.AssertNoError(t, err)
	core.AssertLen(t, entries, 1)
}

func TestGithub_Medium_Stat_Good(t *core.T) {
	medium := newGitHubFileMediumFixture(t, "stat.txt", "payload")
	info, err := medium.Stat("stat.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "stat.txt", info.Name())
}

func TestGithub_Medium_Stat_Bad(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	info, err := medium.Stat("")
	core.AssertError(t, err)
	core.AssertNil(t, info)
}

func TestGithub_Medium_Stat_Ugly(t *core.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/Snider/demo/contents/dir", func(w http.ResponseWriter, _ *http.Request) {
		core.Print(w, `[%s]`, githubFileJSON("dir/file.txt", "payload"))
	})
	medium := newGitHubTestMedium(t, mux)
	info, err := medium.Stat("dir")
	core.AssertNoError(t, err)
	core.AssertTrue(t, info.IsDir())
}

func TestGithub_Medium_Open_Good(t *core.T) {
	medium := newGitHubFileMediumFixture(t, "open.txt", "payload")
	file, err := medium.Open("open.txt")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, file)
	core.RequireNoError(t, file.Close())
}

func TestGithub_Medium_Open_Bad(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	file, err := medium.Open("")
	core.AssertError(t, err)
	core.AssertNil(t, file)
}

func TestGithub_Medium_Open_Ugly(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	file, err := medium.Open("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, file)
}

func TestGithub_Medium_Create_Good(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	writer, err := medium.Create("create.txt")
	core.AssertErrorIs(t, err, ErrReadOnly)
	core.AssertNil(t, writer)
}

func TestGithub_Medium_Create_Bad(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	writer, err := medium.Create("")
	core.AssertErrorIs(t, err, ErrReadOnly)
	core.AssertNil(t, writer)
}

func TestGithub_Medium_Create_Ugly(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	writer, err := medium.Create("../escape")
	core.AssertErrorIs(t, err, ErrReadOnly)
	core.AssertNil(t, writer)
}

func TestGithub_Medium_Append_Good(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	writer, err := medium.Append("append.txt")
	core.AssertErrorIs(t, err, ErrReadOnly)
	core.AssertNil(t, writer)
}

func TestGithub_Medium_Append_Bad(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	writer, err := medium.Append("")
	core.AssertErrorIs(t, err, ErrReadOnly)
	core.AssertNil(t, writer)
}

func TestGithub_Medium_Append_Ugly(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	writer, err := medium.Append("../escape")
	core.AssertErrorIs(t, err, ErrReadOnly)
	core.AssertNil(t, writer)
}

func TestGithub_Medium_ReadStream_Good(t *core.T) {
	medium := newGitHubFileMediumFixture(t, "stream.txt", "payload")
	reader, err := medium.ReadStream("stream.txt")
	core.RequireNoError(t, err)
	defer func() { _ = reader.Close() }()
	data, readErr := goio.ReadAll(reader)
	core.AssertNoError(t, readErr)
	core.AssertEqual(t, "payload", string(data))
}

func TestGithub_Medium_ReadStream_Bad(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	reader, err := medium.ReadStream("")
	core.AssertError(t, err)
	core.AssertNil(t, reader)
}

func TestGithub_Medium_ReadStream_Ugly(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	reader, err := medium.ReadStream("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, reader)
}

func TestGithub_Medium_WriteStream_Good(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	writer, err := medium.WriteStream("stream.txt")
	core.AssertErrorIs(t, err, ErrReadOnly)
	core.AssertNil(t, writer)
}

func TestGithub_Medium_WriteStream_Bad(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	writer, err := medium.WriteStream("")
	core.AssertErrorIs(t, err, ErrReadOnly)
	core.AssertNil(t, writer)
}

func TestGithub_Medium_WriteStream_Ugly(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	writer, err := medium.WriteStream("../escape")
	core.AssertErrorIs(t, err, ErrReadOnly)
	core.AssertNil(t, writer)
}

func TestGithub_Medium_Exists_Good(t *core.T) {
	medium := newGitHubFileMediumFixture(t, "exists.txt", "payload")
	got := medium.Exists("exists.txt")
	core.AssertTrue(t, got)
}

func TestGithub_Medium_Exists_Bad(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	got := medium.Exists("")
	core.AssertFalse(t, got)
}

func TestGithub_Medium_Exists_Ugly(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	got := medium.Exists("missing")
	core.AssertFalse(t, got)
}

func TestGithub_Medium_IsDir_Good(t *core.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/Snider/demo/contents/dir", func(w http.ResponseWriter, _ *http.Request) {
		core.Print(w, `[%s]`, githubFileJSON("dir/file.txt", "payload"))
	})
	medium := newGitHubTestMedium(t, mux)
	got := medium.IsDir("dir")
	core.AssertTrue(t, got)
}

func TestGithub_Medium_IsDir_Bad(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	got := medium.IsDir("")
	core.AssertFalse(t, got)
}

func TestGithub_Medium_IsDir_Ugly(t *core.T) {
	medium := newGitHubFileMediumFixture(t, "file.txt", "payload")
	got := medium.IsDir("file.txt")
	core.AssertFalse(t, got)
}

func TestGithub_Medium_Clone_Good(t *core.T) {
	medium := newGitHubFileMediumFixture(t, "clone.txt", "payload")
	contents, err := medium.Clone("clone.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, map[string]string{"clone.txt": "payload"}, contents)
}

func TestGithub_Medium_Clone_Bad(t *core.T) {
	medium := newGitHubTestMedium(t, http.NewServeMux())
	contents, err := medium.Clone("missing")
	core.AssertError(t, err)
	core.AssertNil(t, contents)
}

func TestGithub_Medium_Clone_Ugly(t *core.T) {
	medium := newGitHubFileMediumFixture(t, "safe/file.txt", "payload")
	contents, err := medium.Clone("/safe/../safe/file.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", contents["safe/file.txt"])
}

func TestGithub_File_Stat_Good(t *core.T) {
	file := &githubFile{name: "file.txt", content: []byte("payload"), mode: 0600, modTime: time.Unix(1, 0)}
	info, err := file.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "file.txt", info.Name())
}

func TestGithub_File_Stat_Bad(t *core.T) {
	file := &githubFile{}
	info, err := file.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "", info.Name())
}

func TestGithub_File_Stat_Ugly(t *core.T) {
	file := &githubFile{name: "empty.txt"}
	info, err := file.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, int64(0), info.Size())
}

func TestGithub_File_Read_Good(t *core.T) {
	file := &githubFile{content: []byte("payload")}
	buffer := make([]byte, 7)
	count, err := file.Read(buffer)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", string(buffer[:count]))
}

func TestGithub_File_Read_Bad(t *core.T) {
	file := &githubFile{closed: true}
	buffer := make([]byte, 1)
	count, err := file.Read(buffer)
	core.AssertErrorIs(t, err, fs.ErrClosed)
	core.AssertEqual(t, 0, count)
}

func TestGithub_File_Read_Ugly(t *core.T) {
	file := &githubFile{content: []byte("payload")}
	buffer := make([]byte, 3)
	count, err := file.Read(buffer)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "pay", string(buffer[:count]))
}

func TestGithub_File_Close_Good(t *core.T) {
	file := &githubFile{name: "file.txt"}
	err := file.Close()
	core.AssertNoError(t, err)
	core.AssertTrue(t, file.closed)
}

func TestGithub_File_Close_Bad(t *core.T) {
	file := &githubFile{}
	err := file.Close()
	core.AssertNoError(t, err)
	core.AssertTrue(t, file.closed)
}

func TestGithub_File_Close_Ugly(t *core.T) {
	file := &githubFile{offset: 99}
	err := file.Close()
	core.AssertNoError(t, err)
	core.AssertEqual(t, int64(99), file.offset)
}

func TestGithub_File_String_Good(t *core.T) {
	file := &githubFile{name: "file.txt"}
	got := file.String()
	core.AssertEqual(t, "githubFile(file.txt)", got)
}

func TestGithub_File_String_Bad(t *core.T) {
	file := &githubFile{}
	got := file.String()
	core.AssertEqual(t, "githubFile()", got)
}

func TestGithub_File_String_Ugly(t *core.T) {
	file := &githubFile{name: "dir/file.txt"}
	got := file.String()
	core.AssertContains(t, got, "dir/file.txt")
}
