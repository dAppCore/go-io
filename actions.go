// SPDX-License-Identifier: EUPL-1.2

// Example: io.RegisterActions(c)
// Example: result := c.Action("core.io.local.read").Run(ctx, core.NewOptions(
// Example:     core.Option{Key: "root", Value: "/srv/app"},
// Example:     core.Option{Key: "path", Value: "config/app.yaml"},
// Example: ))
package io

import (
	"archive/tar"
	"bytes"
	"context"
	goio "io"
	"io/fs"
	"time"

	core "dappco.re/go/core"
	"dappco.re/go/io/local"
	"dappco.re/go/io/sigil"
)

// Named action identifiers used by Core consumers. Each maps to a Medium
// operation with a predictable path name.
//
// Example: result := c.Action(io.ActionLocalRead).Run(ctx, opts)
const (
	ActionLocalRead   = "core.io.local.read"
	ActionLocalWrite  = "core.io.local.write"
	ActionLocalList   = "core.io.local.list"
	ActionLocalDelete = "core.io.local.delete"

	ActionMemoryRead  = "core.io.memory.read"
	ActionMemoryWrite = "core.io.memory.write"

	ActionGitHubClone = "core.io.github.clone"
	ActionGitHubRead  = "core.io.github.read"

	ActionPWAScrape = "core.io.pwa.scrape"

	ActionSFTPRead  = "core.io.sftp.read"
	ActionSFTPWrite = "core.io.sftp.write"

	ActionS3Read  = "core.io.s3.read"
	ActionS3Write = "core.io.s3.write"

	ActionCubeRead   = "core.io.cube.read"
	ActionCubeWrite  = "core.io.cube.write"
	ActionCubePack   = "core.io.cube.pack"
	ActionCubeUnpack = "core.io.cube.unpack"

	ActionCopy = "core.io.copy"
)

// memoryActionStore is the shared in-memory backing for
// core.io.memory.read/core.io.memory.write. Keeping it package-level lets the
// two actions agree on state without the caller supplying a backend.
var memoryActionStore = NewMemoryMedium()

// Example: io.RegisterActions(c)
//
// RegisterActions installs the named actions listed in the go-io RFC §15 on
// the given Core. Consumers call this at service registration time so that any
// agent or CLI can dispatch Medium operations by name.
func RegisterActions(c *core.Core) {
	if c == nil {
		return
	}
	c.Action(ActionLocalRead, localReadAction)
	c.Action(ActionLocalWrite, localWriteAction)
	c.Action(ActionLocalList, localListAction)
	c.Action(ActionLocalDelete, localDeleteAction)
	c.Action(ActionMemoryRead, memoryReadAction)
	c.Action(ActionMemoryWrite, memoryWriteAction)
	c.Action(ActionGitHubClone, githubNotImplementedAction)
	c.Action(ActionGitHubRead, githubNotImplementedAction)
	c.Action(ActionPWAScrape, pwaNotImplementedAction)
	c.Action(ActionSFTPRead, mediumReadAction("io.sftp.readAction"))
	c.Action(ActionSFTPWrite, mediumWriteAction("io.sftp.writeAction"))
	c.Action(ActionS3Read, mediumReadAction("io.s3.readAction"))
	c.Action(ActionS3Write, mediumWriteAction("io.s3.writeAction"))
	c.Action(ActionCubeRead, cubeReadAction)
	c.Action(ActionCubeWrite, cubeWriteAction)
	c.Action(ActionCubePack, cubePackAction)
	c.Action(ActionCubeUnpack, cubeUnpackAction)
	c.Action(ActionCopy, copyAction)
}

// Example: opts := core.NewOptions(core.Option{Key: "root", Value: "/srv/app"}, core.Option{Key: "path", Value: "config/app.yaml"})
func localReadAction(_ context.Context, opts core.Options) core.Result {
	medium, err := localMediumFromOptions(opts)
	if err != nil {
		return core.Result{}.New(err)
	}
	content, err := medium.Read(opts.String("path"))
	if err != nil {
		return core.Result{}.New(err)
	}
	return core.Result{Value: content, OK: true}
}

// Example: opts := core.NewOptions(core.Option{Key: "root", Value: "/srv/app"}, core.Option{Key: "path", Value: "log.txt"}, core.Option{Key: "content", Value: "event"})
func localWriteAction(_ context.Context, opts core.Options) core.Result {
	medium, err := localMediumFromOptions(opts)
	if err != nil {
		return core.Result{}.New(err)
	}
	if err := medium.Write(opts.String("path"), opts.String("content")); err != nil {
		return core.Result{}.New(err)
	}
	return core.Result{OK: true}
}

// Example: opts := core.NewOptions(core.Option{Key: "root", Value: "/srv/app"}, core.Option{Key: "path", Value: "config"})
func localListAction(_ context.Context, opts core.Options) core.Result {
	medium, err := localMediumFromOptions(opts)
	if err != nil {
		return core.Result{}.New(err)
	}
	entries, err := medium.List(opts.String("path"))
	if err != nil {
		return core.Result{}.New(err)
	}
	return core.Result{Value: entries, OK: true}
}

// Example: opts := core.NewOptions(core.Option{Key: "root", Value: "/srv/app"}, core.Option{Key: "path", Value: "tmp/old.log"})
func localDeleteAction(_ context.Context, opts core.Options) core.Result {
	medium, err := localMediumFromOptions(opts)
	if err != nil {
		return core.Result{}.New(err)
	}
	path := opts.String("path")
	recursive := opts.Bool("recursive")
	if recursive {
		if err := medium.DeleteAll(path); err != nil {
			return core.Result{}.New(err)
		}
	} else {
		if err := medium.Delete(path); err != nil {
			return core.Result{}.New(err)
		}
	}
	return core.Result{OK: true}
}

// Example: opts := core.NewOptions(core.Option{Key: "path", Value: "config/app.yaml"})
func memoryReadAction(_ context.Context, opts core.Options) core.Result {
	content, err := memoryActionStore.Read(opts.String("path"))
	if err != nil {
		return core.Result{}.New(err)
	}
	return core.Result{Value: content, OK: true}
}

// Example: opts := core.NewOptions(core.Option{Key: "path", Value: "config/app.yaml"}, core.Option{Key: "content", Value: "port: 8080"})
func memoryWriteAction(_ context.Context, opts core.Options) core.Result {
	if err := memoryActionStore.Write(opts.String("path"), opts.String("content")); err != nil {
		return core.Result{}.New(err)
	}
	return core.Result{OK: true}
}

func githubNotImplementedAction(context.Context, core.Options) core.Result {
	return core.Result{
		OK:    false,
		Value: core.E("io.github", "not implemented — see #633 for backend tracking", nil),
	}
}

func pwaNotImplementedAction(context.Context, core.Options) core.Result {
	return core.Result{
		OK:    false,
		Value: core.E("io.pwa", "not implemented — see #633 for backend tracking", nil),
	}
}

func mediumReadAction(operation string) core.ActionHandler {
	return func(_ context.Context, opts core.Options) core.Result {
		medium, err := mediumFromOptions(opts, operation)
		if err != nil {
			return core.Result{}.New(err)
		}
		content, err := medium.Read(opts.String("path"))
		if err != nil {
			return core.Result{}.New(err)
		}
		return core.Result{Value: content, OK: true}
	}
}

func mediumWriteAction(operation string) core.ActionHandler {
	return func(_ context.Context, opts core.Options) core.Result {
		medium, err := mediumFromOptions(opts, operation)
		if err != nil {
			return core.Result{}.New(err)
		}
		if err := medium.Write(opts.String("path"), opts.String("content")); err != nil {
			return core.Result{}.New(err)
		}
		return core.Result{OK: true}
	}
}

func cubeReadAction(_ context.Context, opts core.Options) core.Result {
	if medium, ok := opts.Get("medium").Value.(Medium); ok {
		content, err := medium.Read(opts.String("path"))
		if err != nil {
			return core.Result{}.New(err)
		}
		return core.Result{Value: content, OK: true}
	}

	inner, err := innerMediumFromOptions(opts, "io.cube.readAction")
	if err != nil {
		return core.Result{}.New(err)
	}
	cipherSigil, err := cubeSigilFromOptions(opts, "io.cube.readAction")
	if err != nil {
		return core.Result{}.New(err)
	}
	ciphertext, err := inner.Read(opts.String("path"))
	if err != nil {
		return core.Result{}.New(err)
	}
	plaintext, err := sigil.Untransmute([]byte(ciphertext), []sigil.Sigil{cipherSigil})
	if err != nil {
		return core.Result{}.New(core.E("io.cube.readAction", "failed to decrypt", err))
	}
	return core.Result{Value: string(plaintext), OK: true}
}

func cubeWriteAction(_ context.Context, opts core.Options) core.Result {
	if medium, ok := opts.Get("medium").Value.(Medium); ok {
		if err := medium.Write(opts.String("path"), opts.String("content")); err != nil {
			return core.Result{}.New(err)
		}
		return core.Result{OK: true}
	}

	inner, err := innerMediumFromOptions(opts, "io.cube.writeAction")
	if err != nil {
		return core.Result{}.New(err)
	}
	cipherSigil, err := cubeSigilFromOptions(opts, "io.cube.writeAction")
	if err != nil {
		return core.Result{}.New(err)
	}
	ciphertext, err := sigil.Transmute([]byte(opts.String("content")), []sigil.Sigil{cipherSigil})
	if err != nil {
		return core.Result{}.New(core.E("io.cube.writeAction", "failed to encrypt", err))
	}
	if err := inner.Write(opts.String("path"), string(ciphertext)); err != nil {
		return core.Result{}.New(err)
	}
	return core.Result{OK: true}
}

func cubePackAction(_ context.Context, opts core.Options) core.Result {
	source, ok := opts.Get("source").Value.(Medium)
	if !ok {
		return core.Result{}.New(core.E("io.cube.packAction", "source medium is required", fs.ErrInvalid))
	}
	key, err := keyFromOptions(opts, "io.cube.packAction")
	if err != nil {
		return core.Result{}.New(err)
	}
	if err := packCubeArchive(opts.String("output"), source, key); err != nil {
		return core.Result{}.New(err)
	}
	return core.Result{OK: true}
}

func cubeUnpackAction(_ context.Context, opts core.Options) core.Result {
	destination, ok := opts.Get("destination").Value.(Medium)
	if !ok {
		return core.Result{}.New(core.E("io.cube.unpackAction", "destination medium is required", fs.ErrInvalid))
	}
	key, err := keyFromOptions(opts, "io.cube.unpackAction")
	if err != nil {
		return core.Result{}.New(err)
	}
	if err := unpackCubeArchive(opts.String("cube"), destination, key); err != nil {
		return core.Result{}.New(err)
	}
	return core.Result{OK: true}
}

// Example: opts := core.NewOptions(
// Example:     core.Option{Key: "source", Value: sourceMedium},
// Example:     core.Option{Key: "sourcePath", Value: "input.txt"},
// Example:     core.Option{Key: "destination", Value: destinationMedium},
// Example:     core.Option{Key: "destinationPath", Value: "backup/input.txt"},
// Example: )
func copyAction(_ context.Context, opts core.Options) core.Result {
	source, ok := opts.Get("source").Value.(Medium)
	if !ok {
		return core.Result{}.New(core.E("io.copyAction", "source medium is required", fs.ErrInvalid))
	}
	destination, ok := opts.Get("destination").Value.(Medium)
	if !ok {
		return core.Result{}.New(core.E("io.copyAction", "destination medium is required", fs.ErrInvalid))
	}
	if err := Copy(source, opts.String("sourcePath"), destination, opts.String("destinationPath")); err != nil {
		return core.Result{}.New(err)
	}
	return core.Result{OK: true}
}

// localMediumFromOptions constructs a sandboxed local Medium using the
// "root" option. An empty root defaults to "/" (unsandboxed).
func localMediumFromOptions(opts core.Options) (Medium, error) {
	root := opts.String("root")
	if root == "" {
		root = "/"
	}
	return local.New(root)
}

func mediumFromOptions(opts core.Options, operation string) (Medium, error) {
	medium, ok := opts.Get("medium").Value.(Medium)
	if !ok {
		return nil, core.E(operation, "medium is required", fs.ErrInvalid)
	}
	return medium, nil
}

func innerMediumFromOptions(opts core.Options, operation string) (Medium, error) {
	inner, ok := opts.Get("inner").Value.(Medium)
	if !ok {
		return nil, core.E(operation, "inner medium is required", fs.ErrInvalid)
	}
	return inner, nil
}

func cubeSigilFromOptions(opts core.Options, operation string) (*sigil.ChaChaPolySigil, error) {
	key, err := keyFromOptions(opts, operation)
	if err != nil {
		return nil, err
	}
	cipherSigil, err := sigil.NewChaChaPolySigil(key, nil)
	if err != nil {
		return nil, core.E(operation, "failed to create cipher", err)
	}
	return cipherSigil, nil
}

func keyFromOptions(opts core.Options, operation string) ([]byte, error) {
	value := opts.Get("key").Value
	switch typed := value.(type) {
	case []byte:
		return typed, nil
	case string:
		return []byte(typed), nil
	case []int:
		key := make([]byte, len(typed))
		for i, item := range typed {
			if item < 0 || item > 255 {
				return nil, core.E(operation, "key byte out of range", fs.ErrInvalid)
			}
			key[i] = byte(item)
		}
		return key, nil
	case []any:
		key := make([]byte, len(typed))
		for i, item := range typed {
			number, ok := item.(int)
			if !ok || number < 0 || number > 255 {
				return nil, core.E(operation, "key must be []byte or string", fs.ErrInvalid)
			}
			key[i] = byte(number)
		}
		return key, nil
	default:
		return nil, core.E(operation, "key must be []byte or string", fs.ErrInvalid)
	}
}

func packCubeArchive(outputPath string, source Medium, key []byte) error {
	if outputPath == "" {
		return core.E("io.cube.packAction", "output path is required", fs.ErrInvalid)
	}

	archiveBytes, err := archiveMediumToTar(source)
	if err != nil {
		return core.E("io.cube.packAction", "failed to build archive", err)
	}

	cipherSigil, err := sigil.NewChaChaPolySigil(key, nil)
	if err != nil {
		return core.E("io.cube.packAction", "failed to create cipher", err)
	}
	ciphertext, err := sigil.Transmute(archiveBytes, []sigil.Sigil{cipherSigil})
	if err != nil {
		return core.E("io.cube.packAction", "failed to encrypt archive", err)
	}

	localMedium, err := local.New("/")
	if err != nil {
		return core.E("io.cube.packAction", "failed to access local filesystem", err)
	}
	return localMedium.WriteMode(outputPath, string(ciphertext), 0600)
}

func unpackCubeArchive(cubePath string, destination Medium, key []byte) error {
	if cubePath == "" {
		return core.E("io.cube.unpackAction", "cube path is required", fs.ErrInvalid)
	}

	localMedium, err := local.New("/")
	if err != nil {
		return core.E("io.cube.unpackAction", "failed to access local filesystem", err)
	}
	ciphertext, err := localMedium.Read(cubePath)
	if err != nil {
		return core.E("io.cube.unpackAction", core.Concat("failed to read cube: ", cubePath), err)
	}

	cipherSigil, err := sigil.NewChaChaPolySigil(key, nil)
	if err != nil {
		return core.E("io.cube.unpackAction", "failed to create cipher", err)
	}
	archiveBytes, err := sigil.Untransmute([]byte(ciphertext), []sigil.Sigil{cipherSigil})
	if err != nil {
		return core.E("io.cube.unpackAction", "failed to decrypt archive", err)
	}

	return extractTarToMedium(archiveBytes, destination)
}

func archiveMediumToTar(source Medium) ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	tarWriter := tar.NewWriter(buffer)

	if err := walkAndArchive(source, "", tarWriter); err != nil {
		_ = tarWriter.Close()
		return nil, err
	}
	if err := tarWriter.Close(); err != nil {
		return nil, core.E("io.cube.archive", "failed to close tar writer", err)
	}
	return buffer.Bytes(), nil
}

func walkAndArchive(source Medium, archivePath string, tarWriter *tar.Writer) error {
	entries, err := source.List(archivePath)
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		childPath := entry.Name()
		if archivePath != "" {
			childPath = core.Concat(archivePath, "/", entry.Name())
		}
		if entry.IsDir() {
			if err := walkAndArchive(source, childPath, tarWriter); err != nil {
				return err
			}
			continue
		}

		content, err := source.Read(childPath)
		if err != nil {
			return core.E("io.cube.archive", core.Concat("failed to read: ", childPath), err)
		}
		modTime := time.Now()
		mode := fs.FileMode(0600)
		if info, err := source.Stat(childPath); err == nil {
			modTime = info.ModTime()
			mode = info.Mode()
		}
		header := &tar.Header{
			Name:    childPath,
			Mode:    int64(mode.Perm()),
			Size:    int64(len(content)),
			ModTime: modTime,
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return core.E("io.cube.archive", core.Concat("failed to write header: ", childPath), err)
		}
		if _, err := tarWriter.Write([]byte(content)); err != nil {
			return core.E("io.cube.archive", core.Concat("failed to write content: ", childPath), err)
		}
	}
	return nil
}

func extractTarToMedium(archiveBytes []byte, destination Medium) error {
	tarReader := tar.NewReader(bytes.NewReader(archiveBytes))
	for {
		header, err := tarReader.Next()
		if err == goio.EOF {
			break
		}
		if err != nil {
			return core.E("io.cube.extract", "failed to read tar entry", err)
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		content, err := goio.ReadAll(tarReader)
		if err != nil {
			return core.E("io.cube.extract", core.Concat("failed to read entry: ", header.Name), err)
		}
		name := core.TrimPrefix(header.Name, "/")
		if name == "" || core.HasSuffix(name, "/") {
			continue
		}
		mode := fs.FileMode(header.Mode)
		if mode == 0 {
			mode = 0644
		}
		if err := destination.WriteMode(name, string(content), mode); err != nil {
			return core.E("io.cube.extract", core.Concat("failed to write entry: ", name), err)
		}
	}
	return nil
}

// ResetMemoryActionStore clears the in-memory state used by memory action
// handlers. Tests call this to isolate runs from each other.
//
// Example: io.ResetMemoryActionStore()
func ResetMemoryActionStore() {
	memoryActionStore = NewMemoryMedium()
}
