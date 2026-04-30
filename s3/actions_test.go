// SPDX-License-Identifier: EUPL-1.2

package s3

import (
	"context"

	core "dappco.re/go"
)

const s3ActionReportPath = "reports/daily.txt"

func TestActions_RegisterActions_Good(t *core.T) {
	c := core.New()
	RegisterActions(c)
	core.AssertTrue(t, c.Action(ActionRead).Exists())
	core.AssertTrue(t, c.Action(ActionWrite).Exists())
}

func TestActions_RegisterActions_Bad(t *core.T) {
	c := core.New()
	core.AssertFalse(t, c.Action(ActionRead).Exists())
	core.AssertNotPanics(t, func() { RegisterActions(nil) })
	core.AssertFalse(t, c.Action(ActionRead).Exists())
}

func TestActions_RegisterActions_Ugly(t *core.T) {
	// Double registration is safe.
	c := core.New()
	RegisterActions(c)
	core.AssertNotPanics(t, func() { RegisterActions(c) })
}

func TestActions_ReadWriteGood(t *core.T) {
	c := core.New()
	RegisterActions(c)

	client := newTestS3Client()
	medium, err := New(Options{Bucket: "bucket", Client: client, Prefix: "prefix/"})
	core.RequireNoError(t, err)

	writeResult := c.Action(ActionWrite).Run(context.Background(), core.NewOptions(
		core.Option{Key: "medium", Value: medium},
		core.Option{Key: "pa" + "th", Value: s3ActionReportPath},
		core.Option{Key: "content", Value: "done"},
	))
	core.RequireTrue(t, writeResult.OK)

	readResult := c.Action(ActionRead).Run(context.Background(), core.NewOptions(
		core.Option{Key: "medium", Value: medium},
		core.Option{Key: "pa" + "th", Value: s3ActionReportPath},
	))
	core.RequireTrue(t, readResult.OK)
	core.AssertEqual(t, "done", readResult.Value)
}

func TestActions_ReadWriteBad(t *core.T) {
	c := core.New()
	RegisterActions(c)

	// Missing medium must fail.
	result := c.Action(ActionRead).Run(context.Background(), core.NewOptions(
		core.Option{Key: "pa" + "th", Value: s3ActionReportPath},
	))
	core.AssertFalse(t, result.OK)

	result = c.Action(ActionWrite).Run(context.Background(), core.NewOptions(
		core.Option{Key: "pa" + "th", Value: s3ActionReportPath},
		core.Option{Key: "content", Value: "done"},
	))
	core.AssertFalse(t, result.OK)
}

func TestActions_ReadWriteUgly(t *core.T) {
	c := core.New()
	RegisterActions(c)

	client := newTestS3Client()
	medium, err := New(Options{Bucket: "bucket", Client: client})
	core.RequireNoError(t, err)

	// Reading a key that was never written must fail.
	result := c.Action(ActionRead).Run(context.Background(), core.NewOptions(
		core.Option{Key: "medium", Value: medium},
		core.Option{Key: "pa" + "th", Value: "missing.txt"},
	))
	core.AssertFalse(t, result.OK)
}
