---
title: Architecture
description: Internal design of go-io — the Medium interface, backend implementations, sigil transformation pipeline, and security model.
---

# Architecture

This document explains how `go-io` is structured internally, how the key types relate to one another, and how data flows through the system.


## Design Principles

1. **One interface, many backends.** All storage operations go through `Medium`. Business logic never imports a specific backend.
2. **Path sandboxing by default.** The local backend validates every path against symlink escapes. Sandboxed mediums cannot reach outside their root.
3. **Composable transforms.** The `sigil` package lets you chain encoding, compression, and encryption steps into reversible pipelines.
4. **No CGO.** The SQLite driver (`modernc.org/sqlite`) is pure Go. The entire module compiles without a C toolchain.


## Core Types

### Medium (root package)

The `Medium` interface is defined in `io.go`. It is the only type that consuming code needs to know about. The root package also provides:

- **`io.Local`** — a package-level variable initialised in `init()` via `local.New("/")`. This gives unsandboxed access to the host filesystem, mirroring the behaviour of the standard `os` package.
- **`io.NewSandboxed(root)`** — creates a `local.Medium` restricted to `root`. All path resolution is confined within that directory.
- **`io.Copy(src, srcPath, dst, dstPath)`** — copies a file between any two mediums by reading from one and writing to the other.
- **`io.NewMemoryMedium()`** — a fully functional in-memory implementation for unit tests. It tracks files, directories, and modification times in plain maps.

### FileInfo and DirEntry (root package)

Simple struct implementations of `fs.FileInfo` and `fs.DirEntry` are exported from the root package for use in mocks and tests. Each backend also defines its own unexported equivalents internally.


## Backend Implementations

### local.Medium

**File:** `local/medium.go`

The local backend wraps the standard `os` package with two layers of path protection:

1. **`path(p string)`** — normalises paths using `filepath.Clean("/" + p)` before joining with the root. This neutralises `..` traversal at the string level. When root is `"/"`, absolute paths pass through unchanged and relative paths resolve against the working directory.

2. **`validatePath(p string)`** — walks each path component, calling `filepath.EvalSymlinks` at every step. If any resolved component lands outside the root, it logs a security event to stderr (including timestamp, root, attempted path, and OS username) and returns `os.ErrPermission`.

Delete operations refuse to remove `"/"` or the user's home directory as an additional safety rail.

```
caller -> path(p) -> validatePath(p) -> os.ReadFile / os.WriteFile / ...
```

### s3.Medium

**File:** `s3/s3.go`

The S3 backend translates `Medium` operations into AWS SDK calls. Key design decisions:

- **Key construction:** `key(p)` uses `path.Clean("/" + p)` to sandbox traversal, then prepends the optional prefix. This means `../secret` resolves to `secret`, not an escape.
- **Directory semantics:** S3 has no real directories. `EnsureDir` is a no-op. `IsDir` and `Exists` for directory-like paths use `ListObjectsV2` with `MaxKeys: 1` to check for objects under the prefix.
- **Rename:** Implemented as copy-then-delete, since S3 has no atomic rename.
- **Append:** Downloads existing content, appends in memory, re-uploads on `Close()`. This is the only viable approach given S3's immutable-object model.
- **Testability:** The `Client` interface abstracts the six SDK methods used. Tests inject a `mockS3` that stores objects in a `map[string][]byte` with a `sync.RWMutex`.

### sqlite.Medium

**File:** `sqlite/sqlite.go`

Stores files and directories as rows in a single SQLite table:

```sql
CREATE TABLE IF NOT EXISTS files (
    path    TEXT PRIMARY KEY,
    content BLOB NOT NULL,
    mode    INTEGER DEFAULT 420,   -- 0644
    is_dir  BOOLEAN DEFAULT FALSE,
    mtime   DATETIME DEFAULT CURRENT_TIMESTAMP
)
```

- **WAL mode** is enabled at connection time for better concurrent read performance.
- **Path cleaning** uses the same `path.Clean("/" + p)` pattern as other backends.
- **Rename** is transactional: it reads the source row, inserts at the destination, deletes the source, and moves all children (if it is a directory) within a single transaction.
- **Custom tables** are supported via `sqlite.Options{Path: ":memory:", Table: "name"}` to allow multiple logical filesystems in one database.
- **`:memory:`** databases work out of the box for tests.

### node.Node

**File:** `node/node.go`

A pure in-memory filesystem that implements both `Medium` and the standard library's `fs.FS`, `fs.StatFS`, `fs.ReadDirFS`, and `fs.ReadFileFS` interfaces. Directories are implicit -- they exist whenever a stored file path contains a `"/"`.

Key capabilities beyond `Medium`:

- **`ToTar()` / `FromTar()`** — serialise the entire tree to a tar archive and back. This enables snapshotting, transport, and archival.
- **`Walk()` with `WalkOptions`** — extends `fs.WalkDir` with `MaxDepth`, `Filter`, and `SkipErrors` controls.
- **`CopyFile(src, dst, perm)`** — copies a file from the in-memory tree to the real filesystem.
- **`CopyTo(target Medium, src, dst)`** — copies a file or directory tree to any other `Medium`.
- **`ReadFile(name)`** — returns a defensive copy of file content, preventing callers from mutating internal state.

### datanode.Medium

**File:** `datanode/medium.go`

A thread-safe `Medium` backed by Borg's `DataNode` (an in-memory `fs.FS` with tar serialisation). It adds:

- **`sync.RWMutex`** on every operation for concurrent safety.
- **Explicit directory tracking** via a `map[string]bool`, since `DataNode` only stores files.
- **`Snapshot()` / `Restore(data)`** — serialise and deserialise the entire filesystem as a tarball.
- **`DataNode()`** — exposes the underlying Borg DataNode for integration with TIM containers.
- **File deletion** is handled by rebuilding the DataNode without the target file, since the upstream type does not expose a `Remove` method.


## store Package

**Files:** `store/store.go`, `store/medium.go`

The store package provides two complementary APIs:

### Store (key-value)

A group-namespaced key-value store backed by SQLite:

```sql
CREATE TABLE IF NOT EXISTS kv (
    grp   TEXT NOT NULL,
    key   TEXT NOT NULL,
    value TEXT NOT NULL,
    PRIMARY KEY (grp, key)
)
```

Operations: `Get`, `Set`, `Delete`, `Count`, `DeleteGroup`, `GetAll`, `Render`.

The `Render` method loads all key-value pairs from a group into a `map[string]string` and executes a Go `text/template` against them:

```go
s.Set("user", "pool", "pool.lthn.io:3333")
s.Set("user", "wallet", "iz...")
out, _ := s.Render(`{"pool":"{{ .pool }}"}`, "user")
// out: {"pool":"pool.lthn.io:3333"}
```

### store.Medium (Medium adapter)

Wraps a `Store` to satisfy the `Medium` interface. Paths are split as `group/key`:

- `Read("config/theme")` calls `Get("config", "theme")`
- `List("")` returns all groups as directories
- `List("config")` returns all keys in the `config` group as files
- `IsDir("config")` returns true if the group has entries

You can create it directly (`NewMedium(":memory:")`) or adapt an existing store (`store.AsMedium()`).


## sigil Package

**Files:** `sigil/sigil.go`, `sigil/sigils.go`, `sigil/crypto_sigil.go`

The sigil package implements composable, reversible data transformations.

### Interface

```go
type Sigil interface {
    In(data []byte) ([]byte, error)   // forward transform
    Out(data []byte) ([]byte, error)  // reverse transform
}
```

Contracts:
- Reversible sigils: `Out(In(x)) == x`
- Irreversible sigils (hashes): `Out` returns input unchanged
- Symmetric sigils (reverse): `In(x) == Out(x)`
- `nil` input returns `nil` without error
- Empty input returns empty without error

### Available Sigils

Created via `NewSigil(name)`:

| Name | Type | Reversible |
|------|------|------------|
| `reverse` | Byte reversal | Yes (symmetric) |
| `hex` | Hexadecimal encoding | Yes |
| `base64` | Base64 encoding | Yes |
| `gzip` | Gzip compression | Yes |
| `json` | JSON compaction | No (`Out` is passthrough) |
| `json-indent` | JSON pretty-printing | No (`Out` is passthrough) |
| `md4`, `md5`, `sha1` | Legacy hashes | No |
| `sha224` .. `sha512` | SHA-2 family | No |
| `sha3-224` .. `sha3-512` | SHA-3 family | No |
| `sha512-224`, `sha512-256` | Truncated SHA-512 | No |
| `ripemd160` | RIPEMD-160 | No |
| `blake2s-256` | BLAKE2s | No |
| `blake2b-256` .. `blake2b-512` | BLAKE2b | No |

### Pipeline Functions

```go
// Apply sigils left-to-right.
encoded, _ := sigil.Transmute(data, []sigil.Sigil{gzipSigil, hexSigil})

// Reverse sigils right-to-left.
original, _ := sigil.Untransmute(encoded, []sigil.Sigil{gzipSigil, hexSigil})
```

### Authenticated Encryption: ChaChaPolySigil

`ChaChaPolySigil` provides XChaCha20-Poly1305 authenticated encryption with a pre-obfuscation layer. It implements the `Sigil` interface, so it composes naturally into pipelines.

**Encryption flow:**

```
plaintext -> obfuscate(nonce) -> XChaCha20-Poly1305 encrypt -> [nonce || ciphertext || tag]
```

**Decryption flow:**

```
[nonce || ciphertext || tag] -> decrypt -> deobfuscate(nonce) -> plaintext
```

The pre-obfuscation layer ensures that raw plaintext patterns are never sent directly to CPU encryption routines, providing defence-in-depth against side-channel attacks. Two obfuscators are provided:

- **`XORObfuscator`** (default) — XORs data with a SHA-256 counter-mode key stream derived from the nonce.
- **`ShuffleMaskObfuscator`** — applies XOR masking followed by a deterministic Fisher-Yates byte shuffle, making both value and position analysis more difficult.

```go
key := make([]byte, 32)
rand.Read(key)

s, _ := sigil.NewChaChaPolySigil(key)
ciphertext, _ := s.In([]byte("secret"))
plaintext, _ := s.Out(ciphertext)

// With stronger obfuscation:
s2, _ := sigil.NewChaChaPolySigilWithObfuscator(key, &sigil.ShuffleMaskObfuscator{})
```

Each call to `In` generates a fresh random nonce, so encrypting the same plaintext twice produces different ciphertexts.


## workspace Package

**File:** `workspace/service.go`

A higher-level service that integrates with the Core DI container (`forge.lthn.ai/core/go`). It manages encrypted workspaces stored under `~/.core/workspaces/`.

Each workspace:
- Is identified by a SHA-256 hash of the user-provided identifier
- Contains subdirectories: `config/`, `log/`, `data/`, `files/`, `keys/`
- Has a PGP keypair generated via the Core crypt service
- Supports file get/set operations on the `files/` subdirectory
- Handles IPC events (`workspace.create`, `workspace.switch`) for integration with the Core message bus

The workspace service implements `core.Workspace` and uses `io.Local` as its storage medium.


## Data Flow Summary

```
Application code
       |
       v
   io.Medium (interface)
       |
       +-- local.Medium  --> os package (with sandbox validation)
       +-- s3.Medium     --> AWS SDK S3 client
       +-- sqlite.Medium --> modernc.org/sqlite
       +-- node.Node     --> in-memory map + tar serialisation
       +-- datanode.Medium --> Borg DataNode + sync.RWMutex
       +-- store.Medium  --> store.Store (SQLite KV) --> Medium adapter
       +-- MemoryMedium   --> map[string]string (for tests)
```

Every backend normalises paths using the same `path.Clean("/" + p)` pattern, ensuring consistent behaviour regardless of which backend is in use.
