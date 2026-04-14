// SPDX-License-Identifier: EUPL-1.2

// Example: cube.RegisterActions(c)
// Example: result := c.Action(cube.ActionPack).Run(ctx, core.NewOptions(
// Example:     core.Option{Key: "source", Value: sourceMedium},
// Example:     core.Option{Key: "output", Value: "app.cube"},
// Example:     core.Option{Key: "key", Value: key},
// Example: ))
package cube

import (
	"context"
	"io/fs"

	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"
)

// Named action identifiers for the Cube Medium. Matches the go-io RFC §15
// registry so any Core-aware agent or CLI can dispatch Cube operations by
// name.
//
// Example: result := c.Action(cube.ActionRead).Run(ctx, opts)
const (
	ActionRead   = "core.io.cube.read"
	ActionWrite  = "core.io.cube.write"
	ActionPack   = "core.io.cube.pack"
	ActionUnpack = "core.io.cube.unpack"
)

// Example: cube.RegisterActions(c)
//
// RegisterActions installs the cube actions listed in the go-io RFC §15 on the
// given Core. Call this during service registration.
func RegisterActions(c *core.Core) {
	if c == nil {
		return
	}
	c.Action(ActionRead, readAction)
	c.Action(ActionWrite, writeAction)
	c.Action(ActionPack, packAction)
	c.Action(ActionUnpack, unpackAction)
}

// Example: opts := core.NewOptions(
// Example:     core.Option{Key: "inner", Value: innerMedium},
// Example:     core.Option{Key: "key",   Value: key},
// Example:     core.Option{Key: "path",  Value: "secret.txt"},
// Example: )
func readAction(_ context.Context, opts core.Options) core.Result {
	inner, ok := opts.Get("inner").Value.(coreio.Medium)
	if !ok {
		return core.Result{}.New(core.E("cube.readAction", "inner medium is required", fs.ErrInvalid))
	}
	key, ok := opts.Get("key").Value.([]byte)
	if !ok {
		return core.Result{}.New(core.E("cube.readAction", "key must be []byte", fs.ErrInvalid))
	}
	medium, err := New(Options{Inner: inner, Key: key})
	if err != nil {
		return core.Result{}.New(err)
	}
	content, err := medium.Read(opts.String("path"))
	if err != nil {
		return core.Result{}.New(err)
	}
	return core.Result{Value: content, OK: true}
}

// Example: opts := core.NewOptions(
// Example:     core.Option{Key: "inner",   Value: innerMedium},
// Example:     core.Option{Key: "key",     Value: key},
// Example:     core.Option{Key: "path",    Value: "secret.txt"},
// Example:     core.Option{Key: "content", Value: "classified"},
// Example: )
func writeAction(_ context.Context, opts core.Options) core.Result {
	inner, ok := opts.Get("inner").Value.(coreio.Medium)
	if !ok {
		return core.Result{}.New(core.E("cube.writeAction", "inner medium is required", fs.ErrInvalid))
	}
	key, ok := opts.Get("key").Value.([]byte)
	if !ok {
		return core.Result{}.New(core.E("cube.writeAction", "key must be []byte", fs.ErrInvalid))
	}
	medium, err := New(Options{Inner: inner, Key: key})
	if err != nil {
		return core.Result{}.New(err)
	}
	if err := medium.Write(opts.String("path"), opts.String("content")); err != nil {
		return core.Result{}.New(err)
	}
	return core.Result{OK: true}
}

// Example: opts := core.NewOptions(
// Example:     core.Option{Key: "source", Value: sourceMedium},
// Example:     core.Option{Key: "output", Value: "app.cube"},
// Example:     core.Option{Key: "key",    Value: key},
// Example: )
func packAction(_ context.Context, opts core.Options) core.Result {
	source, ok := opts.Get("source").Value.(coreio.Medium)
	if !ok {
		return core.Result{}.New(core.E("cube.packAction", "source medium is required", fs.ErrInvalid))
	}
	key, ok := opts.Get("key").Value.([]byte)
	if !ok {
		return core.Result{}.New(core.E("cube.packAction", "key must be []byte", fs.ErrInvalid))
	}
	output := opts.String("output")
	if err := Pack(output, source, key); err != nil {
		return core.Result{}.New(err)
	}
	return core.Result{OK: true}
}

// Example: opts := core.NewOptions(
// Example:     core.Option{Key: "cube",        Value: "app.cube"},
// Example:     core.Option{Key: "destination", Value: destinationMedium},
// Example:     core.Option{Key: "key",         Value: key},
// Example: )
func unpackAction(_ context.Context, opts core.Options) core.Result {
	destination, ok := opts.Get("destination").Value.(coreio.Medium)
	if !ok {
		return core.Result{}.New(core.E("cube.unpackAction", "destination medium is required", fs.ErrInvalid))
	}
	key, ok := opts.Get("key").Value.([]byte)
	if !ok {
		return core.Result{}.New(core.E("cube.unpackAction", "key must be []byte", fs.ErrInvalid))
	}
	cubePath := opts.String("cube")
	if err := Unpack(cubePath, destination, key); err != nil {
		return core.Result{}.New(err)
	}
	return core.Result{OK: true}
}
