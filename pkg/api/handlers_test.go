// SPDX-License-Identifier: EUPL-1.2

package api

import (
	"context"
	"net/http"
	"net/http/httptest"

	. "dappco.re/go"
	coreio "dappco.re/go/io"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

const testPathKey = "pa" + "th"

func TestCreateWorkspace_Good_Delegates(t *T) {
	router := testRouter(NewProvider(nil))

	rec := postJSON(t, router, "/v1/workspace", `{"workspace":"alice"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !Contains(rec.Body.String(), `"success":true`) {
		t.Fatalf("expected success response, got %s", rec.Body.String())
	}
}

func TestCreateWorkspace_Bad_InvalidJSON(t *T) {
	router := testRouter(NewProvider(nil))

	rec := postJSON(t, router, "/v1/workspace", `{`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	assertAPIErrorCode(t, rec, "invalid_request")
}

func TestSwitchWorkspace_Good_Delegates(t *T) {
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

func TestSwitchWorkspace_Bad_EmptyID(t *T) {
	router := testRouter(NewProvider(nil))

	rec := postJSON(t, router, "/v1/workspace/%20/switch", `{}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	assertAPIErrorCode(t, rec, "invalid_request")
}

func TestHandleWorkspaceCommand_Good_Delegates(t *T) {
	router := testRouter(NewProvider(nil))

	create := postJSON(t, router, "/v1/workspace", `{"workspace":"ws-1"}`)
	if create.Code != http.StatusOK {
		t.Fatalf("expected create 200, got %d: %s", create.Code, create.Body.String())
	}
	write := postJSON(t, router, "/v1/workspace/ws-1/command", Sprintf(`{"action":"write",%q:"note.txt","content":"hello"}`, testPathKey))
	if write.Code != http.StatusOK {
		t.Fatalf("expected write 200, got %d: %s", write.Code, write.Body.String())
	}
	read := postJSON(t, router, "/v1/workspace/ws-1/command", Sprintf(`{"action":"read",%q:"note.txt"}`, testPathKey))
	if read.Code != http.StatusOK {
		t.Fatalf("expected read 200, got %d: %s", read.Code, read.Body.String())
	}
	if !Contains(read.Body.String(), "hello") {
		t.Fatalf("expected response to contain read content, got %s", read.Body.String())
	}
}

func TestHandleWorkspaceCommand_Bad_MissingAction(t *T) {
	router := testRouter(NewProvider(nil))

	rec := postJSON(t, router, "/v1/workspace/ws-1/command", `{}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	assertAPIErrorCode(t, rec, "invalid_request")
}

func TestMediumDispatcher_Good_MemoryRoundTrip(t *T) {
	router := testRouter(NewProvider(nil))

	write := postJSON(t, router, "/v1/medium/memory/write", Sprintf(`{%q:"note.txt","content":"hello"}`, testPathKey))
	if write.Code != http.StatusOK {
		t.Fatalf("expected write 200, got %d: %s", write.Code, write.Body.String())
	}

	read := postJSON(t, router, "/v1/medium/memory/read", Sprintf(`{%q:"note.txt"}`, testPathKey))
	if read.Code != http.StatusOK {
		t.Fatalf("expected read 200, got %d: %s", read.Code, read.Body.String())
	}
	if !Contains(read.Body.String(), "hello") {
		t.Fatalf("expected response to contain read content, got %s", read.Body.String())
	}
}

func TestMediumDispatcher_Bad_UnsupportedMedium(t *T) {
	router := testRouter(NewProvider(nil))

	rec := postJSON(t, router, "/v1/medium/github/read", Sprintf(`{%q:"README.md"}`, testPathKey))
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d: %s", rec.Code, rec.Body.String())
	}
	assertAPIErrorCode(t, rec, "not_implemented")
}

func TestActionDispatcher_Good_WiredActionDelegates(t *T) {
	coreio.ResetMemoryActionStore()
	defer coreio.ResetMemoryActionStore()

	router := testRouter(NewProvider(nil))

	write := postJSON(t, router, "/v1/io/core.io.memory.write", Sprintf(`{%q:"config/app.yaml","content":"port: 8080"}`, testPathKey))
	if write.Code != http.StatusOK {
		t.Fatalf("expected write action 200, got %d: %s", write.Code, write.Body.String())
	}

	read := postJSON(t, router, "/v1/io/core.io.memory.read", Sprintf(`{%q:"config/app.yaml"}`, testPathKey))
	if read.Code != http.StatusOK {
		t.Fatalf("expected read action 200, got %d: %s", read.Code, read.Body.String())
	}
	if !Contains(read.Body.String(), "port: 8080") {
		t.Fatalf("expected delegated action content, got %s", read.Body.String())
	}
}

func TestActionDispatcher_Bad_UnknownAction(t *T) {
	router := testRouter(NewProvider(nil))

	rec := postJSON(t, router, "/v1/io/core.io.unknown", `{}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
	assertAPIErrorCode(t, rec, "unknown_action")
}

func TestActionDispatcher_Good_FormerMissingActionDelegates(t *T) {
	c := New()
	provider := NewProvider(c)
	c.Action(coreio.ActionS3Read, func(_ context.Context, opts Options) Result {
		if opts.String("pa"+"th") != "reports/daily.txt" {
			return Result{}.New(E("test", "unexpected path", nil))
		}
		return Ok("delegated s3 read")
	})
	router := testRouter(provider)

	rec := postJSON(t, router, "/v1/io/core.io.s3.read", Sprintf(`{%q:"reports/daily.txt"}`, testPathKey))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !Contains(rec.Body.String(), "delegated s3 read") {
		t.Fatalf("expected delegated response, got %s", rec.Body.String())
	}
}

func testRouter(provider *IOProvider) *gin.Engine {
	router := gin.New()
	provider.RegisterRoutes(router.Group(provider.BasePath()))
	return router
}

func postJSON(t *T, router http.Handler, path string, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func assertAPIErrorCode(t *T, rec *httptest.ResponseRecorder, code string) {
	t.Helper()
	var resp apiResponse
	if result := JSONUnmarshal(rec.Body.Bytes(), &resp); !result.OK {
		t.Fatalf("decode response: %s; body=%s", result.Error(), rec.Body.String())
	}
	if resp.Error == nil {
		t.Fatalf("expected error response, got %s", rec.Body.String())
	}
	if resp.Error.Code != code {
		t.Fatalf("expected error code %q, got %q", code, resp.Error.Code)
	}
}
