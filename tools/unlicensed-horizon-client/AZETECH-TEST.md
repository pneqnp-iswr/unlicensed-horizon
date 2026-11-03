# Test: join Horizon as azetech_ (offline profile)

This is a repeatable manual test for the community fork on Windows. No Microsoft login is needed for the *local nickname*, but actual gameplay still requires properly prepared Minecraft Java Edition 26.3 resources from a copy you are authorized to use. The original-content laboratory is not playable.

The test does not bypass online-mode authentication. On servers configured with online-mode=true, an authenticated Minecraft session is required. Local offline names can be impersonated, so use trusted private servers only.

## 1. Set up a controlled offline-mode server

Obtain/build the matching official Horizon server. In a **separate empty server folder**, initialize an isolated test server:

    horizon-server init --eula --offline --world azetech-test --seed 12345 --port 25565 --no-whitelist --yes
    horizon-server run

See the upstream hosting guide at https://github.com/horizon-rust/horizon/blob/main/docs/hosting.md for full configuration options. The server must be firewalled to localhost or a private LAN; never expose an open offline test server to the public Internet.

## 2. Build our companion launcher and try a real join

From tools/unlicensed-horizon-client in Windows PowerShell with Go 1.22+:

    go test ./...
    go build -o unlicensed-horizon-client.exe ./cmd/uhc
    powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\TEST-AZETECH.ps1 -Nickname azetech_ -Server 127.0.0.1:25565

If the engine or prepared resources are not configured, run:

    .\unlicensed-horizon-client.exe install
    .\unlicensed-horizon-client.exe config --resources "C:\path\to\prepared\26.3"
    .\unlicensed-horizon-client.exe doctor

The real prepared directory is produced by Horizon's horizon-prepare from an authorized Minecraft installation. The launcher does not ship proprietary Mojang data or account entitlements.

Equivalent manual steps, if you do not want to run the PowerShell script:

    .\unlicensed-horizon-client.exe config --name azetech_ --world= --server 127.0.0.1:25565
    .\unlicensed-horizon-client.exe probe-server 127.0.0.1:25565
    .\unlicensed-horizon-client.exe doctor
    .\unlicensed-horizon-client.exe play --wait
    .\unlicensed-horizon-client.exe logs

IMPORTANT: the --world= option is a single nonempty command-line argument containing an empty value. Do not use --world '' in Windows PowerShell 5.1: it can omit the empty argument entirely. If a world name is already configured, it must be cleared for quick-join to work.

## 3. Interpret the test

- tcp_reachable=true: network socket opened successfully. This is **not** proof of Minecraft login.
- status_responded=true: Minecraft server status JSON was read. This is **not** proof of account verification, game protocol compatibility, or spawning.
- online_mode=unknown: intentional. Status responses cannot reliably reveal server's authentication policy.
- doctor passes: prerequisite file checks passed. It does not prove the renderer runs.
- The game enters a multiplayer world as azetech_, and the server logs a successful join: a real offline-mode join succeeded.
- Invalid session / Failed to verify username: the server requires authentication or rejected the current session. A local nickname alone will not meet that requirement.
- Version mismatch: use matching client and server versions.
- Refused connection / timed out: check target address, firewall, server status, and protocol.

The test script backs up the current configuration before changing nickname/server, and does not delete worlds. The included Go status-probe tests run against an isolated mock Minecraft status server, not a real online-mode login. An actual Windows/GPU multiplayer join must be observed separately before claiming a playable release.
