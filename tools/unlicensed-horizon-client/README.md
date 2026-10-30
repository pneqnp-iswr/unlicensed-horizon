# unlicensed-horizon-client v0.7.0

Independent local-profile launcher for the MIT-licensed [Horizon](https://github.com/horizon-rust/horizon) game engine. Windows x64 release; CLI code is in Go (standard library); GUI uses PowerShell 5.1+ Windows Forms. Not affiliated with Horizon, Mojang or Microsoft.

**Important:** This application does not verify a Microsoft/Minecraft account and can start Horizon with an offline nickname (`--name`). It is **NOT a ready-to-play unlicensed copy of Minecraft**. Horizon requires a complete bundle of prepared Minecraft 26.3 data from a legally obtained copy. It cannot be reproduced merely by supplying a texture pack. This ZIP contains no Minecraft client/server jars, proprietary data, sounds, textures, session tokens or bypass for online-mode accounts. To reach a genuinely standalone playable open-content game, the assets and generated gameplay data would require an independent implementation plus testing against Horizon's loaders.

> **Repository integration:** This folder is a self-contained Go/PowerShell addon inside the [unlicensed-horizon fork](https://github.com/pneqnp-iswr/unlicensed-horizon). It is **not** a modification to the original Rust game. Pre-generated `examples/` and Windows binaries from the earlier downloadable ZIP are deliberately not committed here; run `lab-init` to regenerate research fixtures and build the EXE locally. See the root [`FORK_DIFFERENCES.md`](../../FORK_DIFFERENCES.md) for an evidence-based comparison.

## Quick start

1. Unzip on Windows 10/11 x64. Double-click `START-WINDOWS.bat`.
2. Set nickname (3-16 ASCII letters/digits/underscores). Leave **World** and **Server** empty to open the main menu, or specify **one** of them (not both). `Server` is for trusted `online-mode=false` servers only.
3. Click **Install Engine** to fetch a signed Horizon game release from the Horizon GitHub releases; Ed25519 signature and SHA-256 are verified. This downloads only the open game binaries.
4. **Find Prepared Data** or **Browse** to select the prepared Horizon `26.3` resource directory, generated from your Minecraft copy via `horizon-prepare`. **Audit Resources** identifies missing/invalid critical files. A passing audit does *not* guarantee the full asset set is complete.
5. **Check Setup**, then **PLAY**. Examine **View Game Logs** after errors.

For the full legal preparation procedure using your own Minecraft copy, see Horizon's [README](https://github.com/horizon-rust/horizon/blob/main/README.md). As of 2026-10-30 it uses Minecraft Java Edition 26.3 and JDK 25 for data preparation.

## v0.7: Classified diagnostics (building on the v0.6 laboratory)

The **new v3 laboratory** contains 82 independent files plus its provenance manifest (19 block states, 18 model bindings, 36 procedural PNG files). Two new original JSON files implement the **shape** of `data/minecraft/timeline/day.json` and `data/minecraft/dimension_type/overworld.json`, as directly read by Horizon's `Environment::load_dimension`. They have new semantic checks for period/track/keyframe and dimension attributes. They are NOT derived from Mojang's dimension or timeline files.

`lab-probe` launches a **user-provided Horizon executable** with the experimental lab in an **isolated scratch directory**, `--world UHC_LabProbe --test-terrain --offscreen --frames 2 --timeout-seconds 20 --smoke-report ...`. The launcher also enforces a process timeout and saves a full process log, an optional Horizon smoke report, and a machine-readable `probe-result.json`. A failure is preserved for debugging; successful process termination alone does not prove game compatibility. **No real Horizon binary or GPU boot is available in this build environment**, so the bundled tests use fake child processes only. The lab is still intentionally blocked from normal `PLAY`.

```cmd
unlicensed-horizon-client.exe lab-init
unlicensed-horizon-client.exe lab-contract "%APPDATA%\unlicensed-horizon-client\open-content-lab-v0.7"
unlicensed-horizon-client.exe lab-audit "%APPDATA%\unlicensed-horizon-client\open-content-lab-v0.7"
unlicensed-horizon-client.exe lab-probe --timeout 20 --frames 2
```

The GUI has a separate **6. Run isolated Horizon laboratory probe** button; `PROBE-LAB-WINDOWS.bat` performs the same explicit action. The probe requires an installed/selected `horizon-client.exe` and a freshly generated v0.7 lab. Reports live under `%APPDATA%\unlicensed-horizon-client\probe-reports\` and no existing worlds are read, changed or removed. A GPU/display driver may still be needed for the real client, even with `--offscreen`.

These are original-content *research fixtures*, not a version of Minecraft without a license.

## v0.5: Referenced gallery states and cross-file contract checks

The independent, non-playable laboratory now generates **19 block states** (air, 6 material states and 12 inspection-gallery specimens), **18 block model bindings**, `model-states.json`, **36 original PNGs**, and encoded tags, plus one provenance manifest (**80 generated files + manifest**). The extra gallery states reproduce the identifiers and required property combinations from Horizon's published Rust inspection gallery; their textures and placeholder cube geometry are independently authored, not Minecraft assets.

`lab-contract DIR` analyzes the subset of formats and relations described by Horizon's `assets.rs`, `models.rs`, `models/catalog.rs`, `registry/facts.rs`, and `worldgen/tags.rs`: registry default indexes, state descriptors, block tag references, model-state schemas, blockstate/model pointers and decoded PNG dimensions. The `lab-audit DIR` command additionally checks file checksums and the lab's experimental/not-playable provenance. Re-hashing a mutated file no longer makes semantic errors disappear. CLI and Windows GUI both expose these checks.

The structural checker is written in Go and covers only the **documented subset**, not a real Horizon Rust loader. It does **NOT** establish that the game can render, load a title screen or run without Minecraft. `doctor`/`play` intentionally reject labs even when their original lab marker is removed because the resource manifest explicitly marks these files as incomplete. Unmodified v0.4 labs remain readable for checksum verification; regenerate a fresh v0.5 lab to use the new structural checker.

```cmd
unlicensed-horizon-client.exe lab-init
unlicensed-horizon-client.exe lab-contract "%APPDATA%\unlicensed-horizon-client\open-content-lab-v0.7"
unlicensed-horizon-client.exe lab-audit "%APPDATA%\unlicensed-horizon-client\open-content-lab-v0.7"
```

## v0.4 (previous): Original-content resource laboratory and corrected upstream preflight

**In v0.4:** `lab-init` introduced a separate, deterministic original, **non-playable** lab (then 7 states; expanded in v0.5). `lab-audit` verifies recorded checksums, provenance, and, for v0.5 labs, subset contracts. No Minecraft assets or generated Mojang data are included.

This is a concrete first milestone toward an open-content adaptation, **not** a launcher or license workaround that magically supplies Minecraft 26.3 resources. `doctor`/`play` explicitly reject this experimental folder, even if it has the basic startup marker. Do not point Play at it.

**Corrected preflight:** no longer guesses the first required files. Upstream `horizon-client-render::Assets::load` consumes `horizon-resources.json` with `data_version = 5023`, `facts.json` and `block-tags.json`. The game then needs `block-behaviours.json`, biome colormaps and biome definitions (plus many later tables). Static preflight remains an intentionally incomplete approximation. `--render-distance` range updated to upstream 2..128, not the prior 2..64.

Create and validate the lab on Windows in Command Prompt:

```bat
unlicensed-horizon-client.exe lab-init
unlicensed-horizon-client.exe lab-audit "%APPDATA%\unlicensed-horizon-client\open-content-lab-v0.7"
```

You can also use **Create Original Content Lab** in the GUI, or generate a new lab using `lab-init` (no generated files are committed to Git).

## v0.3 (previous): Fixes and features

- **Critical save-path fix:** Horizon reads `saves/` and `options.txt` from the process's current working directory, which the old launcher incorrectly set to the versioned engine directory. v0.3 runs the game from your `game-home`, while keeping the binary and resource paths absolute. This also means existing engine updates can't accidentally place new saves beside disposable binaries.
- **Legacy save locator:** `legacy-worlds` lists recognizable v0.1-v0.2 worlds in the engine folder. Copy worlds manually into `<game-home>/saves/` while the game is closed. No automatic write, deletion, or migration.
- **Real startup marker check:** checks `horizon-resources.json` as well as `registries.json` and `commands.json` and validates basic JSON syntax. Earlier v0.2 didn't check the actual startup marker. Run `audit-resources` for machine-readable JSON (status is *preflight only*).
- **Offline server quick-join:** `--server HOST[:PORT]` supported; server and world are mutually exclusive; invalid or ambiguous server addresses are rejected. Offline usernames are impersonable; use private trusted servers.
- **PowerShell 5.1 profile fix:** GUI now saves config via one base64-encoded JSON argument (`config-json`), avoiding Windows PowerShell's tendency to discard native empty-string arguments, such as a blank world name.
- Maintains v0.2 Ed25519 release authentication, download limits, ZIP traversal checks and exit-code diagnostics.

## Developer CLI

```powershell
.\unlicensed-horizon-client.exe help
.\unlicensed-horizon-client.exe install
.\unlicensed-horizon-client.exe find-resources
.\unlicensed-horizon-client.exe config --name Rafig_123 --resources 'C:\my-data\26.3' --world Survival --server '' --distance 16
.\unlicensed-horizon-client.exe audit-resources
.\unlicensed-horizon-client.exe doctor
.\unlicensed-horizon-client.exe legacy-worlds
.\unlicensed-horizon-client.exe play --wait
.\unlicensed-horizon-client.exe logs
```

For quickjoin, clear World (`--world ''`) and set `--server 127.0.0.1:25565`, or enter the server in GUI (World empty).

Data files: `%APPDATA%\unlicensed-horizon-client\config.json`. Saves and options: `%APPDATA%\unlicensed-horizon-client\game-home\`. Engine release: `%APPDATA%\unlicensed-horizon-client\engines\<tag>\`.

## Probe triage (v0.7)

`lab-probe` now reports `diagnostic_code`, `suggested_action`, and
`smoke_report_json_valid` alongside process status. Typical codes are
`missing_file`, `rust_panic`, `graphics_backend`, `timeout`,
`invalid_smoke_json`, `missing_smoke_report`, and `inconclusive`.

This classification only examines a bounded tail of the process log and the
smoke file's basic JSON syntax. It is deliberately conservative: **even a
zero-exit process with a syntactically valid JSON smoke report is classified
as inconclusive, not playable**. The probe now passes through only a
restricted allowlist of basic OS/graphics environment variables and supplies
isolated home/config directories; it is not an OS-level security sandbox.

## Build and verification

Requires Go 1.22+ to rebuild. The downloaded Horizon game client is in Rust.

```sh
go test -count=1 ./...
go vet ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o unlicensed-horizon-client.exe ./cmd/uhc
```

All project launcher code is MIT-licensed; third-party rights are separate. Tests on Linux with mock Horizon processes are **not** Windows GPU, multiplayer or gameplay tests. No claim of a playable install without Mojang files is made. The lab is checked by Go tests but has not been loaded by a real Rust Horizon binary.
