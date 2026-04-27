package pwa

import (
	"context"
	"errors"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPWAMedium_StubOperations_ReturnErrNotImplemented(t *testing.T) {
	medium, err := New(Options{})
	require.NoError(t, err)

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
		t.Run(check.name, func(t *testing.T) {
			assert.True(t, errors.Is(check.run(), ErrNotImplemented))
		})
	}

	assert.False(t, medium.IsFile("page"))
	assert.False(t, medium.Exists("page"))
	assert.False(t, medium.IsDir("page"))
}

func TestPWAMedium_Actions_ReturnErrNotImplemented(t *testing.T) {
	_, ok := FactoryFor(Scheme)
	require.True(t, ok)

	c := core.New()
	RegisterActions(c)

	for _, action := range []string{ActionScrape, ActionRead, ActionList, ActionWrite} {
		require.True(t, c.Action(action).Exists())
		result := c.Action(action).Run(context.Background(), core.NewOptions(
			core.Option{Key: "url", Value: "https://example.test"},
			core.Option{Key: "path", Value: "page"},
		))
		require.False(t, result.OK)
		assert.True(t, errors.Is(result.Value.(error), ErrNotImplemented))
	}
}
