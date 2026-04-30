package io_test

import (
	"context"
	. "dappco.re/go"
	coreio "dappco.re/go/io"
	"dappco.re/go/io/cube"
	iosftp "dappco.re/go/io/pkg/medium/sftp"
	ios3 "dappco.re/go/io/s3"
	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	pkgsftp "github.com/pkg/sftp"
	goio "io"
	"io/fs"
	"net"
	"sync"
	"time"
)

// SPDX-License-Identifier: EUPL-1.2

const (
	actionMissingPath     = "missing.txt"
	actionConfigPath      = "config/app.yaml"
	actionConfigContent   = "port: 8080"
	actionInputPath       = "input.txt"
	actionBackupInputPath = "backup/input.txt"
	actionSecretPath      = "secret.txt"
)

var actionTestCubeKey = []byte("0123456789abcdef0123456789abcdef")

func TestActions_RegisterActions_Good(t *T) {
	c := New()
	coreio.RegisterActions(c)

	for _, name := range []string{
		coreio.ActionLocalRead, coreio.ActionLocalWrite, coreio.ActionLocalList, coreio.ActionLocalDelete,
		coreio.ActionMemoryRead, coreio.ActionMemoryWrite,
		coreio.ActionGitHubClone, coreio.ActionGitHubRead, coreio.ActionPWAScrape,
		coreio.ActionSFTPRead, coreio.ActionSFTPWrite,
		coreio.ActionS3Read, coreio.ActionS3Write,
		coreio.ActionCopy,
	} {
		AssertTrue(t, c.Action(name).Exists(), name)
	}
}

func TestActions_RegisterActions_Bad(t *T) {
	c := New()
	AssertFalse(t, c.Action(coreio.ActionMemoryRead).Exists())
	AssertNotPanics(t, func() { coreio.RegisterActions(nil) })
	AssertFalse(t, c.Action(coreio.ActionMemoryRead).Exists())
}

func TestActions_RegisterActions_Ugly(t *T) {
	// Calling RegisterActions twice on the same Core is safe (idempotent overwrite).
	c := New()
	coreio.RegisterActions(c)
	AssertNotPanics(t, func() { coreio.RegisterActions(c) })
	AssertTrue(t, c.Action(coreio.ActionMemoryRead).Exists())
}

func TestActions_LocalReadGood(t *T) {
	tempDir := t.TempDir()
	c := New()
	coreio.RegisterActions(c)

	// Prime a file via the write action, then read it back via the read action.
	writeResult := c.Action(coreio.ActionLocalWrite).Run(context.Background(), NewOptions(
		Option{Key: "root", Value: tempDir},
		Option{Key: "pa" + "th", Value: "hello.txt"},
		Option{Key: "content", Value: "world"},
	))
	RequireTrue(t, writeResult.OK)

	readResult := c.Action(coreio.ActionLocalRead).Run(context.Background(), NewOptions(
		Option{Key: "root", Value: tempDir},
		Option{Key: "pa" + "th", Value: "hello.txt"},
	))
	RequireTrue(t, readResult.OK)
	AssertEqual(t, "world", readResult.Value)
}

func TestActions_LocalReadBad(t *T) {
	tempDir := t.TempDir()
	c := New()
	coreio.RegisterActions(c)

	// Reading a missing file returns !OK and an error in Value.
	result := c.Action(coreio.ActionLocalRead).Run(context.Background(), NewOptions(
		Option{Key: "root", Value: tempDir},
		Option{Key: "pa" + "th", Value: actionMissingPath},
	))
	AssertFalse(t, result.OK)
}

func TestActions_LocalReadUgly(t *T) {
	c := New()
	coreio.RegisterActions(c)

	// Empty path — read attempts to read the sandbox root which is not a file.
	result := c.Action(coreio.ActionLocalRead).Run(context.Background(), NewOptions(
		Option{Key: "root", Value: t.TempDir()},
		Option{Key: "pa" + "th", Value: ""},
	))
	AssertFalse(t, result.OK)
}

func TestActions_LocalListGood(t *T) {
	tempDir := t.TempDir()
	c := New()
	coreio.RegisterActions(c)

	RequireTrue(t, c.Action(coreio.ActionLocalWrite).Run(context.Background(), NewOptions(
		Option{Key: "root", Value: tempDir},
		Option{Key: "pa" + "th", Value: "a.txt"},
		Option{Key: "content", Value: "alpha"},
	)).OK)
	RequireTrue(t, c.Action(coreio.ActionLocalWrite).Run(context.Background(), NewOptions(
		Option{Key: "root", Value: tempDir},
		Option{Key: "pa" + "th", Value: "b.txt"},
		Option{Key: "content", Value: "beta"},
	)).OK)

	listResult := c.Action(coreio.ActionLocalList).Run(context.Background(), NewOptions(
		Option{Key: "root", Value: tempDir},
		Option{Key: "pa" + "th", Value: ""},
	))
	RequireTrue(t, listResult.OK)
	entries, ok := listResult.Value.([]fs.DirEntry)
	RequireTrue(t, ok)
	AssertLen(t, entries, 2)
}

func TestActions_LocalListBad(t *T) {
	tempDir := t.TempDir()
	c := New()
	coreio.RegisterActions(c)

	// Listing a path that does not exist returns !OK.
	result := c.Action(coreio.ActionLocalList).Run(context.Background(), NewOptions(
		Option{Key: "root", Value: tempDir},
		Option{Key: "pa" + "th", Value: "missing"},
	))
	AssertFalse(t, result.OK)
}

func TestActions_LocalListUgly(t *T) {
	tempDir := t.TempDir()
	c := New()
	coreio.RegisterActions(c)

	// Missing root must fail instead of falling back to host root.
	result := c.Action(coreio.ActionLocalList).Run(context.Background(), NewOptions(
		Option{Key: "pa" + "th", Value: tempDir},
	))
	AssertFalse(t, result.OK)
}

func TestActions_LocalDeleteGood(t *T) {
	tempDir := t.TempDir()
	c := New()
	coreio.RegisterActions(c)

	RequireTrue(t, c.Action(coreio.ActionLocalWrite).Run(context.Background(), NewOptions(
		Option{Key: "root", Value: tempDir},
		Option{Key: "pa" + "th", Value: "temp.txt"},
		Option{Key: "content", Value: "ephemeral"},
	)).OK)

	result := c.Action(coreio.ActionLocalDelete).Run(context.Background(), NewOptions(
		Option{Key: "root", Value: tempDir},
		Option{Key: "pa" + "th", Value: "temp.txt"},
	))
	AssertTrue(t, result.OK)
}

func TestActions_LocalDeleteBad(t *T) {
	tempDir := t.TempDir()
	c := New()
	coreio.RegisterActions(c)

	// Deleting a missing file returns !OK.
	result := c.Action(coreio.ActionLocalDelete).Run(context.Background(), NewOptions(
		Option{Key: "root", Value: tempDir},
		Option{Key: "pa" + "th", Value: actionMissingPath},
	))
	AssertFalse(t, result.OK)
}

func TestActions_LocalDeleteUgly(t *T) {
	tempDir := t.TempDir()
	c := New()
	coreio.RegisterActions(c)

	// Recursive delete of a subtree.
	RequireTrue(t, c.Action(coreio.ActionLocalWrite).Run(context.Background(), NewOptions(
		Option{Key: "root", Value: tempDir},
		Option{Key: "pa" + "th", Value: "branch/a.txt"},
		Option{Key: "content", Value: "a"},
	)).OK)
	RequireTrue(t, c.Action(coreio.ActionLocalWrite).Run(context.Background(), NewOptions(
		Option{Key: "root", Value: tempDir},
		Option{Key: "pa" + "th", Value: "branch/b.txt"},
		Option{Key: "content", Value: "b"},
	)).OK)

	result := c.Action(coreio.ActionLocalDelete).Run(context.Background(), NewOptions(
		Option{Key: "root", Value: tempDir},
		Option{Key: "pa" + "th", Value: "branch"},
		Option{Key: "recursive", Value: true},
	))
	AssertTrue(t, result.OK)
}

func TestActions_MemoryRoundTripGood(t *T) {
	c := New()
	coreio.RegisterActions(c)
	defer coreio.ResetMemoryActionStore()
	coreio.ResetMemoryActionStore()

	writeResult := c.Action(coreio.ActionMemoryWrite).Run(context.Background(), NewOptions(
		Option{Key: "pa" + "th", Value: actionConfigPath},
		Option{Key: "content", Value: actionConfigContent},
	))
	RequireTrue(t, writeResult.OK)

	readResult := c.Action(coreio.ActionMemoryRead).Run(context.Background(), NewOptions(
		Option{Key: "pa" + "th", Value: actionConfigPath},
	))
	RequireTrue(t, readResult.OK)
	AssertEqual(t, actionConfigContent, readResult.Value)
}

func TestActions_MemoryRoundTripBad(t *T) {
	c := New()
	coreio.RegisterActions(c)
	coreio.ResetMemoryActionStore()

	// Reading a missing path returns !OK.
	result := c.Action(coreio.ActionMemoryRead).Run(context.Background(), NewOptions(
		Option{Key: "pa" + "th", Value: actionMissingPath},
	))
	AssertFalse(t, result.OK)
}

func TestActions_MemoryRoundTripUgly(t *T) {
	c := New()
	coreio.RegisterActions(c)
	coreio.ResetMemoryActionStore()

	// ResetMemoryActionStore clears previous state between actions.
	writeResult := c.Action(coreio.ActionMemoryWrite).Run(context.Background(), NewOptions(
		Option{Key: "pa" + "th", Value: "tmp.txt"},
		Option{Key: "content", Value: "payload"},
	))
	RequireTrue(t, writeResult.OK)

	coreio.ResetMemoryActionStore()

	readResult := c.Action(coreio.ActionMemoryRead).Run(context.Background(), NewOptions(
		Option{Key: "pa" + "th", Value: "tmp.txt"},
	))
	AssertFalse(t, readResult.OK)
}

func TestActions_CopyGood(t *T) {
	c := New()
	coreio.RegisterActions(c)

	source := coreio.NewMemoryMedium()
	destination := coreio.NewMemoryMedium()
	RequireNoError(t, source.Write(actionInputPath, "payload"))

	result := c.Action(coreio.ActionCopy).Run(context.Background(), NewOptions(
		Option{Key: "source", Value: coreio.Medium(source)},
		Option{Key: "sourcePath", Value: actionInputPath},
		Option{Key: "destination", Value: coreio.Medium(destination)},
		Option{Key: "destinationPath", Value: actionBackupInputPath},
	))
	RequireTrue(t, result.OK)

	content, err := destination.Read(actionBackupInputPath)
	RequireNoError(t, err)
	AssertEqual(t, "payload", content)
}

func TestActions_CopyBad(t *T) {
	c := New()
	coreio.RegisterActions(c)

	// Missing source medium must fail.
	result := c.Action(coreio.ActionCopy).Run(context.Background(), NewOptions(
		Option{Key: "sourcePath", Value: actionInputPath},
		Option{Key: "destination", Value: coreio.Medium(coreio.NewMemoryMedium())},
		Option{Key: "destinationPath", Value: actionBackupInputPath},
	))
	AssertFalse(t, result.OK)
}

func TestActions_CopyUgly(t *T) {
	c := New()
	coreio.RegisterActions(c)

	source := coreio.NewMemoryMedium()
	// Source file does not exist — copy must surface the read error.
	result := c.Action(coreio.ActionCopy).Run(context.Background(), NewOptions(
		Option{Key: "source", Value: coreio.Medium(source)},
		Option{Key: "sourcePath", Value: actionMissingPath},
		Option{Key: "destination", Value: coreio.Medium(coreio.NewMemoryMedium())},
		Option{Key: "destinationPath", Value: "dest.txt"},
	))
	AssertFalse(t, result.OK)
}

func TestActions_S3ReadWriteGood(t *T) {
	c := New()
	coreio.RegisterActions(c)
	medium := newActionS3Medium(t)

	writeResult := c.Action(coreio.ActionS3Write).Run(context.Background(), NewOptions(
		Option{Key: "medium", Value: medium},
		Option{Key: "pa" + "th", Value: "reports/daily.txt"},
		Option{Key: "content", Value: "done"},
	))
	RequireTrue(t, writeResult.OK)

	readResult := c.Action(coreio.ActionS3Read).Run(context.Background(), NewOptions(
		Option{Key: "medium", Value: medium},
		Option{Key: "pa" + "th", Value: "reports/daily.txt"},
	))
	RequireTrue(t, readResult.OK)
	AssertEqual(t, "done", readResult.Value)
}

func TestActions_S3ReadWriteUgly(t *T) {
	c := New()
	coreio.RegisterActions(c)
	medium := newActionS3Medium(t)

	readResult := c.Action(coreio.ActionS3Read).Run(context.Background(), NewOptions(
		Option{Key: "medium", Value: medium},
		Option{Key: "pa" + "th", Value: actionMissingPath},
	))
	AssertFalse(t, readResult.OK)

	writeResult := c.Action(coreio.ActionS3Write).Run(context.Background(), NewOptions(
		Option{Key: "medium", Value: medium},
		Option{Key: "pa" + "th", Value: ""},
		Option{Key: "content", Value: "payload"},
	))
	AssertFalse(t, writeResult.OK)
}

func TestActions_SFTPReadWriteGood(t *T) {
	c := New()
	coreio.RegisterActions(c)
	medium := newActionSFTPTestMedium(t)

	writeResult := c.Action(coreio.ActionSFTPWrite).Run(context.Background(), NewOptions(
		Option{Key: "medium", Value: medium},
		Option{Key: "pa" + "th", Value: "notes/todo.txt"},
		Option{Key: "content", Value: "ship sftp"},
	))
	RequireTrue(t, writeResult.OK)

	readResult := c.Action(coreio.ActionSFTPRead).Run(context.Background(), NewOptions(
		Option{Key: "medium", Value: medium},
		Option{Key: "pa" + "th", Value: "notes/todo.txt"},
	))
	RequireTrue(t, readResult.OK)
	AssertEqual(t, "ship sftp", readResult.Value)
}

func TestActions_SFTPReadWriteUgly(t *T) {
	c := New()
	coreio.RegisterActions(c)
	medium := newActionSFTPTestMedium(t)

	readResult := c.Action(coreio.ActionSFTPRead).Run(context.Background(), NewOptions(
		Option{Key: "medium", Value: medium},
		Option{Key: "pa" + "th", Value: actionMissingPath},
	))
	AssertFalse(t, readResult.OK)

	writeResult := c.Action(coreio.ActionSFTPWrite).Run(context.Background(), NewOptions(
		Option{Key: "medium", Value: medium},
		Option{Key: "pa" + "th", Value: ""},
		Option{Key: "content", Value: "payload"},
	))
	AssertFalse(t, writeResult.OK)
}

func TestActions_CubeReadWritePackUnpackGood(t *T) {
	tempDir := t.TempDir()
	c := New()
	coreio.RegisterActions(c)
	cube.RegisterActions(c)

	inner := coreio.NewMemoryMedium()
	cubeMedium, err := cube.New(cube.Options{Inner: inner, Key: actionTestCubeKey})
	RequireNoError(t, err)

	writeResult := c.Action(coreio.ActionCubeWrite).Run(context.Background(), NewOptions(
		Option{Key: "medium", Value: coreio.Medium(cubeMedium)},
		Option{Key: "pa" + "th", Value: actionSecretPath},
		Option{Key: "content", Value: "classified"},
	))
	RequireTrue(t, writeResult.OK)

	readResult := c.Action(coreio.ActionCubeRead).Run(context.Background(), NewOptions(
		Option{Key: "medium", Value: coreio.Medium(cubeMedium)},
		Option{Key: "pa" + "th", Value: actionSecretPath},
	))
	RequireTrue(t, readResult.OK)
	AssertEqual(t, "classified", readResult.Value)

	innerContract := coreio.NewMemoryMedium()
	contractWrite := c.Action(coreio.ActionCubeWrite).Run(context.Background(), NewOptions(
		Option{Key: "inner", Value: coreio.Medium(innerContract)},
		Option{Key: "key", Value: actionTestCubeKey},
		Option{Key: "pa" + "th", Value: "inner.txt"},
		Option{Key: "content", Value: "via inner"},
	))
	RequireTrue(t, contractWrite.OK)

	contractRead := c.Action(coreio.ActionCubeRead).Run(context.Background(), NewOptions(
		Option{Key: "inner", Value: coreio.Medium(innerContract)},
		Option{Key: "key", Value: actionTestCubeKey},
		Option{Key: "pa" + "th", Value: "inner.txt"},
	))
	RequireTrue(t, contractRead.OK)
	AssertEqual(t, "via inner", contractRead.Value)

	source := coreio.NewMemoryMedium()
	RequireNoError(t, source.Write(actionConfigPath, actionConfigContent))
	outputPath := tempDir + "/app.cube"
	packResult := c.Action(coreio.ActionCubePack).Run(context.Background(), NewOptions(
		Option{Key: "source", Value: coreio.Medium(source)},
		Option{Key: "output", Value: outputPath},
		Option{Key: "key", Value: actionTestCubeKey},
	))
	RequireTrue(t, packResult.OK)

	destination := coreio.NewMemoryMedium()
	unpackResult := c.Action(coreio.ActionCubeUnpack).Run(context.Background(), NewOptions(
		Option{Key: "cube", Value: outputPath},
		Option{Key: "destination", Value: coreio.Medium(destination)},
		Option{Key: "key", Value: actionTestCubeKey},
	))
	RequireTrue(t, unpackResult.OK)

	content, err := destination.Read(actionConfigPath)
	RequireNoError(t, err)
	AssertEqual(t, actionConfigContent, content)
}

func TestActions_CubeReadWritePackUnpackUgly(t *T) {
	c := New()
	coreio.RegisterActions(c)
	cube.RegisterActions(c)

	readResult := c.Action(coreio.ActionCubeRead).Run(context.Background(), NewOptions(
		Option{Key: "inner", Value: coreio.Medium(coreio.NewMemoryMedium())},
		Option{Key: "key", Value: actionTestCubeKey},
		Option{Key: "pa" + "th", Value: actionMissingPath},
	))
	AssertFalse(t, readResult.OK)

	writeResult := c.Action(coreio.ActionCubeWrite).Run(context.Background(), NewOptions(
		Option{Key: "inner", Value: coreio.Medium(coreio.NewMemoryMedium())},
		Option{Key: "key", Value: []byte("short")},
		Option{Key: "pa" + "th", Value: actionSecretPath},
		Option{Key: "content", Value: "payload"},
	))
	AssertFalse(t, writeResult.OK)

	packResult := c.Action(coreio.ActionCubePack).Run(context.Background(), NewOptions(
		Option{Key: "output", Value: t.TempDir() + "/app.cube"},
		Option{Key: "key", Value: actionTestCubeKey},
	))
	AssertFalse(t, packResult.OK)

	unpackResult := c.Action(coreio.ActionCubeUnpack).Run(context.Background(), NewOptions(
		Option{Key: "cube", Value: t.TempDir() + "/missing.cube"},
		Option{Key: "destination", Value: coreio.Medium(coreio.NewMemoryMedium())},
		Option{Key: "key", Value: actionTestCubeKey},
	))
	AssertFalse(t, unpackResult.OK)
}

func TestActions_GitHubPWAStubsBad(t *T) {
	c := New()
	coreio.RegisterActions(c)

	for _, name := range []string{coreio.ActionGitHubClone, coreio.ActionGitHubRead, coreio.ActionPWAScrape} {
		result := c.Action(name).Run(context.Background(), NewOptions())
		AssertFalse(t, result.OK, name)
		err, ok := result.Value.(error)
		RequireTrue(t, ok, name)
		AssertContains(t, err.Error(), "not implemented", name)
		AssertContains(t, err.Error(), "#633", name)
	}
}

type actionTestS3Client struct {
	mu      sync.RWMutex
	objects map[string][]byte
}

func newActionS3Medium(t *T) *ios3.Medium {
	t.Helper()
	medium, err := ios3.New(ios3.Options{
		Bucket: "bucket",
		Client: &actionTestS3Client{objects: make(map[string][]byte)},
	})
	RequireNoError(t, err)
	return medium
}

func (client *actionTestS3Client) GetObject(_ context.Context, params *awss3.GetObjectInput, _ ...func(*awss3.Options)) (*awss3.GetObjectOutput, error) {
	client.mu.RLock()
	defer client.mu.RUnlock()

	key := aws.ToString(params.Key)
	data, ok := client.objects[key]
	if !ok {
		return nil, E("actionsTest.s3.GetObject", "key not found", fs.ErrNotExist)
	}
	return &awss3.GetObjectOutput{
		Body:          goio.NopCloser(NewReader(string(data))),
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
		return nil, E("actionsTest.s3.HeadObject", "key not found", fs.ErrNotExist)
	}
	return &awss3.HeadObjectOutput{ContentLength: aws.Int64(int64(len(data)))}, nil
}

func (client *actionTestS3Client) ListObjectsV2(context.Context, *awss3.ListObjectsV2Input, ...func(*awss3.Options)) (*awss3.ListObjectsV2Output, error) {
	return &awss3.ListObjectsV2Output{}, nil
}

func (client *actionTestS3Client) CopyObject(_ context.Context, params *awss3.CopyObjectInput, _ ...func(*awss3.Options)) (*awss3.CopyObjectOutput, error) {
	client.mu.Lock()
	defer client.mu.Unlock()

	parts := SplitN(aws.ToString(params.CopySource), "/", 2)
	if len(parts) != 2 {
		return nil, E("actionsTest.s3.CopyObject", "invalid copy source", fs.ErrInvalid)
	}
	sourceKey := parts[1]
	data, ok := client.objects[sourceKey]
	if !ok {
		return nil, E("actionsTest.s3.CopyObject", "source not found", fs.ErrNotExist)
	}
	client.objects[aws.ToString(params.Key)] = append([]byte(nil), data...)
	return &awss3.CopyObjectOutput{}, nil
}

func newActionSFTPTestMedium(t *T) *iosftp.Medium {
	t.Helper()

	serverConn, clientConn := net.Pipe()
	server := pkgsftp.NewRequestServer(serverConn, pkgsftp.InMemHandler())
	done := make(chan error, 1)
	go func() {
		done <- server.Serve()
	}()

	client, err := pkgsftp.NewClientPipe(clientConn, clientConn)
	RequireNoError(t, err)

	medium, err := iosftp.New(iosftp.Options{Client: client})
	RequireNoError(t, err)

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

func TestActions_ResetMemoryActionStore_Good(t *T) {
	c := New()
	coreio.RegisterActions(c)
	coreio.ResetMemoryActionStore()

	writeResult := c.Action(coreio.ActionMemoryWrite).Run(context.Background(), NewOptions(
		Option{Key: "pa" + "th", Value: "file.txt"},
		Option{Key: "content", Value: "payload"},
	))
	RequireTrue(t, writeResult.OK)

	coreio.ResetMemoryActionStore()
	readResult := c.Action(coreio.ActionMemoryRead).Run(context.Background(), NewOptions(
		Option{Key: "pa" + "th", Value: "file.txt"},
	))
	AssertFalse(t, readResult.OK)
}

func TestActions_ResetMemoryActionStore_Bad(t *T) {
	c := New()
	coreio.RegisterActions(c)
	coreio.ResetMemoryActionStore()
	coreio.ResetMemoryActionStore()

	readResult := c.Action(coreio.ActionMemoryRead).Run(context.Background(), NewOptions(
		Option{Key: "pa" + "th", Value: actionMissingPath},
	))
	AssertFalse(t, readResult.OK)
}

func TestActions_ResetMemoryActionStore_Ugly(t *T) {
	c := New()
	coreio.RegisterActions(c)
	coreio.ResetMemoryActionStore()

	writeResult := c.Action(coreio.ActionMemoryWrite).Run(context.Background(), NewOptions(
		Option{Key: "pa" + "th", Value: "old.txt"},
		Option{Key: "content", Value: "payload"},
	))
	RequireTrue(t, writeResult.OK)

	readBeforeReset := c.Action(coreio.ActionMemoryRead).Run(context.Background(), NewOptions(
		Option{Key: "pa" + "th", Value: "old.txt"},
	))
	RequireTrue(t, readBeforeReset.OK)

	coreio.ResetMemoryActionStore()
	readAfterReset := c.Action(coreio.ActionMemoryRead).Run(context.Background(), NewOptions(
		Option{Key: "pa" + "th", Value: "old.txt"},
	))
	AssertFalse(t, readAfterReset.OK)
}
