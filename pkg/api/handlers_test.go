// SPDX-License-Identifier: EUPL-1.2

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	goapi "dappco.re/go/api"
	core "dappco.re/go/core"
	coreio "dappco.re/go/io"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestCreateWorkspace_Good_Delegates(t *testing.T) {
	router := testRouter(NewProvider(nil))

	rec := postJSON(t, router, "/v1/workspace", `{"workspace":"alice"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"success":true`) {
		t.Fatalf("expected success response, got %s", rec.Body.String())
	}
}

func TestCreateWorkspace_Bad_InvalidJSON(t *testing.T) {
	router := testRouter(NewProvider(nil))

	rec := postJSON(t, router, "/v1/workspace", `{`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	assertAPIErrorCode(t, rec, "invalid_request")
}

func TestSwitchWorkspace_Good_Delegates(t *testing.T) {
	router := testRouter(NewProvider(nil))

	create := postJSON(t, router, "/v1/workspace", `{"workspace":"ws-1"}`)
	if create.Code != http.StatusOK {
		t.Fatalf("expected create 200, got %d: %s", create.Code, create.Body.String())
	}
	rec := postJSON(t, router, "/v1/workspace/ws-1/switch", `{}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSwitchWorkspace_Bad_EmptyID(t *testing.T) {
	router := testRouter(NewProvider(nil))

	rec := postJSON(t, router, "/v1/workspace/%20/switch", `{}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	assertAPIErrorCode(t, rec, "invalid_request")
}

func TestHandleWorkspaceCommand_Good_Delegates(t *testing.T) {
	router := testRouter(NewProvider(nil))

	create := postJSON(t, router, "/v1/workspace", `{"workspace":"ws-1"}`)
	if create.Code != http.StatusOK {
		t.Fatalf("expected create 200, got %d: %s", create.Code, create.Body.String())
	}
	write := postJSON(t, router, "/v1/workspace/ws-1/command", `{"action":"write","path":"note.txt","content":"hello"}`)
	if write.Code != http.StatusOK {
		t.Fatalf("expected write 200, got %d: %s", write.Code, write.Body.String())
	}
	read := postJSON(t, router, "/v1/workspace/ws-1/command", `{"action":"read","path":"note.txt"}`)
	if read.Code != http.StatusOK {
		t.Fatalf("expected read 200, got %d: %s", read.Code, read.Body.String())
	}
	if !strings.Contains(read.Body.String(), "hello") {
		t.Fatalf("expected response to contain read content, got %s", read.Body.String())
	}
}

func TestHandleWorkspaceCommand_Bad_MissingAction(t *testing.T) {
	router := testRouter(NewProvider(nil))

	rec := postJSON(t, router, "/v1/workspace/ws-1/command", `{}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	assertAPIErrorCode(t, rec, "invalid_request")
}

func TestMediumDispatcher_Good_MemoryRoundTrip(t *testing.T) {
	router := testRouter(NewProvider(nil))

	write := postJSON(t, router, "/v1/medium/memory/write", `{"path":"note.txt","content":"hello"}`)
	if write.Code != http.StatusOK {
		t.Fatalf("expected write 200, got %d: %s", write.Code, write.Body.String())
	}

	read := postJSON(t, router, "/v1/medium/memory/read", `{"path":"note.txt"}`)
	if read.Code != http.StatusOK {
		t.Fatalf("expected read 200, got %d: %s", read.Code, read.Body.String())
	}
	if !strings.Contains(read.Body.String(), "hello") {
		t.Fatalf("expected response to contain read content, got %s", read.Body.String())
	}
}

func TestMediumDispatcher_Bad_UnsupportedMedium(t *testing.T) {
	router := testRouter(NewProvider(nil))

	rec := postJSON(t, router, "/v1/medium/github/read", `{"path":"README.md"}`)
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d: %s", rec.Code, rec.Body.String())
	}
	assertAPIErrorCode(t, rec, "not_implemented")
}

func TestActionDispatcher_Good_WiredActionDelegates(t *testing.T) {
	coreio.ResetMemoryActionStore()
	defer coreio.ResetMemoryActionStore()

	router := testRouter(NewProvider(nil))

	write := postJSON(t, router, "/v1/io/core.io.memory.write", `{"path":"config/app.yaml","content":"port: 8080"}`)
	if write.Code != http.StatusOK {
		t.Fatalf("expected write action 200, got %d: %s", write.Code, write.Body.String())
	}

	read := postJSON(t, router, "/v1/io/core.io.memory.read", `{"path":"config/app.yaml"}`)
	if read.Code != http.StatusOK {
		t.Fatalf("expected read action 200, got %d: %s", read.Code, read.Body.String())
	}
	if !strings.Contains(read.Body.String(), "port: 8080") {
		t.Fatalf("expected delegated action content, got %s", read.Body.String())
	}
}

func TestActionDispatcher_Bad_UnknownAction(t *testing.T) {
	router := testRouter(NewProvider(nil))

	rec := postJSON(t, router, "/v1/io/core.io.unknown", `{}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
	assertAPIErrorCode(t, rec, "unknown_action")
}

func TestActionDispatcher_Good_FormerMissingActionDelegates(t *testing.T) {
	c := core.New()
	provider := NewProvider(c)
	c.Action(coreio.ActionS3Read, func(_ context.Context, opts core.Options) core.Result {
		if opts.String("path") != "reports/daily.txt" {
			return core.Result{}.New(core.E("test", "unexpected path", nil))
		}
		return core.Result{OK: true, Value: "delegated s3 read"}
	})
	router := testRouter(provider)

	rec := postJSON(t, router, "/v1/io/core.io.s3.read", `{"path":"reports/daily.txt"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "delegated s3 read") {
		t.Fatalf("expected delegated response, got %s", rec.Body.String())
	}
}

func testRouter(provider *IOProvider) *gin.Engine {
	router := gin.New()
	provider.RegisterRoutes(router.Group(provider.BasePath()))
	return router
}

func postJSON(t *testing.T, router http.Handler, path string, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func assertAPIErrorCode(t *testing.T, rec *httptest.ResponseRecorder, code string) {
	t.Helper()
	var resp goapi.Response[any]
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, rec.Body.String())
	}
	if resp.Error == nil {
		t.Fatalf("expected error response, got %s", rec.Body.String())
	}
	if resp.Error.Code != code {
		t.Fatalf("expected error code %q, got %q", code, resp.Error.Code)
	}
}
