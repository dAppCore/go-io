// SPDX-License-Identifier: EUPL-1.2

// Example: s3.RegisterActions(c)
// Example: result := c.Action(s3.ActionRead).Run(ctx, core.NewOptions(
// Example:     core.Option{Key: "medium", Value: s3Medium},
// Example:     core.Option{Key: "path",   Value: "reports/daily.txt"},
// Example: ))
package s3

import (
	"context"
	"io/fs"

	core "dappco.re/go/core"
)

// Named action identifiers for the S3 Medium. Matches the go-io RFC §15
// registry so any Core-aware agent or CLI can dispatch S3 operations by name.
//
// Example: result := c.Action(s3.ActionRead).Run(ctx, opts)
const (
	ActionRead  = "core.io.s3.read"
	ActionWrite = "core.io.s3.write"
)

// Example: s3.RegisterActions(c)
//
// RegisterActions installs the S3 actions listed in the go-io RFC §15 on the
// given Core. Call this during service registration.
func RegisterActions(c *core.Core) {
	if c == nil {
		return
	}
	c.Action(ActionRead, readAction)
	c.Action(ActionWrite, writeAction)
}

// Example: opts := core.NewOptions(
// Example:     core.Option{Key: "medium", Value: s3Medium},
// Example:     core.Option{Key: "path",   Value: "reports/daily.txt"},
// Example: )
func readAction(_ context.Context, opts core.Options) core.Result {
	medium, ok := opts.Get("medium").Value.(*Medium)
	if !ok {
		return core.Result{}.New(core.E("s3.readAction", "medium is required", fs.ErrInvalid))
	}
	content, err := medium.Read(opts.String("path"))
	if err != nil {
		return core.Result{}.New(err)
	}
	return core.Result{Value: content, OK: true}
}

// Example: opts := core.NewOptions(
// Example:     core.Option{Key: "medium",  Value: s3Medium},
// Example:     core.Option{Key: "path",    Value: "reports/daily.txt"},
// Example:     core.Option{Key: "content", Value: "done"},
// Example: )
func writeAction(_ context.Context, opts core.Options) core.Result {
	medium, ok := opts.Get("medium").Value.(*Medium)
	if !ok {
		return core.Result{}.New(core.E("s3.writeAction", "medium is required", fs.ErrInvalid))
	}
	if err := medium.Write(opts.String("path"), opts.String("content")); err != nil {
		return core.Result{}.New(err)
	}
	return core.Result{OK: true}
}
