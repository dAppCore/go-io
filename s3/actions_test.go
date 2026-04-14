// SPDX-License-Identifier: EUPL-1.2

package s3

import (
	"context"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActions_RegisterActions_Good(t *testing.T) {
	c := core.New()
	RegisterActions(c)
	assert.True(t, c.Action(ActionRead).Exists())
	assert.True(t, c.Action(ActionWrite).Exists())
}

func TestActions_RegisterActions_Bad(t *testing.T) {
	// Nil Core must not panic.
	assert.NotPanics(t, func() { RegisterActions(nil) })
}

func TestActions_RegisterActions_Ugly(t *testing.T) {
	// Double registration is safe.
	c := core.New()
	RegisterActions(c)
	assert.NotPanics(t, func() { RegisterActions(c) })
}

func TestActions_ReadWrite_Good(t *testing.T) {
	c := core.New()
	RegisterActions(c)

	client := newTestS3Client()
	medium, err := New(Options{Bucket: "bucket", Client: client, Prefix: "prefix/"})
	require.NoError(t, err)

	writeResult := c.Action(ActionWrite).Run(context.Background(), core.NewOptions(
		core.Option{Key: "medium", Value: medium},
		core.Option{Key: "path", Value: "reports/daily.txt"},
		core.Option{Key: "content", Value: "done"},
	))
	require.True(t, writeResult.OK)

	readResult := c.Action(ActionRead).Run(context.Background(), core.NewOptions(
		core.Option{Key: "medium", Value: medium},
		core.Option{Key: "path", Value: "reports/daily.txt"},
	))
	require.True(t, readResult.OK)
	assert.Equal(t, "done", readResult.Value)
}

func TestActions_ReadWrite_Bad(t *testing.T) {
	c := core.New()
	RegisterActions(c)

	// Missing medium must fail.
	result := c.Action(ActionRead).Run(context.Background(), core.NewOptions(
		core.Option{Key: "path", Value: "reports/daily.txt"},
	))
	assert.False(t, result.OK)

	result = c.Action(ActionWrite).Run(context.Background(), core.NewOptions(
		core.Option{Key: "path", Value: "reports/daily.txt"},
		core.Option{Key: "content", Value: "done"},
	))
	assert.False(t, result.OK)
}

func TestActions_ReadWrite_Ugly(t *testing.T) {
	c := core.New()
	RegisterActions(c)

	client := newTestS3Client()
	medium, err := New(Options{Bucket: "bucket", Client: client})
	require.NoError(t, err)

	// Reading a key that was never written must fail.
	result := c.Action(ActionRead).Run(context.Background(), core.NewOptions(
		core.Option{Key: "medium", Value: medium},
		core.Option{Key: "path", Value: "missing.txt"},
	))
	assert.False(t, result.OK)
}
