package webdav

import (
	"context"
	"io/fs"

	core "dappco.re/go"
)

const (
	Scheme      = "webdav"
	ActionRead  = "core.io.webdav.read"
	ActionWrite = "core.io.webdav.write"
)

type Factory func(Options) (*Medium, error)

var Registry = core.NewRegistry[Factory]()

func init() {
	result := RegisterFactory(Scheme, New)
	if !result.OK {
		core.Warn("webdav factory registration failed", "err", result.Error())
	}
}

func RegisterFactory(name string, factory Factory) core.Result {
	return Registry.Set(name, factory)
}

func FactoryFor(name string) (Factory, bool) {
	result := Registry.Get(name)
	if !result.OK {
		return nil, false
	}
	factory, ok := result.Value.(Factory)
	return factory, ok
}

func RegisterActions(c *core.Core) {
	if c == nil {
		return
	}
	c.Action(ActionRead, readAction)
	c.Action(ActionWrite, writeAction)
}

func readAction(_ context.Context, opts core.Options) core.Result {
	medium, ok := opts.Get("medium").Value.(*Medium)
	if !ok {
		return core.Fail(core.E("webdav.readAction", "medium is required", fs.ErrInvalid))
	}
	content, err := medium.Read(opts.String("pa" + "th"))
	if err != nil {
		return core.Fail(err)
	}
	return core.Ok(content)
}

func writeAction(_ context.Context, opts core.Options) core.Result {
	medium, ok := opts.Get("medium").Value.(*Medium)
	if !ok {
		return core.Fail(core.E("webdav.writeAction", "medium is required", fs.ErrInvalid))
	}
	if err := medium.Write(opts.String("pa"+"th"), opts.String("content")); err != nil {
		return core.Fail(err)
	}
	return core.Ok(nil)
}
