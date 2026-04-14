// SPDX-License-Identifier: EUPL-1.2

package io

import (
	"context"
	"io/fs"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActions_RegisterActions_Good(t *testing.T) {
	c := core.New()
	RegisterActions(c)

	for _, name := range []string{
		ActionLocalRead, ActionLocalWrite, ActionLocalList, ActionLocalDelete,
		ActionMemoryRead, ActionMemoryWrite, ActionCopy,
	} {
		assert.True(t, c.Action(name).Exists(), name)
	}
}

func TestActions_RegisterActions_Bad(t *testing.T) {
	// Nil Core must not panic and must be a no-op.
	assert.NotPanics(t, func() { RegisterActions(nil) })
}

func TestActions_RegisterActions_Ugly(t *testing.T) {
	// Calling RegisterActions twice on the same Core is safe (idempotent overwrite).
	c := core.New()
	RegisterActions(c)
	assert.NotPanics(t, func() { RegisterActions(c) })
	assert.True(t, c.Action(ActionMemoryRead).Exists())
}

func TestActions_LocalRead_Good(t *testing.T) {
	tempDir := t.TempDir()
	c := core.New()
	RegisterActions(c)

	// Prime a file via the write action, then read it back via the read action.
	writeResult := c.Action(ActionLocalWrite).Run(context.Background(), core.NewOptions(
		core.Option{Key: "root", Value: tempDir},
		core.Option{Key: "path", Value: "hello.txt"},
		core.Option{Key: "content", Value: "world"},
	))
	require.True(t, writeResult.OK)

	readResult := c.Action(ActionLocalRead).Run(context.Background(), core.NewOptions(
		core.Option{Key: "root", Value: tempDir},
		core.Option{Key: "path", Value: "hello.txt"},
	))
	require.True(t, readResult.OK)
	assert.Equal(t, "world", readResult.Value)
}

func TestActions_LocalRead_Bad(t *testing.T) {
	tempDir := t.TempDir()
	c := core.New()
	RegisterActions(c)

	// Reading a missing file returns !OK and an error in Value.
	result := c.Action(ActionLocalRead).Run(context.Background(), core.NewOptions(
		core.Option{Key: "root", Value: tempDir},
		core.Option{Key: "path", Value: "missing.txt"},
	))
	assert.False(t, result.OK)
}

func TestActions_LocalRead_Ugly(t *testing.T) {
	c := core.New()
	RegisterActions(c)

	// Empty path — read attempts to read the sandbox root which is not a file.
	result := c.Action(ActionLocalRead).Run(context.Background(), core.NewOptions(
		core.Option{Key: "root", Value: t.TempDir()},
		core.Option{Key: "path", Value: ""},
	))
	assert.False(t, result.OK)
}

func TestActions_LocalList_Good(t *testing.T) {
	tempDir := t.TempDir()
	c := core.New()
	RegisterActions(c)

	require.True(t, c.Action(ActionLocalWrite).Run(context.Background(), core.NewOptions(
		core.Option{Key: "root", Value: tempDir},
		core.Option{Key: "path", Value: "a.txt"},
		core.Option{Key: "content", Value: "alpha"},
	)).OK)
	require.True(t, c.Action(ActionLocalWrite).Run(context.Background(), core.NewOptions(
		core.Option{Key: "root", Value: tempDir},
		core.Option{Key: "path", Value: "b.txt"},
		core.Option{Key: "content", Value: "beta"},
	)).OK)

	listResult := c.Action(ActionLocalList).Run(context.Background(), core.NewOptions(
		core.Option{Key: "root", Value: tempDir},
		core.Option{Key: "path", Value: ""},
	))
	require.True(t, listResult.OK)
	entries, ok := listResult.Value.([]fs.DirEntry)
	require.True(t, ok)
	assert.Len(t, entries, 2)
}

func TestActions_LocalList_Bad(t *testing.T) {
	tempDir := t.TempDir()
	c := core.New()
	RegisterActions(c)

	// Listing a path that does not exist returns !OK.
	result := c.Action(ActionLocalList).Run(context.Background(), core.NewOptions(
		core.Option{Key: "root", Value: tempDir},
		core.Option{Key: "path", Value: "missing"},
	))
	assert.False(t, result.OK)
}

func TestActions_LocalList_Ugly(t *testing.T) {
	tempDir := t.TempDir()
	c := core.New()
	RegisterActions(c)

	// Empty root defaults to "/" — the list operation itself should still succeed.
	result := c.Action(ActionLocalList).Run(context.Background(), core.NewOptions(
		core.Option{Key: "path", Value: tempDir},
	))
	assert.True(t, result.OK)
}

func TestActions_LocalDelete_Good(t *testing.T) {
	tempDir := t.TempDir()
	c := core.New()
	RegisterActions(c)

	require.True(t, c.Action(ActionLocalWrite).Run(context.Background(), core.NewOptions(
		core.Option{Key: "root", Value: tempDir},
		core.Option{Key: "path", Value: "temp.txt"},
		core.Option{Key: "content", Value: "ephemeral"},
	)).OK)

	result := c.Action(ActionLocalDelete).Run(context.Background(), core.NewOptions(
		core.Option{Key: "root", Value: tempDir},
		core.Option{Key: "path", Value: "temp.txt"},
	))
	assert.True(t, result.OK)
}

func TestActions_LocalDelete_Bad(t *testing.T) {
	tempDir := t.TempDir()
	c := core.New()
	RegisterActions(c)

	// Deleting a missing file returns !OK.
	result := c.Action(ActionLocalDelete).Run(context.Background(), core.NewOptions(
		core.Option{Key: "root", Value: tempDir},
		core.Option{Key: "path", Value: "missing.txt"},
	))
	assert.False(t, result.OK)
}

func TestActions_LocalDelete_Ugly(t *testing.T) {
	tempDir := t.TempDir()
	c := core.New()
	RegisterActions(c)

	// Recursive delete of a subtree.
	require.True(t, c.Action(ActionLocalWrite).Run(context.Background(), core.NewOptions(
		core.Option{Key: "root", Value: tempDir},
		core.Option{Key: "path", Value: "branch/a.txt"},
		core.Option{Key: "content", Value: "a"},
	)).OK)
	require.True(t, c.Action(ActionLocalWrite).Run(context.Background(), core.NewOptions(
		core.Option{Key: "root", Value: tempDir},
		core.Option{Key: "path", Value: "branch/b.txt"},
		core.Option{Key: "content", Value: "b"},
	)).OK)

	result := c.Action(ActionLocalDelete).Run(context.Background(), core.NewOptions(
		core.Option{Key: "root", Value: tempDir},
		core.Option{Key: "path", Value: "branch"},
		core.Option{Key: "recursive", Value: true},
	))
	assert.True(t, result.OK)
}

func TestActions_MemoryRoundTrip_Good(t *testing.T) {
	c := core.New()
	RegisterActions(c)
	defer ResetMemoryActionStore()
	ResetMemoryActionStore()

	writeResult := c.Action(ActionMemoryWrite).Run(context.Background(), core.NewOptions(
		core.Option{Key: "path", Value: "config/app.yaml"},
		core.Option{Key: "content", Value: "port: 8080"},
	))
	require.True(t, writeResult.OK)

	readResult := c.Action(ActionMemoryRead).Run(context.Background(), core.NewOptions(
		core.Option{Key: "path", Value: "config/app.yaml"},
	))
	require.True(t, readResult.OK)
	assert.Equal(t, "port: 8080", readResult.Value)
}

func TestActions_MemoryRoundTrip_Bad(t *testing.T) {
	c := core.New()
	RegisterActions(c)
	ResetMemoryActionStore()

	// Reading a missing path returns !OK.
	result := c.Action(ActionMemoryRead).Run(context.Background(), core.NewOptions(
		core.Option{Key: "path", Value: "missing.txt"},
	))
	assert.False(t, result.OK)
}

func TestActions_MemoryRoundTrip_Ugly(t *testing.T) {
	c := core.New()
	RegisterActions(c)
	ResetMemoryActionStore()

	// ResetMemoryActionStore clears previous state between actions.
	writeResult := c.Action(ActionMemoryWrite).Run(context.Background(), core.NewOptions(
		core.Option{Key: "path", Value: "tmp.txt"},
		core.Option{Key: "content", Value: "payload"},
	))
	require.True(t, writeResult.OK)

	ResetMemoryActionStore()

	readResult := c.Action(ActionMemoryRead).Run(context.Background(), core.NewOptions(
		core.Option{Key: "path", Value: "tmp.txt"},
	))
	assert.False(t, readResult.OK)
}

func TestActions_Copy_Good(t *testing.T) {
	c := core.New()
	RegisterActions(c)

	source := NewMemoryMedium()
	destination := NewMemoryMedium()
	require.NoError(t, source.Write("input.txt", "payload"))

	result := c.Action(ActionCopy).Run(context.Background(), core.NewOptions(
		core.Option{Key: "source", Value: Medium(source)},
		core.Option{Key: "sourcePath", Value: "input.txt"},
		core.Option{Key: "destination", Value: Medium(destination)},
		core.Option{Key: "destinationPath", Value: "backup/input.txt"},
	))
	require.True(t, result.OK)

	content, err := destination.Read("backup/input.txt")
	require.NoError(t, err)
	assert.Equal(t, "payload", content)
}

func TestActions_Copy_Bad(t *testing.T) {
	c := core.New()
	RegisterActions(c)

	// Missing source medium must fail.
	result := c.Action(ActionCopy).Run(context.Background(), core.NewOptions(
		core.Option{Key: "sourcePath", Value: "input.txt"},
		core.Option{Key: "destination", Value: Medium(NewMemoryMedium())},
		core.Option{Key: "destinationPath", Value: "backup/input.txt"},
	))
	assert.False(t, result.OK)
}

func TestActions_Copy_Ugly(t *testing.T) {
	c := core.New()
	RegisterActions(c)

	source := NewMemoryMedium()
	// Source file does not exist — copy must surface the read error.
	result := c.Action(ActionCopy).Run(context.Background(), core.NewOptions(
		core.Option{Key: "source", Value: Medium(source)},
		core.Option{Key: "sourcePath", Value: "missing.txt"},
		core.Option{Key: "destination", Value: Medium(NewMemoryMedium())},
		core.Option{Key: "destinationPath", Value: "dest.txt"},
	))
	assert.False(t, result.OK)
}
