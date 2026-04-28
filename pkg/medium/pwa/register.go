package pwa

import (
	"context"

	core "dappco.re/go"
)

const (
	Scheme       = "pwa"
	ActionScrape = "core.io.pwa.scrape"
	ActionRead   = "core.io.pwa.read"
	ActionList   = "core.io.pwa.list"
	ActionWrite  = "core.io.pwa.write"
)

type Factory func(Options) (*Medium, error)

var Registry = core.NewRegistry[Factory]()

func init() {
	RegisterFactory(Scheme, New)
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
	c.Action(ActionScrape, scrapeAction)
	c.Action(ActionRead, readAction)
	c.Action(ActionList, listAction)
	c.Action(ActionWrite, writeAction)
}

func mediumFromOptions(opts core.Options) (*Medium, error) {
	if medium, ok := opts.Get("medium").Value.(*Medium); ok {
		return medium, nil
	}
	return New(Options{URL: opts.String("url")})
}

func readAction(_ context.Context, opts core.Options) core.Result {
	medium, err := mediumFromOptions(opts)
	if err != nil {
		return core.Fail(err)
	}
	content, err := medium.Read(opts.String("path"))
	if err != nil {
		return core.Fail(err)
	}
	return core.Ok(content)
}

func scrapeAction(_ context.Context, opts core.Options) core.Result {
	medium, err := mediumFromOptions(opts)
	if err != nil {
		return core.Fail(err)
	}
	if _, err := medium.ensureDataNode(); err != nil {
		return core.Fail(err)
	}
	return core.Ok(medium)
}

func listAction(_ context.Context, opts core.Options) core.Result {
	medium, err := mediumFromOptions(opts)
	if err != nil {
		return core.Fail(err)
	}
	entries, err := medium.List(opts.String("path"))
	if err != nil {
		return core.Fail(err)
	}
	return core.Ok(entries)
}

func writeAction(_ context.Context, opts core.Options) core.Result {
	medium, err := mediumFromOptions(opts)
	if err != nil {
		return core.Fail(err)
	}
	if err := medium.Write(opts.String("path"), opts.String("content")); err != nil {
		return core.Fail(err)
	}
	return core.Ok(nil)
}
