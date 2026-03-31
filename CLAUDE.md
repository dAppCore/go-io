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
core go qa                # fmt + vet + lint + test
```

If running `go` directly (outside `core`), set `GOWORK=off` to avoid workspace resolution errors:
```bash
GOWORK=off go test -cover ./...
```

## Architecture

### Core Interface

`io.Medium` — 17 methods: Read, Write, WriteMode, EnsureDir, IsFile, Delete, DeleteAll, Rename, List, Stat, Open, Create, Append, ReadStream, WriteStream, Exists, IsDir.

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
| `MemoryMedium` | In-memory map | Testing — no filesystem needed |

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
- `forge.lthn.ai/core/go` — Core DI (workspace service only)
- `forge.lthn.ai/core/go-crypt` — PGP key generation (workspace service only)
- `aws-sdk-go-v2` — S3 backend
- `golang.org/x/crypto` — XChaCha20-Poly1305, BLAKE2, SHA-3 (sigil package)
- `modernc.org/sqlite` — SQLite backends (pure Go, no CGO)
- `github.com/stretchr/testify` — test assertions

### Sentinel Errors

Sentinel errors (`var ErrNotFound`, `var ErrInvalidKey`, etc.) use standard `errors.New()` — this is correct Go convention. Only inline error returns in functions should use `coreerr.E()`.

## Testing

Use `io.NewMemoryMedium()` or `io.NewSandboxed(t.TempDir())` in tests — never hit real S3/SQLite unless integration testing. S3 tests use an interface-based mock (`s3.Client`).
