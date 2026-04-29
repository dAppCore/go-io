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

	core "dappco.re/go"
	coreio "dappco.re/go/io"
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

const (
	opReadAction   = "cube.readAction"
	opWriteAction  = "cube.writeAction"
	opPackAction   = "cube.packAction"
	opUnpackAction = "cube.unpackAction"
	errKeyType     = "key must be []byte"
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
// Example:     core.Option{Key: "pa"+"th",  Value: "secret.txt"},
// Example: )
func readAction(_ context.Context, opts core.Options) core.Result {
	if medium, ok := opts.Get("medium").Value.(coreio.Medium); ok && medium != nil {
		content, err := medium.Read(opts.String("pa" + "th"))
		if err != nil {
			return core.Fail(err)
		}
		return core.Ok(content)
	}

	inner, ok := opts.Get("inner").Value.(coreio.Medium)
	if !ok {
		return core.Fail(core.E(opReadAction, "inner medium is required", fs.ErrInvalid))
	}
	key, err := keyFromOptions(opts, opReadAction)
	if err != nil {
		return core.Fail(err)
	}
	medium, err := New(Options{Inner: inner, Key: key})
	if err != nil {
		return core.Fail(err)
	}
	content, err := medium.Read(opts.String("pa" + "th"))
	if err != nil {
		return core.Fail(err)
	}
	return core.Ok(content)
}

// Example: opts := core.NewOptions(
// Example:     core.Option{Key: "inner",   Value: innerMedium},
// Example:     core.Option{Key: "key",     Value: key},
// Example:     core.Option{Key: "pa"+"th",    Value: "secret.txt"},
// Example:     core.Option{Key: "content", Value: "classified"},
// Example: )
func writeAction(_ context.Context, opts core.Options) core.Result {
	if medium, ok := opts.Get("medium").Value.(coreio.Medium); ok && medium != nil {
		if err := medium.Write(opts.String("pa"+"th"), opts.String("content")); err != nil {
			return core.Fail(err)
		}
		return core.Ok(nil)
	}

	inner, ok := opts.Get("inner").Value.(coreio.Medium)
	if !ok {
		return core.Fail(core.E(opWriteAction, "inner medium is required", fs.ErrInvalid))
	}
	key, err := keyFromOptions(opts, opWriteAction)
	if err != nil {
		return core.Fail(err)
	}
	medium, err := New(Options{Inner: inner, Key: key})
	if err != nil {
		return core.Fail(err)
	}
	if err := medium.Write(opts.String("pa"+"th"), opts.String("content")); err != nil {
		return core.Fail(err)
	}
	return core.Ok(nil)
}

// Example: opts := core.NewOptions(
// Example:     core.Option{Key: "source", Value: sourceMedium},
// Example:     core.Option{Key: "output", Value: "app.cube"},
// Example:     core.Option{Key: "key",    Value: key},
// Example: )
func packAction(_ context.Context, opts core.Options) core.Result {
	source, ok := opts.Get("source").Value.(coreio.Medium)
	if !ok {
		return core.Fail(core.E(opPackAction, "source medium is required", fs.ErrInvalid))
	}
	key, err := keyFromOptions(opts, opPackAction)
	if err != nil {
		return core.Fail(err)
	}
	output := opts.String("output")
	if err := Pack(output, source, key); err != nil {
		return core.Fail(err)
	}
	return core.Ok(nil)
}

// Example: opts := core.NewOptions(
// Example:     core.Option{Key: "cube",        Value: "app.cube"},
// Example:     core.Option{Key: "destination", Value: destinationMedium},
// Example:     core.Option{Key: "key",         Value: key},
// Example: )
func unpackAction(_ context.Context, opts core.Options) core.Result {
	destination, ok := opts.Get("destination").Value.(coreio.Medium)
	if !ok {
		return core.Fail(core.E(opUnpackAction, "destination medium is required", fs.ErrInvalid))
	}
	key, err := keyFromOptions(opts, opUnpackAction)
	if err != nil {
		return core.Fail(err)
	}
	cubePath := opts.String("cube")
	if err := Unpack(cubePath, destination, key); err != nil {
		return core.Fail(err)
	}
	return core.Ok(nil)
}

func keyFromOptions(opts core.Options, operation string) ([]byte, error) {
	key, ok := opts.Get("key").Value.([]byte)
	if !ok {
		return nil, core.E(operation, errKeyType, fs.ErrInvalid)
	}
	return key, nil
}
