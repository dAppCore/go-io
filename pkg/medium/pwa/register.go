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

func mediumFromOptions(opts core.Options) *Medium {
	if medium, ok := opts.Get("medium").Value.(*Medium); ok {
		return medium
	}
	return &Medium{url: opts.String("url")}
}

func readAction(_ context.Context, opts core.Options) core.Result {
	content, err := mediumFromOptions(opts).Read(opts.String("path"))
	if err != nil {
		return core.Fail(err)
	}
	return core.Ok(content)
}

func scrapeAction(_ context.Context, opts core.Options) core.Result {
	target := opts.String("url")
	if target == "" {
		target = opts.String("path")
	}
	content, err := mediumFromOptions(opts).Read(target)
	if err != nil {
		return core.Fail(err)
	}
	return core.Ok(content)
}

func listAction(_ context.Context, opts core.Options) core.Result {
	entries, err := mediumFromOptions(opts).List(opts.String("path"))
	if err != nil {
		return core.Fail(err)
	}
	return core.Ok(entries)
}

func writeAction(_ context.Context, opts core.Options) core.Result {
	if err := mediumFromOptions(opts).Write(opts.String("path"), opts.String("content")); err != nil {
		return core.Fail(err)
	}
	return core.Ok(nil)
}
