# Changelog

## 0.7.0 (2026-10-30)

- Added conservative machine-readable `lab-probe` diagnosis and concrete next action for missing files, Rust panics, graphics-backend problems, timeouts and malformed/missing smoke reports.
- A nonempty smoke report is no longer treated as valid JSON automatically; JSON syntax is checked separately, and zero-exit smoke results remain inconclusive.
- Restricted the child process to an allowlist of essential OS/graphics environment variables to reduce accidental credentials leakage.
- Added diagnosis matrix tests, secret-environment regression tests, and bounded log-tail test.
- No claim of successful Rust-loader boot or fully independent gameplay.


## 0.6.0 (2026-10-30)

- Added an isolated, time-bounded `lab-probe` that executes a user-provided real Horizon client with a copy of experimental resources and no account/profile data, with process log, JSON report and optional Horizon smoke report. The launcher itself enforces a timeout and does not open a normal game world.
- Added original day timeline and overworld dimension attributes JSON; independently authored, never Mojang-derived. v3 laboratory: 82 files + manifest, still non-playable.
- Added semantic checks and mutation tests for the two new environment files, along with probe subprocess isolation and failure-reporting tests. v2 laboratory audits remain supported.
- Added an explicit GUI test button and PROBE-LAB-WINDOWS.bat. Neither starts a normal Play session.
- Windows build and CLI tests executed; true Rust/Horizon integration boot remains untested in this environment.


## 0.5.0 (2026-10-30)

- Added 12 independently-authored gallery test block states (19 total including air), 18 model mappings, and `model-states.json`. Original procedural textures total 36 PNG files.
- Added `lab-contract DIR`: cross-file schema checker for registry defaults, block tags, variant/model pointers, model-state properties, and PNG dimensions / full decoding.
- Updated `lab-audit` to detect semantic inconsistencies even when a corrupt file is re-hashed in the provenance manifest.
- Hardened lab reads against filesystem symlink traversal, and disabled the manual removal of the lab marker as a way to pass basic startup preflight.
- Added GUI contract inspection, additional mutation/security tests, and fixed CLI preflight wording to avoid claiming a tested game boot.
- Preserved earlier projects/archive compatibility; regenerated the bundled sample lab for v0.5.


## 0.4.0 (2026-10-30)

- Added deterministic, original-content `lab-init` generator with authored textures, 7-state facts registry, tag bundle, models and basic biome data. No Mojang files.
- Added `lab-audit` provenance and SHA-256 checks; original labs explicitly blocked from `doctor` and `play`.
- Corrected static prepared-resource audit using actual upstream 26.3 early-loader files and data_version 5023 instead of mistaken marker-only assumptions.
- Updated render-distance validation to 2..128, matching current source.
- Exposed lab creation as an experimental GUI button.
- Added deterministic generation, tampering, overwrite and lab-block unit tests.


## 0.3.0 (2026-10-30)

- Fixed a major bug: game's process working directory is now configured `game-home`, not the engine install directory. Saves and `options.txt` now reside in the profile folder.
- Added `legacy-worlds` locator for prior releases' accidentally engine-local saves; does not change anything.
- Fixed real Horizon prepared-data preflight by checking `horizon-resources.json`, plus the existing essential registries and commands JSON. JSON audit mode added.
- Added offline server startup config `--server HOST[:PORT]`, input validation, world/server collision detection, GUI box.
- Fixed Windows PowerShell 5.1 empty native argument handling: GUI saves with `config-json` base64-encoded JSON.
- Added mock working-directory and migration-discovery regression tests.

## 0.2.0 (2026-10-30)

- Pinned Ed25519 signature verification before checksum verification and extraction.
- Prepared game-file discovery, stricter resource preflight, process exit diagnostics, safer installs.

## 0.1.0

- Initial Windows wrapper and local-profile Horizon launcher.
