# v0.6.0 test report — 2026-10-30

Environment: Linux amd64, Go 1.23.2; cross-compilation target Windows x64. No Windows GUI, Rust compiler, actual Horizon game executable, graphics device, official Minecraft files, or Mojang-provided resources are available in this environment.

## Executed

- `go test -count=1 ./...` — PASS (25 named test cases across 2 Go packages).
- `go vet ./...` — PASS.
- `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o unlicensed-horizon-client.exe ./cmd/uhc` — PASS; inspected PE32+ Windows x64 output.
- `go run ./cmd/uhc lab-init examples/original-resource-lab-v0.6` — PASS. Generated **82** independent files plus `open-content-lab.json`.
- `go run ./cmd/uhc lab-audit examples/original-resource-lab-v0.6` — PASS. Version 3 lab manifest, checksum and subset-contract checks passed.
- `go run ./cmd/uhc lab-contract examples/original-resource-lab-v0.6` — PASS. 19 states, 19 block names, 3 required tags, 18 model bindings, 36 decoded PNG files, no subset issues. Environment timeline and dimension contract checked.
- New mocked subprocess tests: `TestSafeProbeEnvRedactsAuthenticationAndHome`, `TestProbeLabIsolationAndReports`, `TestProbeLabFailsClosed`, `TestProbeLabCannotWriteInsideSource`, `TestV3EnvironmentContractRejectsCorruptTimeline` — PASS. Tests use synthetic executable shell scripts, **not Horizon**.

## NOT tested / limitations

- Real `horizon-client.exe` executing a laboratory boot probe — **NOT RUN**. The command is implemented but must be executed on a user's own GPU-equipped Windows machine. The Go tests exercise its isolation/copy/report pipeline using mock processes.
- Rust `Assets::load`, `RenderAssets::load`, full `Game::load_resources`, title UI, graphics, terrain, saves, multiplayer — **NOT TESTED** with v3 original content. Real game playability is not established.
- PowerShell GUI runtime and actual Windows EXE execution — **NOT TESTED** on Windows (cross-compiled only).
- `go test -race` — **not completed in v0.6**: race-enabled Go build exceeded test execution time limit; regular tests and vet passed. Do not interpret this as a passed race check.
- No live GitHub engine download or real-release Ed25519 signature validation took place in this cycle (existing unit tests use synthetic signed fixtures).
- No Mojang data is distributed. This is not a complete unlicensed game build.

## Upstream source references verified on 2026-10-30

- https://raw.githubusercontent.com/horizon-rust/horizon/main/crates/horizon-client/src/game.rs (`Options::read`, synchronous startup, resource loading)
- https://raw.githubusercontent.com/horizon-rust/horizon/main/crates/horizon-client-render/src/environment.rs (`Environment::load_dimension`)
- https://raw.githubusercontent.com/horizon-rust/horizon/main/crates/horizon-client-render/src/assets.rs (`Assets::load`)

## Next concrete integration step

On a user Windows PC with a working Horizon executable: create v0.6 lab, run `PROBE-LAB-WINDOWS.bat`, then inspect `process.log` and optional `horizon-smoke.json`. Capture the first genuine Rust loader error and iterate based on that evidence. The probe is strictly separate from normal `PLAY` and preserves personal worlds.
