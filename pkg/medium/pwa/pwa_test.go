package pwa

import (
	"context"
	"fmt"
	goio "io"
	"net/http"
	"net/http/httptest"

	core "dappco.re/go"
)

func newPWATestServer(t *core.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(w, `<html><head><link rel="manifest" href="%s/manifest.json"></head><body><script src="/app.js"></script></body></html>`, server.URL)
	})
	mux.HandleFunc("/manifest.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/manifest+json")
		_, _ = fmt.Fprint(w, `{"name":"Test PWA","start_url":"/index.html","icons":[{"src":"/icon.png"}]}`)
	})
	mux.HandleFunc("/index.html", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprint(w, `<html><body><img src="/icon.png"></body></html>`)
	})
	mux.HandleFunc("/app.js", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `console.log("pwa")`)
	})
	mux.HandleFunc("/icon.png", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("png"))
	})
	t.Cleanup(server.Close)
	return server
}

func TestPWAMedium_BorgDataNodeOperations_Good(t *core.T) {
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
		core.Option{Key: "path", Value: "index.html"},
	))
	core.AssertTrue(t, read.OK)
	core.AssertContains(t, read.Value.(string), "icon.png")

	list := c.Action(ActionList).Run(context.Background(), core.NewOptions(
		core.Option{Key: "medium", Value: medium},
		core.Option{Key: "path", Value: ""},
	))
	core.AssertTrue(t, list.OK)

	write := c.Action(ActionWrite).Run(context.Background(), core.NewOptions(
		core.Option{Key: "medium", Value: medium},
		core.Option{Key: "path", Value: "index.html"},
		core.Option{Key: "content", Value: "mutate"},
	))
	core.AssertFalse(t, write.OK)
}
