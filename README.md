# go-io

`dappco.re/go/io` is the Core storage boundary. It gives callers one `Medium`
interface for file-like operations while letting the backing store be local
disk, memory, S3, SQLite, WebDAV, SFTP, GitHub, a Borg DataNode, or an encrypted
workspace.

The point of the module is containment. Application code can read, write, list,
stream, rename, and delete through a `Medium` without taking a direct dependency
on host filesystem APIs. Backends decide how paths are normalized, how metadata
is represented, and which operations are no-ops or remote calls.

## Quick Start

```go
memory := io.NewMemoryMedium()
_ = memory.Write("config/app.yaml", "port: 8080")

content, err := io.Read(memory, "config/app.yaml")
if err != nil {
    return err
}
_ = content
```

For host files, use `io.NewSandboxed(root)` or `local.New(root)` so paths stay
under an explicit root. `io.Local` exists for process-level access, but service
code should prefer scoped mediums.

## Packages

- `io`: the `Medium` interface, copy helpers, in-memory medium, file info and
  directory entry helpers, and Core action registration.
- `local`: sandboxed local filesystem backend.
- `s3`, `pkg/medium/sftp`, `pkg/medium/webdav`, `pkg/medium/github`,
  `pkg/medium/pwa`: remote or protocol-backed mediums.
- `sqlite` and `store`: database-backed storage and key-value storage.
- `node` and `datanode`: in-memory tree backends with tar snapshot support.
- `cube`: encrypted medium wrapper and archive packing helpers.
- `sigil`: reversible and one-way byte transformations used by cube and
  workspace encryption.
- `workspace`: medium-backed workspace services and RFC command DTOs.
- `pkg/api`: Gin provider exposing the I/O action and medium surfaces over HTTP.

## Development Gate

Run the repository with workspace mode disabled:

```bash
GOWORK=off go mod tidy
GOWORK=off go vet ./...
GOWORK=off go test -count=1 ./...
gofmt -l .
bash /Users/snider/Code/core/go/tests/cli/v090-upgrade/audit.sh .
```

The audit is part of the contract. Public symbols need sibling triplet tests and
sibling examples, test files use Core assertions, and direct imports of wrapped
stdlib packages are not accepted in repository Go files.
