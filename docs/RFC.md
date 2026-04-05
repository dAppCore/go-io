---
title: API Reference
description: Complete API reference for go-io.
---

# API Reference

This document enumerates every exported type, function, method, and variable in go-io, with short usage examples.

Examples use the import paths from `docs/index.md` (`dappco.re/go/core/io`). Adjust paths if your module path differs.

## Package io (`dappco.re/go/core/io`)

Defines the `Medium` interface, helper functions, and in-memory `MemoryMedium` implementation.

### Medium (interface)

The common storage abstraction implemented by every backend.

Example:
```go
var m io.Medium = io.NewMemoryMedium()
_ = m.Write("notes.txt", "hello")
```

**Read(path string) (string, error)**
Reads a file as a string.
Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("notes.txt", "hello")
value, _ := m.Read("notes.txt")
```

**Write(path, content string) error**
Writes content to a file, creating it if needed.
Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("notes.txt", "hello")
```

**WriteMode(path, content string, mode fs.FileMode) error**
Writes content with explicit permissions.
Example:
```go
m := io.NewMemoryMedium()
_ = m.WriteMode("secret.txt", "secret", 0600)
```

**EnsureDir(path string) error**
Ensures a directory exists.
Example:
```go
m := io.NewMemoryMedium()
_ = m.EnsureDir("config")
```

**IsFile(path string) bool**
Reports whether a path is a regular file.
Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("notes.txt", "hello")
ok := m.IsFile("notes.txt")
```

**Delete(path string) error**
Deletes a file or empty directory.
Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("old.txt", "data")
_ = m.Delete("old.txt")
```

**DeleteAll(path string) error**
Deletes a file or directory tree recursively.
Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("logs/run.txt", "started")
_ = m.DeleteAll("logs")
```

**Rename(oldPath, newPath string) error**
Moves or renames a file or directory.
Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("old.txt", "data")
_ = m.Rename("old.txt", "new.txt")
```

**List(path string) ([]fs.DirEntry, error)**
Lists immediate directory entries.
Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("dir/file.txt", "data")
entries, _ := m.List("dir")
```

**Stat(path string) (fs.FileInfo, error)**
Returns file metadata.
Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("notes.txt", "hello")
info, _ := m.Stat("notes.txt")
```

**Open(path string) (fs.File, error)**
Opens a file for reading.
Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("notes.txt", "hello")
f, _ := m.Open("notes.txt")
defer f.Close()
```

**Create(path string) (io.WriteCloser, error)**
Creates or truncates a file and returns a writer.
Example:
```go
m := io.NewMemoryMedium()
w, _ := m.Create("notes.txt")
_, _ = w.Write([]byte("hello"))
_ = w.Close()
```

**Append(path string) (io.WriteCloser, error)**
Opens a file for appending, creating it if needed.
Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("notes.txt", "hello")
w, _ := m.Append("notes.txt")
_, _ = w.Write([]byte(" world"))
_ = w.Close()
```

**ReadStream(path string) (io.ReadCloser, error)**
Opens a streaming reader for a file.
Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("notes.txt", "hello")
r, _ := m.ReadStream("notes.txt")
defer r.Close()
```

**WriteStream(path string) (io.WriteCloser, error)**
Opens a streaming writer for a file.
Example:
```go
m := io.NewMemoryMedium()
w, _ := m.WriteStream("notes.txt")
_, _ = w.Write([]byte("hello"))
_ = w.Close()
```

**Exists(path string) bool**
Reports whether a path exists.
Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("notes.txt", "hello")
ok := m.Exists("notes.txt")
```

**IsDir(path string) bool**
Reports whether a path is a directory.
Example:
```go
m := io.NewMemoryMedium()
_ = m.EnsureDir("config")
ok := m.IsDir("config")
```

### FileInfo

Lightweight `fs.FileInfo` implementation used by `MemoryMedium`.

**Name() string**
Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("file.txt", "data")
info, _ := m.Stat("file.txt")
_ = info.Name()
```

**Size() int64**
Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("file.txt", "data")
info, _ := m.Stat("file.txt")
_ = info.Size()
```

**Mode() fs.FileMode**
Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("file.txt", "data")
info, _ := m.Stat("file.txt")
_ = info.Mode()
```

**ModTime() time.Time**
Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("file.txt", "data")
info, _ := m.Stat("file.txt")
_ = info.ModTime()
```

**IsDir() bool**
Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("file.txt", "data")
info, _ := m.Stat("file.txt")
_ = info.IsDir()
```

**Sys() any**
Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("file.txt", "data")
info, _ := m.Stat("file.txt")
_ = info.Sys()
```

### DirEntry

Lightweight `fs.DirEntry` implementation used by `MemoryMedium` listings.

**Name() string**
Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("dir/file.txt", "data")
entries, _ := m.List("dir")
_ = entries[0].Name()
```

**IsDir() bool**
Example:
```go
m := io.NewMemoryMedium()
_ = m.EnsureDir("dir")
entries, _ := m.List("")
_ = entries[0].IsDir()
```

**Type() fs.FileMode**
Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("dir/file.txt", "data")
entries, _ := m.List("dir")
_ = entries[0].Type()
```

**Info() (fs.FileInfo, error)**
Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("dir/file.txt", "data")
entries, _ := m.List("dir")
info, _ := entries[0].Info()
_ = info.Name()
```

### Local

Pre-initialised local filesystem medium rooted at `/`.

Example:
```go
content, _ := io.Local.Read("/etc/hostname")
```

### NewSandboxed(root string) (Medium, error)

Creates a local filesystem medium sandboxed to `root`.

Example:
```go
m, _ := io.NewSandboxed("/srv/app")
_ = m.Write("config/app.yaml", "port: 8080")
```

### Read(m Medium, path string) (string, error)

Helper that calls `Medium.Read` on a supplied backend.

Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("notes.txt", "hello")
value, _ := io.Read(m, "notes.txt")
```

### Write(m Medium, path, content string) error

Helper that calls `Medium.Write` on a supplied backend.

Example:
```go
m := io.NewMemoryMedium()
_ = io.Write(m, "notes.txt", "hello")
```

### ReadStream(m Medium, path string) (io.ReadCloser, error)

Helper that calls `Medium.ReadStream` on a supplied backend.

Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("notes.txt", "hello")
r, _ := io.ReadStream(m, "notes.txt")
defer r.Close()
```

### WriteStream(m Medium, path string) (io.WriteCloser, error)

Helper that calls `Medium.WriteStream` on a supplied backend.

Example:
```go
m := io.NewMemoryMedium()
w, _ := io.WriteStream(m, "notes.txt")
_, _ = w.Write([]byte("hello"))
_ = w.Close()
```

### EnsureDir(m Medium, path string) error

Helper that calls `Medium.EnsureDir` on a supplied backend.

Example:
```go
m := io.NewMemoryMedium()
_ = io.EnsureDir(m, "config")
```

### IsFile(m Medium, path string) bool

Helper that calls `Medium.IsFile` on a supplied backend.

Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("notes.txt", "hello")
ok := io.IsFile(m, "notes.txt")
```

### Copy(src Medium, srcPath string, dst Medium, dstPath string) error

Copies a file between two mediums.

Example:
```go
src := io.NewMemoryMedium()
dst := io.NewMemoryMedium()
_ = src.Write("source.txt", "data")
_ = io.Copy(src, "source.txt", dst, "dest.txt")
```

### MemoryMedium

In-memory `Medium` implementation for tests.

Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("seed.txt", "seeded")
```

**Read(path string) (string, error)**
Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("notes.txt", "hello")
value, _ := m.Read("notes.txt")
```

**Write(path, content string) error**
Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("notes.txt", "hello")
```

**WriteMode(path, content string, mode fs.FileMode) error**
Example:
```go
m := io.NewMemoryMedium()
_ = m.WriteMode("secret.txt", "secret", 0600)
```

**EnsureDir(path string) error**
Example:
```go
m := io.NewMemoryMedium()
_ = m.EnsureDir("config")
```

**IsFile(path string) bool**
Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("notes.txt", "hello")
ok := m.IsFile("notes.txt")
```

**Delete(path string) error**
Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("old.txt", "data")
_ = m.Delete("old.txt")
```

**DeleteAll(path string) error**
Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("logs/run.txt", "started")
_ = m.DeleteAll("logs")
```

**Rename(oldPath, newPath string) error**
Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("old.txt", "data")
_ = m.Rename("old.txt", "new.txt")
```

**List(path string) ([]fs.DirEntry, error)**
Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("dir/file.txt", "data")
entries, _ := m.List("dir")
```

**Stat(path string) (fs.FileInfo, error)**
Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("notes.txt", "hello")
info, _ := m.Stat("notes.txt")
```

**Open(path string) (fs.File, error)**
Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("notes.txt", "hello")
f, _ := m.Open("notes.txt")
defer f.Close()
```

**Create(path string) (io.WriteCloser, error)**
Example:
```go
m := io.NewMemoryMedium()
w, _ := m.Create("notes.txt")
_, _ = w.Write([]byte("hello"))
_ = w.Close()
```

**Append(path string) (io.WriteCloser, error)**
Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("notes.txt", "hello")
w, _ := m.Append("notes.txt")
_, _ = w.Write([]byte(" world"))
_ = w.Close()
```

**ReadStream(path string) (io.ReadCloser, error)**
Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("notes.txt", "hello")
r, _ := m.ReadStream("notes.txt")
defer r.Close()
```

**WriteStream(path string) (io.WriteCloser, error)**
Example:
```go
m := io.NewMemoryMedium()
w, _ := m.WriteStream("notes.txt")
_, _ = w.Write([]byte("hello"))
_ = w.Close()
```

**Exists(path string) bool**
Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("notes.txt", "hello")
ok := m.Exists("notes.txt")
```

**IsDir(path string) bool**
Example:
```go
m := io.NewMemoryMedium()
_ = m.EnsureDir("config")
ok := m.IsDir("config")
```

### NewMemoryMedium() *MemoryMedium

Creates a new empty in-memory medium.

Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("notes.txt", "hello")
```

### MemoryFile

`fs.File` implementation returned by `MemoryMedium.Open`.

**Stat() (fs.FileInfo, error)**
Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("notes.txt", "hello")
f, _ := m.Open("notes.txt")
info, _ := f.Stat()
_ = info.Name()
```

**Read(b []byte) (int, error)**
Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("notes.txt", "hello")
f, _ := m.Open("notes.txt")
buf := make([]byte, 5)
_, _ = f.Read(buf)
```

**Close() error**
Example:
```go
m := io.NewMemoryMedium()
_ = m.Write("notes.txt", "hello")
f, _ := m.Open("notes.txt")
_ = f.Close()
```

### MemoryWriteCloser

`io.WriteCloser` implementation returned by `MemoryMedium.Create` and `MemoryMedium.Append`.

**Write(p []byte) (int, error)**
Example:
```go
m := io.NewMemoryMedium()
w, _ := m.Create("notes.txt")
_, _ = w.Write([]byte("hello"))
```

**Close() error**
Example:
```go
m := io.NewMemoryMedium()
w, _ := m.Create("notes.txt")
_, _ = w.Write([]byte("hello"))
_ = w.Close()
```

## Package local (`dappco.re/go/core/io/local`)

Local filesystem backend with sandboxed roots and symlink-escape protection.

### New(root string) (*Medium, error)

Creates a new local filesystem medium rooted at `root`.

Example:
```go
m, _ := local.New("/srv/app")
_ = m.Write("config/app.yaml", "port: 8080")
```

### Medium

Local filesystem implementation of `io.Medium`.

Example:
```go
m, _ := local.New("/srv/app")
_ = m.EnsureDir("config")
```

**Read(path string) (string, error)**
Example:
```go
m, _ := local.New("/srv/app")
_ = m.Write("notes.txt", "hello")
value, _ := m.Read("notes.txt")
```

**Write(path, content string) error**
Example:
```go
m, _ := local.New("/srv/app")
_ = m.Write("notes.txt", "hello")
```

**WriteMode(path, content string, mode fs.FileMode) error**
Example:
```go
m, _ := local.New("/srv/app")
_ = m.WriteMode("secret.txt", "secret", 0600)
```

**EnsureDir(path string) error**
Example:
```go
m, _ := local.New("/srv/app")
_ = m.EnsureDir("config")
```

**IsDir(path string) bool**
Example:
```go
m, _ := local.New("/srv/app")
_ = m.EnsureDir("config")
ok := m.IsDir("config")
```

**IsFile(path string) bool**
Example:
```go
m, _ := local.New("/srv/app")
_ = m.Write("notes.txt", "hello")
ok := m.IsFile("notes.txt")
```

**Exists(path string) bool**
Example:
```go
m, _ := local.New("/srv/app")
_ = m.Write("notes.txt", "hello")
ok := m.Exists("notes.txt")
```

**List(path string) ([]fs.DirEntry, error)**
Example:
```go
m, _ := local.New("/srv/app")
_ = m.Write("dir/file.txt", "data")
entries, _ := m.List("dir")
```

**Stat(path string) (fs.FileInfo, error)**
Example:
```go
m, _ := local.New("/srv/app")
_ = m.Write("notes.txt", "hello")
info, _ := m.Stat("notes.txt")
```

**Open(path string) (fs.File, error)**
Example:
```go
m, _ := local.New("/srv/app")
_ = m.Write("notes.txt", "hello")
f, _ := m.Open("notes.txt")
defer f.Close()
```

**Create(path string) (io.WriteCloser, error)**
Example:
```go
m, _ := local.New("/srv/app")
w, _ := m.Create("notes.txt")
_, _ = w.Write([]byte("hello"))
_ = w.Close()
```

**Append(path string) (io.WriteCloser, error)**
Example:
```go
m, _ := local.New("/srv/app")
_ = m.Write("notes.txt", "hello")
w, _ := m.Append("notes.txt")
_, _ = w.Write([]byte(" world"))
_ = w.Close()
```

**ReadStream(path string) (io.ReadCloser, error)**
Example:
```go
m, _ := local.New("/srv/app")
_ = m.Write("notes.txt", "hello")
r, _ := m.ReadStream("notes.txt")
defer r.Close()
```

**WriteStream(path string) (io.WriteCloser, error)**
Example:
```go
m, _ := local.New("/srv/app")
w, _ := m.WriteStream("notes.txt")
_, _ = w.Write([]byte("hello"))
_ = w.Close()
```

**Delete(path string) error**
Example:
```go
m, _ := local.New("/srv/app")
_ = m.Write("old.txt", "data")
_ = m.Delete("old.txt")
```

**DeleteAll(path string) error**
Example:
```go
m, _ := local.New("/srv/app")
_ = m.Write("logs/run.txt", "started")
_ = m.DeleteAll("logs")
```

**Rename(oldPath, newPath string) error**
Example:
```go
m, _ := local.New("/srv/app")
_ = m.Write("old.txt", "data")
_ = m.Rename("old.txt", "new.txt")
```

## Package node (`dappco.re/go/core/io/node`)

In-memory filesystem implementing `io.Medium` and `fs.FS`, with tar serialisation.

### New() *Node

Creates a new empty in-memory filesystem.

Example:
```go
n := node.New()
```

### FromTar(data []byte) (*Node, error)

Creates a new `Node` by loading a tar archive.

Example:
```go
tarball := []byte{}
n, _ := node.FromTar(tarball)
```

### WalkOptions

Options for `Node.Walk`.

Example:
```go
options := node.WalkOptions{MaxDepth: 1, SkipErrors: true}
_ = options.MaxDepth
```

### Node

In-memory filesystem with implicit directories and tar support.

Example:
```go
n := node.New()
n.AddData("config/app.yaml", []byte("port: 8080"))
```

**AddData(name string, content []byte)**
Stages content in the in-memory filesystem.
Example:
```go
n := node.New()
n.AddData("config/app.yaml", []byte("port: 8080"))
```

**ToTar() ([]byte, error)**
Serialises the tree to a tar archive.
Example:
```go
n := node.New()
_ = n.Write("a.txt", "alpha")
blob, _ := n.ToTar()
```

**LoadTar(data []byte) error**
Replaces the tree with a tar archive.
Example:
```go
n := node.New()
_ = n.LoadTar([]byte{})
```

**Walk(root string, fn fs.WalkDirFunc, options WalkOptions) error**
Walks the tree with optional depth or filter controls.
Example:
```go
n := node.New()
options := node.WalkOptions{MaxDepth: 1, SkipErrors: true}
_ = n.Walk(".", func(path string, d fs.DirEntry, err error) error {
	return nil
}, options)
```

**ReadFile(name string) ([]byte, error)**
Reads file content as bytes.
Example:
```go
n := node.New()
_ = n.Write("file.txt", "data")
b, _ := n.ReadFile("file.txt")
```

**ExportFile(sourcePath, destinationPath string, perm fs.FileMode) error**
Exports a file from the in-memory tree to the local filesystem. Operates on coreio.Local directly — use CopyTo for Medium-agnostic transfers.
Example:
```go
n := node.New()
_ = n.Write("file.txt", "data")
_ = n.ExportFile("file.txt", "/tmp/file.txt", 0644)
```

**CopyTo(target io.Medium, sourcePath, destPath string) error**
Copies a file or directory tree to another medium.
Example:
```go
n := node.New()
_ = n.Write("config/app.yaml", "port: 8080")
copyTarget := io.NewMemoryMedium()
_ = n.CopyTo(copyTarget, "config", "backup/config")
```

**Open(name string) (fs.File, error)**
Opens a file for reading.
Example:
```go
n := node.New()
_ = n.Write("file.txt", "data")
f, _ := n.Open("file.txt")
defer f.Close()
```

**Stat(name string) (fs.FileInfo, error)**
Returns metadata for a file or directory.
Example:
```go
n := node.New()
_ = n.Write("file.txt", "data")
info, _ := n.Stat("file.txt")
```

**ReadDir(name string) ([]fs.DirEntry, error)**
Lists directory entries.
Example:
```go
n := node.New()
_ = n.Write("dir/file.txt", "data")
entries, _ := n.ReadDir("dir")
```

**Read(p string) (string, error)**
Reads content as a string.
Example:
```go
n := node.New()
_ = n.Write("file.txt", "data")
value, _ := n.Read("file.txt")
```

**Write(p, content string) error**
Writes content to a file.
Example:
```go
n := node.New()
_ = n.Write("file.txt", "data")
```

**WriteMode(p, content string, mode fs.FileMode) error**
Writes content with explicit permissions (no-op in memory).
Example:
```go
n := node.New()
_ = n.WriteMode("file.txt", "data", 0600)
```

**Read(p string) (string, error)**
Alias for `Read`.
Example:
```go
n := node.New()
_ = n.Write("file.txt", "data")
value, _ := n.Read("file.txt")
```

**Write(p, content string) error**
Alias for `Write`.
Example:
```go
n := node.New()
_ = n.Write("file.txt", "data")
```

**EnsureDir(path string) error**
No-op (directories are implicit).
Example:
```go
n := node.New()
_ = n.EnsureDir("dir")
```

**Exists(p string) bool**
Reports whether a path exists.
Example:
```go
n := node.New()
_ = n.Write("file.txt", "data")
ok := n.Exists("file.txt")
```

**IsFile(p string) bool**
Reports whether a path is a file.
Example:
```go
n := node.New()
_ = n.Write("file.txt", "data")
ok := n.IsFile("file.txt")
```

**IsDir(p string) bool**
Reports whether a path is a directory.
Example:
```go
n := node.New()
_ = n.Write("dir/file.txt", "data")
ok := n.IsDir("dir")
```

**Delete(p string) error**
Deletes a file.
Example:
```go
n := node.New()
_ = n.Write("old.txt", "data")
_ = n.Delete("old.txt")
```

**DeleteAll(p string) error**
Deletes a file or directory tree.
Example:
```go
n := node.New()
_ = n.Write("logs/run.txt", "started")
_ = n.DeleteAll("logs")
```

**Rename(oldPath, newPath string) error**
Moves a file within the node.
Example:
```go
n := node.New()
_ = n.Write("old.txt", "data")
_ = n.Rename("old.txt", "new.txt")
```

**List(p string) ([]fs.DirEntry, error)**
Lists directory entries.
Example:
```go
n := node.New()
_ = n.Write("dir/file.txt", "data")
entries, _ := n.List("dir")
```

**Create(p string) (io.WriteCloser, error)**
Creates or truncates a file and returns a writer.
Example:
```go
n := node.New()
w, _ := n.Create("file.txt")
_, _ = w.Write([]byte("data"))
_ = w.Close()
```

**Append(p string) (io.WriteCloser, error)**
Appends to a file and returns a writer.
Example:
```go
n := node.New()
_ = n.Write("file.txt", "data")
w, _ := n.Append("file.txt")
_, _ = w.Write([]byte(" more"))
_ = w.Close()
```

**ReadStream(p string) (io.ReadCloser, error)**
Opens a streaming reader.
Example:
```go
n := node.New()
_ = n.Write("file.txt", "data")
r, _ := n.ReadStream("file.txt")
defer r.Close()
```

**WriteStream(p string) (io.WriteCloser, error)**
Opens a streaming writer.
Example:
```go
n := node.New()
w, _ := n.WriteStream("file.txt")
_, _ = w.Write([]byte("data"))
_ = w.Close()
```

## Package store (`dappco.re/go/core/io/store`)

Group-namespaced key-value store backed by SQLite, plus a `Medium` adapter.

### NotFoundError

Returned when a key does not exist.

Example:
```go
keyValueStore, _ := store.New(store.Options{Path: ":memory:"})
_, err := keyValueStore.Get("config", "missing")
if core.Is(err, store.NotFoundError) {
	// handle missing key
}
```

### Options

Configures the SQLite database path used by the store.

Example:
```go
options := store.Options{Path: ":memory:"}
_ = options
```

### New(options Options) (*KeyValueStore, error)

Creates a new `KeyValueStore` backed by the configured SQLite path.

Example:
```go
keyValueStore, _ := store.New(store.Options{Path: ":memory:"})
_ = keyValueStore.Set("config", "theme", "midnight")
```

### KeyValueStore

Group-namespaced key-value store.

Example:
```go
keyValueStore, _ := store.New(store.Options{Path: ":memory:"})
_ = keyValueStore.Set("config", "theme", "midnight")
```

**Close() error**
Example:
```go
keyValueStore, _ := store.New(store.Options{Path: ":memory:"})
_ = keyValueStore.Close()
```

**Get(group, key string) (string, error)**
Example:
```go
keyValueStore, _ := store.New(store.Options{Path: ":memory:"})
_ = keyValueStore.Set("config", "theme", "midnight")
value, _ := keyValueStore.Get("config", "theme")
```

**Set(group, key, value string) error**
Example:
```go
keyValueStore, _ := store.New(store.Options{Path: ":memory:"})
_ = keyValueStore.Set("config", "theme", "midnight")
```

**Delete(group, key string) error**
Example:
```go
keyValueStore, _ := store.New(store.Options{Path: ":memory:"})
_ = keyValueStore.Set("config", "theme", "midnight")
_ = keyValueStore.Delete("config", "theme")
```

**Count(group string) (int, error)**
Example:
```go
keyValueStore, _ := store.New(store.Options{Path: ":memory:"})
_ = keyValueStore.Set("config", "theme", "midnight")
count, _ := keyValueStore.Count("config")
```

**DeleteGroup(group string) error**
Example:
```go
keyValueStore, _ := store.New(store.Options{Path: ":memory:"})
_ = keyValueStore.Set("config", "theme", "midnight")
_ = keyValueStore.DeleteGroup("config")
```

**GetAll(group string) (map[string]string, error)**
Example:
```go
keyValueStore, _ := store.New(store.Options{Path: ":memory:"})
_ = keyValueStore.Set("config", "theme", "midnight")
all, _ := keyValueStore.GetAll("config")
```

**Render(tmplStr, group string) (string, error)**
Example:
```go
keyValueStore, _ := store.New(store.Options{Path: ":memory:"})
_ = keyValueStore.Set("user", "name", "alice")
renderedText, _ := keyValueStore.Render("hello {{ .name }}", "user")
```

**AsMedium() *Medium**
Example:
```go
keyValueStore, _ := store.New(store.Options{Path: ":memory:"})
medium := keyValueStore.AsMedium()
_ = medium.Write("config/theme", "midnight")
```

### NewMedium(options Options) (*Medium, error)

Creates an `io.Medium` backed by a SQLite key-value store.

Example:
```go
m, _ := store.NewMedium(store.Options{Path: "config.db"})
_ = m.Write("config/theme", "midnight")
```

### Medium

Adapter that maps `group/key` paths onto a `KeyValueStore`.

Example:
```go
m, _ := store.NewMedium(store.Options{Path: ":memory:"})
_ = m.Write("config/theme", "midnight")
```

**KeyValueStore() *KeyValueStore**
Example:
```go
m, _ := store.NewMedium(store.Options{Path: ":memory:"})
keyValueStore := m.KeyValueStore()
_ = keyValueStore.Set("config", "theme", "midnight")
```

**Close() error**
Example:
```go
m, _ := store.NewMedium(store.Options{Path: ":memory:"})
_ = m.Close()
```

**Read(p string) (string, error)**
Example:
```go
m, _ := store.NewMedium(store.Options{Path: ":memory:"})
_ = m.Write("config/theme", "midnight")
value, _ := m.Read("config/theme")
```

**Write(p, content string) error**
Example:
```go
m, _ := store.NewMedium(store.Options{Path: ":memory:"})
_ = m.Write("config/theme", "midnight")
```

**EnsureDir(path string) error**
No-op (groups are implicit).
Example:
```go
m, _ := store.NewMedium(store.Options{Path: ":memory:"})
_ = m.EnsureDir("config")
```

**IsFile(p string) bool**
Example:
```go
m, _ := store.NewMedium(store.Options{Path: ":memory:"})
_ = m.Write("config/theme", "midnight")
ok := m.IsFile("config/theme")
```

**Read(p string) (string, error)**
Example:
```go
m, _ := store.NewMedium(store.Options{Path: ":memory:"})
_ = m.Write("config/theme", "midnight")
value, _ := m.Read("config/theme")
```

**Write(p, content string) error**
Example:
```go
m, _ := store.NewMedium(store.Options{Path: ":memory:"})
_ = m.Write("config/theme", "midnight")
```

**Delete(p string) error**
Example:
```go
m, _ := store.NewMedium(store.Options{Path: ":memory:"})
_ = m.Write("config/theme", "midnight")
_ = m.Delete("config/theme")
```

**DeleteAll(p string) error**
Example:
```go
m, _ := store.NewMedium(store.Options{Path: ":memory:"})
_ = m.Write("config/theme", "midnight")
_ = m.DeleteAll("config")
```

**Rename(oldPath, newPath string) error**
Example:
```go
m, _ := store.NewMedium(store.Options{Path: ":memory:"})
_ = m.Write("old/theme", "midnight")
_ = m.Rename("old/theme", "new/theme")
```

**List(p string) ([]fs.DirEntry, error)**
Example:
```go
m, _ := store.NewMedium(store.Options{Path: ":memory:"})
_ = m.Write("config/theme", "midnight")
entries, _ := m.List("")
```

**Stat(p string) (fs.FileInfo, error)**
Example:
```go
m, _ := store.NewMedium(store.Options{Path: ":memory:"})
_ = m.Write("config/theme", "midnight")
info, _ := m.Stat("config/theme")
```

**Open(p string) (fs.File, error)**
Example:
```go
m, _ := store.NewMedium(store.Options{Path: ":memory:"})
_ = m.Write("config/theme", "midnight")
f, _ := m.Open("config/theme")
defer f.Close()
```

**Create(p string) (io.WriteCloser, error)**
Example:
```go
m, _ := store.NewMedium(store.Options{Path: ":memory:"})
w, _ := m.Create("config/theme")
_, _ = w.Write([]byte("midnight"))
_ = w.Close()
```

**Append(p string) (io.WriteCloser, error)**
Example:
```go
m, _ := store.NewMedium(store.Options{Path: ":memory:"})
_ = m.Write("config/theme", "midnight")
w, _ := m.Append("config/theme")
_, _ = w.Write([]byte(" plus"))
_ = w.Close()
```

**ReadStream(p string) (io.ReadCloser, error)**
Example:
```go
m, _ := store.NewMedium(store.Options{Path: ":memory:"})
_ = m.Write("config/theme", "midnight")
r, _ := m.ReadStream("config/theme")
defer r.Close()
```

**WriteStream(p string) (io.WriteCloser, error)**
Example:
```go
m, _ := store.NewMedium(store.Options{Path: ":memory:"})
w, _ := m.WriteStream("config/theme")
_, _ = w.Write([]byte("midnight"))
_ = w.Close()
```

**Exists(p string) bool**
Example:
```go
m, _ := store.NewMedium(store.Options{Path: ":memory:"})
_ = m.Write("config/theme", "midnight")
ok := m.Exists("config")
```

**IsDir(p string) bool**
Example:
```go
m, _ := store.NewMedium(store.Options{Path: ":memory:"})
_ = m.Write("config/theme", "midnight")
ok := m.IsDir("config")
```

## Package sqlite (`dappco.re/go/core/io/sqlite`)

SQLite-backed `io.Medium` implementation using the pure-Go driver.

### Options

Configures the SQLite database path and optional table name.

Example:
```go
options := sqlite.Options{Path: ":memory:", Table: "files"}
_ = options
```

### New(options Options) (*Medium, error)

Creates a new SQLite-backed medium.

Example:
```go
m, _ := sqlite.New(sqlite.Options{Path: ":memory:"})
_ = m.Write("notes.txt", "hello")
```

### Medium

SQLite-backed storage backend.

Example:
```go
m, _ := sqlite.New(sqlite.Options{Path: ":memory:"})
_ = m.Write("notes.txt", "hello")
```

**Close() error**
Example:
```go
m, _ := sqlite.New(sqlite.Options{Path: ":memory:"})
_ = m.Close()
```

**Read(p string) (string, error)**
Example:
```go
m, _ := sqlite.New(sqlite.Options{Path: ":memory:"})
_ = m.Write("notes.txt", "hello")
value, _ := m.Read("notes.txt")
```

**Write(p, content string) error**
Example:
```go
m, _ := sqlite.New(sqlite.Options{Path: ":memory:"})
_ = m.Write("notes.txt", "hello")
```

**EnsureDir(p string) error**
Example:
```go
m, _ := sqlite.New(sqlite.Options{Path: ":memory:"})
_ = m.EnsureDir("config")
```

**IsFile(p string) bool**
Example:
```go
m, _ := sqlite.New(sqlite.Options{Path: ":memory:"})
_ = m.Write("notes.txt", "hello")
ok := m.IsFile("notes.txt")
```

**Read(p string) (string, error)**
Example:
```go
m, _ := sqlite.New(sqlite.Options{Path: ":memory:"})
_ = m.Write("notes.txt", "hello")
value, _ := m.Read("notes.txt")
```

**Write(p, content string) error**
Example:
```go
m, _ := sqlite.New(sqlite.Options{Path: ":memory:"})
_ = m.Write("notes.txt", "hello")
```

**Delete(p string) error**
Example:
```go
m, _ := sqlite.New(sqlite.Options{Path: ":memory:"})
_ = m.Write("old.txt", "data")
_ = m.Delete("old.txt")
```

**DeleteAll(p string) error**
Example:
```go
m, _ := sqlite.New(sqlite.Options{Path: ":memory:"})
_ = m.Write("logs/run.txt", "started")
_ = m.DeleteAll("logs")
```

**Rename(oldPath, newPath string) error**
Example:
```go
m, _ := sqlite.New(sqlite.Options{Path: ":memory:"})
_ = m.Write("old.txt", "data")
_ = m.Rename("old.txt", "new.txt")
```

**List(p string) ([]fs.DirEntry, error)**
Example:
```go
m, _ := sqlite.New(sqlite.Options{Path: ":memory:"})
_ = m.Write("dir/file.txt", "data")
entries, _ := m.List("dir")
```

**Stat(p string) (fs.FileInfo, error)**
Example:
```go
m, _ := sqlite.New(sqlite.Options{Path: ":memory:"})
_ = m.Write("notes.txt", "hello")
info, _ := m.Stat("notes.txt")
```

**Open(p string) (fs.File, error)**
Example:
```go
m, _ := sqlite.New(sqlite.Options{Path: ":memory:"})
_ = m.Write("notes.txt", "hello")
f, _ := m.Open("notes.txt")
defer f.Close()
```

**Create(p string) (io.WriteCloser, error)**
Example:
```go
m, _ := sqlite.New(sqlite.Options{Path: ":memory:"})
w, _ := m.Create("notes.txt")
_, _ = w.Write([]byte("hello"))
_ = w.Close()
```

**Append(p string) (io.WriteCloser, error)**
Example:
```go
m, _ := sqlite.New(sqlite.Options{Path: ":memory:"})
_ = m.Write("notes.txt", "hello")
w, _ := m.Append("notes.txt")
_, _ = w.Write([]byte(" world"))
_ = w.Close()
```

**ReadStream(p string) (io.ReadCloser, error)**
Example:
```go
m, _ := sqlite.New(sqlite.Options{Path: ":memory:"})
_ = m.Write("notes.txt", "hello")
r, _ := m.ReadStream("notes.txt")
defer r.Close()
```

**WriteStream(p string) (io.WriteCloser, error)**
Example:
```go
m, _ := sqlite.New(sqlite.Options{Path: ":memory:"})
w, _ := m.WriteStream("notes.txt")
_, _ = w.Write([]byte("hello"))
_ = w.Close()
```

**Exists(p string) bool**
Example:
```go
m, _ := sqlite.New(sqlite.Options{Path: ":memory:"})
_ = m.Write("notes.txt", "hello")
ok := m.Exists("notes.txt")
```

**IsDir(p string) bool**
Example:
```go
m, _ := sqlite.New(sqlite.Options{Path: ":memory:"})
_ = m.EnsureDir("config")
ok := m.IsDir("config")
```

## Package s3 (`dappco.re/go/core/io/s3`)

Amazon S3-backed `io.Medium` implementation.

### Options

Configures the bucket, client, and optional key prefix.

Example:
```go
client := awss3.NewFromConfig(aws.Config{Region: "us-east-1"})
options := s3.Options{Bucket: "bucket", Client: client, Prefix: "daily/"}
_ = options
```

### Client

Supplies an AWS SDK S3 client.

Example:
```go
client := awss3.NewFromConfig(aws.Config{Region: "us-east-1"})
m, _ := s3.New(s3.Options{Bucket: "bucket", Client: client})
```

### New(options Options) (*Medium, error)

Creates a new S3-backed medium.

Example:
```go
client := awss3.NewFromConfig(aws.Config{Region: "us-east-1"})
m, _ := s3.New(s3.Options{Bucket: "bucket", Client: client})
```

### Medium

S3-backed storage backend.

Example:
```go
client := awss3.NewFromConfig(aws.Config{Region: "us-east-1"})
m, _ := s3.New(s3.Options{Bucket: "bucket", Client: client})
```

**Read(p string) (string, error)**
Example:
```go
client := awss3.NewFromConfig(aws.Config{Region: "us-east-1"})
m, _ := s3.New(s3.Options{Bucket: "bucket", Client: client})
value, _ := m.Read("notes.txt")
```

**Write(p, content string) error**
Example:
```go
client := awss3.NewFromConfig(aws.Config{Region: "us-east-1"})
m, _ := s3.New(s3.Options{Bucket: "bucket", Client: client})
_ = m.Write("notes.txt", "hello")
```

**EnsureDir(path string) error**
No-op (S3 has no directories).
Example:
```go
client := awss3.NewFromConfig(aws.Config{Region: "us-east-1"})
m, _ := s3.New(s3.Options{Bucket: "bucket", Client: client})
_ = m.EnsureDir("config")
```

**IsFile(p string) bool**
Example:
```go
client := awss3.NewFromConfig(aws.Config{Region: "us-east-1"})
m, _ := s3.New(s3.Options{Bucket: "bucket", Client: client})
ok := m.IsFile("notes.txt")
```

**Read(p string) (string, error)**
Example:
```go
client := awss3.NewFromConfig(aws.Config{Region: "us-east-1"})
m, _ := s3.New(s3.Options{Bucket: "bucket", Client: client})
value, _ := m.Read("notes.txt")
```

**Write(p, content string) error**
Example:
```go
client := awss3.NewFromConfig(aws.Config{Region: "us-east-1"})
m, _ := s3.New(s3.Options{Bucket: "bucket", Client: client})
_ = m.Write("notes.txt", "hello")
```

**Delete(p string) error**
Example:
```go
client := awss3.NewFromConfig(aws.Config{Region: "us-east-1"})
m, _ := s3.New(s3.Options{Bucket: "bucket", Client: client})
_ = m.Delete("old.txt")
```

**DeleteAll(p string) error**
Example:
```go
client := awss3.NewFromConfig(aws.Config{Region: "us-east-1"})
m, _ := s3.New(s3.Options{Bucket: "bucket", Client: client})
_ = m.DeleteAll("logs")
```

**Rename(oldPath, newPath string) error**
Example:
```go
client := awss3.NewFromConfig(aws.Config{Region: "us-east-1"})
m, _ := s3.New(s3.Options{Bucket: "bucket", Client: client})
_ = m.Rename("old.txt", "new.txt")
```

**List(p string) ([]fs.DirEntry, error)**
Example:
```go
client := awss3.NewFromConfig(aws.Config{Region: "us-east-1"})
m, _ := s3.New(s3.Options{Bucket: "bucket", Client: client})
entries, _ := m.List("dir")
```

**Stat(p string) (fs.FileInfo, error)**
Example:
```go
client := awss3.NewFromConfig(aws.Config{Region: "us-east-1"})
m, _ := s3.New(s3.Options{Bucket: "bucket", Client: client})
info, _ := m.Stat("notes.txt")
```

**Open(p string) (fs.File, error)**
Example:
```go
client := awss3.NewFromConfig(aws.Config{Region: "us-east-1"})
m, _ := s3.New(s3.Options{Bucket: "bucket", Client: client})
f, _ := m.Open("notes.txt")
defer f.Close()
```

**Create(p string) (io.WriteCloser, error)**
Example:
```go
client := awss3.NewFromConfig(aws.Config{Region: "us-east-1"})
m, _ := s3.New(s3.Options{Bucket: "bucket", Client: client})
w, _ := m.Create("notes.txt")
_, _ = w.Write([]byte("hello"))
_ = w.Close()
```

**Append(p string) (io.WriteCloser, error)**
Example:
```go
client := awss3.NewFromConfig(aws.Config{Region: "us-east-1"})
m, _ := s3.New(s3.Options{Bucket: "bucket", Client: client})
w, _ := m.Append("notes.txt")
_, _ = w.Write([]byte(" world"))
_ = w.Close()
```

**ReadStream(p string) (io.ReadCloser, error)**
Example:
```go
client := awss3.NewFromConfig(aws.Config{Region: "us-east-1"})
m, _ := s3.New(s3.Options{Bucket: "bucket", Client: client})
r, _ := m.ReadStream("notes.txt")
defer r.Close()
```

**WriteStream(p string) (io.WriteCloser, error)**
Example:
```go
client := awss3.NewFromConfig(aws.Config{Region: "us-east-1"})
m, _ := s3.New(s3.Options{Bucket: "bucket", Client: client})
w, _ := m.WriteStream("notes.txt")
_, _ = w.Write([]byte("hello"))
_ = w.Close()
```

**Exists(p string) bool**
Example:
```go
client := awss3.NewFromConfig(aws.Config{Region: "us-east-1"})
m, _ := s3.New(s3.Options{Bucket: "bucket", Client: client})
ok := m.Exists("notes.txt")
```

**IsDir(p string) bool**
Example:
```go
client := awss3.NewFromConfig(aws.Config{Region: "us-east-1"})
m, _ := s3.New(s3.Options{Bucket: "bucket", Client: client})
ok := m.IsDir("logs")
```

## Package datanode (`dappco.re/go/core/io/datanode`)

In-memory `io.Medium` backed by Borg's DataNode, with tar snapshot/restore support.

### New() *Medium

Creates a new empty DataNode-backed medium.

Example:
```go
m := datanode.New()
_ = m.Write("jobs/run.log", "started")
```

### FromTar(data []byte) (*Medium, error)

Restores a medium from a tar archive.

Example:
```go
m, _ := datanode.FromTar([]byte{})
```

### Medium

Thread-safe in-memory medium using Borg DataNode.

Example:
```go
m := datanode.New()
_ = m.Write("jobs/run.log", "started")
```

**Snapshot() ([]byte, error)**
Serialises the filesystem to a tarball.
Example:
```go
m := datanode.New()
_ = m.Write("jobs/run.log", "started")
snap, _ := m.Snapshot()
```

**Restore(data []byte) error**
Replaces the filesystem from a tarball.
Example:
```go
m := datanode.New()
_ = m.Restore([]byte{})
```

**DataNode() *datanode.DataNode**
Returns the underlying Borg DataNode.
Example:
```go
m := datanode.New()
dn := m.DataNode()
_ = dn
```

**Read(p string) (string, error)**
Example:
```go
m := datanode.New()
_ = m.Write("notes.txt", "hello")
value, _ := m.Read("notes.txt")
```

**Write(p, content string) error**
Example:
```go
m := datanode.New()
_ = m.Write("notes.txt", "hello")
```

**WriteMode(p, content string, mode fs.FileMode) error**
Example:
```go
m := datanode.New()
_ = m.WriteMode("notes.txt", "hello", 0600)
```

**EnsureDir(p string) error**
Example:
```go
m := datanode.New()
_ = m.EnsureDir("config")
```

**IsFile(p string) bool**
Example:
```go
m := datanode.New()
_ = m.Write("notes.txt", "hello")
ok := m.IsFile("notes.txt")
```

**Read(p string) (string, error)**
Example:
```go
m := datanode.New()
_ = m.Write("notes.txt", "hello")
value, _ := m.Read("notes.txt")
```

**Write(p, content string) error**
Example:
```go
m := datanode.New()
_ = m.Write("notes.txt", "hello")
```

**Delete(p string) error**
Example:
```go
m := datanode.New()
_ = m.Write("old.txt", "data")
_ = m.Delete("old.txt")
```

**DeleteAll(p string) error**
Example:
```go
m := datanode.New()
_ = m.Write("logs/run.txt", "started")
_ = m.DeleteAll("logs")
```

**Rename(oldPath, newPath string) error**
Example:
```go
m := datanode.New()
_ = m.Write("old.txt", "data")
_ = m.Rename("old.txt", "new.txt")
```

**List(p string) ([]fs.DirEntry, error)**
Example:
```go
m := datanode.New()
_ = m.Write("dir/file.txt", "data")
entries, _ := m.List("dir")
```

**Stat(p string) (fs.FileInfo, error)**
Example:
```go
m := datanode.New()
_ = m.Write("notes.txt", "hello")
info, _ := m.Stat("notes.txt")
```

**Open(p string) (fs.File, error)**
Example:
```go
m := datanode.New()
_ = m.Write("notes.txt", "hello")
f, _ := m.Open("notes.txt")
defer f.Close()
```

**Create(p string) (io.WriteCloser, error)**
Example:
```go
m := datanode.New()
w, _ := m.Create("notes.txt")
_, _ = w.Write([]byte("hello"))
_ = w.Close()
```

**Append(p string) (io.WriteCloser, error)**
Example:
```go
m := datanode.New()
_ = m.Write("notes.txt", "hello")
w, _ := m.Append("notes.txt")
_, _ = w.Write([]byte(" world"))
_ = w.Close()
```

**ReadStream(p string) (io.ReadCloser, error)**
Example:
```go
m := datanode.New()
_ = m.Write("notes.txt", "hello")
r, _ := m.ReadStream("notes.txt")
defer r.Close()
```

**WriteStream(p string) (io.WriteCloser, error)**
Example:
```go
m := datanode.New()
w, _ := m.WriteStream("notes.txt")
_, _ = w.Write([]byte("hello"))
_ = w.Close()
```

**Exists(p string) bool**
Example:
```go
m := datanode.New()
_ = m.Write("notes.txt", "hello")
ok := m.Exists("notes.txt")
```

**IsDir(p string) bool**
Example:
```go
m := datanode.New()
_ = m.EnsureDir("config")
ok := m.IsDir("config")
```

## Package workspace (`dappco.re/go/core/io/workspace`)

Encrypted user workspace management.

### Workspace (interface)

Creates and operates on encrypted workspaces through a service.

Example:
```go
service, _ := workspace.New(workspace.Options{KeyPairProvider: stubKeyPairProvider{}})
_ = service
```

### KeyPairProvider

Creates key pairs for workspace provisioning.

Example:
```go
keyPairProvider := stubKeyPairProvider{}
keyPair, _ := keyPairProvider.CreateKeyPair("alice", "pass123")
_ = keyPair
```

### WorkspaceCreateAction

Action value used to create a workspace.

Example:
```go
command := workspace.WorkspaceCommand{Action: workspace.WorkspaceCreateAction, Identifier: "alice", Password: "pass123"}
_ = command
```

### WorkspaceSwitchAction

Action value used to switch to an existing workspace.

Example:
```go
command := workspace.WorkspaceCommand{Action: workspace.WorkspaceSwitchAction, WorkspaceID: "f3f0d7"}
_ = command
```

### WorkspaceCommand

Command envelope consumed by the service.

Example:
```go
command := workspace.WorkspaceCommand{Action: workspace.WorkspaceCreateAction, Identifier: "alice", Password: "pass123"}
_ = command
```

### Options

Configures the workspace service.

Example:
```go
service, _ := workspace.New(workspace.Options{KeyPairProvider: stubKeyPairProvider{}})
_ = service
```

### New(options Options) (*Service, error)

Creates a new workspace service.

Example:
```go
type stubKeyPairProvider struct{}

func (stubKeyPairProvider) CreateKeyPair(name, passphrase string) (string, error) {
	return "key", nil
}

service, _ := workspace.New(workspace.Options{KeyPairProvider: stubKeyPairProvider{}})
_ = service
```

### Service

Implements `Workspace` and handles Core messages.

Example:
```go
service, _ := workspace.New(workspace.Options{KeyPairProvider: stubKeyPairProvider{}})
_ = service
```

**CreateWorkspace(identifier, passphrase string) (string, error)**
Example:
```go
service, _ := workspace.New(workspace.Options{KeyPairProvider: stubKeyPairProvider{}})
workspaceID, _ := service.CreateWorkspace("alice", "pass123")
_ = workspaceID
```

**SwitchWorkspace(workspaceID string) error**
Example:
```go
service, _ := workspace.New(workspace.Options{KeyPairProvider: stubKeyPairProvider{}})
_ = service.SwitchWorkspace("f3f0d7")
```

**ReadWorkspaceFile(workspaceFilePath string) (string, error)**
Example:
```go
service, _ := workspace.New(workspace.Options{KeyPairProvider: stubKeyPairProvider{}})
content, _ := service.ReadWorkspaceFile("notes/todo.txt")
_ = content
```

**WriteWorkspaceFile(workspaceFilePath, content string) error**
Example:
```go
service, _ := workspace.New(workspace.Options{KeyPairProvider: stubKeyPairProvider{}})
_ = service.WriteWorkspaceFile("notes/todo.txt", "ship it")
```

**HandleWorkspaceCommand(command WorkspaceCommand) core.Result**
Example:
```go
service, _ := workspace.New(workspace.Options{KeyPairProvider: stubKeyPairProvider{}})
result := service.HandleWorkspaceCommand(workspace.WorkspaceCommand{Action: workspace.WorkspaceCreateAction, Identifier: "alice", Password: "pass123"})
_ = result.OK
```

**HandleWorkspaceMessage(_ *core.Core, message core.Message) core.Result**
Example:
```go
service, _ := workspace.New(workspace.Options{KeyPairProvider: stubKeyPairProvider{}})
result := service.HandleWorkspaceMessage(core.New(), workspace.WorkspaceCommand{Action: workspace.WorkspaceSwitchAction, WorkspaceID: "f3f0d7"})
_ = result.OK
```

## Package sigil (`dappco.re/go/core/io/sigil`)

Composable data-transformation sigils for encoding, compression, hashing, and encryption.

### Sigil (interface)

Defines the transformation contract.

Example:
```go
var s sigil.Sigil = &sigil.HexSigil{}
out, _ := s.In([]byte("hello"))
_ = out
```

**In(data []byte) ([]byte, error)**
Applies the forward transformation.
Example:
```go
s := &sigil.HexSigil{}
encoded, _ := s.In([]byte("hello"))
```

**Out(data []byte) ([]byte, error)**
Applies the reverse transformation.
Example:
```go
s := &sigil.HexSigil{}
encoded, _ := s.In([]byte("hello"))
decoded, _ := s.Out(encoded)
```

### Transmute(data []byte, sigils []Sigil) ([]byte, error)

Applies `In` across a chain of sigils.

Example:
```go
hexSigil, _ := sigil.NewSigil("hex")
base64Sigil, _ := sigil.NewSigil("base64")
encoded, _ := sigil.Transmute([]byte("hello"), []sigil.Sigil{hexSigil, base64Sigil})
```

### Untransmute(data []byte, sigils []Sigil) ([]byte, error)

Reverses a transmutation by applying `Out` in reverse order.

Example:
```go
hexSigil, _ := sigil.NewSigil("hex")
base64Sigil, _ := sigil.NewSigil("base64")
encoded, _ := sigil.Transmute([]byte("hello"), []sigil.Sigil{hexSigil, base64Sigil})
plain, _ := sigil.Untransmute(encoded, []sigil.Sigil{hexSigil, base64Sigil})
```

### ReverseSigil

Reverses byte order (symmetric).

Example:
```go
s := &sigil.ReverseSigil{}
```

**In(data []byte) ([]byte, error)**
Example:
```go
s := &sigil.ReverseSigil{}
reversed, _ := s.In([]byte("hello"))
```

**Out(data []byte) ([]byte, error)**
Example:
```go
s := &sigil.ReverseSigil{}
reversed, _ := s.In([]byte("hello"))
restored, _ := s.Out(reversed)
```

### HexSigil

Encodes/decodes hexadecimal.

Example:
```go
s := &sigil.HexSigil{}
```

**In(data []byte) ([]byte, error)**
Example:
```go
s := &sigil.HexSigil{}
encoded, _ := s.In([]byte("hello"))
```

**Out(data []byte) ([]byte, error)**
Example:
```go
s := &sigil.HexSigil{}
encoded, _ := s.In([]byte("hello"))
decoded, _ := s.Out(encoded)
```

### Base64Sigil

Encodes/decodes base64.

Example:
```go
s := &sigil.Base64Sigil{}
```

**In(data []byte) ([]byte, error)**
Example:
```go
s := &sigil.Base64Sigil{}
encoded, _ := s.In([]byte("hello"))
```

**Out(data []byte) ([]byte, error)**
Example:
```go
s := &sigil.Base64Sigil{}
encoded, _ := s.In([]byte("hello"))
decoded, _ := s.Out(encoded)
```

### GzipSigil

Compresses/decompresses gzip payloads.

Example:
```go
s := &sigil.GzipSigil{}
```

**In(data []byte) ([]byte, error)**
Example:
```go
s := &sigil.GzipSigil{}
compressed, _ := s.In([]byte("hello"))
```

**Out(data []byte) ([]byte, error)**
Example:
```go
s := &sigil.GzipSigil{}
compressed, _ := s.In([]byte("hello"))
plain, _ := s.Out(compressed)
```

### JSONSigil

Compacts or indents JSON (depending on `Indent`).

Example:
```go
s := &sigil.JSONSigil{Indent: true}
```

**In(data []byte) ([]byte, error)**
Example:
```go
s := &sigil.JSONSigil{Indent: false}
compacted, _ := s.In([]byte(`{"key":"value"}`))
```

**Out(data []byte) ([]byte, error)**
No-op for `JSONSigil`.
Example:
```go
s := &sigil.JSONSigil{Indent: false}
pass, _ := s.Out([]byte(`{"key":"value"}`))
```

### HashSigil

Hashes input using the configured `crypto.Hash`.

Example:
```go
s := sigil.NewHashSigil(crypto.SHA256)
```

**In(data []byte) ([]byte, error)**
Example:
```go
s := sigil.NewHashSigil(crypto.SHA256)
digest, _ := s.In([]byte("hello"))
```

**Out(data []byte) ([]byte, error)**
No-op for hash sigils.
Example:
```go
s := sigil.NewHashSigil(crypto.SHA256)
digest, _ := s.In([]byte("hello"))
pass, _ := s.Out(digest)
```

### NewHashSigil(h crypto.Hash) *HashSigil

Creates a new `HashSigil`.

Example:
```go
s := sigil.NewHashSigil(crypto.SHA256)
```

### NewSigil(name string) (Sigil, error)

Factory for built-in sigils.

Example:
```go
s, _ := sigil.NewSigil("hex")
```

### InvalidKeyError

Returned when an encryption key is not 32 bytes.

Example:
```go
_, err := sigil.NewChaChaPolySigil([]byte("short"), nil)
if errors.Is(err, sigil.InvalidKeyError) {
	// handle invalid key
}
```

### CiphertextTooShortError

Returned when ciphertext is too short to decrypt.

Example:
```go
_, err := sigil.NonceFromCiphertext([]byte("short"))
if errors.Is(err, sigil.CiphertextTooShortError) {
	// handle truncated payload
}
```

### DecryptionFailedError

Returned when decryption or authentication fails.

Example:
```go
key := make([]byte, 32)
s, _ := sigil.NewChaChaPolySigil(key, nil)
_, err := s.Out([]byte("tampered"))
if errors.Is(err, sigil.DecryptionFailedError) {
	// handle failed decrypt
}
```

### NoKeyConfiguredError

Returned when a `ChaChaPolySigil` has no key.

Example:
```go
s := &sigil.ChaChaPolySigil{}
_, err := s.In([]byte("data"))
if errors.Is(err, sigil.NoKeyConfiguredError) {
	// handle missing key
}
```

### PreObfuscator (interface)

Defines pre-obfuscation hooks for encryption sigils.

Example:
```go
var ob sigil.PreObfuscator = &sigil.XORObfuscator{}
_ = ob
```

**Obfuscate(data []byte, entropy []byte) []byte**
Example:
```go
ob := &sigil.XORObfuscator{}
masked := ob.Obfuscate([]byte("hello"), []byte("nonce"))
```

**Deobfuscate(data []byte, entropy []byte) []byte**
Example:
```go
ob := &sigil.XORObfuscator{}
masked := ob.Obfuscate([]byte("hello"), []byte("nonce"))
plain := ob.Deobfuscate(masked, []byte("nonce"))
```

### XORObfuscator

XOR-based pre-obfuscator.

Example:
```go
ob := &sigil.XORObfuscator{}
```

**Obfuscate(data []byte, entropy []byte) []byte**
Example:
```go
ob := &sigil.XORObfuscator{}
masked := ob.Obfuscate([]byte("hello"), []byte("nonce"))
```

**Deobfuscate(data []byte, entropy []byte) []byte**
Example:
```go
ob := &sigil.XORObfuscator{}
masked := ob.Obfuscate([]byte("hello"), []byte("nonce"))
plain := ob.Deobfuscate(masked, []byte("nonce"))
```

### ShuffleMaskObfuscator

Shuffle + mask pre-obfuscator.

Example:
```go
ob := &sigil.ShuffleMaskObfuscator{}
```

**Obfuscate(data []byte, entropy []byte) []byte**
Example:
```go
ob := &sigil.ShuffleMaskObfuscator{}
masked := ob.Obfuscate([]byte("hello"), []byte("nonce"))
```

**Deobfuscate(data []byte, entropy []byte) []byte**
Example:
```go
ob := &sigil.ShuffleMaskObfuscator{}
masked := ob.Obfuscate([]byte("hello"), []byte("nonce"))
plain := ob.Deobfuscate(masked, []byte("nonce"))
```

### ChaChaPolySigil

XChaCha20-Poly1305 encryption sigil with optional pre-obfuscation.

Example:
```go
key := make([]byte, 32)
s, _ := sigil.NewChaChaPolySigil(key, nil)
```

**In(data []byte) ([]byte, error)**
Example:
```go
key := make([]byte, 32)
s, _ := sigil.NewChaChaPolySigil(key, nil)
ciphertext, _ := s.In([]byte("hello"))
```

**Out(data []byte) ([]byte, error)**
Example:
```go
key := make([]byte, 32)
s, _ := sigil.NewChaChaPolySigil(key, nil)
ciphertext, _ := s.In([]byte("hello"))
plain, _ := s.Out(ciphertext)
```

### NewChaChaPolySigil(key []byte, obfuscator PreObfuscator) (*ChaChaPolySigil, error)

Creates an encryption sigil with an optional pre-obfuscator. Pass `nil` to use the default XOR obfuscator.

Example:
```go
key := make([]byte, 32)
ob := &sigil.ShuffleMaskObfuscator{}
s, _ := sigil.NewChaChaPolySigil(key, ob)
```

### NonceFromCiphertext(ciphertext []byte) ([]byte, error)

Extracts the XChaCha20 nonce from encrypted output.

Example:
```go
key := make([]byte, 32)
s, _ := sigil.NewChaChaPolySigil(key, nil)
ciphertext, _ := s.In([]byte("hello"))
nonce, _ := sigil.NonceFromCiphertext(ciphertext)
```
