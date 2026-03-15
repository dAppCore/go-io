# go-io

Module: `forge.lthn.ai/core/go-io`

File I/O abstraction layer providing a `Medium` interface for interchangeable storage backends. Includes local filesystem, S3, SQLite, and in-memory mock implementations. Used throughout the Core ecosystem for file operations.

## Architecture

| File/Dir | Purpose |
|----------|---------|
| `io.go` | `Medium` interface, `Local` singleton, `NewSandboxed()`, `MockMedium`, helper functions |
| `local/client.go` | Local filesystem `Medium` implementation |
| `s3/s3.go` | S3-compatible storage `Medium` |
| `sqlite/sqlite.go` | SQLite-backed `Medium` |
| `store/medium.go` | Store medium abstraction |
| `store/store.go` | Key-value store interface |
| `sigil/` | Cryptographic sigil system (content addressing) |
| `datanode/client.go` | Data node client |
| `node/node.go` | Node abstraction |
| `workspace/service.go` | Workspace service |

## Key Types

### Medium Interface

```go
type Medium interface {
    Read(path string) (string, error)
    Write(path, content string) error
    EnsureDir(path string) error
    IsFile(path string) bool
    FileGet(path string) (string, error)
    FileSet(path, content string) error
    Delete(path string) error
    DeleteAll(path string) error
    Rename(oldPath, newPath string) error
    List(path string) ([]fs.DirEntry, error)
    Stat(path string) (fs.FileInfo, error)
    Open(path string) (fs.File, error)
    Create(path string) (io.WriteCloser, error)
    Append(path string) (io.WriteCloser, error)
    ReadStream(path string) (io.ReadCloser, error)
    WriteStream(path string) (io.WriteCloser, error)
    Exists(path string) bool
    IsDir(path string) bool
}
```

### Implementations

- **`io.Local`** — Pre-initialised local filesystem medium (root: "/")
- **`NewSandboxed(root)`** — Sandboxed local filesystem restricted to a root directory
- **`MockMedium`** — In-memory implementation for testing. Tracks files, dirs, and modification times.

### Helper Functions

- `Read(m, path)`, `Write(m, path, content)` — Convenience wrappers
- `ReadStream(m, path)`, `WriteStream(m, path)` — Stream wrappers
- `EnsureDir(m, path)`, `IsFile(m, path)` — Wrappers
- `Copy(src, srcPath, dst, dstPath)` — Cross-medium file copy

### Testing Types

- **`FileInfo`** — Simple `fs.FileInfo` implementation
- **`DirEntry`** — Simple `fs.DirEntry` implementation
- **`MockFile`** — `fs.File` for `MockMedium`
- **`MockWriteCloser`** — `io.WriteCloser` for `MockMedium`

## Usage

```go
import io "forge.lthn.ai/core/go-io"

// Use pre-initialised local filesystem
content, _ := io.Local.Read("/etc/hostname")
io.Local.Write("/tmp/output.txt", "hello")

// Sandboxed medium
sandbox, _ := io.NewSandboxed("/app/data")
sandbox.Write("config.yaml", yamlContent)

// Testing
mock := io.NewMockMedium()
mock.Files["/test.txt"] = "content"
mock.Dirs["/data"] = true
```

## Dependencies

- `forge.lthn.ai/core/go-log` — Error handling (`log.E()`)
