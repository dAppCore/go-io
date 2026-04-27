// SPDX-License-Identifier: EUPL-1.2

// Package api exposes go-io primitives as a Core API service provider.
package api

import (
	"net/http"

	core "dappco.re/go/core"
	coreio "dappco.re/go/io"
	"dappco.re/go/io/cube"
	"github.com/gin-gonic/gin"
)

// ParameterDescription describes a single HTTP route parameter.
type ParameterDescription struct {
	Name     string
	In       string
	Required bool
	Schema   map[string]any
}

// RouteDescription describes an HTTP route exposed by IOProvider.
type RouteDescription struct {
	Method      string
	Path        string
	Summary     string
	Description string
	Tags        []string
	StatusCode  int
	Parameters  []ParameterDescription
	RequestBody map[string]any
	Response    map[string]any
}

// IOProvider wraps go-io's library-only surface as HTTP routes.
type IOProvider struct {
	core   *core.Core
	local  coreio.Medium
	memory coreio.Medium
}

// NewProvider creates an IO provider backed by a Core action registry.
//
// Pass nil or no Core to create a private registry with go-io and cube actions
// registered. The variadic form keeps the provider easy to mount from core/api
// while still allowing tests and callers to inject a Core.
func NewProvider(cores ...*core.Core) *IOProvider {
	var c *core.Core
	if len(cores) > 0 {
		c = cores[0]
	}
	if c == nil {
		c = core.New()
	}
	coreio.RegisterActions(c)
	cube.RegisterActions(c)
	return &IOProvider{
		core:   c,
		local:  configuredLocalMedium(),
		memory: coreio.NewMemoryMedium(),
	}
}

// Name implements api.RouteGroup.
func (p *IOProvider) Name() string { return "io" }

// BasePath implements api.RouteGroup.
func (p *IOProvider) BasePath() string { return "/v1" }

// Register mounts the provider on a Gin router using the provider base path.
func (p *IOProvider) Register(r gin.IRouter) {
	if p == nil || r == nil {
		return
	}
	p.RegisterRoutes(r.Group(p.BasePath()))
}

// RegisterRoutes implements api.RouteGroup.
func (p *IOProvider) RegisterRoutes(rg *gin.RouterGroup) {
	if p == nil || rg == nil {
		return
	}
	rg.POST("/workspace", p.createWorkspace)
	rg.POST("/workspace/:id/switch", p.switchWorkspace)
	rg.POST("/workspace/:id/command", p.handleWorkspaceCommand)
	rg.POST("/medium/:type/:op", p.dispatchMedium)
	rg.POST("/io/:action", p.dispatchAction)
}

// Describe implements api.DescribableGroup.
func (p *IOProvider) Describe() []RouteDescription {
	actionNames := make([]any, 0, len(rfc15Actions))
	for _, action := range rfc15Actions {
		actionNames = append(actionNames, action.Name)
	}

	return []RouteDescription{
		{
			Method:      http.MethodPost,
			Path:        "/workspace",
			Summary:     "Create workspace",
			Description: "RFC §5 workspace creation route.",
			Tags:        []string{"io", "workspace"},
			StatusCode:  http.StatusOK,
			RequestBody: map[string]any{
				"type":     "object",
				"required": []string{"workspace"},
				"properties": map[string]any{
					"workspace":   map[string]any{"type": "string"},
					"name":        map[string]any{"type": "string"},
					"identifier":  map[string]any{"type": "string"},
					"workspaceID": map[string]any{"type": "string"},
				},
			},
			Response: map[string]any{"type": "object"},
		},
		{
			Method:      http.MethodPost,
			Path:        "/workspace/:id/switch",
			Summary:     "Switch workspace",
			Description: "RFC §5 workspace switch route.",
			Tags:        []string{"io", "workspace"},
			StatusCode:  http.StatusOK,
			Parameters: []ParameterDescription{
				{Name: "id", In: "path", Required: true, Schema: map[string]any{"type": "string"}},
			},
			Response: map[string]any{"type": "object"},
		},
		{
			Method:      http.MethodPost,
			Path:        "/workspace/:id/command",
			Summary:     "Handle workspace command",
			Description: "RFC §5 workspace command route.",
			Tags:        []string{"io", "workspace"},
			StatusCode:  http.StatusOK,
			Parameters: []ParameterDescription{
				{Name: "id", In: "path", Required: true, Schema: map[string]any{"type": "string"}},
			},
			RequestBody: map[string]any{
				"type":     "object",
				"required": []string{"action"},
				"properties": map[string]any{
					"action":      map[string]any{"type": "string"},
					"workspace":   map[string]any{"type": "string"},
					"name":        map[string]any{"type": "string"},
					"identifier":  map[string]any{"type": "string"},
					"workspaceID": map[string]any{"type": "string"},
					"path":        map[string]any{"type": "string"},
					"content":     map[string]any{"type": "string"},
				},
			},
			Response: map[string]any{"type": "object"},
		},
		{
			Method:      http.MethodPost,
			Path:        "/medium/:type/:op",
			Summary:     "Dispatch Medium operation",
			Description: "Dispatches HTTP requests to configured go-io Medium primitives.",
			Tags:        []string{"io", "medium"},
			StatusCode:  http.StatusOK,
			Parameters: []ParameterDescription{
				{Name: "type", In: "path", Required: true, Schema: map[string]any{"type": "string"}},
				{Name: "op", In: "path", Required: true, Schema: map[string]any{"type": "string"}},
			},
			RequestBody: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path":      map[string]any{"type": "string"},
					"oldPath":   map[string]any{"type": "string"},
					"newPath":   map[string]any{"type": "string"},
					"content":   map[string]any{"type": "string"},
					"mode":      map[string]any{"type": "integer"},
					"recursive": map[string]any{"type": "boolean"},
				},
			},
			Response: map[string]any{"type": "object"},
		},
		{
			Method:      http.MethodPost,
			Path:        "/io/:action",
			Summary:     "Dispatch RFC §15 IO action",
			Description: "Dispatches registered go-io RFC §15 actions.",
			Tags:        []string{"io", "actions"},
			StatusCode:  http.StatusOK,
			Parameters: []ParameterDescription{
				{
					Name:     "action",
					In:       "path",
					Required: true,
					Schema: map[string]any{
						"type": "string",
						"enum": actionNames,
					},
				},
			},
			RequestBody: map[string]any{"type": "object"},
			Response:    map[string]any{"type": "object"},
		},
	}
}

func configuredLocalMedium() coreio.Medium {
	root := core.Env("CORE_IO_LOCAL_ROOT")
	if root == "" {
		return nil
	}
	medium, err := coreio.NewSandboxed(root)
	if err != nil {
		return nil
	}
	return medium
}

func errorResponseSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"success": map[string]any{"type": "boolean"},
			"error": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"code":    map[string]any{"type": "string"},
					"message": map[string]any{"type": "string"},
				},
			},
		},
	}
}
