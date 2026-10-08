"""JSON-only integration check of the running Windows executable (no browser)."""
import json
import socket
import urllib.error
import urllib.request

BASE = "http://127.0.0.1:18088"


def api(path, token=None, data=None, expected=200):
    headers = {"Content-Type": "application/json", "X-Nexo-Client": "1"}
    if token:
        headers["Authorization"] = "Bearer " + token
    body = None if data is None else json.dumps(data).encode()
    request = urllib.request.Request(BASE + path, data=body, headers=headers)
    try:
        response = urllib.request.urlopen(request, timeout=5)
    except urllib.error.HTTPError as error:
        response = error
    assert response.status == expected, (path, response.status)
    return json.load(response)


w = api("/api/session", data={})["token"]
b = api("/api/session", data={})["token"]
viewer = api("/api/session", data={})["token"]
s = api("/api/rooms", w, {"name": "Prueba nativa", "playerName": "Blancas", "color": "w"})
room = s["id"]
s = api("/api/join", b, {"room": room, "playerName": "Negras", "color": "b"})
spectator = api("/api/join", viewer, {"room": room, "playerName": "Visita", "color": "spectator"})
assert spectator["you"] == "spectator"
assert api("/api/lobby", w)["activeRoom"] == room
s = api("/api/state?room=" + room, w)

for token, start, end in [(w, "f2", "f3"), (b, "e7", "e5"), (w, "g2", "g4"), (b, "d8", "h4")]:
    s = api("/api/action", token, {"room": room, "action": "move", "from": start, "to": end, "version": s["version"]})
assert s["outcome"] == "0-1" and s["check"] and s["reason"] == "Jaque mate"
assert len(s["history"]) == 4 and s["history"][-1]["san"] == "Qh4#"
assert "0-1" in s["pgn"]

s = api("/api/action", w, {"room": room, "action": "undo", "version": s["version"]})
s = api("/api/action", b, {"room": room, "action": "respond", "accept": True, "version": s["version"]})
assert len(s["history"]) == 3 and s["outcome"] == "*" and s["turn"] == "b"
assert api("/api/state?room=" + room, viewer)["fen"] == s["fen"]
assert api("/api/state?room=" + room, w)["you"] == "w"

api("/api/shutdown", w, {})
print("EXE: two players, spectator, mate, SAN/PGN, mutual undo, state recovery, shutdown PASS")
