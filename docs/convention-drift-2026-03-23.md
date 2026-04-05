<!-- SPDX-License-Identifier: EUPL-1.2 -->

# Convention Drift Audit

Date: 2026-03-23

Scope: tracked module files in the main repo surface (`*.go`, `*.md`), excluding `.core/`, `.github/`, `.idea/`, `go.mod`, `go.sum`, and generated coverage output.

Conventions used: `CLAUDE.md`, `docs/development.md`, `docs/index.md`, and `docs/architecture.md`.

Limitation: `CODEX.md` is not present in this repository. The `stdlib -> core.*` and usage-example findings below are therefore inferred from the documented guidance already in-tree.

## Missing SPDX Headers

- `CLAUDE.md:1`
- `bench_test.go:1`
- `client_test.go:1`
- `datanode/client.go:1`
- `datanode/client_test.go:1`
- `docs/architecture.md:1`
- `docs/development.md:1`
- `docs/index.md:1`
- `io.go:1`
- `local/client.go:1`
- `local/client_test.go:1`
- `node/node.go:1`
- `node/node_test.go:1`
- `s3/s3.go:1`
- `s3/s3_test.go:1`
- `sigil/crypto_sigil.go:1`
- `sigil/crypto_sigil_test.go:1`
- `sigil/sigil.go:1`
- `sigil/sigil_test.go:1`
- `sigil/sigils.go:1`
- `sqlite/sqlite.go:1`
- `sqlite/sqlite_test.go:1`
- `store/medium.go:1`
- `store/medium_test.go:1`
- `store/store.go:1`
- `store/store_test.go:1`
- `workspace/service.go:1`
- `workspace/service_test.go:1`

## `stdlib -> core.*` Drift

Interpretation note: `CLAUDE.md` only makes one direct stdlib replacement rule explicit: do not use raw `os` / `filepath` outside the backend boundary. The concrete drift in this repo therefore falls into two buckets: stale pre-`forge.lthn.ai` core import paths, and direct host-filesystem/path handling in non-backend production code.

- `go.mod:1` still declares `module dappco.re/go/core/io` while the repo documentation identifies the module as `forge.lthn.ai/core/go-io`.
- `go.mod:6` still depends on `dappco.re/go/core` while the repo docs list `forge.lthn.ai/core/go` as the current Core dependency.
- `io.go:12` imports `dappco.re/go/core/io/local` instead of the documented `forge.lthn.ai/core/go-io/local`.
- `node/node.go:18` imports `dappco.re/go/core/io` instead of the documented `forge.lthn.ai/core/go-io`.
- `workspace/service.go:10` imports `dappco.re/go/core` instead of the documented Core package path.
- `workspace/service.go:13` imports `dappco.re/go/core/io` instead of the documented `forge.lthn.ai/core/go-io`.
- `workspace/service_test.go:7` still imports `dappco.re/go/core`.
- `datanode/client_test.go:7` still imports `dappco.re/go/core/io`.
- `workspace/service.go:6` uses raw `os.UserHomeDir()` in non-backend production code, despite the repo guidance that filesystem access must go through the `io.Medium` abstraction.
- `workspace/service.go:7` builds runtime filesystem paths with `filepath.Join()` in non-backend production code, again bypassing the documented abstraction boundary.

## UK English Drift

- `datanode/client.go:3` uses `serializes`; `docs/development.md` calls for UK English (`serialises`).
- `datanode/client.go:52` uses `serializes`; `docs/development.md` calls for UK English (`serialises`).
- `sigil/crypto_sigil.go:3` uses `defense-in-depth`; `docs/development.md` calls for UK English (`defence-in-depth`).
- `sigil/crypto_sigil.go:38` uses `defense`; `docs/development.md` calls for UK English (`defence`).

## Missing Tests

Basis: `GOWORK=off go test -coverprofile=coverage.out ./...` and `go tool cover -func=coverage.out` on 2026-03-23. This list focuses on public or semantically meaningful API entrypoints at `0.0%` coverage and omits trivial one-line accessor helpers.

- `io.go:126` `NewSandboxed`
- `io.go:143` `ReadStream`
- `io.go:148` `WriteStream`
- `io.go:208` `(*MockMedium).WriteMode`
- `io.go:358` `(*MockMedium).Open`
- `io.go:370` `(*MockMedium).Create`
- `io.go:378` `(*MockMedium).Append`
- `io.go:388` `(*MockMedium).ReadStream`
- `io.go:393` `(*MockMedium).WriteStream`
- `datanode/client.go:138` `(*Medium).WriteMode`
- `local/client.go:231` `(*Medium).Append`
- `node/node.go:128` `(*Node).WalkNode`
- `node/node.go:218` `(*Node).CopyTo`
- `node/node.go:349` `(*Node).Read`
- `node/node.go:359` `(*Node).Write`
- `node/node.go:365` `(*Node).WriteMode`
- `node/node.go:370` `(*Node).FileGet`
- `node/node.go:375` `(*Node).FileSet`
- `node/node.go:380` `(*Node).EnsureDir`
- `node/node.go:393` `(*Node).IsFile`
- `node/node.go:400` `(*Node).IsDir`
- `node/node.go:411` `(*Node).Delete`
- `node/node.go:421` `(*Node).DeleteAll`
- `node/node.go:445` `(*Node).Rename`
- `node/node.go:461` `(*Node).List`
- `node/node.go:473` `(*Node).Create`
- `node/node.go:480` `(*Node).Append`
- `node/node.go:491` `(*Node).ReadStream`
- `node/node.go:500` `(*Node).WriteStream`
- `s3/s3.go:55` `WithClient`
- `store/medium.go:37` `(*Medium).Store`
- `store/medium.go:80` `(*Medium).EnsureDir`
- `store/medium.go:95` `(*Medium).FileGet`
- `store/medium.go:100` `(*Medium).FileSet`
- `store/medium.go:246` `(*Medium).ReadStream`
- `store/medium.go:259` `(*Medium).WriteStream`
- `workspace/service.go:150` `(*Service).HandleIPCEvents`

## Missing Usage-Example Comments

Interpretation note: because `CODEX.md` is absent, this section flags public entrypoints that expose the package's main behaviour but do not have a nearby comment block showing concrete usage. `sigil/sigil.go` is the only production file in the repo that currently includes an explicit `Example usage:` comment block.

- `io.go:123` `NewSandboxed`
- `local/client.go:22` `New`
- `s3/s3.go:68` `New`
- `sqlite/sqlite.go:35` `New`
- `node/node.go:32` `New`
- `node/node.go:217` `CopyTo`
- `datanode/client.go:32` `New`
- `datanode/client.go:40` `FromTar`
- `store/store.go:21` `New`
- `store/store.go:124` `Render`
- `store/medium.go:22` `NewMedium`
- `workspace/service.go:39` `New`
- `sigil/crypto_sigil.go:247` `NewChaChaPolySigil`
- `sigil/crypto_sigil.go:263` `NewChaChaPolySigilWithObfuscator`
