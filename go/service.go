// SPDX-License-Identifier: EUPL-1.2

// Service registration for the go-io package. Exposes a Medium-backed
// service whose actions wrap the canonical filesystem ops so consumers
// can call them through Core's Action plumbing.
//
// Example:
//
//	c, _ := core.New(
//	    core.WithName("io", io.NewService(io.IOConfig{})),
//	)
//	r := c.Action("io.read").Run(ctx, core.NewOptions(
//	    core.Option{Key: "path", Value: "config/app.yaml"},
//	))

package io

import (
	"context"
	"io/fs" // AX-6-exception: fs.FileMode is the structural type the Medium interface returns; no core equivalent.

	core "dappco.re/go"
)

// IOConfig configures the service-backed Medium. Empty/zero config
// means "use the unsandboxed package-level Local medium". Setting Root
// switches to a sandboxed local Medium rooted at that directory.
//
// Example:
//
//	cfg := io.IOConfig{Root: "/srv/app"}
type IOConfig struct {
	// Example: cfg := io.IOConfig{Root: "/srv/app"}
	Root string
}

// Service is the registerable handle for the io package — embeds
// *core.ServiceRuntime[IOConfig] for typed options access and holds
// the active Medium for direct method use or action dispatch.
//
// Example:
//
//	svc := core.MustServiceFor[*io.Service](c, "io")
//	content, err := svc.Medium.Read("config/app.yaml")
type Service struct {
	*core.ServiceRuntime[IOConfig]
	// Medium is the live filesystem-abstraction the service was constructed with.
	// Example: content, _ := svc.Medium.Read("config/app.yaml")
	Medium        Medium
	registrations core.Once
}

// NewService returns a factory that resolves a Medium (sandboxed when
// IOConfig.Root is non-empty, otherwise the package Local) and produces
// a *Service ready for c.Service() registration.
//
// Example:
//
//	c, _ := core.New(
//	    core.WithName("io", io.NewService(io.IOConfig{Root: "/srv/app"})),
//	)
func NewService(config IOConfig) func(*core.Core) core.Result {
	return func(c *core.Core) core.Result {
		medium := Local
		if config.Root != "" {
			sandboxed, err := NewSandboxed(config.Root)
			if err != nil {
				return core.Fail(core.E("io.NewService", "create sandboxed medium", err))
			}
			medium = sandboxed
		}
		return core.Ok(&Service{
			ServiceRuntime: core.NewServiceRuntime(c, config),
			Medium:         medium,
		})
	}
}

// OnStartup registers the io action handlers on the attached Core.
// Implements core.Startable. Idempotent via core.Once — multiple
// startups will not double-register.
//
// Example:
//
//	r := svc.OnStartup(ctx)
func (s *Service) OnStartup(context.Context) core.Result {
	if s == nil {
		return core.Ok(nil)
	}
	s.registrations.Do(func() {
		c := s.Core()
		if c == nil {
			return
		}
		c.Action("io.read", s.handleRead)
		c.Action("io.write", s.handleWrite)
		c.Action("io.delete", s.handleDelete)
		c.Action("io.delete_all", s.handleDeleteAll)
		c.Action("io.rename", s.handleRename)
		c.Action("io.exists", s.handleExists)
		c.Action("io.is_file", s.handleIsFile)
		c.Action("io.is_dir", s.handleIsDir)
		c.Action("io.ensure_dir", s.handleEnsureDir)
		c.Action("io.list", s.handleList)
	})
	return core.Ok(nil)
}

// OnShutdown is a no-op — Local and sandboxed mediums hold no
// closable resources. Implements core.Stoppable.
//
// Example:
//
//	r := svc.OnShutdown(ctx)
func (s *Service) OnShutdown(context.Context) core.Result {
	return core.Ok(nil)
}

// handleRead — `io.read` action handler. Reads opts.path and returns
// the file content as a string in r.Value.
//
// Example:
//
//	r := c.Action("io.read").Run(ctx, core.NewOptions(
//	    core.Option{Key: "path", Value: "config/app.yaml"},
//	))
//	content, _ := r.Value.(string)
func (s *Service) handleRead(_ core.Context, opts core.Options) core.Result {
	if s == nil || s.Medium == nil {
		return core.Fail(core.E("io.read", "service not initialised", nil))
	}
	content, err := s.Medium.Read(opts.String("path"))
	if err != nil {
		return core.Fail(core.E("io.read", opts.String("path"), err))
	}
	return core.Ok(content)
}

// handleWrite — `io.write` action handler. Writes opts.content to
// opts.path. Optional opts.mode (octal int) switches to mode-aware
// write.
//
// Example:
//
//	r := c.Action("io.write").Run(ctx, core.NewOptions(
//	    core.Option{Key: "path", Value: "config/app.yaml"},
//	    core.Option{Key: "content", Value: "port: 8080"},
//	))
func (s *Service) handleWrite(_ core.Context, opts core.Options) core.Result {
	if s == nil || s.Medium == nil {
		return core.Fail(core.E("io.write", "service not initialised", nil))
	}
	path := opts.String("path")
	content := opts.String("content")
	if mode := opts.Int("mode"); mode != 0 {
		if err := s.Medium.WriteMode(path, content, fs.FileMode(mode)); err != nil {
			return core.Fail(core.E("io.write", path, err))
		}
		return core.Ok(nil)
	}
	if err := s.Medium.Write(path, content); err != nil {
		return core.Fail(core.E("io.write", path, err))
	}
	return core.Ok(nil)
}

// handleDelete — `io.delete` action handler. Removes the file at
// opts.path; succeeds idempotently when the path already absent (per
// underlying Medium semantics).
//
// Example:
//
//	r := c.Action("io.delete").Run(ctx, core.NewOptions(
//	    core.Option{Key: "path", Value: "config/old.yaml"},
//	))
func (s *Service) handleDelete(_ core.Context, opts core.Options) core.Result {
	if s == nil || s.Medium == nil {
		return core.Fail(core.E("io.delete", "service not initialised", nil))
	}
	path := opts.String("path")
	if err := s.Medium.Delete(path); err != nil {
		return core.Fail(core.E("io.delete", path, err))
	}
	return core.Ok(nil)
}

// handleDeleteAll — `io.delete_all` action handler. Recursively
// removes opts.path and any descendants.
//
// Example:
//
//	r := c.Action("io.delete_all").Run(ctx, core.NewOptions(
//	    core.Option{Key: "path", Value: "logs/archive"},
//	))
func (s *Service) handleDeleteAll(_ core.Context, opts core.Options) core.Result {
	if s == nil || s.Medium == nil {
		return core.Fail(core.E("io.delete_all", "service not initialised", nil))
	}
	path := opts.String("path")
	if err := s.Medium.DeleteAll(path); err != nil {
		return core.Fail(core.E("io.delete_all", path, err))
	}
	return core.Ok(nil)
}

// handleRename — `io.rename` action handler. Renames opts.old to
// opts.new.
//
// Example:
//
//	r := c.Action("io.rename").Run(ctx, core.NewOptions(
//	    core.Option{Key: "old", Value: "drafts/todo.txt"},
//	    core.Option{Key: "new", Value: "archive/todo.txt"},
//	))
func (s *Service) handleRename(_ core.Context, opts core.Options) core.Result {
	if s == nil || s.Medium == nil {
		return core.Fail(core.E("io.rename", "service not initialised", nil))
	}
	if err := s.Medium.Rename(opts.String("old"), opts.String("new")); err != nil {
		return core.Fail(core.E("io.rename", opts.String("old"), err))
	}
	return core.Ok(nil)
}

// handleExists — `io.exists` action handler. Returns bool in r.Value.
//
// Example:
//
//	r := c.Action("io.exists").Run(ctx, core.NewOptions(
//	    core.Option{Key: "path", Value: "config/app.yaml"},
//	))
//	exists, _ := r.Value.(bool)
func (s *Service) handleExists(_ core.Context, opts core.Options) core.Result {
	if s == nil || s.Medium == nil {
		return core.Fail(core.E("io.exists", "service not initialised", nil))
	}
	return core.Ok(s.Medium.Exists(opts.String("path")))
}

// handleIsFile — `io.is_file` action handler. Returns bool in r.Value.
//
// Example:
//
//	r := c.Action("io.is_file").Run(ctx, core.NewOptions(
//	    core.Option{Key: "path", Value: "config/app.yaml"},
//	))
func (s *Service) handleIsFile(_ core.Context, opts core.Options) core.Result {
	if s == nil || s.Medium == nil {
		return core.Fail(core.E("io.is_file", "service not initialised", nil))
	}
	return core.Ok(s.Medium.IsFile(opts.String("path")))
}

// handleIsDir — `io.is_dir` action handler. Returns bool in r.Value.
//
// Example:
//
//	r := c.Action("io.is_dir").Run(ctx, core.NewOptions(
//	    core.Option{Key: "path", Value: "config"},
//	))
func (s *Service) handleIsDir(_ core.Context, opts core.Options) core.Result {
	if s == nil || s.Medium == nil {
		return core.Fail(core.E("io.is_dir", "service not initialised", nil))
	}
	return core.Ok(s.Medium.IsDir(opts.String("path")))
}

// handleEnsureDir — `io.ensure_dir` action handler. Creates opts.path
// recursively if it does not exist.
//
// Example:
//
//	r := c.Action("io.ensure_dir").Run(ctx, core.NewOptions(
//	    core.Option{Key: "path", Value: "config/app"},
//	))
func (s *Service) handleEnsureDir(_ core.Context, opts core.Options) core.Result {
	if s == nil || s.Medium == nil {
		return core.Fail(core.E("io.ensure_dir", "service not initialised", nil))
	}
	if err := s.Medium.EnsureDir(opts.String("path")); err != nil {
		return core.Fail(core.E("io.ensure_dir", opts.String("path"), err))
	}
	return core.Ok(nil)
}

// handleList — `io.list` action handler. Returns directory entry
// names as []string in r.Value (the underlying fs.DirEntry slice is
// projected to names for IPC-friendly transport).
//
// Example:
//
//	r := c.Action("io.list").Run(ctx, core.NewOptions(
//	    core.Option{Key: "path", Value: "config"},
//	))
//	names, _ := r.Value.([]string)
func (s *Service) handleList(_ core.Context, opts core.Options) core.Result {
	if s == nil || s.Medium == nil {
		return core.Fail(core.E("io.list", "service not initialised", nil))
	}
	path := opts.String("path")
	entries, err := s.Medium.List(path)
	if err != nil {
		return core.Fail(core.E("io.list", path, err))
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return core.Ok(names)
}

