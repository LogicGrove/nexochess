NEXO CHESS 1.0 - CHESS ON YOUR NETWORK
64-bit Windows 10/11. No Python, Node, .NET, or Go installation needed.

TO PLAY
1. Extract the ZIP to a folder and open NexoChess.exe.
2. Your browser opens. Enter a nickname, create a table, and choose a color.
3. Share the network address shown by the app or console window.
   Example: http://192.168.1.20:8088 (use your own, not this example).
4. On the other device, open that address and join the same table.
5. Keep the program running on the host computer.

No internet required. Only the host runs the EXE; everyone else needs a
current browser. Phones and tablets can also join.

IF THE OTHER DEVICE CANNOT CONNECT
- Both devices must use the same network. Avoid guest Wi-Fi: some routers
  isolate clients even when they share a network name.
- If Windows asks, allow Nexo Chess on private networks.
- VPNs and virtual adapters may show several addresses. Choose the address
  for the adapter connected to your Wi-Fi or Ethernet.
- If you change networks while the app runs, reload the host page to refresh
  the addresses you can share.
- Do not forward router ports: the app is intended for a LAN.
- If the usual port is occupied, the app looks for the next free one.

THE BOARD
Select a piece to see dots on legal destinations and rings on captures.
Select a destination to move. Promotion always offers queen, rook, bishop,
or knight. Castling and en passant are validated.
The app shows check, checkmate, stalemate, draws, and the game result.

REWIND
History arrows review moves without changing the game. Return to "En directo"
(Live) to move. Undo requests removal of the last half-move and requires the
opponent's acceptance. Restarting and agreeing draws also require acceptance.
You can export a PGN file whenever you wish.

CLOSING AND DATA
Shut down the server from the host page, press Ctrl+C in its console, or close
the window. Games disappear when it closes; export a PGN to keep them.
Reloading a tab keeps your seat during the server session. Closing a tab
removes its temporary session.

If you close a tab without leaving the table, its seat is reserved for two
minutes of disconnection. Another player can then take it. Leaving the table
releases the seat immediately.

The app does not install services, modify the registry, create configuration
files, store games, or download resources while running. Everything needed
is inside the EXE. To remove it, close the app and delete its extracted folder.

Windows and your browser may retain their own traces, such as history and
recent files. If you approved a firewall rule, Windows may retain it after
the EXE is deleted; manage it in Windows Defender Firewall > Allow an app.
Deleting an executable cannot guarantee removal of all system-managed traces.
The app does not add or remove firewall rules itself.

Common insufficient-material cases are detected. Exceptional dead positions
with completely blocked pawns may require an agreed draw. Repetition and
50-move claims are available once the position is reached; the app does not
declare an intended move to an arbiter.

Options: NexoChess.exe --no-browser / --port 8099 / --version
Project license: MIT. Dependencies and notices: THIRD_PARTY_NOTICES.txt.
The interface of this version is in Spanish.

GitHub: https://github.com/LogicGrove/nexochess
Quick guide: README.en.md
Advanced guide: docs/README.advanced.en.md
