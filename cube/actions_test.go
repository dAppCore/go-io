// SPDX-License-Identifier: EUPL-1.2

package cube

import (
	"context"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
)

func TestActions_RegisterActions_Good(t *core.T) {
	c := core.New()
	RegisterActions(c)

	for _, name := range []string{ActionRead, ActionWrite, ActionPack, ActionUnpack} {
		core.AssertTrue(t, c.Action(name).Exists(), name)
	}
}

func TestActions_RegisterActions_Bad(t *core.T) {
	c := core.New()
	core.AssertFalse(t, c.Action(ActionRead).Exists())
	core.AssertNotPanics(t, func() { RegisterActions(nil) })
	core.AssertFalse(t, c.Action(ActionRead).Exists())
}

func TestActions_RegisterActions_Ugly(t *core.T) {
	// Double registration is safe (idempotent overwrite).
	c := core.New()
	RegisterActions(c)
	core.AssertNotPanics(t, func() { RegisterActions(c) })
}

func TestActions_Write_Read_Good(t *core.T) {
	c := core.New()
	RegisterActions(c)
	inner := coreio.NewMemoryMedium()

	writeResult := c.Action(ActionWrite).Run(context.Background(), core.NewOptions(
		core.Option{Key: "inner", Value: coreio.Medium(inner)},
		core.Option{Key: "key", Value: testKey},
		core.Option{Key: "path", Value: "secret.txt"},
		core.Option{Key: "content", Value: "classified"},
	))
	core.RequireTrue(t, writeResult.OK)

	readResult := c.Action(ActionRead).Run(context.Background(), core.NewOptions(
		core.Option{Key: "inner", Value: coreio.Medium(inner)},
		core.Option{Key: "key", Value: testKey},
		core.Option{Key: "path", Value: "secret.txt"},
	))
	core.RequireTrue(t, readResult.OK)
	core.AssertEqual(t, "classified", readResult.Value)
}

func TestActions_Write_Read_Bad(t *core.T) {
	c := core.New()
	RegisterActions(c)

	// Missing inner medium must fail.
	result := c.Action(ActionWrite).Run(context.Background(), core.NewOptions(
		core.Option{Key: "key", Value: testKey},
		core.Option{Key: "path", Value: "secret.txt"},
		core.Option{Key: "content", Value: "classified"},
	))
	core.AssertFalse(t, result.OK)
}

func TestActions_Write_Read_Ugly(t *core.T) {
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
	core.AssertFalse(t, result.OK)
}

func TestActions_PackUnpack_Good(t *core.T) {
	tempDir := t.TempDir()
	c := core.New()
	RegisterActions(c)

	source := coreio.NewMemoryMedium()
	core.RequireNoError(t, source.Write("config/app.yaml", "port: 8080"))

	outputPath := tempDir + "/app.cube"
	packResult := c.Action(ActionPack).Run(context.Background(), core.NewOptions(
		core.Option{Key: "source", Value: coreio.Medium(source)},
		core.Option{Key: "output", Value: outputPath},
		core.Option{Key: "key", Value: testKey},
	))
	core.RequireTrue(t, packResult.OK)

	destination := coreio.NewMemoryMedium()
	unpackResult := c.Action(ActionUnpack).Run(context.Background(), core.NewOptions(
		core.Option{Key: "cube", Value: outputPath},
		core.Option{Key: "destination", Value: coreio.Medium(destination)},
		core.Option{Key: "key", Value: testKey},
	))
	core.RequireTrue(t, unpackResult.OK)

	content, err := destination.Read("config/app.yaml")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "port: 8080", content)
}

func TestActions_PackUnpack_Bad(t *core.T) {
	c := core.New()
	RegisterActions(c)

	// Pack without a source medium.
	result := c.Action(ActionPack).Run(context.Background(), core.NewOptions(
		core.Option{Key: "output", Value: "/tmp/anywhere.cube"},
		core.Option{Key: "key", Value: testKey},
	))
	core.AssertFalse(t, result.OK)

	// Unpack without a destination medium.
	result = c.Action(ActionUnpack).Run(context.Background(), core.NewOptions(
		core.Option{Key: "cube", Value: "missing.cube"},
		core.Option{Key: "key", Value: testKey},
	))
	core.AssertFalse(t, result.OK)
}

func TestActions_PackUnpack_Ugly(t *core.T) {
	tempDir := t.TempDir()
	c := core.New()
	RegisterActions(c)

	source := coreio.NewMemoryMedium()
	core.RequireNoError(t, source.Write("a.txt", "alpha"))

	// Wrong-type key on pack must fail without writing anything.
	outputPath := tempDir + "/bad.cube"
	result := c.Action(ActionPack).Run(context.Background(), core.NewOptions(
		core.Option{Key: "source", Value: coreio.Medium(source)},
		core.Option{Key: "output", Value: outputPath},
		core.Option{Key: "key", Value: "not a byte slice"},
	))
	core.AssertFalse(t, result.OK)
}
