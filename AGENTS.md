# AGENTS.md

This repository implements `dappco.re/go/io`, the Core I/O abstraction layer.
Agents working here should treat `Medium` as the central contract and should
avoid bypassing it with direct filesystem or JSON convenience imports.

## Repository Shape

The root package defines the shared interface, in-memory medium, action
registration, mock medium, and cross-medium helpers. Backend packages sit under
their storage domain:

- `local`, `s3`, `sqlite`, `node`, `datanode`, `store`, `cube`, and `workspace`
  are first-class backend or service packages.
- `pkg/medium/*` contains protocol integrations for SFTP, WebDAV, GitHub, and
  PWA scraping.
- `pkg/api` adapts the library surface into HTTP routes for Core providers.
- `sigil` contains transformation primitives used by encryption and archive
  flows.
- `internal/fsutil` holds shared filesystem helpers that should not become part
  of the public API.

## Compliance Rules

Every public function or method in a production `<file>.go` has its tests in the
matching `<file>_test.go` file. The required triplets are named
`Test<File>_<Symbol>_Good`, `Test<File>_<Symbol>_Bad`, and
`Test<File>_<Symbol>_Ugly`. Do not create AX7, versioned, generated, or
catch-all test files.

Examples live beside the source in `<file>_example_test.go`. They use
`Println` from `dappco.re/go`, not `fmt`.

The upgrade audit also rejects direct imports or exact quoted use of wrapped
stdlib names such as `fmt`, `errors`, `strings`, `path`, `os`, `encoding/json`,
and `bytes`. Use Core wrappers such as `core.E`, `core.NewError`, `core.Is`,
`core.JSONUnmarshal`, `core.Sprintf`, `core.NewBuffer`, string helpers, and path
helpers.

## Local Workflow

Before handing work back, run:

```bash
GOWORK=off go mod tidy
GOWORK=off go vet ./...
GOWORK=off go test -count=1 ./...
gofmt -l .
bash /Users/snider/Code/core/go/tests/cli/v090-upgrade/audit.sh .
```

The target state is a clean audit with `verdict: COMPLIANT` and zero counters.
Do not touch `BRIEF.md`, `.git/`, `.codex/`, or any `third_party` directory.
