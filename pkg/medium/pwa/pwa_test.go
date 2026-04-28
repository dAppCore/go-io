package pwa

import (
	"context"
	"errors"

	core "dappco.re/go"
)

func TestPWAMedium_StubOperations_ReturnErrNotImplemented(t *core.T) {
	medium, err := New(Options{})
	core.RequireNoError(t, err)

	checks := []struct {
		name string
		run  func() error
	}{
		{name: "Read", run: func() error { _, err := medium.Read("page"); return err }},
		{name: "Write", run: func() error { return medium.Write("page", "content") }},
		{name: "WriteMode", run: func() error { return medium.WriteMode("page", "content", 0644) }},
		{name: "EnsureDir", run: func() error { return medium.EnsureDir("page") }},
		{name: "Delete", run: func() error { return medium.Delete("page") }},
		{name: "DeleteAll", run: func() error { return medium.DeleteAll("page") }},
		{name: "Rename", run: func() error { return medium.Rename("old", "new") }},
		{name: "List", run: func() error { _, err := medium.List("page"); return err }},
		{name: "Stat", run: func() error { _, err := medium.Stat("page"); return err }},
		{name: "Open", run: func() error { _, err := medium.Open("page"); return err }},
		{name: "Create", run: func() error { _, err := medium.Create("page"); return err }},
		{name: "Append", run: func() error { _, err := medium.Append("page"); return err }},
		{name: "ReadStream", run: func() error { _, err := medium.ReadStream("page"); return err }},
		{name: "WriteStream", run: func() error { _, err := medium.WriteStream("page"); return err }},
	}

	for _, check := range checks {
		t.Run(check.name, func(t *core.T) {
			core.AssertTrue(t, errors.Is(check.run(), ErrNotImplemented))
		})
	}

	core.AssertFalse(t, medium.IsFile("page"))
	core.AssertFalse(t, medium.Exists("page"))
	core.AssertFalse(t, medium.IsDir("page"))
}

func TestPWAMedium_Actions_ReturnErrNotImplemented(t *core.T) {
	_, ok := FactoryFor(Scheme)
	core.RequireTrue(t, ok)

	c := core.New()
	RegisterActions(c)

	for _, action := range []string{ActionScrape, ActionRead, ActionList, ActionWrite} {
		core.RequireTrue(t, c.Action(action).Exists())
		result := c.Action(action).Run(context.Background(), core.NewOptions(
			core.Option{Key: "url", Value: "https://example.test"},
			core.Option{Key: "path", Value: "page"},
		))
		core.AssertFalse(t, result.OK)
		core.AssertTrue(t, errors.Is(result.Value.(error), ErrNotImplemented))
	}
}
