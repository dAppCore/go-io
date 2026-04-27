// SPDX-License-Identifier: EUPL-1.2

package io_test

import (
	"bytes"
	"context"
	goio "io"
	"io/fs"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	core "dappco.re/go/core"
	coreio "dappco.re/go/io"
	"dappco.re/go/io/cube"
	iosftp "dappco.re/go/io/pkg/medium/sftp"
	ios3 "dappco.re/go/io/s3"
	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	pkgsftp "github.com/pkg/sftp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var actionTestCubeKey = []byte("0123456789abcdef0123456789abcdef")

func TestActions_RegisterActions_Good(t *testing.T) {
	c := core.New()
	coreio.RegisterActions(c)

	for _, name := range []string{
		coreio.ActionLocalRead, coreio.ActionLocalWrite, coreio.ActionLocalList, coreio.ActionLocalDelete,
		coreio.ActionMemoryRead, coreio.ActionMemoryWrite,
		coreio.ActionGitHubClone, coreio.ActionGitHubRead, coreio.ActionPWAScrape,
		coreio.ActionSFTPRead, coreio.ActionSFTPWrite,
		coreio.ActionS3Read, coreio.ActionS3Write,
		coreio.ActionCopy,
	} {
		assert.True(t, c.Action(name).Exists(), name)
	}
}

func TestActions_RegisterActions_Bad(t *testing.T) {
	// Nil Core must not panic and must be a no-op.
	assert.NotPanics(t, func() { coreio.RegisterActions(nil) })
}

func TestActions_RegisterActions_Ugly(t *testing.T) {
	// Calling RegisterActions twice on the same Core is safe (idempotent overwrite).
	c := core.New()
	coreio.RegisterActions(c)
	assert.NotPanics(t, func() { coreio.RegisterActions(c) })
	assert.True(t, c.Action(coreio.ActionMemoryRead).Exists())
}

func TestActions_LocalRead_Good(t *testing.T) {
	tempDir := t.TempDir()
	c := core.New()
	coreio.RegisterActions(c)

	// Prime a file via the write action, then read it back via the read action.
	writeResult := c.Action(coreio.ActionLocalWrite).Run(context.Background(), core.NewOptions(
		core.Option{Key: "root", Value: tempDir},
		core.Option{Key: "path", Value: "hello.txt"},
		core.Option{Key: "content", Value: "world"},
	))
	require.True(t, writeResult.OK)

	readResult := c.Action(coreio.ActionLocalRead).Run(context.Background(), core.NewOptions(
		core.Option{Key: "root", Value: tempDir},
		core.Option{Key: "path", Value: "hello.txt"},
	))
	require.True(t, readResult.OK)
	assert.Equal(t, "world", readResult.Value)
}

func TestActions_LocalRead_Bad(t *testing.T) {
	tempDir := t.TempDir()
	c := core.New()
	coreio.RegisterActions(c)

	// Reading a missing file returns !OK and an error in Value.
	result := c.Action(coreio.ActionLocalRead).Run(context.Background(), core.NewOptions(
		core.Option{Key: "root", Value: tempDir},
		core.Option{Key: "path", Value: "missing.txt"},
	))
	assert.False(t, result.OK)
}

func TestActions_LocalRead_Ugly(t *testing.T) {
	c := core.New()
	coreio.RegisterActions(c)

	// Empty path — read attempts to read the sandbox root which is not a file.
	result := c.Action(coreio.ActionLocalRead).Run(context.Background(), core.NewOptions(
		core.Option{Key: "root", Value: t.TempDir()},
		core.Option{Key: "path", Value: ""},
	))
	assert.False(t, result.OK)
}

func TestActions_LocalList_Good(t *testing.T) {
	tempDir := t.TempDir()
	c := core.New()
	coreio.RegisterActions(c)

	require.True(t, c.Action(coreio.ActionLocalWrite).Run(context.Background(), core.NewOptions(
		core.Option{Key: "root", Value: tempDir},
		core.Option{Key: "path", Value: "a.txt"},
		core.Option{Key: "content", Value: "alpha"},
	)).OK)
	require.True(t, c.Action(coreio.ActionLocalWrite).Run(context.Background(), core.NewOptions(
		core.Option{Key: "root", Value: tempDir},
		core.Option{Key: "path", Value: "b.txt"},
		core.Option{Key: "content", Value: "beta"},
	)).OK)

	listResult := c.Action(coreio.ActionLocalList).Run(context.Background(), core.NewOptions(
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
	coreio.RegisterActions(c)

	// Listing a path that does not exist returns !OK.
	result := c.Action(coreio.ActionLocalList).Run(context.Background(), core.NewOptions(
		core.Option{Key: "root", Value: tempDir},
		core.Option{Key: "path", Value: "missing"},
	))
	assert.False(t, result.OK)
}

func TestActions_LocalList_Ugly(t *testing.T) {
	tempDir := t.TempDir()
	c := core.New()
	coreio.RegisterActions(c)

	// Missing root must fail instead of falling back to host root.
	result := c.Action(coreio.ActionLocalList).Run(context.Background(), core.NewOptions(
		core.Option{Key: "path", Value: tempDir},
	))
	assert.False(t, result.OK)
}

func TestActions_LocalDelete_Good(t *testing.T) {
	tempDir := t.TempDir()
	c := core.New()
	coreio.RegisterActions(c)

	require.True(t, c.Action(coreio.ActionLocalWrite).Run(context.Background(), core.NewOptions(
		core.Option{Key: "root", Value: tempDir},
		core.Option{Key: "path", Value: "temp.txt"},
		core.Option{Key: "content", Value: "ephemeral"},
	)).OK)

	result := c.Action(coreio.ActionLocalDelete).Run(context.Background(), core.NewOptions(
		core.Option{Key: "root", Value: tempDir},
		core.Option{Key: "path", Value: "temp.txt"},
	))
	assert.True(t, result.OK)
}

func TestActions_LocalDelete_Bad(t *testing.T) {
	tempDir := t.TempDir()
	c := core.New()
	coreio.RegisterActions(c)

	// Deleting a missing file returns !OK.
	result := c.Action(coreio.ActionLocalDelete).Run(context.Background(), core.NewOptions(
		core.Option{Key: "root", Value: tempDir},
		core.Option{Key: "path", Value: "missing.txt"},
	))
	assert.False(t, result.OK)
}

func TestActions_LocalDelete_Ugly(t *testing.T) {
	tempDir := t.TempDir()
	c := core.New()
	coreio.RegisterActions(c)

	// Recursive delete of a subtree.
	require.True(t, c.Action(coreio.ActionLocalWrite).Run(context.Background(), core.NewOptions(
		core.Option{Key: "root", Value: tempDir},
		core.Option{Key: "path", Value: "branch/a.txt"},
		core.Option{Key: "content", Value: "a"},
	)).OK)
	require.True(t, c.Action(coreio.ActionLocalWrite).Run(context.Background(), core.NewOptions(
		core.Option{Key: "root", Value: tempDir},
		core.Option{Key: "path", Value: "branch/b.txt"},
		core.Option{Key: "content", Value: "b"},
	)).OK)

	result := c.Action(coreio.ActionLocalDelete).Run(context.Background(), core.NewOptions(
		core.Option{Key: "root", Value: tempDir},
		core.Option{Key: "path", Value: "branch"},
		core.Option{Key: "recursive", Value: true},
	))
	assert.True(t, result.OK)
}

func TestActions_MemoryRoundTrip_Good(t *testing.T) {
	c := core.New()
	coreio.RegisterActions(c)
	defer coreio.ResetMemoryActionStore()
	coreio.ResetMemoryActionStore()

	writeResult := c.Action(coreio.ActionMemoryWrite).Run(context.Background(), core.NewOptions(
		core.Option{Key: "path", Value: "config/app.yaml"},
		core.Option{Key: "content", Value: "port: 8080"},
	))
	require.True(t, writeResult.OK)

	readResult := c.Action(coreio.ActionMemoryRead).Run(context.Background(), core.NewOptions(
		core.Option{Key: "path", Value: "config/app.yaml"},
	))
	require.True(t, readResult.OK)
	assert.Equal(t, "port: 8080", readResult.Value)
}

func TestActions_MemoryRoundTrip_Bad(t *testing.T) {
	c := core.New()
	coreio.RegisterActions(c)
	coreio.ResetMemoryActionStore()

	// Reading a missing path returns !OK.
	result := c.Action(coreio.ActionMemoryRead).Run(context.Background(), core.NewOptions(
		core.Option{Key: "path", Value: "missing.txt"},
	))
	assert.False(t, result.OK)
}

func TestActions_MemoryRoundTrip_Ugly(t *testing.T) {
	c := core.New()
	coreio.RegisterActions(c)
	coreio.ResetMemoryActionStore()

	// ResetMemoryActionStore clears previous state between actions.
	writeResult := c.Action(coreio.ActionMemoryWrite).Run(context.Background(), core.NewOptions(
		core.Option{Key: "path", Value: "tmp.txt"},
		core.Option{Key: "content", Value: "payload"},
	))
	require.True(t, writeResult.OK)

	coreio.ResetMemoryActionStore()

	readResult := c.Action(coreio.ActionMemoryRead).Run(context.Background(), core.NewOptions(
		core.Option{Key: "path", Value: "tmp.txt"},
	))
	assert.False(t, readResult.OK)
}

func TestActions_Copy_Good(t *testing.T) {
	c := core.New()
	coreio.RegisterActions(c)

	source := coreio.NewMemoryMedium()
	destination := coreio.NewMemoryMedium()
	require.NoError(t, source.Write("input.txt", "payload"))

	result := c.Action(coreio.ActionCopy).Run(context.Background(), core.NewOptions(
		core.Option{Key: "source", Value: coreio.Medium(source)},
		core.Option{Key: "sourcePath", Value: "input.txt"},
		core.Option{Key: "destination", Value: coreio.Medium(destination)},
		core.Option{Key: "destinationPath", Value: "backup/input.txt"},
	))
	require.True(t, result.OK)

	content, err := destination.Read("backup/input.txt")
	require.NoError(t, err)
	assert.Equal(t, "payload", content)
}

func TestActions_Copy_Bad(t *testing.T) {
	c := core.New()
	coreio.RegisterActions(c)

	// Missing source medium must fail.
	result := c.Action(coreio.ActionCopy).Run(context.Background(), core.NewOptions(
		core.Option{Key: "sourcePath", Value: "input.txt"},
		core.Option{Key: "destination", Value: coreio.Medium(coreio.NewMemoryMedium())},
		core.Option{Key: "destinationPath", Value: "backup/input.txt"},
	))
	assert.False(t, result.OK)
}

func TestActions_Copy_Ugly(t *testing.T) {
	c := core.New()
	coreio.RegisterActions(c)

	source := coreio.NewMemoryMedium()
	// Source file does not exist — copy must surface the read error.
	result := c.Action(coreio.ActionCopy).Run(context.Background(), core.NewOptions(
		core.Option{Key: "source", Value: coreio.Medium(source)},
		core.Option{Key: "sourcePath", Value: "missing.txt"},
		core.Option{Key: "destination", Value: coreio.Medium(coreio.NewMemoryMedium())},
		core.Option{Key: "destinationPath", Value: "dest.txt"},
	))
	assert.False(t, result.OK)
}

func TestActions_S3ReadWrite_Good(t *testing.T) {
	c := core.New()
	coreio.RegisterActions(c)
	medium := newActionS3Medium(t)

	writeResult := c.Action(coreio.ActionS3Write).Run(context.Background(), core.NewOptions(
		core.Option{Key: "medium", Value: medium},
		core.Option{Key: "path", Value: "reports/daily.txt"},
		core.Option{Key: "content", Value: "done"},
	))
	require.True(t, writeResult.OK)

	readResult := c.Action(coreio.ActionS3Read).Run(context.Background(), core.NewOptions(
		core.Option{Key: "medium", Value: medium},
		core.Option{Key: "path", Value: "reports/daily.txt"},
	))
	require.True(t, readResult.OK)
	assert.Equal(t, "done", readResult.Value)
}

func TestActions_S3ReadWrite_Ugly(t *testing.T) {
	c := core.New()
	coreio.RegisterActions(c)
	medium := newActionS3Medium(t)

	readResult := c.Action(coreio.ActionS3Read).Run(context.Background(), core.NewOptions(
		core.Option{Key: "medium", Value: medium},
		core.Option{Key: "path", Value: "missing.txt"},
	))
	assert.False(t, readResult.OK)

	writeResult := c.Action(coreio.ActionS3Write).Run(context.Background(), core.NewOptions(
		core.Option{Key: "medium", Value: medium},
		core.Option{Key: "path", Value: ""},
		core.Option{Key: "content", Value: "payload"},
	))
	assert.False(t, writeResult.OK)
}

func TestActions_SFTPReadWrite_Good(t *testing.T) {
	c := core.New()
	coreio.RegisterActions(c)
	medium := newActionSFTPTestMedium(t)

	writeResult := c.Action(coreio.ActionSFTPWrite).Run(context.Background(), core.NewOptions(
		core.Option{Key: "medium", Value: medium},
		core.Option{Key: "path", Value: "notes/todo.txt"},
		core.Option{Key: "content", Value: "ship sftp"},
	))
	require.True(t, writeResult.OK)

	readResult := c.Action(coreio.ActionSFTPRead).Run(context.Background(), core.NewOptions(
		core.Option{Key: "medium", Value: medium},
		core.Option{Key: "path", Value: "notes/todo.txt"},
	))
	require.True(t, readResult.OK)
	assert.Equal(t, "ship sftp", readResult.Value)
}

func TestActions_SFTPReadWrite_Ugly(t *testing.T) {
	c := core.New()
	coreio.RegisterActions(c)
	medium := newActionSFTPTestMedium(t)

	readResult := c.Action(coreio.ActionSFTPRead).Run(context.Background(), core.NewOptions(
		core.Option{Key: "medium", Value: medium},
		core.Option{Key: "path", Value: "missing.txt"},
	))
	assert.False(t, readResult.OK)

	writeResult := c.Action(coreio.ActionSFTPWrite).Run(context.Background(), core.NewOptions(
		core.Option{Key: "medium", Value: medium},
		core.Option{Key: "path", Value: ""},
		core.Option{Key: "content", Value: "payload"},
	))
	assert.False(t, writeResult.OK)
}

func TestActions_CubeReadWritePackUnpack_Good(t *testing.T) {
	tempDir := t.TempDir()
	c := core.New()
	coreio.RegisterActions(c)
	cube.RegisterActions(c)

	inner := coreio.NewMemoryMedium()
	cubeMedium, err := cube.New(cube.Options{Inner: inner, Key: actionTestCubeKey})
	require.NoError(t, err)

	writeResult := c.Action(coreio.ActionCubeWrite).Run(context.Background(), core.NewOptions(
		core.Option{Key: "medium", Value: coreio.Medium(cubeMedium)},
		core.Option{Key: "path", Value: "secret.txt"},
		core.Option{Key: "content", Value: "classified"},
	))
	require.True(t, writeResult.OK)

	readResult := c.Action(coreio.ActionCubeRead).Run(context.Background(), core.NewOptions(
		core.Option{Key: "medium", Value: coreio.Medium(cubeMedium)},
		core.Option{Key: "path", Value: "secret.txt"},
	))
	require.True(t, readResult.OK)
	assert.Equal(t, "classified", readResult.Value)

	innerContract := coreio.NewMemoryMedium()
	contractWrite := c.Action(coreio.ActionCubeWrite).Run(context.Background(), core.NewOptions(
		core.Option{Key: "inner", Value: coreio.Medium(innerContract)},
		core.Option{Key: "key", Value: actionTestCubeKey},
		core.Option{Key: "path", Value: "inner.txt"},
		core.Option{Key: "content", Value: "via inner"},
	))
	require.True(t, contractWrite.OK)

	contractRead := c.Action(coreio.ActionCubeRead).Run(context.Background(), core.NewOptions(
		core.Option{Key: "inner", Value: coreio.Medium(innerContract)},
		core.Option{Key: "key", Value: actionTestCubeKey},
		core.Option{Key: "path", Value: "inner.txt"},
	))
	require.True(t, contractRead.OK)
	assert.Equal(t, "via inner", contractRead.Value)

	source := coreio.NewMemoryMedium()
	require.NoError(t, source.Write("config/app.yaml", "port: 8080"))
	outputPath := tempDir + "/app.cube"
	packResult := c.Action(coreio.ActionCubePack).Run(context.Background(), core.NewOptions(
		core.Option{Key: "source", Value: coreio.Medium(source)},
		core.Option{Key: "output", Value: outputPath},
		core.Option{Key: "key", Value: actionTestCubeKey},
	))
	require.True(t, packResult.OK)

	destination := coreio.NewMemoryMedium()
	unpackResult := c.Action(coreio.ActionCubeUnpack).Run(context.Background(), core.NewOptions(
		core.Option{Key: "cube", Value: outputPath},
		core.Option{Key: "destination", Value: coreio.Medium(destination)},
		core.Option{Key: "key", Value: actionTestCubeKey},
	))
	require.True(t, unpackResult.OK)

	content, err := destination.Read("config/app.yaml")
	require.NoError(t, err)
	assert.Equal(t, "port: 8080", content)
}

func TestActions_CubeReadWritePackUnpack_Ugly(t *testing.T) {
	c := core.New()
	coreio.RegisterActions(c)
	cube.RegisterActions(c)

	readResult := c.Action(coreio.ActionCubeRead).Run(context.Background(), core.NewOptions(
		core.Option{Key: "inner", Value: coreio.Medium(coreio.NewMemoryMedium())},
		core.Option{Key: "key", Value: actionTestCubeKey},
		core.Option{Key: "path", Value: "missing.txt"},
	))
	assert.False(t, readResult.OK)

	writeResult := c.Action(coreio.ActionCubeWrite).Run(context.Background(), core.NewOptions(
		core.Option{Key: "inner", Value: coreio.Medium(coreio.NewMemoryMedium())},
		core.Option{Key: "key", Value: []byte("short")},
		core.Option{Key: "path", Value: "secret.txt"},
		core.Option{Key: "content", Value: "payload"},
	))
	assert.False(t, writeResult.OK)

	packResult := c.Action(coreio.ActionCubePack).Run(context.Background(), core.NewOptions(
		core.Option{Key: "output", Value: t.TempDir() + "/app.cube"},
		core.Option{Key: "key", Value: actionTestCubeKey},
	))
	assert.False(t, packResult.OK)

	unpackResult := c.Action(coreio.ActionCubeUnpack).Run(context.Background(), core.NewOptions(
		core.Option{Key: "cube", Value: t.TempDir() + "/missing.cube"},
		core.Option{Key: "destination", Value: coreio.Medium(coreio.NewMemoryMedium())},
		core.Option{Key: "key", Value: actionTestCubeKey},
	))
	assert.False(t, unpackResult.OK)
}

func TestActions_GitHubPWAStubs_Bad(t *testing.T) {
	c := core.New()
	coreio.RegisterActions(c)

	for _, name := range []string{coreio.ActionGitHubClone, coreio.ActionGitHubRead, coreio.ActionPWAScrape} {
		result := c.Action(name).Run(context.Background(), core.NewOptions())
		require.False(t, result.OK, name)
		err, ok := result.Value.(error)
		require.True(t, ok, name)
		assert.Contains(t, err.Error(), "not implemented", name)
		assert.Contains(t, err.Error(), "#633", name)
	}
}

type actionTestS3Client struct {
	mu      sync.RWMutex
	objects map[string][]byte
}

func newActionS3Medium(t *testing.T) *ios3.Medium {
	t.Helper()
	medium, err := ios3.New(ios3.Options{
		Bucket: "bucket",
		Client: &actionTestS3Client{objects: make(map[string][]byte)},
	})
	require.NoError(t, err)
	return medium
}

func (client *actionTestS3Client) GetObject(_ context.Context, params *awss3.GetObjectInput, _ ...func(*awss3.Options)) (*awss3.GetObjectOutput, error) {
	client.mu.RLock()
	defer client.mu.RUnlock()

	key := aws.ToString(params.Key)
	data, ok := client.objects[key]
	if !ok {
		return nil, core.E("actionsTest.s3.GetObject", "key not found", fs.ErrNotExist)
	}
	return &awss3.GetObjectOutput{
		Body:          goio.NopCloser(bytes.NewReader(data)),
		ContentLength: aws.Int64(int64(len(data))),
	}, nil
}

func (client *actionTestS3Client) PutObject(_ context.Context, params *awss3.PutObjectInput, _ ...func(*awss3.Options)) (*awss3.PutObjectOutput, error) {
	client.mu.Lock()
	defer client.mu.Unlock()

	data, err := goio.ReadAll(params.Body)
	if err != nil {
		return nil, err
	}
	client.objects[aws.ToString(params.Key)] = data
	return &awss3.PutObjectOutput{}, nil
}

func (client *actionTestS3Client) DeleteObject(_ context.Context, params *awss3.DeleteObjectInput, _ ...func(*awss3.Options)) (*awss3.DeleteObjectOutput, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	delete(client.objects, aws.ToString(params.Key))
	return &awss3.DeleteObjectOutput{}, nil
}

func (client *actionTestS3Client) DeleteObjects(_ context.Context, params *awss3.DeleteObjectsInput, _ ...func(*awss3.Options)) (*awss3.DeleteObjectsOutput, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	for _, object := range params.Delete.Objects {
		delete(client.objects, aws.ToString(object.Key))
	}
	return &awss3.DeleteObjectsOutput{}, nil
}

func (client *actionTestS3Client) HeadObject(_ context.Context, params *awss3.HeadObjectInput, _ ...func(*awss3.Options)) (*awss3.HeadObjectOutput, error) {
	client.mu.RLock()
	defer client.mu.RUnlock()
	data, ok := client.objects[aws.ToString(params.Key)]
	if !ok {
		return nil, core.E("actionsTest.s3.HeadObject", "key not found", fs.ErrNotExist)
	}
	return &awss3.HeadObjectOutput{ContentLength: aws.Int64(int64(len(data)))}, nil
}

func (client *actionTestS3Client) ListObjectsV2(context.Context, *awss3.ListObjectsV2Input, ...func(*awss3.Options)) (*awss3.ListObjectsV2Output, error) {
	return &awss3.ListObjectsV2Output{}, nil
}

func (client *actionTestS3Client) CopyObject(_ context.Context, params *awss3.CopyObjectInput, _ ...func(*awss3.Options)) (*awss3.CopyObjectOutput, error) {
	client.mu.Lock()
	defer client.mu.Unlock()

	_, sourceKey, ok := strings.Cut(aws.ToString(params.CopySource), "/")
	if !ok {
		return nil, core.E("actionsTest.s3.CopyObject", "invalid copy source", fs.ErrInvalid)
	}
	data, ok := client.objects[sourceKey]
	if !ok {
		return nil, core.E("actionsTest.s3.CopyObject", "source not found", fs.ErrNotExist)
	}
	client.objects[aws.ToString(params.Key)] = append([]byte(nil), data...)
	return &awss3.CopyObjectOutput{}, nil
}

func newActionSFTPTestMedium(t *testing.T) *iosftp.Medium {
	t.Helper()

	serverConn, clientConn := net.Pipe()
	server := pkgsftp.NewRequestServer(serverConn, pkgsftp.InMemHandler())
	done := make(chan error, 1)
	go func() {
		done <- server.Serve()
	}()

	client, err := pkgsftp.NewClientPipe(clientConn, clientConn)
	require.NoError(t, err)

	medium, err := iosftp.New(iosftp.Options{Client: client})
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = client.Close()
		_ = clientConn.Close()
		_ = serverConn.Close()
		select {
		case <-done:
		case <-time.After(time.Second):
		}
	})

	return medium
}
