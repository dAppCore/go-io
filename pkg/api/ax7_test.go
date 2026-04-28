package api

import (
	"net/http/httptest"

	core "dappco.re/go"
	"github.com/gin-gonic/gin"
)

func TestAX7_NewProvider_Good(t *core.T) {
	c := core.New()
	provider := NewProvider(c)
	core.AssertNotNil(t, provider)
	core.AssertEqual(t, "io", provider.Name())
}

func TestAX7_NewProvider_Bad(t *core.T) {
	provider := NewProvider(nil)
	core.AssertNotNil(t, provider)
	core.AssertNotNil(t, provider.core)
}

func TestAX7_NewProvider_Ugly(t *core.T) {
	c := core.New()
	provider := NewProvider(c)
	second := NewProvider(c)
	core.AssertEqual(t, provider.Name(), second.Name())
}

func TestAX7_IOProvider_Name_Good(t *core.T) {
	provider := NewProvider(core.New())
	got := provider.Name()
	core.AssertEqual(t, "io", got)
}

func TestAX7_IOProvider_Name_Bad(t *core.T) {
	provider := &IOProvider{}
	got := provider.Name()
	core.AssertEqual(t, "io", got)
}

func TestAX7_IOProvider_Name_Ugly(t *core.T) {
	provider := NewProvider(nil)
	got := provider.Name()
	core.AssertEqual(t, "io", got)
}

func TestAX7_IOProvider_BasePath_Good(t *core.T) {
	provider := NewProvider(core.New())
	got := provider.BasePath()
	core.AssertEqual(t, "/v1", got)
}

func TestAX7_IOProvider_BasePath_Bad(t *core.T) {
	provider := &IOProvider{}
	got := provider.BasePath()
	core.AssertEqual(t, "/v1", got)
}

func TestAX7_IOProvider_BasePath_Ugly(t *core.T) {
	provider := NewProvider(nil)
	got := provider.BasePath()
	core.AssertTrue(t, core.HasPrefix(got, "/"))
}

func TestAX7_IOProvider_Describe_Good(t *core.T) {
	provider := NewProvider(core.New())
	routes := provider.Describe()
	core.AssertLen(t, routes, 5)
}

func TestAX7_IOProvider_Describe_Bad(t *core.T) {
	provider := &IOProvider{}
	routes := provider.Describe()
	core.AssertLen(t, routes, 5)
}

func TestAX7_IOProvider_Describe_Ugly(t *core.T) {
	provider := NewProvider(nil)
	routes := provider.Describe()
	core.AssertTrue(t, core.HasPrefix(routes[0].Path, "/"))
}

func TestAX7_IOProvider_Register_Good(t *core.T) {
	provider := NewProvider(core.New())
	router := gin.New()
	provider.Register(router)
	core.AssertNotNil(t, router)
}

func TestAX7_IOProvider_Register_Bad(t *core.T) {
	provider := NewProvider(nil)
	core.AssertNotPanics(t, func() { provider.Register(nil) })
	core.AssertNotNil(t, provider.core)
}

func TestAX7_IOProvider_Register_Ugly(t *core.T) {
	var provider *IOProvider
	router := gin.New()
	core.AssertNotPanics(t, func() { provider.Register(router) })
}

func TestAX7_IOProvider_RegisterRoutes_Good(t *core.T) {
	provider := NewProvider(nil)
	router := gin.New()
	provider.RegisterRoutes(router.Group(provider.BasePath()))
	rec := httptest.NewRecorder()
	core.AssertNotNil(t, rec)
}

func TestAX7_IOProvider_RegisterRoutes_Bad(t *core.T) {
	provider := NewProvider(nil)
	router := gin.New()
	core.AssertNotPanics(t, func() { provider.RegisterRoutes(router.Group("/bad")) })
}

func TestAX7_IOProvider_RegisterRoutes_Ugly(t *core.T) {
	var provider *IOProvider
	router := gin.New()
	group := router.Group("/v1")
	core.AssertNotPanics(t, func() { provider.RegisterRoutes(group) })
}
