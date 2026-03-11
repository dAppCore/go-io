---
title: go-io
description: Unified storage abstraction for Go with pluggable backends — local filesystem, S3, SQLite, in-memory, and key-value.
---

# go-io

`forge.lthn.ai/core/go-io` is a storage abstraction library that provides a single `Medium` interface for reading and writing files across different backends. Write your code against `Medium` once, then swap between local disk, S3, SQLite, or in-memory storage without changing a line of business logic.

The library also includes `sigil`, a composable data-transformation pipeline for encoding, compression, hashing, and authenticated encryption.


## Quick Start

```go
import (
    io "forge.lthn.ai/core/go-io"
    "forge.lthn.ai/core/go-io/s3"
    "forge.lthn.ai/core/go-io/node"
)

// Use the pre-initialised local filesystem (unsandboxed, rooted at "/").
content, _ := io.Local.Read("/etc/hostname")

// Create a sandboxed medium restricted to a single directory.
sandbox, _ := io.NewSandboxed("/var/data/myapp")
_ = sandbox.Write("config.yaml", "key: value")

// In-memory filesystem with tar serialisation.
mem := node.New()
mem.AddData("hello.txt", []byte("world"))
tarball, _ := mem.ToTar()

// S3 backend (requires an *s3.Client from the AWS SDK).
bucket, _ := s3.New("my-bucket", s3.WithClient(awsClient), s3.WithPrefix("uploads/"))
_ = bucket.Write("photo.jpg", rawData)
```


## Package Layout

| Package | Import Path | Purpose |
|---------|-------------|---------|
| `io` (root) | `forge.lthn.ai/core/go-io` | `Medium` interface, helper functions, `MockMedium` for tests |
| `local` | `forge.lthn.ai/core/go-io/local` | Local filesystem backend with path sandboxing and symlink-escape protection |
| `s3` | `forge.lthn.ai/core/go-io/s3` | Amazon S3 / S3-compatible backend (Garage, MinIO, etc.) |
| `sqlite` | `forge.lthn.ai/core/go-io/sqlite` | SQLite-backed virtual filesystem (pure Go driver, no CGO) |
| `node` | `forge.lthn.ai/core/go-io/node` | In-memory filesystem implementing both `Medium` and `fs.FS`, with tar round-tripping |
| `datanode` | `forge.lthn.ai/core/go-io/datanode` | Thread-safe in-memory `Medium` backed by Borg's DataNode, with snapshot/restore |
| `store` | `forge.lthn.ai/core/go-io/store` | Group-namespaced key-value store (SQLite), with a `Medium` adapter and Go template rendering |
| `sigil` | `forge.lthn.ai/core/go-io/sigil` | Composable data transformations: encoding, compression, hashing, XChaCha20-Poly1305 encryption |
| `workspace` | `forge.lthn.ai/core/go-io/workspace` | Encrypted workspace service integrated with the Core DI container |


## The Medium Interface

Every storage backend implements the same 18-method interface:

```go
type Medium interface {
    // Content operations
    Read(path string) (string, error)
    Write(path, content string) error
    FileGet(path string) (string, error)   // alias for Read
    FileSet(path, content string) error    // alias for Write

    // Streaming (for large files)
    ReadStream(path string) (io.ReadCloser, error)
    WriteStream(path string) (io.WriteCloser, error)
    Open(path string) (fs.File, error)
    Create(path string) (io.WriteCloser, error)
    Append(path string) (io.WriteCloser, error)

    // Directory operations
    EnsureDir(path string) error
    List(path string) ([]fs.DirEntry, error)

    // Metadata
    Stat(path string) (fs.FileInfo, error)
    Exists(path string) bool
    IsFile(path string) bool
    IsDir(path string) bool

    // Mutation
    Delete(path string) error
    DeleteAll(path string) error
    Rename(oldPath, newPath string) error
}
```

All backends implement this interface fully. Backends where a method has no natural equivalent (e.g., `EnsureDir` on S3) provide a safe no-op.


## Cross-Medium Operations

The root package provides helper functions that accept any `Medium`:

```go
// Copy a file between any two backends.
err := io.Copy(localMedium, "source.txt", s3Medium, "dest.txt")

// Read/Write wrappers that take an explicit medium.
content, err := io.Read(medium, "path")
err := io.Write(medium, "path", "content")
```


## Dependencies

| Dependency | Role |
|------------|------|
| `forge.lthn.ai/core/go-log` | Structured error helper (`E()`) |
| `forge.lthn.ai/Snider/Borg` | DataNode in-memory FS (used by `datanode` package) |
| `github.com/aws/aws-sdk-go-v2` | S3 client (used by `s3` package) |
| `golang.org/x/crypto` | BLAKE2, SHA-3, RIPEMD-160, XChaCha20-Poly1305 (used by `sigil`) |
| `modernc.org/sqlite` | Pure Go SQLite driver (used by `sqlite` and `store`) |
| `github.com/stretchr/testify` | Test assertions |

Go version: **1.26.0**

Licence: **EUPL-1.2**
