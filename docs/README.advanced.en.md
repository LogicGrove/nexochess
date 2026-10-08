# Nexo Chess · Advanced guide

[Español](README.advanced.md) | **English** · [Back to the start](../README.en.md)

This publication contains **Nexo Chess 1.0.0**, the original version for games between people on a LAN, with no clock or AI. The original application code and executable ZIP are preserved; GitHub presentation is added.

## Usage and connections

The host runs `NexoChess.exe` on Windows 10/11 x64. The app opens the local browser and displays network addresses that players can share. Other devices need only a current browser with JavaScript.

The default port is **8088**; if occupied, the program looks for the next available port. Always use the address shown by your instance. For example, `http://192.168.1.20:8088` is only an example, not every computer's address.

Options in PowerShell, from the executable's folder:

```powershell
.\NexoChess.exe --no-browser
.\NexoChess.exe --port 8099
.\NexoChess.exe --version
```

If a connection fails, check that both devices are on the same network, that the router does not isolate clients, and that Windows allows the app on private networks. VPNs and virtual adapters can add addresses that do not work for your Wi-Fi. After changing networks, reload the host page to update the displayed addresses.

The app uses local HTTP and is intended for a trusted network. It needs neither internet access nor router port forwarding. Only the host, through its local connection, can shut down the server using the web button; it can also press Ctrl+C or close the window.

## Tables and board

You can create multiple tables, choose a color, take a free seat, or watch as a spectator. Selecting a piece shows dots on legal destinations and rings on captures. Promotion offers four pieces.

Use the history arrows to review positions and return to **En directo** (Live) to move. Rewinding does not change the game. **Undo** removes one half-move after the opponent accepts; restarting and agreeing a draw also require their acceptance.

Reloading a tab keeps your seat during the server session. Closing it without leaving the table reserves the seat for two minutes of disconnection; another player can then take it. Leaving the table releases it immediately.

## Data and portability

Tables and games are held in memory. They disappear when the server closes and do not return when reopened. Export a PGN before finishing if you want to save or study a game.

Everything needed to run the app is inside the EXE. It does not install services, modify the registry, store configuration, or download resources while running. To remove it, close the program and delete its folder.

Windows and the browser may retain their own history, recent files, or other traces. If you approved a firewall rule, Windows may retain it; manage it through “Windows Defender Firewall → Allow an app.” The app does not create or remove these rules itself.

## Rules and limitations

Castling, en passant, promotion, check, checkmate, and stalemate are validated. Common insufficient-material cases are detected; exceptional dead positions with blocked pawns may require an agreed draw. Repetition and 50-move claims are made once the position has been reached, without declaring an intended move to an arbiter.

The bundled rules dependency includes two local fixes: position identity for repetition with legal en passant captures, and resignation against an opponent with only a king. See [the original patch notes](vendor-patches.md). The app is intended for informal games.

## Source and building

| Path | Contents |
| --- | --- |
| `main.go` | Windows startup, embedded resources, and local HTTP server |
| `api.go` | Sessions, tables, requests, and in-memory state |
| `game.go` | Rules, history, and PGN export |
| `web/` | HTML, CSS, and JavaScript interface; original SVG pieces |
| `vendor/` | Bundled chess dependency and local fixes |
| `*_test.go`, `tests/` | Server, rules, and interface tests |
| `docs/` | This guide and original development and verification reports |

Development requires **Go compatible with 1.25 or later** and PowerShell on Windows. Playing requires no Go, Python, Node, or .NET.

From the project root:

```powershell
.\build.ps1
```

The script runs tests and creates `dist\NexoChess.exe` for Windows x64, with embedded resources and CGO disabled. It uses `-mod=vendor`; **`go mod vendor` would replace the local fixes**, so you would need to reapply and check them. The startup code is Windows-specific.

Additional checks:

```powershell
go test -mod=vendor ./... -count=1
go vet -mod=vendor ./...
node --check web/app.js
node --check web/pieces.js
# Install JSDOM 27 as a development tool only:
npm install --no-save --package-lock=false jsdom@27
node --test tests/frontend.test.cjs
```

`tests/native_smoke.py` can check the API with the EXE running with `--no-browser --port 18088`; the test shuts down the server when finished. See [the original verification report](verification.md) for those results and limitations, including the absence of a test between two physical computers. These reports describe the original delivery, not guarantees for every network.

## Licenses

The project retains its [MIT license](../LICENSE) and [third-party notices](../THIRD_PARTY_NOTICES.txt). The vendored dependency retains its own license. The release ZIP is published exactly as received in the original sources.
