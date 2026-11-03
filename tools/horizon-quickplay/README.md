# Horizon QuickPlay v1.8 — single EXE for Windows

**Double-click `Horizon-QuickPlay-v1.8.exe`.** Type any valid nickname (default `azetech_`), choose Singleplayer or a trusted offline-mode server, and press **PLAY**. The launcher remembers your nickname and downloads the **open-source Horizon engine** from its verified release channel, then checks for prepared game data. If missing, it detects an existing Minecraft Java Edition 26.3 installation in `%APPDATA%\.minecraft` (or a custom folder selected in the GUI), downloads the version-pinned Minecraft server jar from Mojang after validating its SHA-1 and size, and runs the release `horizon-prepare` tool automatically. It also validates prepared data before starting. The GUI remains responsive and displays progress; logs are saved under `%APPDATA%\unlicensed-horizon-client\quickplay-logs`. A separate Microsoft login is not required to set an offline nickname.

**Not a license bypass:** the playable Rust engine needs prepared Minecraft Java Edition 26.3 resources from a copy you are permitted to use. The EXE does not contain or download protected client jars/assets or bypass ownership verification, and it cannot sign in to Minecraft online-mode servers without valid Minecraft credentials. The locally installed Java Edition 26.3 files are a prerequisite for automatic preparation; if they are absent, install and run the game once using its official launcher with an authorized account. Java 25 may also be required for resource preparation. This is not a ready-to-play license-free Minecraft distribution. The experimental original-content lab in the underlying fork is not a playable game.

The downloaded file is a single Windows x64 EXE which embeds the [unlicensed-horizon companion CLI](https://github.com/pneqnp-iswr/unlicensed-horizon/tree/main/tools/unlicensed-horizon-client) and a Windows Forms interface plus `AutoPrepare.ps1` for automatic local resource preparation. Both PowerShell scripts have UTF-8 BOM so Windows PowerShell 5.1 displays Russian text correctly. The GUI uses Windows PowerShell 5.1 (included with Windows 10/11) and the installed .NET Framework. The EXE extracts its verified bundled files under `%LOCALAPPDATA%\Horizon-QuickPlay\payload\` and starts the built-in GUI. Saved profiles and game data remain under `%APPDATA%\unlicensed-horizon-client\`.

## Build from source (requires Go 1.23+)

First build the companion Go CLI for Windows x64 from `tools/unlicensed-horizon-client` in the project repository:

```powershell
$env:GOOS = 'windows'
$env:GOARCH = 'amd64'
go build -o path\to\horizon-quickplay-src\assets\unlicensed-horizon-client.exe ./cmd/uhc
```

Then from the directory containing this README and `main.go`:

```powershell
go build -trimpath -ldflags='-s -w -H windowsgui' -o Horizon-QuickPlay.exe .
```

To cross-compile from Linux, set `GOOS=windows GOARCH=amd64 CGO_ENABLED=0` before the final build.

## Implementation boundaries

The UI is a WinForms PowerShell 5.1 script embedded inside a Go Windows GUI executable. Engine installation/verification, profile persistence, resource checking and game spawning are delegated to the separately tested companion launcher. The GUI's direct local game boot is not testable in the Linux build environment. Real Windows/GPU gameplay and a successful multiplayer spawn must be tested on a host with the required assets and server.

## v1.8: identify missing Minecraft 26.3 inputs in readable UTF-8

QuickPlay v1.7 correctly finds a local Horizon game executable, but automatic resource preparation can still fail when the user has **no locally installed Minecraft Java Edition 26.3 data**. Previously Windows PowerShell 5.1 stderr redirection turned Cyrillic diagnostics into unreadable text.

The preparer now explicitly checks for `versions/26.3/26.3.json`, `versions/26.3/26.3.jar`, `assets/indexes`, `assets/objects` and `libraries` in the selected Minecraft folder. If any are absent, it writes UTF-8 structured details to `%APPDATA%\unlicensed-horizon-client\quickplay-logs\prepare-diagnostic.json` (error code `MINECRAFT_26_3_INPUTS_MISSING`, message and exact missing paths). The GUI reads that file with explicit UTF-8 decoding and displays the concrete requirements and the **Обзор…** recovery option, instead of a corrupted PowerShell trace. New Windows PowerShell 5.1 regression tests verify this failure path without downloading Minecraft data.

**This is diagnostics, not a licensing workaround.** An existing `horizon-client.exe` is not sufficient to play Minecraft content. To prepare legally, select an existing authorized Minecraft Java Edition 26.3 installation or install/run the game once from the official launcher on an account entitled to play. QuickPlay does not supply or fetch missing protected client jars/assets without such a source; offline-mode servers do not waive this resource prerequisite.

## v1.7: fix "engine selected" followed by "install Horizon first"

The previous GUI selected an existing `horizon-client.exe` and confirmed it, but the next automatic `Save-Profile` could replace the selected `client_path` with a missing or stale value from the CLI's `show` output. This produced a misleading "install Horizon first" error *even though the game binary exists*, as reproduced by a Windows user using the official Horizon install folder.

QuickPlay now maintains the selected engine path in its own session state, persists that exact path in each profile save, and reads the result back before claiming success. A successful engine download also updates the same state. On Windows, a new PowerShell 5.1 regression runs actual profile persistence functions against the compiled companion CLI and verifies that a locally selected `horizon-client.exe` survives repeated saves, including paths containing spaces.

This only fixes engine selection and persistence. The next step may still require generating prepared Minecraft 26.3 resources. An engine executable on disk does not imply the user has the proprietary game data or that an authenticated server accepts the offline nickname.

## v1.6: detailed installer errors and manually selected game engine

The v1.5 fix for unauthenticated GitHub API rate limiting remains in effect. If a download still fails, the window now shows the exit code and actual last lines of both install logs, with their directory path, instead of just saying "check internet".

Before downloading, QuickPlay looks for an existing \`horizon-client.exe\` in known Horizon folders. A new **Локальный Horizon…** button also allows selecting the EXE from a previously downloaded and **fully extracted** official Horizon ZIP. Download from https://github.com/horizon-rust/horizon/releases/tag/nightly . Choose only a trusted executable: manual selection does **not** verify its release signature, unlike automatic installation. Keep all neighboring files from the archive.

An existing game engine alone is not playable without prepared Minecraft Java Edition 26.3 resources from a copy you are authorized to use. Server online-mode login still requires Minecraft account authentication. The GUI and game boot have not been tested end-to-end on Windows.

## v1.5: fix HTTP 403 from api.github.com

The Horizon engine installer now reads the official rolling `nightly` release assets directly from `github.com/horizon-rust/horizon/releases/download/nightly/`, **without calling the unauthenticated GitHub REST API**. This avoids a common `HTTP 403 from api.github.com` caused by shared-IP API limits. The archive is still accepted only after validating the pinned upstream Ed25519 signature on `SHA256SUMS` and the matching SHA-256 digest. The launcher does not disable security verification, download from arbitrary mirrors, or request the player's GitHub credentials.

If the download from **github.com** itself fails, check access to GitHub release assets or use a network that can reach them. A blocked site, TLS interception, missing release asset or unavailable network cannot be fixed merely by removing the API request. The client resources are still required from an authorized Minecraft installation, and no online-mode server authentication is bypassed.

## One-click install sequence

1. Double-click the EXE, enter `azetech_` or another nickname and click Play.
2. If the official Horizon engine binary is absent, QuickPlay downloads it from GitHub and verifies its Ed25519 signature and SHA-256 (existing companion behavior).
3. If valid prepared resources are not found, the new AutoPrepare helper checks the **locally installed** Minecraft Java Edition 26.3 directory, downloads **only** the matching Minecraft server jar from Mojang (verified against the local version manifest), and runs `horizon-prepare --prepare --complete` using the local authorized Minecraft files.
4. The launcher performs its resource preflight and opens Horizon. Login on `online-mode=true` servers still requires authenticated Minecraft credentials.

This build cannot be integration-tested against a real Windows GPU or Microsoft/Mojang download in our Linux test environment. Automatic game-resource preparation is best-effort; when Java 25, the version's installed data or other prerequisites are absent, click **Логи** to inspect the exact error. Never share downloaded Minecraft data or account tokens.

## v1.2 enhancements

- **Choose the Minecraft installation folder** instead of assuming `%APPDATA%\.minecraft`. The chosen path is stored in a dedicated settings file and reloaded on the next launch. The folder need not exist to download the open-source Horizon engine; the resource stage explains that dependency when reached.
- **Validate the local client JAR's SHA-1 and size** against the user's existing Minecraft version manifest before invoking the Horizon preparer. The bootstrap still downloads only the server JAR, never proprietary client resources.
- **Visible download status** from the game engine installer; indeterminate progress during lengthy downloads, JSON fraction from the preparer when available.
- **Cancel install/preparation**, terminating the launched subprocess tree while leaving stored worlds intact; restarting may reuse previously verified downloads.
- Improve handling of native CLI stderr with PowerShell 5.1 and avoid claiming an unverified selected resource directory is playable.
- Build tested for Windows x64 with Go 1.23, and checked for PE format, basic script delimiter balance, and `go vet`. Windows Forms and game boot were not executed in this Linux environment.

For a real test, open the executable and enter `azetech_`; the first Play downloads the engine and uses the chosen Minecraft folder to prepare the game. If no authorized game is installed, the GUI stops with an actionable message. For remote servers, only a true in-game join proves success; the status probe alone never verifies authentication.

## v1.3 hotfix: saved preferences crash

- Fix the reproducible Play failure shown as `Exception calling "Replace" with "3" argument(s): "The path is not of a legal form"`. The former PowerShell 5.1 `.NET File.Replace(..., $null)` in `Save-Preferences` was executed when `quickplay-preferences.json` already existed.
- Persist settings with validated absolute paths, UTF-8 temp-file generation and a previous-version backup, and recover from a damaged JSON file when a valid `.bak` is available. No third-argument `.Replace()` remains in QuickPlay.
- The EXE embeds the patched PowerShell UI and its underlying Go CLI; it still requires prepared, legitimately obtained Minecraft data for gameplay. Windows Forms and actual Minecraft multiplayer cannot be exercised on the Linux build host, so the fix is compiled/static-checked, not yet GUI-tested.

## v1.5 hotfix: PowerShell HOME is read-only

- Fix reproduced from the Windows GUI: 
`Cannot overwrite variable HOME because it is read-only or constant` (Russian: 
`Не удается перезаписать переменную HOME, так как она является постоянной либо доступна только для чтения.`).
- The `Save-Profile` function used `$home` as a local variable. PowerShell variable names are case-insensitive, so it collided with the built-in read-only `$HOME` variable. Renamed to `$gameHomeDir`, retaining existing configured home and fallback behavior.
- Added a static regression check to reject writes to protected automatic variables in bundled scripts. The Windows executable was cross-compiled on Linux and inspected as a PE64 binary; actual Windows gameplay is still not verified.
## GitHub Actions release artifact

The repository's `quickplay-windows.yml` workflow rebuilds the embedded companion CLI from source, validates PowerShell regression checks, and cross-compiles the self-contained Windows QuickPlay executable. Its uploaded Actions artifact is a build output, not a packaged Minecraft installation. Compiled binaries and Mojang content are not tracked in Git.
