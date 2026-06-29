# RFC — Windows support for go-io (local Medium)

**Status:** draft · **Author:** Cladius · **Date:** 2026-05-12
**Triggered by:** Lethean Desktop (`lthn/desktop`) Windows cross-compile blocked at `dappco.re/go/io/local/medium.go` syscall references.

## Problem

`GOOS=windows go build` fails with three error clusters in `go/local/medium.go`:

```
medium.go:465-473   syscall.Stat_t  / syscall.Lstat            [lstat helper]
medium.go:478-480   syscall.S_IFMT  / syscall.S_IFLNK           [isSymlink helper]
medium.go:482-499   syscall.Readlink                            [readlink helper]
```

The two `syscall.ENOENT` references (lines 121, 248) are **not blockers** — `syscall.ENOENT` is defined on Windows (POSIX errno compatibility) and compiles fine.

All three failing helpers are private to `package local` and called only via the three internal helpers `lstat()`, `isSymlink()`, `readlink()`. Used in `medium.go:119,127,146` from `walkSymlink()`-style traversal code.

## Scope

In: `go/local/medium.go` (the three private helpers and their type contract).

Out of scope:
- Public API changes (`Medium` interface methods stay identical)
- Symlink **creation** on Windows (the existing code only reads/walks; create paths use `os.Symlink` already)
- Windows reparse points beyond standard symlinks (junction points, mount points) — Phase 2

## Constraints

- `lstat()` currently returns `*syscall.Stat_t` — a Unix-specific struct. The replacement must be cross-platform.
- `isSymlink()` is called as `isSymlink(uint32(info.Mode))` from line 127 — the call-site casts `info.Mode` to `uint32`. The new abstraction must preserve a callable that takes the relevant Mode representation.
- The existing code lives in `package local`. No new sub-package — keep it flat.

## Design

Replace the three syscall-leaning helpers with three platform-abstracted ones, splitting Unix-specific implementations behind build tags. Match the precedent already in `dappco.re/go/process` (`pidfile_unix.go` / `pidfile_windows.go`).

### New abstraction: `linkInfo`

```go
// linkInfo carries the fields we actually inspect after lstat.
// Replaces the Unix-only *syscall.Stat_t handle so callers stay
// cross-platform.
type linkInfo struct {
    IsSymlink bool        // true if the entry is a symbolic link
    Mode      os.FileMode // standard os.FileMode (Unix mode bits + Go's portable mode bits)
    Size      int64
    ModTime   time.Time
}
```

Callers in `medium.go:119-148` only check `isSymlink(...)` and the existing flow doesn't read inode/uid/gid/etc — so the wider `syscall.Stat_t` payload isn't load-bearing. If a future caller needs Unix-only fields, add a `Stat *syscall.Stat_t` field on Unix builds (zero on Windows).

### File layout

```
go/local/
├── medium.go             # body unchanged except call sites moving to abstracted helpers
├── medium_link.go        # NEW — linkInfo type + signatures + doc
├── medium_link_unix.go   # //go:build !windows
└── medium_link_windows.go # //go:build windows
```

### Helper signatures

```go
// in medium_link.go
func lstat(path string) (linkInfo, error)
func readlink(path string) (string, error)
```

`isSymlink()` collapses into `linkInfo.IsSymlink` — no separate function needed.

### Per-platform implementations

**`medium_link_unix.go`** — `//go:build !windows`

```go
func lstat(path string) (linkInfo, error) {
    var st syscall.Stat_t
    if err := syscall.Lstat(path, &st); err != nil {
        return linkInfo{}, err
    }
    return linkInfo{
        IsSymlink: uint32(st.Mode)&syscall.S_IFMT == syscall.S_IFLNK,
        Mode:      os.FileMode(st.Mode),
        Size:      st.Size,
        ModTime:   time.Unix(st.Mtim.Sec, st.Mtim.Nsec),  // platform-specific field name; see note
    }, nil
}

func readlink(path string) (string, error) {
    // existing buffer-grow loop — call syscall.Readlink directly here
}
```

Note: `Stat_t.Mtim` vs `Stat_t.Mtimespec` varies between Linux and Darwin. Existing macOS code in this file already handles that — copy whatever convention is in use. (If unclear, use `os.Lstat()` for the timestamp and skip `syscall.Stat_t` entirely on macOS too — the only reason to keep `Stat_t` is the inode-level mode bits which we don't actually use beyond symlink detection.)

**`medium_link_windows.go`** — `//go:build windows`

```go
func lstat(path string) (linkInfo, error) {
    fi, err := os.Lstat(path)
    if err != nil {
        return linkInfo{}, err
    }
    return linkInfo{
        IsSymlink: fi.Mode()&os.ModeSymlink != 0,
        Mode:      fi.Mode(),
        Size:      fi.Size(),
        ModTime:   fi.ModTime(),
    }, nil
}

func readlink(path string) (string, error) {
    return os.Readlink(path)
}
```

### Call-site rewrites in `medium.go`

| Line | Before | After |
|---|---|---|
| 119 | `info, err := lstat(next)` (returns `*syscall.Stat_t`) | `info, err := lstat(next)` (returns `linkInfo`) — call site unchanged |
| 127 | `if !isSymlink(uint32(info.Mode)) {` | `if !info.IsSymlink {` |
| 146 | `target, err := readlink(next)` | unchanged (signature compatible) |

The 460-499 block in `medium.go` gets fully removed (lstat/isSymlink/readlink definitions moved to the new files).

### Why this shape

- **Cross-platform `os.FileMode`** carries both Unix mode bits and Go's portable mode bits (`os.ModeSymlink`, `os.ModeDir`, `os.ModeNamedPipe` etc.) — so the abstraction works on Linux/Darwin/Windows uniformly without leaking syscall types.
- **One struct, four fields** — minimal surface area; easy to extend if callers later need uid/gid (Unix-only field guarded by build tag).
- **No conditional imports in `medium.go`** — that file ends up cleanly portable.

## Tests

Existing tests should pass on Unix unchanged. For Windows:
- Add a Windows-only smoke test in `medium_link_windows_test.go` covering:
  - `lstat` on a regular file → `IsSymlink == false`, `Mode != 0`
  - `lstat` on a Windows symlink (created via `os.Symlink` in the test setup, with mklink-equivalent permissions assumed in CI) → `IsSymlink == true`
  - `lstat` on a non-existent path → error wraps `syscall.ENOENT` or `fs.ErrNotExist`
  - `readlink` round-trip via `os.Symlink` → returns the target
- Any existing symlink test that depends on Unix mode bits (e.g. asserting exact `S_IFLNK` value) — annotate `//go:build !windows`.

## Acceptance criteria

1. `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./...` succeeds.
2. `GOOS=windows go vet ./...` clean.
3. `go test ./...` passes on Linux, Darwin, and Windows (the latter via CI runner — see `lthn/desktop/.github/workflows/build.yml` for the pattern).
4. Lethean Desktop's `wails3 build GOOS=windows` reaches the next CGO/duckdb step without choking on this package.
5. No regression in symlink-walking behaviour on Unix — existing `walkSymlink` integration tests pass byte-for-byte.

## Estimated effort

**~1h** — one file split, one struct, mechanical call-site updates, Windows smoke test.

## Cross-references

- Existing platform-split precedent: `dappco.re/go/process/pidfile_{unix,windows}.go`
- Triggered from: Lethean Desktop Windows cross-compile audit (parent issue at `lthn/desktop` GitHub)
- Companion RFC: `~/Code/core/go-process/RFC.md` (the other Windows source-level blocker, larger scope)
