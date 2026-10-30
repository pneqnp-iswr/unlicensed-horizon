# v0.7.0 test report — 2026-10-30

Environment: Linux amd64, Go 1.23.2; Windows x64 cross-build target. The test environment does not have a Windows desktop, the real Rust Horizon executable, official game assets, or a GPU-backed Horizon runtime.

## Executed successfully

- `go test -count=1 ./...` — PASS, 27 named test functions across two Go packages (including table-driven diagnostic subtests).
- `go test -race -count=1 ./...` — PASS; no detected data races in the exercised Go code.
- `go vet ./...` — PASS.
- `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o unlicensed-horizon-client.exe ./cmd/uhc` — PASS; PE32+ Windows x64 binary created.
- The existing original-content lab tests create and audit 82 independently authored files plus a provenance manifest; test contract includes 19 block states and 36 PNGs.
- v0.7: matrix coverage for missing file, panic, graphics backend, nonzero exit, timeout, invalid/missing smoke JSON, and inconclusive process success; environment-token exclusion and bounded log-tail checks — PASS.
- Isolated launch tests use **synthetic shell processes**, not a real Horizon binary.

## What the new diagnostics mean

A `diagnostic_code` is a **heuristic classification of a bounded process-log tail**. A `smoke_report_json_valid` value only says the file contains a nonempty JSON object; it is not a check of Horizon's full smoke schema or evidence of rendering. A zero-exit synthetic process is explicitly classified `inconclusive`. `suggested_action` offers one possible next investigation step.

The probe inherits only selected operating-system/graphics variables and supplies private scratch home paths. **It is not a secure OS sandbox:** a malicious or compromised executable would still be unsafe to run.

## Not tested / remaining barriers

- Actual Rust `horizon-client` loading experimental assets — **NOT TESTED**.
- Graphics rendering, independent world generation, world saves, and multiplayer — **NOT TESTED**.
- Real Windows PowerShell GUI execution — **NOT TESTED**. Only Windows cross-build was verified.
- Live signed upstream engine download/install — **NOT RECHECKED** in this iteration.
- Full gameplay without a licensed Minecraft installation or proprietary resources — **NOT IMPLEMENTED**.

## Next integration step

On a dedicated test machine with a genuine Horizon executable, create the v0.7 lab using `lab-init` and execute `lab-probe` (or `PROBE-LAB-WINDOWS.bat`). Inspect `probe-result.json`, `process.log`, and optional `horizon-smoke.json` to identify the first real Rust-loader blocker. Never point a lab probe at existing personal worlds or Minecraft credentials.
