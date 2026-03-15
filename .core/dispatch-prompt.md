You are working on the go-io repository.

## Project Context (CLAUDE.md)

# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`forge.lthn.ai/core/go-io` is the **mandatory I/O abstraction layer** for the CoreGO ecosystem. All data access — files, configs, journals, state — MUST go through the `io.Medium` interface. Never use raw `os`, `filepath`, or `ioutil` calls.

### The Premise

**The directory you start your binary in becomes the immutable root.** `io.NewSandboxed(".")` (or the CWD at launch) defines the filesystem boundary — everything the process sees is relative to that root. This is the SASE containment model.

If you need a top-level system process (root at `/`), you literally run it from `/` — but that should only be for internal services never publicly exposed. Any user-facing or agent-facing process runs sandboxed to its project directory.

Swap the Medium and the same code runs against S3, SQLite, a Borg DataNode, or a runc rootFS — the binary doesn't know or care. This is what makes LEM model execution safe: the runner is an apartment with walls, not an open box.

## Commands

```bash
core go test              # Run all tests
core go test --run Name   # Single test
core go fmt               # Format
core go lint              # Lint
core go vet               # Vet
```

## Architecture

### Core Interface

`io.Medium` — 18 methods: Read, Write, EnsureDir, IsFile, FileGet, FileSet, Delete, DeleteAll, Rename, List, Stat, Open, Create, Append, ReadStream, WriteStream, Exists, IsDir.

```go
// Sandboxed to a project directory
m, _ := io.NewSandboxed("/home/user/projects/example.com")
m.Write("config/app.yaml", content)  // writes inside sandbox
m.Read("../../../etc/passwd")        // blocked — escape detected

// Unsandboxed system access (use sparingly)
io.Local.Read("/etc/hostname")

// Copy between any two mediums
io.Copy(s3Medium, "backup.tar", localMedium, "restore/backup.tar")
```

### Backends (8 implementations)

| Package | Backend | Use Case |
|---------|---------|----------|
| `local` | Local filesystem | Default, sandboxed path validation, symlink escape detection |
| `s3` | AWS S3 | Cloud storage, prefix-scoped |
| `sqlite` | SQLite (WAL mode) | Embedded database storage |
| `node` | In-memory + tar | Borg DataNode port, also implements `fs.FS`/`fs.ReadFileFS`/`fs.ReadDirFS` |
| `datanode` | Borg DataNode | Thread-safe (RWMutex) in-memory, snapshot/restore via tar |
| `store` | SQLite KV store | Group-namespaced key-value with Go template rendering |
| `workspace` | Core service | Encrypted workspaces, SHA-256 IDs, PGP keypairs |
| `MockMedium` | In-memory map | Testing — no filesystem needed |

`store.Medium` maps filesystem paths as `group/key` — first path segment is the group, remainder is the key. `List("")` returns groups as directories.

### Sigil Transformation Framework (`sigil/`)

Composable data transformations applied in chains:

| Sigil | Purpose |
|-------|---------|
| `ReverseSigil` | Byte reversal (symmetric) |
| `HexSigil` | Base16 encoding |
| `Base64Sigil` | Base64 encoding |
| `GzipSigil` | Compression |
| `JSONSigil` | JSON formatting |
| `HashSigil` | Cryptographic hashing (SHA-256, SHA-512, BLAKE2, etc.) |
| `ChaChaPolySigil` | XChaCha20-Poly1305 encryption with pre-obfuscation |

Pre-obfuscation strategies: `XORObfuscator`, `ShuffleMaskObfuscator`.

```go
// Encrypt then compress
encrypted, _ := sigil.Transmute(data, []sigil.Sigil{chacha, gzip})
// Decompress then decrypt (reverse order)
plain, _ := sigil.Untransmute(encrypted, []sigil.Sigil{chacha, gzip})
```

Sigils can be created by name via `sigil.NewSigil("hex")`, `sigil.NewSigil("sha256")`, etc.

### Security

- `local.Medium.validatePath()` follows symlinks component-by-component, checks each resolved path is still under root
- Sandbox escape attempts log `[SECURITY]` to stderr with timestamp, root, attempted path, username
- `Delete` and `DeleteAll` refuse `/` and `$HOME`
- `io.NewSandboxed(root)` enforces containment — this is the SASE boundary

## Conventions

### Import Aliasing

Standard `io` is always aliased to avoid collision with this package:
```go
goio "io"
coreerr "forge.lthn.ai/core/go-log"
coreio "forge.lthn.ai/core/go-io"  // when imported from subpackages
```

### Error Handling

All errors use `coreerr.E("pkg.Method", "description", wrappedErr)` from `forge.lthn.ai/core/go-log`. Follow this pattern in new code.

### Compile-Time Interface Checks

Backend packages use `var _ io.Medium = (*Medium)(nil)` to verify interface compliance at compile time.

## Dependencies

- `forge.lthn.ai/Snider/Borg` — DataNode container
- `forge.lthn.ai/core/go-log` — error handling (`coreerr.E()`)
- `forge.lthn.ai/core/go/pkg/core` — Core DI (workspace service only)
- `aws-sdk-go-v2` — S3 backend
- `modernc.org/sqlite` — SQLite backends (pure Go, no CGO)

## Testing

All backends have full test coverage. Use `io.MockMedium` or `io.NewSandboxed(t.TempDir())` in tests — never hit real S3/SQLite unless integration testing.


# Agent Context — go-io

> Auto-generated by agentic_prep_workspace MCP tool.



# Consumers of go-io

These modules import `forge.lthn.ai/core/go-io`:

- agent
- api
- cli
- config
- go-ai
- go-ansible
- go-blockchain
- go-build
- go-cache
- go-container
- go-crypt
- go-devops
- go-infra
- go-ml
- go-rag
- go-scm
- gui
- ide
- lint
- mcp
- php
- ts
- core
- LEM

**Breaking change risk: 24 consumers.**


# Recent Changes

```
7950b56 fix: update stale import paths and dependency versions from extraction
b2f017e docs: add CLAUDE.md project instructions
a97bbc4 docs: add human-friendly documentation
af78c9d fix: improve Delete safety guard and init resilience
08f8272 chore: add .core/ build and release configs
6b7b626 refactor: swap pkg/framework imports to pkg/core
9bb9ec7 feat: add workspace subpackage (moved from core/go/pkg/workspace)
65b39b0 feat(store): add KV store subpackage with io.Medium adapter
c282ba0 refactor: swap pkg/{io,log,i18n} imports to go-io/go-log/go-i18n
739898e fix: use forge.lthn.ai/Snider/Borg v0.3.1
ea23438 feat: standalone io.Medium abstraction
```


## Your Task

Fix UK English spelling in all Go files. Change: sanitizes→sanitises, sanitized→sanitised, Sanitized→Sanitised, normalizes→normalises, initialized→initialised, initialize→initialise. Fix in both comments and function/test names. Commit with message: fix: use UK English spelling throughout. Push to forge.

## Conventions

- UK English (colour, organisation, centre)
- Conventional commits: type(scope): description
- Co-Author: Co-Authored-By: Virgil <virgil@lethean.io>
- Licence: EUPL-1.2
- Push to forge: ssh://git@forge.lthn.ai:2223/core/go-io.git
