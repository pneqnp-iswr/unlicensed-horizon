# Unlicensed Horizon — fork differentiation and verification

> **Project status (2026-10-30): experimental fork and companion tooling.** This is a community fork of [horizon-rust/horizon](https://github.com/horizon-rust/horizon), not an official Horizon, Mojang, or Microsoft release. The upstream game engine remains the basis of the project. **No standalone playable Minecraft replacement has been demonstrated.**

## Purpose and scope

This fork explores an **offline-profile workflow** and an **independently authored data compatibility laboratory** for the Horizon Rust engine. The current repository-specific additions live in [`tools/unlicensed-horizon-client/`](tools/unlicensed-horizon-client/). They are built in Go and PowerShell and are *separate from* the upstream Rust game and official Horizon launcher. The word *unlicensed* refers to local profile entry without a Microsoft session; it does not grant a Minecraft license or supply proprietary game resources.

## Evidence-based differences

| Area | Upstream Horizon | This fork / companion tooling | Evidence and limits |
| --- | --- | --- | --- |
| Core engine, renderer, world mechanics | Original upstream Rust implementation | **Inherited** from upstream; not claimed as new | No custom Rust gameplay changes claimed in the initial fork-specific commit |
| Default launcher identity | Microsoft sign-in and ownership verification for a fully supported installation | **Optional separate** Go/PowerShell local-profile launcher using the engine's `--name` argument | Launcher tests pass; real Windows/GPU end-to-end boot not yet verified |
| Game assets/data | Preparation from a user's legally obtained Minecraft installation | Adds a deterministic **original-content research fixture** and validators | Laboratory fixture is incomplete, deliberately blocked from normal Play |
| Local servers | Upstream supports multiplayer, including Microsoft-authenticated online-mode | Provides quick-join targeting trusted `online-mode=false` servers | No authentication or impersonation resistance; online-mode servers still require valid credentials |
| Diagnostics | Upstream's own smoke-test flags and logs | Adds an isolated `lab-probe` harness, process timeout, JSON result, and log capture | Real Rust loader integration probe not yet executed; tests use synthetic processes |
| Release acquisition | Official launcher manages the engine | Companion installer downloads upstream binary releases and verifies pinned Ed25519 signing key + SHA-256 | Network delivery and real release were not tested in the container; verifier has unit tests |
| World storage | Standard upstream game save handling | Companion launcher separates the engine from per-user game-home and locates older saves | File/path tests pass; recovery is intentionally manual |
| Source/toolchain | Rust workspaces plus official UI | A separately buildable Go 1.22+ standard-library CLI and Windows PowerShell 5.1 GUI | Linux unit tests, `go vet` and Windows x64 cross-build pass |

## Verification matrix

| Claim | Verification level (2026-10-30) |
| --- | --- |
| Companion launcher source builds | **Verified**: `go test -count=1 ./...`, `go vet ./...`, Windows x64 cross-compilation |
| Original-content laboratory schema subset | **Verified at Go checker level**: 19 block states, 18 model bindings, 36 procedural PNGs, environment fixtures |
| Rust game loads independent laboratory | **Not verified** |
| Playable worlds with no Minecraft data | **Not implemented** |
| Windows GUI on real PC / GPU renderer | **Not verified** |
| Online-mode server authentication without license | **Not provided** |

## Intellectual-property and security policy

We do **not** redistribute Mojang Minecraft client/server jars, textures, sounds, proprietary resource exports, paid game files or Microsoft credentials. Open-source Rust code under the upstream MIT license and original Go/PowerShell launcher code are distinct from proprietary Minecraft assets. Users preparing complete official Horizon resources must use files they are authorized to access. The project does not claim to defeat server-side online-mode authentication.

## Architecture

- `crates/`, `launcher/`, `mods/`, etc. — inherited upstream Horizon source, maintain upstream compatibility.
- [`tools/unlicensed-horizon-client/`](tools/unlicensed-horizon-client/) — this fork's local-profile launcher, independent-data experiment, tests and Windows interface.
- `FORK_DIFFERENCES.md` — this document, tracking *actual code differences and validation status*.

## Development policy

1. Keep upstream credit and preserve the original MIT license and notices.
2. Keep fork-specific experiments isolated from the official launcher and game saves until a real-loader test proves compatibility.
3. Document each claimed difference with code changes, reproducible tests and limitations; never advertise the 19-state fixture as a functioning game.
4. Sync upstream changes deliberately and separately from fork-owned features, preserving a clear Git history.
5. Do not commit generated/proprietary Mojang files, credentials or compiled Windows executables to the source tree.

## Reproduce companion checks

```sh
cd tools/unlicensed-horizon-client
go test -count=1 ./...
go vet ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o unlicensed-horizon-client.exe ./cmd/uhc
```

For real integration work, run `lab-probe` on a compatible host with a real Horizon client and inspect the generated report. A Go validator's success is not proof that Horizon can load the fixture.

**Attribution:** Original Horizon engine and launcher: [horizon-rust/horizon](https://github.com/horizon-rust/horizon). Fork-specific companion tooling: this repository's maintainers. All respective licenses, trademarks and third-party rights remain with their owners.
