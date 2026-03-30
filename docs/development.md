---
title: Development
description: How to build, test, and contribute to go-io.
---

# Development

This guide covers everything needed to work on `go-io` locally.


## Prerequisites

- **Go 1.26.0** or later
- **No C compiler required** -- all dependencies (including SQLite) are pure Go
- The module is part of the Go workspace at `~/Code/go.work`. If you are working outside that workspace, ensure `GOPRIVATE=forge.lthn.ai/*` is set so the Go toolchain can fetch private dependencies.


## Building

`go-io` is a library with no binary output. To verify it compiles:

```bash
cd /path/to/go-io
go build ./...
```

If using the Core CLI:

```bash
core go fmt       # format
core go vet       # static analysis
core go lint      # linter
core go test      # run all tests
core go qa        # fmt + vet + lint + test
core go qa full   # + race detector, vulnerability scan, security audit
```


## Running Tests

All packages have thorough test suites. Tests use `testify/assert` and `testify/require` for assertions.

```bash
# All tests
go test ./...

# A single package
go test ./sigil/

# A single test by name
go test ./local/ -run TestValidatePath_Security

# With race detector
go test -race ./...

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

Or via the Core CLI:

```bash
core go test
core go test --run TestChaChaPolySigil_Good_RoundTrip
core go cov --open
```


## Test Naming Convention

Tests follow the `_Good`, `_Bad`, `_Ugly` suffix pattern:

| Suffix | Meaning |
|--------|---------|
| `_Good` | Happy path -- the operation succeeds as expected |
| `_Bad` | Expected error conditions -- missing files, invalid input, permission denied |
| `_Ugly` | Edge cases and boundary conditions -- nil input, empty paths, panics |

Example:

```go
func TestDelete_Good(t *testing.T)           { /* deletes a file successfully */ }
func TestDelete_Bad_NotFound(t *testing.T)   { /* returns error for missing file */ }
func TestDelete_Bad_DirNotEmpty(t *testing.T) { /* returns error for non-empty dir */ }
```


## Writing Tests Against Medium

Use `MockMedium` from the root package for unit tests that need a storage backend but should not touch disk:

```go
func TestMyFeature(t *testing.T) {
    m := io.NewMockMedium()
    m.Files["config.yaml"] = "key: value"
    m.Dirs["data"] = true

    // Your code under test receives m as an io.Medium
    result, err := myFunction(m)
    assert.NoError(t, err)
    assert.Equal(t, "expected", m.Files["output.txt"])
}
```

For tests that need a real but ephemeral filesystem, use `local.New` with `t.TempDir()`:

```go
func TestWithRealFS(t *testing.T) {
    m, err := local.New(t.TempDir())
    require.NoError(t, err)

    _ = m.Write("file.txt", "hello")
    content, _ := m.Read("file.txt")
    assert.Equal(t, "hello", content)
}
```

For SQLite-backed tests, use `:memory:`:

```go
func TestWithSQLite(t *testing.T) {
    m, err := sqlite.New(sqlite.Options{Path: ":memory:"})
    require.NoError(t, err)
    defer m.Close()

    _ = m.Write("file.txt", "hello")
}
```


## Adding a New Backend

To add a new `Medium` implementation:

1. Create a new package directory (e.g., `sftp/`).
2. Define a struct that implements all 18 methods of `io.Medium`.
3. Add a compile-time check at the top of your file:

```go
var _ coreio.Medium = (*Medium)(nil)
```

4. Normalise paths using `path.Clean("/" + p)` to prevent traversal escapes. This is the convention followed by every existing backend.
5. Handle `nil` and empty input consistently: check how `MockMedium` and `local.Medium` behave and match that behaviour.
6. Write tests using the `_Good` / `_Bad` / `_Ugly` naming convention.
7. Add your package to the table in `docs/index.md`.


## Adding a New Sigil

To add a new data transformation:

1. Create a struct in `sigil/` that implements the `Sigil` interface (`In` and `Out`).
2. Handle `nil` input by returning `nil, nil`.
3. Handle empty input by returning `[]byte{}, nil`.
4. Register it in the `NewSigil` factory function in `sigils.go`.
5. Add tests covering `_Good` (round-trip), `_Bad` (invalid input), and `_Ugly` (nil/empty edge cases).


## Code Style

- **UK English** in comments and documentation: colour, organisation, centre, serialise, defence.
- **`declare(strict_types=1)`** equivalent: all functions have explicit parameter and return types.
- Errors use the `go-log` helper: `coreerr.E("package.Method", "what failed", underlyingErr)`.
- No blank imports except for database drivers (`_ "modernc.org/sqlite"`).
- Formatting: standard `gofmt` / `goimports`.


## Project Structure

```
go-io/
├── io.go               # Medium interface, helpers, MockMedium
├── client_test.go      # Tests for MockMedium and helpers
├── bench_test.go       # Benchmarks
├── go.mod
├── local/
│   ├── client.go       # Local filesystem backend
│   └── client_test.go
├── s3/
│   ├── s3.go           # S3 backend
│   └── s3_test.go
├── sqlite/
│   ├── sqlite.go       # SQLite virtual filesystem
│   └── sqlite_test.go
├── node/
│   ├── node.go         # In-memory fs.FS + Medium
│   └── node_test.go
├── datanode/
│   ├── client.go       # Borg DataNode Medium wrapper
│   └── client_test.go
├── store/
│   ├── store.go        # KV store
│   ├── medium.go       # Medium adapter for KV store
│   ├── store_test.go
│   └── medium_test.go
├── sigil/
│   ├── sigil.go        # Sigil interface, Transmute/Untransmute
│   ├── sigils.go       # Built-in sigils (hex, base64, gzip, hash, etc.)
│   ├── crypto_sigil.go # ChaChaPolySigil + obfuscators
│   ├── sigil_test.go
│   └── crypto_sigil_test.go
├── workspace/
│   ├── service.go      # Encrypted workspace service
│   └── service_test.go
├── docs/               # This documentation
└── .core/
    ├── build.yaml      # Build configuration
    └── release.yaml    # Release configuration
```


## Licence

EUPL-1.2
