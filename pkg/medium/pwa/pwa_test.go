package pwa

import (
	"context"
	core "dappco.re/go"
	borgdatanode "forge.lthn.ai/Snider/Borg/pkg/datanode"
	goio "io"
	"io/fs"
	"net/http"
	"net/http/httptest"
)

func newPWATestServer(t *core.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		core.Print(w, `<html><head><link rel="manifest" href="%s/manifest.json"></head><body><script src="/app.js"></script></body></html>`, server.URL)
	})
	mux.HandleFunc("/manifest.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/manifest+json")
		core.Print(w, `{"name":"Test PWA","start_url":"/index.html","icons":[{"src":"/icon.png"}]}`)
	})
	mux.HandleFunc("/index.html", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		core.Print(w, `<html><body><img src="/icon.png"></body></html>`)
	})
	mux.HandleFunc("/app.js", func(w http.ResponseWriter, _ *http.Request) {
		core.Print(w, `console.log("pwa")`)
	})
	mux.HandleFunc("/icon.png", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("png"))
	})
	t.Cleanup(server.Close)
	return server
}

func TestPWAMedium_BorgDataNodeOperationsGood(t *core.T) {
	server := newPWATestServer(t)
	medium, err := New(Options{URL: server.URL})
	core.RequireNoError(t, err)

	content, err := medium.Read("index.html")
	core.RequireNoError(t, err)
	core.AssertContains(t, content, "icon.png")

	entries, err := medium.List("")
	core.RequireNoError(t, err)
	core.AssertNotEmpty(t, entries)

	info, err := medium.Stat("manifest.json")
	core.RequireNoError(t, err)
	core.AssertFalse(t, info.IsDir())

	reader, err := medium.ReadStream("icon.png")
	core.RequireNoError(t, err)
	data, readErr := goio.ReadAll(reader)
	core.RequireNoError(t, readErr)
	core.RequireNoError(t, reader.Close())
	core.AssertEqual(t, "png", string(data))

	core.AssertTrue(t, medium.IsFile("icon.png"))
	core.AssertError(t, medium.Write("index.html", "mutate"))
}

func TestPWAMedium_Actions_UseBorgDataNode(t *core.T) {
	_, ok := FactoryFor(Scheme)
	core.RequireTrue(t, ok)

	server := newPWATestServer(t)
	c := core.New()
	RegisterActions(c)

	scrape := c.Action(ActionScrape).Run(context.Background(), core.NewOptions(
		core.Option{Key: "url", Value: server.URL},
	))
	core.RequireTrue(t, scrape.OK)
	medium, ok := scrape.Value.(*Medium)
	core.RequireTrue(t, ok)

	read := c.Action(ActionRead).Run(context.Background(), core.NewOptions(
		core.Option{Key: "medium", Value: medium},
		core.Option{Key: "pa" + "th", Value: "index.html"},
	))
	core.AssertTrue(t, read.OK)
	core.AssertContains(t, read.Value.(string), "icon.png")

	list := c.Action(ActionList).Run(context.Background(), core.NewOptions(
		core.Option{Key: "medium", Value: medium},
		core.Option{Key: "pa" + "th", Value: ""},
	))
	core.AssertTrue(t, list.OK)

	write := c.Action(ActionWrite).Run(context.Background(), core.NewOptions(
		core.Option{Key: "medium", Value: medium},
		core.Option{Key: "pa" + "th", Value: "index.html"},
		core.Option{Key: "content", Value: "mutate"},
	))
	core.AssertFalse(t, write.OK)
}

func newPWAMediumFixture() *Medium {
	dataNode := borgdatanode.New()
	dataNode.AddData("page", []byte("payload"))
	dataNode.AddData("pages/item.txt", []byte("item"))
	return &Medium{url: "https://example.test", dataNode: dataNode}
}

func assertAX7PWANotImplemented(t *core.T, err error) {
	t.Helper()
	core.AssertError(t, err)
}

func TestPwa_New_Good(t *core.T) {
	medium, err := New(Options{URL: "https://example.test"})
	core.AssertNoError(t, err)
	core.AssertEqual(t, "https://example.test", medium.url)
}

func TestPwa_New_Bad(t *core.T) {
	medium, err := New(Options{})
	core.AssertNoError(t, err)
	core.AssertEqual(t, "", medium.url)
}

func TestPwa_New_Ugly(t *core.T) {
	medium, err := New(Options{URL: "  "})
	core.AssertNoError(t, err)
	core.AssertEqual(t, "  ", medium.url)
}

func TestPwa_Medium_Read_Good(t *core.T) {
	medium := newPWAMediumFixture()
	got, err := medium.Read("page")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", got)
}

func TestPwa_Medium_Read_Bad(t *core.T) {
	medium := newPWAMediumFixture()
	got, err := medium.Read("")
	assertAX7PWANotImplemented(t, err)
	core.AssertEqual(t, "", got)
}

func TestPwa_Medium_Read_Ugly(t *core.T) {
	medium := newPWAMediumFixture()
	got, err := medium.Read("../page")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", got)
}

func TestPwa_Medium_Write_Good(t *core.T) {
	medium := newPWAMediumFixture()
	err := medium.Write("page", "content")
	assertAX7PWANotImplemented(t, err)
	core.AssertTrue(t, medium.Exists("page"))
}

func TestPwa_Medium_Write_Bad(t *core.T) {
	medium := newPWAMediumFixture()
	err := medium.Write("", "content")
	assertAX7PWANotImplemented(t, err)
	core.AssertFalse(t, medium.IsFile(""))
}

func TestPwa_Medium_Write_Ugly(t *core.T) {
	medium := newPWAMediumFixture()
	err := medium.Write("../page", "content")
	assertAX7PWANotImplemented(t, err)
	core.AssertTrue(t, medium.Exists("../page"))
}

func TestPwa_Medium_WriteMode_Good(t *core.T) {
	medium := newPWAMediumFixture()
	err := medium.WriteMode("page", "content", 0600)
	assertAX7PWANotImplemented(t, err)
	core.AssertTrue(t, medium.Exists("page"))
}

func TestPwa_Medium_WriteMode_Bad(t *core.T) {
	medium := newPWAMediumFixture()
	err := medium.WriteMode("", "content", fs.FileMode(0))
	assertAX7PWANotImplemented(t, err)
	core.AssertFalse(t, medium.IsFile(""))
}

func TestPwa_Medium_WriteMode_Ugly(t *core.T) {
	medium := newPWAMediumFixture()
	err := medium.WriteMode("page", "content", fs.ModeDir)
	assertAX7PWANotImplemented(t, err)
	core.AssertTrue(t, medium.Exists("page"))
}

func TestPwa_Medium_EnsureDir_Good(t *core.T) {
	medium := newPWAMediumFixture()
	err := medium.EnsureDir("pages")
	assertAX7PWANotImplemented(t, err)
	core.AssertTrue(t, medium.IsDir("pages"))
}

func TestPwa_Medium_EnsureDir_Bad(t *core.T) {
	medium := newPWAMediumFixture()
	err := medium.EnsureDir("")
	assertAX7PWANotImplemented(t, err)
	core.AssertTrue(t, medium.IsDir(""))
}

func TestPwa_Medium_EnsureDir_Ugly(t *core.T) {
	medium := newPWAMediumFixture()
	err := medium.EnsureDir("../pages")
	assertAX7PWANotImplemented(t, err)
	core.AssertTrue(t, medium.Exists("../pages"))
}

func TestPwa_Medium_IsFile_Good(t *core.T) {
	medium := newPWAMediumFixture()
	got := medium.IsFile("page")
	core.AssertTrue(t, got)
}

func TestPwa_Medium_IsFile_Bad(t *core.T) {
	medium := newPWAMediumFixture()
	got := medium.IsFile("")
	core.AssertFalse(t, got)
}

func TestPwa_Medium_IsFile_Ugly(t *core.T) {
	medium := newPWAMediumFixture()
	got := medium.IsFile("../page")
	core.AssertTrue(t, got)
}

func TestPwa_Medium_Delete_Good(t *core.T) {
	medium := newPWAMediumFixture()
	err := medium.Delete("page")
	assertAX7PWANotImplemented(t, err)
	core.AssertTrue(t, medium.Exists("page"))
}

func TestPwa_Medium_Delete_Bad(t *core.T) {
	medium := newPWAMediumFixture()
	err := medium.Delete("")
	assertAX7PWANotImplemented(t, err)
	core.AssertTrue(t, medium.Exists(""))
}

func TestPwa_Medium_Delete_Ugly(t *core.T) {
	medium := newPWAMediumFixture()
	err := medium.Delete("../page")
	assertAX7PWANotImplemented(t, err)
	core.AssertTrue(t, medium.Exists("../page"))
}

func TestPwa_Medium_DeleteAll_Good(t *core.T) {
	medium := newPWAMediumFixture()
	err := medium.DeleteAll("pages")
	assertAX7PWANotImplemented(t, err)
	core.AssertTrue(t, medium.Exists("pages"))
}

func TestPwa_Medium_DeleteAll_Bad(t *core.T) {
	medium := newPWAMediumFixture()
	err := medium.DeleteAll("")
	assertAX7PWANotImplemented(t, err)
	core.AssertTrue(t, medium.Exists(""))
}

func TestPwa_Medium_DeleteAll_Ugly(t *core.T) {
	medium := newPWAMediumFixture()
	err := medium.DeleteAll("../pages")
	assertAX7PWANotImplemented(t, err)
	core.AssertTrue(t, medium.Exists("../pages"))
}

func TestPwa_Medium_Rename_Good(t *core.T) {
	medium := newPWAMediumFixture()
	err := medium.Rename("old", "new")
	assertAX7PWANotImplemented(t, err)
	core.AssertFalse(t, medium.Exists("new"))
}

func TestPwa_Medium_Rename_Bad(t *core.T) {
	medium := newPWAMediumFixture()
	err := medium.Rename("", "new")
	assertAX7PWANotImplemented(t, err)
	core.AssertFalse(t, medium.Exists("new"))
}

func TestPwa_Medium_Rename_Ugly(t *core.T) {
	medium := newPWAMediumFixture()
	err := medium.Rename("old", "../new")
	assertAX7PWANotImplemented(t, err)
	core.AssertFalse(t, medium.Exists("../new"))
}

func TestPwa_Medium_List_Good(t *core.T) {
	medium := newPWAMediumFixture()
	entries, err := medium.List("pages")
	core.AssertNoError(t, err)
	core.AssertLen(t, entries, 1)
}

func TestPwa_Medium_List_Bad(t *core.T) {
	medium := newPWAMediumFixture()
	entries, err := medium.List("")
	core.AssertNoError(t, err)
	core.AssertNotEmpty(t, entries)
}

func TestPwa_Medium_List_Ugly(t *core.T) {
	medium := newPWAMediumFixture()
	entries, err := medium.List("../pages")
	core.AssertNoError(t, err)
	core.AssertLen(t, entries, 1)
}

func TestPwa_Medium_Stat_Good(t *core.T) {
	medium := newPWAMediumFixture()
	info, err := medium.Stat("page")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "page", info.Name())
}

func TestPwa_Medium_Stat_Bad(t *core.T) {
	medium := newPWAMediumFixture()
	info, err := medium.Stat("")
	core.AssertNoError(t, err)
	core.AssertTrue(t, info.IsDir())
}

func TestPwa_Medium_Stat_Ugly(t *core.T) {
	medium := newPWAMediumFixture()
	info, err := medium.Stat("../page")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "page", info.Name())
}

func TestPwa_Medium_Open_Good(t *core.T) {
	medium := newPWAMediumFixture()
	file, err := medium.Open("page")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, file)
	core.RequireNoError(t, file.Close())
}

func TestPwa_Medium_Open_Bad(t *core.T) {
	medium := newPWAMediumFixture()
	file, err := medium.Open("")
	assertAX7PWANotImplemented(t, err)
	core.AssertNil(t, file)
}

func TestPwa_Medium_Open_Ugly(t *core.T) {
	medium := newPWAMediumFixture()
	file, err := medium.Open("../page")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, file)
	core.RequireNoError(t, file.Close())
}

func TestPwa_Medium_Create_Good(t *core.T) {
	medium := newPWAMediumFixture()
	writer, err := medium.Create("page")
	assertAX7PWANotImplemented(t, err)
	core.AssertNil(t, writer)
}

func TestPwa_Medium_Create_Bad(t *core.T) {
	medium := newPWAMediumFixture()
	writer, err := medium.Create("")
	assertAX7PWANotImplemented(t, err)
	core.AssertNil(t, writer)
}

func TestPwa_Medium_Create_Ugly(t *core.T) {
	medium := newPWAMediumFixture()
	writer, err := medium.Create("../page")
	assertAX7PWANotImplemented(t, err)
	core.AssertNil(t, writer)
}

func TestPwa_Medium_Append_Good(t *core.T) {
	medium := newPWAMediumFixture()
	writer, err := medium.Append("page")
	assertAX7PWANotImplemented(t, err)
	core.AssertNil(t, writer)
}

func TestPwa_Medium_Append_Bad(t *core.T) {
	medium := newPWAMediumFixture()
	writer, err := medium.Append("")
	assertAX7PWANotImplemented(t, err)
	core.AssertNil(t, writer)
}

func TestPwa_Medium_Append_Ugly(t *core.T) {
	medium := newPWAMediumFixture()
	writer, err := medium.Append("../page")
	assertAX7PWANotImplemented(t, err)
	core.AssertNil(t, writer)
}

func TestPwa_Medium_ReadStream_Good(t *core.T) {
	medium := newPWAMediumFixture()
	reader, err := medium.ReadStream("page")
	core.RequireNoError(t, err)
	data, readErr := goio.ReadAll(reader)
	core.AssertNoError(t, readErr)
	core.AssertEqual(t, "payload", string(data))
	core.RequireNoError(t, reader.Close())
}

func TestPwa_Medium_ReadStream_Bad(t *core.T) {
	medium := newPWAMediumFixture()
	reader, err := medium.ReadStream("")
	assertAX7PWANotImplemented(t, err)
	core.AssertNil(t, reader)
}

func TestPwa_Medium_ReadStream_Ugly(t *core.T) {
	medium := newPWAMediumFixture()
	reader, err := medium.ReadStream("../page")
	core.RequireNoError(t, err)
	data, readErr := goio.ReadAll(reader)
	core.AssertNoError(t, readErr)
	core.AssertEqual(t, "payload", string(data))
	core.RequireNoError(t, reader.Close())
}

func TestPwa_Medium_WriteStream_Good(t *core.T) {
	medium := newPWAMediumFixture()
	writer, err := medium.WriteStream("page")
	assertAX7PWANotImplemented(t, err)
	core.AssertNil(t, writer)
}

func TestPwa_Medium_WriteStream_Bad(t *core.T) {
	medium := newPWAMediumFixture()
	writer, err := medium.WriteStream("")
	assertAX7PWANotImplemented(t, err)
	core.AssertNil(t, writer)
}

func TestPwa_Medium_WriteStream_Ugly(t *core.T) {
	medium := newPWAMediumFixture()
	writer, err := medium.WriteStream("../page")
	assertAX7PWANotImplemented(t, err)
	core.AssertNil(t, writer)
}

func TestPwa_Medium_Exists_Good(t *core.T) {
	medium := newPWAMediumFixture()
	got := medium.Exists("page")
	core.AssertTrue(t, got)
}

func TestPwa_Medium_Exists_Bad(t *core.T) {
	medium := newPWAMediumFixture()
	got := medium.Exists("")
	core.AssertTrue(t, got)
}

func TestPwa_Medium_Exists_Ugly(t *core.T) {
	medium := newPWAMediumFixture()
	got := medium.Exists("../page")
	core.AssertTrue(t, got)
}

func TestPwa_Medium_IsDir_Good(t *core.T) {
	medium := newPWAMediumFixture()
	got := medium.IsDir("pages")
	core.AssertTrue(t, got)
}

func TestPwa_Medium_IsDir_Bad(t *core.T) {
	medium := newPWAMediumFixture()
	got := medium.IsDir("")
	core.AssertTrue(t, got)
}

func TestPwa_Medium_IsDir_Ugly(t *core.T) {
	medium := newPWAMediumFixture()
	got := medium.IsDir("../pages")
	core.AssertTrue(t, got)
}
