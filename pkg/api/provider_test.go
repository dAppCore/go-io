package api

import (
	. "dappco.re/go"
	coreio "dappco.re/go/io"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
)

// SPDX-License-Identifier: EUPL-1.2

func TestNewProvider_Good(t *T) {
	c := New()
	provider := NewProvider(c)

	if provider.Name() != "io" {
		t.Fatalf("expected provider name io, got %q", provider.Name())
	}
	if provider.BasePath() != "/v1" {
		t.Fatalf("expected base path /v1, got %q", provider.BasePath())
	}
	if !c.Action(coreio.ActionMemoryRead).Exists() {
		t.Fatalf("expected %s to be registered", coreio.ActionMemoryRead)
	}
	if got := len(rfc15Actions); got != 18 {
		t.Fatalf("expected 18 RFC actions, got %d", got)
	}
}

func TestNewProvider_Bad(t *T) {
	provider := NewProvider(nil)
	if provider == nil {
		t.Fatal("expected provider")
		return
	}
	if provider.core == nil {
		t.Fatal("expected provider core registry")
		return
	}
	if !provider.core.Action(coreio.ActionLocalRead).Exists() {
		t.Fatalf("expected %s to be registered on default core", coreio.ActionLocalRead)
	}
}

func TestNewProvider_Ugly(t *T) {
	c := New()
	coreio.RegisterActions(c)

	provider := NewProvider(c)
	if provider == nil {
		t.Fatal("expected provider")
		return
	}
	if !provider.core.Action(coreio.ActionCopy).Exists() {
		t.Fatalf("expected %s to remain registered after duplicate registration", coreio.ActionCopy)
	}
	if len(provider.Describe()) != 5 {
		t.Fatalf("expected 5 route descriptions, got %d", len(provider.Describe()))
	}
}

func TestProvider_NewProvider_Good(t *T) {
	c := New()
	provider := NewProvider(c)
	AssertNotNil(t, provider)
	AssertEqual(t, "io", provider.Name())
}

func TestProvider_NewProvider_Bad(t *T) {
	provider := NewProvider(nil)
	AssertNotNil(t, provider)
	AssertNotNil(t, provider.core)
}

func TestProvider_NewProvider_Ugly(t *T) {
	c := New()
	provider := NewProvider(c)
	second := NewProvider(c)
	AssertEqual(t, provider.Name(), second.Name())
}

func TestProvider_IOProvider_Name_Good(t *T) {
	provider := NewProvider(New())
	got := provider.Name()
	AssertEqual(t, "io", got)
}

func TestProvider_IOProvider_Name_Bad(t *T) {
	provider := &IOProvider{}
	got := provider.Name()
	AssertEqual(t, "io", got)
}

func TestProvider_IOProvider_Name_Ugly(t *T) {
	provider := NewProvider(nil)
	got := provider.Name()
	AssertEqual(t, "io", got)
}

func TestProvider_IOProvider_BasePath_Good(t *T) {
	provider := NewProvider(New())
	got := provider.BasePath()
	AssertEqual(t, "/v1", got)
}

func TestProvider_IOProvider_BasePath_Bad(t *T) {
	provider := &IOProvider{}
	got := provider.BasePath()
	AssertEqual(t, "/v1", got)
}

func TestProvider_IOProvider_BasePath_Ugly(t *T) {
	provider := NewProvider(nil)
	got := provider.BasePath()
	AssertTrue(t, HasPrefix(got, "/"))
}

func TestProvider_IOProvider_Describe_Good(t *T) {
	provider := NewProvider(New())
	routes := provider.Describe()
	AssertLen(t, routes, 5)
}

func TestProvider_IOProvider_Describe_Bad(t *T) {
	provider := &IOProvider{}
	routes := provider.Describe()
	AssertLen(t, routes, 5)
}

func TestProvider_IOProvider_Describe_Ugly(t *T) {
	provider := NewProvider(nil)
	routes := provider.Describe()
	AssertTrue(t, HasPrefix(routes[0].Path, "/"))
}

func TestProvider_IOProvider_Register_Good(t *T) {
	provider := NewProvider(New())
	router := gin.New()
	provider.Register(router)
	AssertNotNil(t, router)
}

func TestProvider_IOProvider_Register_Bad(t *T) {
	provider := NewProvider(nil)
	AssertNotPanics(t, func() { provider.Register(nil) })
	AssertNotNil(t, provider.core)
}

func TestProvider_IOProvider_Register_Ugly(t *T) {
	var provider *IOProvider
	router := gin.New()
	AssertNotPanics(t, func() { provider.Register(router) })
}

func TestProvider_IOProvider_RegisterRoutes_Good(t *T) {
	provider := NewProvider(nil)
	router := gin.New()
	provider.RegisterRoutes(router.Group(provider.BasePath()))
	rec := httptest.NewRecorder()
	AssertNotNil(t, rec)
}

func TestProvider_IOProvider_RegisterRoutes_Bad(t *T) {
	provider := NewProvider(nil)
	router := gin.New()
	AssertNotPanics(t, func() { provider.RegisterRoutes(router.Group("/bad")) })
}

func TestProvider_IOProvider_RegisterRoutes_Ugly(t *T) {
	var provider *IOProvider
	router := gin.New()
	group := router.Group("/v1")
	AssertNotPanics(t, func() { provider.RegisterRoutes(group) })
}
