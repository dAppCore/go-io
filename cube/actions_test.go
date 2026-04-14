// SPDX-License-Identifier: EUPL-1.2

package cube

import (
	"context"
	"testing"

	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActions_RegisterActions_Good(t *testing.T) {
	c := core.New()
	RegisterActions(c)

	for _, name := range []string{ActionRead, ActionWrite, ActionPack, ActionUnpack} {
		assert.True(t, c.Action(name).Exists(), name)
	}
}

func TestActions_RegisterActions_Bad(t *testing.T) {
	// Nil Core must not panic.
	assert.NotPanics(t, func() { RegisterActions(nil) })
}

func TestActions_RegisterActions_Ugly(t *testing.T) {
	// Double registration is safe (idempotent overwrite).
	c := core.New()
	RegisterActions(c)
	assert.NotPanics(t, func() { RegisterActions(c) })
}

func TestActions_Write_Read_Good(t *testing.T) {
	c := core.New()
	RegisterActions(c)
	inner := coreio.NewMemoryMedium()

	writeResult := c.Action(ActionWrite).Run(context.Background(), core.NewOptions(
		core.Option{Key: "inner", Value: coreio.Medium(inner)},
		core.Option{Key: "key", Value: testKey},
		core.Option{Key: "path", Value: "secret.txt"},
		core.Option{Key: "content", Value: "classified"},
	))
	require.True(t, writeResult.OK)

	readResult := c.Action(ActionRead).Run(context.Background(), core.NewOptions(
		core.Option{Key: "inner", Value: coreio.Medium(inner)},
		core.Option{Key: "key", Value: testKey},
		core.Option{Key: "path", Value: "secret.txt"},
	))
	require.True(t, readResult.OK)
	assert.Equal(t, "classified", readResult.Value)
}

func TestActions_Write_Read_Bad(t *testing.T) {
	c := core.New()
	RegisterActions(c)

	// Missing inner medium must fail.
	result := c.Action(ActionWrite).Run(context.Background(), core.NewOptions(
		core.Option{Key: "key", Value: testKey},
		core.Option{Key: "path", Value: "secret.txt"},
		core.Option{Key: "content", Value: "classified"},
	))
	assert.False(t, result.OK)
}

func TestActions_Write_Read_Ugly(t *testing.T) {
	c := core.New()
	RegisterActions(c)
	inner := coreio.NewMemoryMedium()

	// Wrong-type key (string instead of []byte) must fail.
	result := c.Action(ActionWrite).Run(context.Background(), core.NewOptions(
		core.Option{Key: "inner", Value: coreio.Medium(inner)},
		core.Option{Key: "key", Value: "not a byte slice"},
		core.Option{Key: "path", Value: "secret.txt"},
		core.Option{Key: "content", Value: "classified"},
	))
	assert.False(t, result.OK)
}

func TestActions_PackUnpack_Good(t *testing.T) {
	tempDir := t.TempDir()
	c := core.New()
	RegisterActions(c)

	source := coreio.NewMemoryMedium()
	require.NoError(t, source.Write("config/app.yaml", "port: 8080"))

	outputPath := tempDir + "/app.cube"
	packResult := c.Action(ActionPack).Run(context.Background(), core.NewOptions(
		core.Option{Key: "source", Value: coreio.Medium(source)},
		core.Option{Key: "output", Value: outputPath},
		core.Option{Key: "key", Value: testKey},
	))
	require.True(t, packResult.OK)

	destination := coreio.NewMemoryMedium()
	unpackResult := c.Action(ActionUnpack).Run(context.Background(), core.NewOptions(
		core.Option{Key: "cube", Value: outputPath},
		core.Option{Key: "destination", Value: coreio.Medium(destination)},
		core.Option{Key: "key", Value: testKey},
	))
	require.True(t, unpackResult.OK)

	content, err := destination.Read("config/app.yaml")
	require.NoError(t, err)
	assert.Equal(t, "port: 8080", content)
}

func TestActions_PackUnpack_Bad(t *testing.T) {
	c := core.New()
	RegisterActions(c)

	// Pack without a source medium.
	result := c.Action(ActionPack).Run(context.Background(), core.NewOptions(
		core.Option{Key: "output", Value: "/tmp/anywhere.cube"},
		core.Option{Key: "key", Value: testKey},
	))
	assert.False(t, result.OK)

	// Unpack without a destination medium.
	result = c.Action(ActionUnpack).Run(context.Background(), core.NewOptions(
		core.Option{Key: "cube", Value: "missing.cube"},
		core.Option{Key: "key", Value: testKey},
	))
	assert.False(t, result.OK)
}

func TestActions_PackUnpack_Ugly(t *testing.T) {
	tempDir := t.TempDir()
	c := core.New()
	RegisterActions(c)

	source := coreio.NewMemoryMedium()
	require.NoError(t, source.Write("a.txt", "alpha"))

	// Wrong-type key on pack must fail without writing anything.
	outputPath := tempDir + "/bad.cube"
	result := c.Action(ActionPack).Run(context.Background(), core.NewOptions(
		core.Option{Key: "source", Value: coreio.Medium(source)},
		core.Option{Key: "output", Value: outputPath},
		core.Option{Key: "key", Value: "not a byte slice"},
	))
	assert.False(t, result.OK)
}
