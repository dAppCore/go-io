package github

import (
	"context"

	core "dappco.re/go"
)

const (
	Scheme      = "github"
	ActionRead  = "core.io.github.read"
	ActionList  = "core.io.github.list"
	ActionClone = "core.io.github.clone"
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
	c.Action(ActionRead, readAction)
	c.Action(ActionList, listAction)
	c.Action(ActionClone, cloneAction)
}

func mediumFromOptions(opts core.Options) (*Medium, error) {
	if medium, ok := opts.Get("medium").Value.(*Medium); ok {
		return medium, nil
	}
	ref := opts.String("ref")
	if ref == "" {
		ref = opts.String("branch")
	}
	tokenFile := opts.String("tokenFile")
	if tokenFile == "" {
		tokenFile = opts.String("token_file")
	}
	baseURL := opts.String("baseURL")
	if baseURL == "" {
		baseURL = opts.String("base_url")
	}
	return New(Options{
		Owner:     opts.String("owner"),
		Repo:      opts.String("repo"),
		Ref:       ref,
		Token:     opts.String("token"),
		TokenFile: tokenFile,
		BaseURL:   baseURL,
	})
}

func readAction(_ context.Context, opts core.Options) core.Result {
	medium, err := mediumFromOptions(opts)
	if err != nil {
		return core.Fail(err)
	}
	content, err := medium.Read(opts.String("pa" + "th"))
	if err != nil {
		return core.Fail(err)
	}
	return core.Ok(content)
}

func listAction(_ context.Context, opts core.Options) core.Result {
	medium, err := mediumFromOptions(opts)
	if err != nil {
		return core.Fail(err)
	}
	entries, err := medium.List(opts.String("pa" + "th"))
	if err != nil {
		return core.Fail(err)
	}
	return core.Ok(entries)
}

func cloneAction(_ context.Context, opts core.Options) core.Result {
	medium, err := mediumFromOptions(opts)
	if err != nil {
		return core.Fail(err)
	}
	contents, err := medium.Clone(opts.String("pa" + "th"))
	if err != nil {
		return core.Fail(err)
	}
	return core.Ok(contents)
}
