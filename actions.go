// SPDX-License-Identifier: EUPL-1.2

// Example: io.RegisterActions(c)
// Example: result := c.Action("core.io.local.read").Run(ctx, core.NewOptions(
// Example:     core.Option{Key: "root", Value: "/srv/app"},
// Example:     core.Option{Key: "path", Value: "config/app.yaml"},
// Example: ))
package io

import (
	"context"
	"io/fs"

	core "dappco.re/go"
)

// Named action identifiers used by Core consumers. Each maps to a Medium
// operation with a predictable path name.
//
// Example: result := c.Action(io.ActionLocalRead).Run(ctx, opts)
const (
	ActionLocalRead   = "core.io.local.read"
	ActionLocalWrite  = "core.io.local.write"
	ActionLocalList   = "core.io.local.list"
	ActionLocalDelete = "core.io.local.delete"

	ActionMemoryRead  = "core.io.memory.read"
	ActionMemoryWrite = "core.io.memory.write"

	ActionGitHubClone = "core.io.github.clone"
	ActionGitHubRead  = "core.io.github.read"

	ActionPWAScrape = "core.io.pwa.scrape"

	ActionSFTPRead  = "core.io.sftp.read"
	ActionSFTPWrite = "core.io.sftp.write"

	ActionS3Read  = "core.io.s3.read"
	ActionS3Write = "core.io.s3.write"

	ActionCubeRead   = "core.io.cube.read"
	ActionCubeWrite  = "core.io.cube.write"
	ActionCubePack   = "core.io.cube.pack"
	ActionCubeUnpack = "core.io.cube.unpack"

	ActionCopy = "core.io.copy"
)

// memoryActionStore is the shared in-memory backing for
// core.io.memory.read/core.io.memory.write. Keeping it package-level lets the
// two actions agree on state without the caller supplying a backend.
var memoryActionStore = NewMemoryMedium()

// Example: io.RegisterActions(c)
//
// RegisterActions installs the named actions listed in the go-io RFC §15 on
// the given Core. Consumers call this at service registration time so that any
// agent or CLI can dispatch Medium operations by name.
func RegisterActions(c *core.Core) {
	if c == nil {
		return
	}
	c.Action(ActionLocalRead, localReadAction)
	c.Action(ActionLocalWrite, localWriteAction)
	c.Action(ActionLocalList, localListAction)
	c.Action(ActionLocalDelete, localDeleteAction)
	c.Action(ActionMemoryRead, memoryReadAction)
	c.Action(ActionMemoryWrite, memoryWriteAction)
	c.Action(ActionGitHubClone, githubNotImplementedAction)
	c.Action(ActionGitHubRead, githubNotImplementedAction)
	c.Action(ActionPWAScrape, pwaNotImplementedAction)
	c.Action(ActionSFTPRead, mediumReadAction("io.sftp.readAction"))
	c.Action(ActionSFTPWrite, mediumWriteAction("io.sftp.writeAction"))
	c.Action(ActionS3Read, mediumReadAction("io.s3.readAction"))
	c.Action(ActionS3Write, mediumWriteAction("io.s3.writeAction"))
	c.Action(ActionCopy, copyAction)
}

// Example: opts := core.NewOptions(core.Option{Key: "root", Value: "/srv/app"}, core.Option{Key: "path", Value: "config/app.yaml"})
func localReadAction(_ context.Context, opts core.Options) core.Result {
	medium, err := localMediumFromOptions(opts)
	if err != nil {
		return core.Fail(err)
	}
	content, err := medium.Read(opts.String("path"))
	if err != nil {
		return core.Fail(err)
	}
	return core.Ok(content)
}

// Example: opts := core.NewOptions(core.Option{Key: "root", Value: "/srv/app"}, core.Option{Key: "path", Value: "log.txt"}, core.Option{Key: "content", Value: "event"})
func localWriteAction(_ context.Context, opts core.Options) core.Result {
	medium, err := localMediumFromOptions(opts)
	if err != nil {
		return core.Fail(err)
	}
	if err := medium.Write(opts.String("path"), opts.String("content")); err != nil {
		return core.Fail(err)
	}
	return core.Ok(nil)
}

// Example: opts := core.NewOptions(core.Option{Key: "root", Value: "/srv/app"}, core.Option{Key: "path", Value: "config"})
func localListAction(_ context.Context, opts core.Options) core.Result {
	medium, err := localMediumFromOptions(opts)
	if err != nil {
		return core.Fail(err)
	}
	entries, err := medium.List(opts.String("path"))
	if err != nil {
		return core.Fail(err)
	}
	return core.Ok(entries)
}

// Example: opts := core.NewOptions(core.Option{Key: "root", Value: "/srv/app"}, core.Option{Key: "path", Value: "tmp/old.log"})
func localDeleteAction(_ context.Context, opts core.Options) core.Result {
	medium, err := localMediumFromOptions(opts)
	if err != nil {
		return core.Fail(err)
	}
	path := opts.String("path")
	recursive := opts.Bool("recursive")
	if recursive {
		if err := medium.DeleteAll(path); err != nil {
			return core.Fail(err)
		}
	} else {
		if err := medium.Delete(path); err != nil {
			return core.Fail(err)
		}
	}
	return core.Ok(nil)
}

// Example: opts := core.NewOptions(core.Option{Key: "path", Value: "config/app.yaml"})
func memoryReadAction(_ context.Context, opts core.Options) core.Result {
	content, err := memoryActionStore.Read(opts.String("path"))
	if err != nil {
		return core.Fail(err)
	}
	return core.Ok(content)
}

// Example: opts := core.NewOptions(core.Option{Key: "path", Value: "config/app.yaml"}, core.Option{Key: "content", Value: "port: 8080"})
func memoryWriteAction(_ context.Context, opts core.Options) core.Result {
	if err := memoryActionStore.Write(opts.String("path"), opts.String("content")); err != nil {
		return core.Fail(err)
	}
	return core.Ok(nil)
}

func githubNotImplementedAction(context.Context, core.Options) core.Result {
	return core.Fail(core.E("io.github", "not implemented — see #633 for backend tracking", nil))
}

func pwaNotImplementedAction(context.Context, core.Options) core.Result {
	return core.Fail(core.E("io.pwa", "not implemented — see #633 for backend tracking", nil))
}

func mediumReadAction(operation string) core.ActionHandler {
	return func(_ context.Context, opts core.Options) core.Result {
		medium, err := mediumFromOptions(opts, operation)
		if err != nil {
			return core.Fail(err)
		}
		content, err := medium.Read(opts.String("path"))
		if err != nil {
			return core.Fail(err)
		}
		return core.Ok(content)
	}
}

func mediumWriteAction(operation string) core.ActionHandler {
	return func(_ context.Context, opts core.Options) core.Result {
		medium, err := mediumFromOptions(opts, operation)
		if err != nil {
			return core.Fail(err)
		}
		if err := medium.Write(opts.String("path"), opts.String("content")); err != nil {
			return core.Fail(err)
		}
		return core.Ok(nil)
	}
}

// Example: opts := core.NewOptions(
// Example:     core.Option{Key: "source", Value: sourceMedium},
// Example:     core.Option{Key: "sourcePath", Value: "input.txt"},
// Example:     core.Option{Key: "destination", Value: destinationMedium},
// Example:     core.Option{Key: "destinationPath", Value: "backup/input.txt"},
// Example: )
func copyAction(_ context.Context, opts core.Options) core.Result {
	source, ok := opts.Get("source").Value.(Medium)
	if !ok {
		return core.Fail(core.E("io.copyAction", "source medium is required", fs.ErrInvalid))
	}
	destination, ok := opts.Get("destination").Value.(Medium)
	if !ok {
		return core.Fail(core.E("io.copyAction", "destination medium is required", fs.ErrInvalid))
	}
	if err := Copy(source, opts.String("sourcePath"), destination, opts.String("destinationPath")); err != nil {
		return core.Fail(err)
	}
	return core.Ok(nil)
}

// localMediumFromOptions constructs a sandboxed local Medium using the
// "root" option.
func localMediumFromOptions(opts core.Options) (Medium, error) {
	root := opts.String("root")
	if root == "" {
		return nil, core.E("io.localMediumFromOptions", "root is required", fs.ErrInvalid)
	}
	return NewSandboxed(root)
}

func mediumFromOptions(opts core.Options, operation string) (Medium, error) {
	medium, ok := opts.Get("medium").Value.(Medium)
	if !ok {
		return nil, core.E(operation, "medium is required", fs.ErrInvalid)
	}
	return medium, nil
}

// ResetMemoryActionStore clears the in-memory state used by memory action
// handlers. Tests call this to isolate runs from each other.
//
// Example: io.ResetMemoryActionStore()
func ResetMemoryActionStore() {
	memoryActionStore = NewMemoryMedium()
}
