package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func requestAPI(a http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Nexo-Client", "1")
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	return w
}

func decodeSnapshot(t *testing.T, w *httptest.ResponseRecorder) Snapshot {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	var s Snapshot
	if err := json.Unmarshal(w.Body.Bytes(), &s); err != nil {
		t.Fatal(err)
	}
	return s
}

func sessionFor(t *testing.T, a *App) string {
	t.Helper()
	w := requestAPI(a, "POST", "/api/session", "", "{}")
	if w.Code != 200 {
		t.Fatalf("session: %d %s", w.Code, w.Body.String())
	}
	var v struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil || len(v.Token) < 32 {
		t.Fatalf("invalid token: %v %q", err, v.Token)
	}
	return v.Token
}

func seatedRoom(t *testing.T, a *App) (string, string, string) {
	t.Helper()
	w, b := sessionFor(t, a), sessionFor(t, a)
	s := decodeSnapshot(t, requestAPI(a, "POST", "/api/rooms", w, `{"name":"Mesa","playerName":"Blancas","color":"w"}`))
	decodeSnapshot(t, requestAPI(a, "POST", "/api/join", b, fmt.Sprintf(`{"room":%q,"playerName":"Negras","color":"b"}`, s.ID)))
	return s.ID, w, b
}

func actionAPI(a *App, room, token, action, extra string) *httptest.ResponseRecorder {
	return requestAPI(a, "POST", "/api/action", token, fmt.Sprintf(`{"room":%q,"action":%q%s}`, room, action, extra))
}

func TestSessionsSeatsAndAuthoritativeMoves(t *testing.T) {
	a := NewApp()
	id, w, b := seatedRoom(t, a)
	spectator := sessionFor(t, a)
	s := decodeSnapshot(t, requestAPI(a, "POST", "/api/join", spectator, fmt.Sprintf(`{"room":%q,"playerName":"Observador","color":"spectator"}`, id)))
	if s.You != "spectator" || len(s.History) != 0 || len(s.Legal) != 0 {
		t.Fatalf("spectator snapshot: %+v", s)
	}
	for _, tok := range []string{b, spectator} {
		if got := actionAPI(a, id, tok, "move", `,"from":"e2","to":"e4"`); got.Code < 400 {
			t.Fatal("non-turn move accepted")
		}
	}
	if got := actionAPI(a, id, w, "move", `,"from":"e2","to":"e5"`); got.Code < 400 {
		t.Fatal("illegal move accepted")
	}
	s = decodeSnapshot(t, actionAPI(a, id, w, "move", `,"from":"e2","to":"e4"`))
	if s.Turn != "b" || len(s.History) != 1 || s.History[0].SAN != "e4" {
		t.Fatalf("bad move snapshot: %+v", s)
	}
	s = decodeSnapshot(t, requestAPI(a, "GET", "/api/state?room="+id, w, ""))
	if s.You != "w" || s.White == nil || !s.White.Online {
		t.Fatal("seat lost on refresh")
	}
	outsider := sessionFor(t, a)
	if got := requestAPI(a, "GET", "/api/state?room="+id, outsider, ""); got.Code != 403 {
		t.Fatal("outsider read allowed")
	}
	if got := requestAPI(a, "POST", "/api/join", outsider, fmt.Sprintf(`{"room":%q,"playerName":"Usurpador","color":"w"}`, id)); got.Code < 400 {
		t.Fatal("occupied seat stolen")
	}
}

func TestMutualProposalsUndoRestartAndDraw(t *testing.T) {
	a := NewApp()
	id, w, b := seatedRoom(t, a)
	decodeSnapshot(t, actionAPI(a, id, w, "move", `,"from":"e2","to":"e4"`))
	s := decodeSnapshot(t, actionAPI(a, id, w, "undo", ""))
	if s.Pending == nil || len(s.History) != 1 {
		t.Fatal("undo did not require consent")
	}
	if got := actionAPI(a, id, w, "respond", `,"accept":true`); got.Code < 400 {
		t.Fatal("own proposal accepted")
	}
	s = decodeSnapshot(t, actionAPI(a, id, b, "respond", `,"accept":true`))
	if s.Pending != nil || len(s.History) != 0 || s.FEN != s.InitialFEN {
		t.Fatal("undo did not rebuild position")
	}
	decodeSnapshot(t, actionAPI(a, id, w, "draw", ""))
	s = decodeSnapshot(t, actionAPI(a, id, b, "respond", `,"accept":false`))
	if s.Pending != nil || s.Outcome != "*" {
		t.Fatal("rejected draw ended game")
	}
	decodeSnapshot(t, actionAPI(a, id, b, "draw", ""))
	s = decodeSnapshot(t, actionAPI(a, id, w, "respond", `,"accept":true`))
	if s.Outcome != "1/2-1/2" {
		t.Fatal("agreed draw missing")
	}
	decodeSnapshot(t, actionAPI(a, id, w, "restart", ""))
	s = decodeSnapshot(t, actionAPI(a, id, b, "respond", `,"accept":true`))
	if s.Outcome != "*" || len(s.History) != 0 {
		t.Fatal("restart failed")
	}
	s = decodeSnapshot(t, actionAPI(a, id, b, "resign", ""))
	if s.Outcome != "1-0" || !strings.Contains(s.PGN, "1-0") {
		t.Fatal("resignation missing")
	}
}

func TestDuplicateMoveConcurrentOnlyAppliesOnce(t *testing.T) {
	a := NewApp()
	id, w, _ := seatedRoom(t, a)
	var wg sync.WaitGroup
	results := make(chan int, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); results <- actionAPI(a, id, w, "move", `,"from":"e2","to":"e4"`).Code }()
	}
	wg.Wait()
	close(results)
	n := 0
	for status := range results {
		if status == 200 {
			n++
		}
	}
	s := decodeSnapshot(t, requestAPI(a, "GET", "/api/state?room="+id, w, ""))
	if n != 1 || len(s.History) != 1 {
		t.Fatalf("duplicate applied: success=%d ply=%d", n, len(s.History))
	}
}

func TestMutationProtectionAndMalformedInputs(t *testing.T) {
	a := NewApp()
	cases := []struct{ body, contentType, marker, origin string }{
		{`{}`, "text/plain", "1", ""}, {`{}`, "application/json", "", ""},
		{`{}`, "application/json", "1", "http://evil.test"}, {`{}`, "application/json", "1", "https://example.com"},
		{`null`, "application/json", "1", ""},
		{`{} {}`, "application/json", "1", ""}, {`{"unknown":1}`, "application/json", "1", ""},
		{`{`, "application/json", "1", ""}, {strings.Repeat(" ", 8192) + `{}`, "application/json", "1", ""},
	}
	for _, c := range cases {
		r := httptest.NewRequest("POST", "http://example.com/api/session", strings.NewReader(c.body))
		r.Header.Set("Content-Type", c.contentType)
		r.Header.Set("X-Nexo-Client", c.marker)
		if c.origin != "" {
			r.Header.Set("Origin", c.origin)
		}
		wr := httptest.NewRecorder()
		a.ServeHTTP(wr, r)
		if wr.Code < 400 {
			t.Fatalf("malformed/protected accepted: %+v", c)
		}
		if !bytes.Contains(wr.Body.Bytes(), []byte(`"error"`)) {
			t.Fatal("error not JSON")
		}
	}
	r := httptest.NewRequest("POST", "http://example.com/api/session", strings.NewReader(`{}`))
	r.Header.Set("Content-Type", "application/json; charset=utf-8")
	r.Header.Set("X-Nexo-Client", "1")
	r.Header.Set("Origin", "http://example.com")
	wr := httptest.NewRecorder()
	a.ServeHTTP(wr, r)
	if wr.Code != 200 {
		t.Fatal(wr.Body.String())
	}
	if got := requestAPI(a, "GET", "/api/lobby", "", ""); got.Code != 401 {
		t.Fatal("unauthenticated lobby allowed")
	}
	if got := requestAPI(a, "GET", "/api/session", "", ""); got.Code != 405 {
		t.Fatal("wrong method allowed")
	}
}

func TestSeatReconnectAndOfflineReclaim(t *testing.T) {
	a := NewApp()
	id, oldWhite, black := seatedRoom(t, a)
	fresh := sessionFor(t, a)
	join := fmt.Sprintf(`{"room":%q,"playerName":"Nuevo","color":"w"}`, id)
	if got := requestAPI(a, "POST", "/api/join", fresh, join); got.Code != 409 {
		t.Fatal("connected seat reclaimed")
	}
	a.mu.Lock()
	a.sessions[oldWhite].lastSeen = time.Now().Add(-90 * time.Second)
	a.mu.Unlock()
	if got := requestAPI(a, "POST", "/api/join", fresh, join); got.Code != 409 {
		t.Fatal("recent disconnected seat reclaimed")
	}
	s := decodeSnapshot(t, requestAPI(a, "GET", "/api/state?room="+id, oldWhite, ""))
	if s.You != "w" || s.White.Name != "Blancas" {
		t.Fatal("reconnection lost seat/name")
	}
	a.mu.Lock()
	a.sessions[oldWhite].lastSeen = time.Now().Add(-3 * time.Minute)
	a.mu.Unlock()
	s = decodeSnapshot(t, requestAPI(a, "POST", "/api/join", fresh, join))
	if s.You != "w" || s.White.Name != "Nuevo" || s.Black.Name != "Negras" {
		t.Fatal("stale seat not reclaimed")
	}
	if got := requestAPI(a, "GET", "/api/state?room="+id, oldWhite, ""); got.Code != 403 {
		t.Fatal("old owner retained authority")
	}
	decodeSnapshot(t, actionAPI(a, id, fresh, "leave", ""))
	s = decodeSnapshot(t, requestAPI(a, "GET", "/api/state?room="+id, black, ""))
	if s.White != nil {
		t.Fatal("explicit leave did not release seat")
	}
}

func TestRoomAndSessionBounds(t *testing.T) {
	a := NewApp()
	for i := 0; i < maxRooms; i++ {
		tok := sessionFor(t, a)
		decodeSnapshot(t, requestAPI(a, "POST", "/api/rooms", tok, `{"name":"Mesa","playerName":"J","color":"w"}`))
	}
	spare := sessionFor(t, a)
	if got := requestAPI(a, "POST", "/api/rooms", spare, `{"name":"Mesa","playerName":"J","color":"w"}`); got.Code != 409 {
		t.Fatal("room cap missing")
	}
	a.mu.Lock()
	for len(a.sessions) < maxSessions {
		a.sessions[fmt.Sprint("dummy-", len(a.sessions))] = &sessionState{lastSeen: time.Now()}
	}
	a.mu.Unlock()
	if got := requestAPI(a, "POST", "/api/session", "", `{}`); got.Code != 503 {
		t.Fatal("session cap missing")
	}
	a.mu.Lock()
	for _, room := range a.rooms {
		room.lastActive = time.Now().Add(-25 * time.Hour)
	}
	a.mu.Unlock()
	wr := requestAPI(a, "GET", "/api/lobby", spare, "")
	if wr.Code != 200 || !strings.Contains(wr.Body.String(), `"rooms":[]`) {
		t.Fatal("idle rooms not removed")
	}
}

type blockingWriter struct {
	headers          http.Header
	entered, release chan struct{}
}

func (w *blockingWriter) Header() http.Header  { return w.headers }
func (w *blockingWriter) WriteHeader(code int) {}
func (w *blockingWriter) Write(body []byte) (int, error) {
	close(w.entered)
	<-w.release
	return len(body), nil
}

func TestSlowResponseDoesNotBlockOtherRooms(t *testing.T) {
	a := NewApp()
	token := sessionFor(t, a)
	w := &blockingWriter{headers: make(http.Header), entered: make(chan struct{}), release: make(chan struct{})}
	r := httptest.NewRequest("GET", "/api/lobby", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	done := make(chan struct{})
	go func() { a.ServeHTTP(w, r); close(done) }()
	<-w.entered
	defer func() { close(w.release); <-done }()
	other := make(chan int, 1)
	go func() { other <- requestAPI(a, "POST", "/api/session", "", `{}`).Code }()
	select {
	case status := <-other:
		if status != 200 {
			t.Fatalf("session status %d", status)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("slow response held global mutex")
	}
}

func TestConcurrentStateReadsAndProposalResponses(t *testing.T) {
	a := NewApp()
	id, w, b := seatedRoom(t, a)
	decodeSnapshot(t, actionAPI(a, id, w, "move", `,"from":"e2","to":"e4"`))
	proposal := decodeSnapshot(t, actionAPI(a, id, w, "undo", ""))
	var wg sync.WaitGroup
	statuses := make(chan int, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses <- actionAPI(a, id, b, "respond", fmt.Sprintf(`,"accept":true,"version":%d`, proposal.Version)).Code
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				if got := requestAPI(a, "GET", "/api/state?room="+id, w, ""); got.Code != 200 {
					t.Errorf("state=%d", got.Code)
				}
				if got := requestAPI(a, "GET", "/api/lobby", b, ""); got.Code != 200 {
					t.Errorf("lobby=%d", got.Code)
				}
			}
		}()
	}
	wg.Wait()
	close(statuses)
	successes := 0
	for status := range statuses {
		if status == 200 {
			successes++
		} else if status != 409 {
			t.Fatalf("unexpected respond status%d", status)
		}
	}
	s := decodeSnapshot(t, requestAPI(a, "GET", "/api/state?room="+id, w, ""))
	if successes != 1 || len(s.History) != 0 || s.Pending != nil || s.FEN != s.InitialFEN {
		t.Fatalf("concurrent undo applied%d times", successes)
	}
}

func TestStaleVersionCannotAcceptNewProposal(t *testing.T) {
	a := NewApp()
	id, w, b := seatedRoom(t, a)
	old := decodeSnapshot(t, actionAPI(a, id, w, "draw", ""))
	decodeSnapshot(t, actionAPI(a, id, w, "move", `,"from":"e2","to":"e4"`))
	current := decodeSnapshot(t, actionAPI(a, id, b, "draw", ""))
	stale := fmt.Sprintf(`,"accept":true,"version":%d`, old.Version)
	if got := actionAPI(a, id, w, "respond", stale); got.Code != 409 {
		t.Fatal("stale response accepted replacement proposal")
	}
	s := decodeSnapshot(t, requestAPI(a, "GET", "/api/state?room="+id, w, ""))
	if s.Pending == nil || s.Outcome != "*" {
		t.Fatal("stale response modified game")
	}
	s = decodeSnapshot(t, actionAPI(a, id, w, "respond", fmt.Sprintf(`,"accept":true,"version":%d`, current.Version)))
	if s.Outcome != "1/2-1/2" {
		t.Fatal("current proposal rejected")
	}
}

func TestPresenceAndReclaimWithClock(t *testing.T) {
	a := NewApp()
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	a.now = func() time.Time { return now }
	id, w, b := seatedRoom(t, a)
	now = now.Add(16 * time.Second)
	s := decodeSnapshot(t, requestAPI(a, "GET", "/api/state?room="+id, b, ""))
	if s.White.Online {
		t.Fatal("15 second presence not expired")
	}
	newWhite := sessionFor(t, a)
	join := fmt.Sprintf(`{"room":%q,"playerName":"Nuevo","color":"w"}`, id)
	if got := requestAPI(a, "POST", "/api/join", newWhite, join); got.Code != 409 {
		t.Fatal("reserved seat replaced")
	}
	now = now.Add(105 * time.Second)
	s = decodeSnapshot(t, requestAPI(a, "POST", "/api/join", newWhite, join))
	if s.You != "w" {
		t.Fatal("seat not reclaimed after2minutes")
	}
	if got := requestAPI(a, "GET", "/api/state?room="+id, w, ""); got.Code != 403 {
		t.Fatal("old owner retained seat")
	}
}

func TestResignationDrawReasonExplainsImpossibleMate(t *testing.T) {
	for _, caseData := range []struct{ fen, role string }{
		{"7k/8/8/8/8/8/R7/K7 w - - 0 1", "w"},
		{"7k/7r/8/8/8/8/8/K7 b - - 0 1", "b"},
	} {
		a := NewApp()
		id, w, b := seatedRoom(t, a)
		m, err := newMatch(caseData.fen)
		if err != nil {
			t.Fatal(err)
		}
		a.mu.Lock()
		a.rooms[id].match = m
		a.mu.Unlock()
		token := w
		if caseData.role == "b" {
			token = b
		}
		s := decodeSnapshot(t, actionAPI(a, id, token, "resign", ""))
		if s.Outcome != "1/2-1/2" || s.Reason != "Tablas por rendición: rival sin posibilidad de mate" {
			t.Fatalf("unclear resignation draw: %s %q", s.Outcome, s.Reason)
		}
	}
}

type lostResponseWriter struct {
	headers http.Header
	writes  int
}

func (w *lostResponseWriter) Header() http.Header            { return w.headers }
func (w *lostResponseWriter) WriteHeader(status int)         {}
func (w *lostResponseWriter) Write(body []byte) (int, error) { w.writes++; return 0, io.ErrClosedPipe }

func TestLobbyRecoversOwnRoomAfterLostCreateResponse(t *testing.T) {
	a := NewApp()
	owner := sessionFor(t, a)
	other := sessionFor(t, a)
	r := httptest.NewRequest("POST", "/api/rooms", strings.NewReader(`{"name":"Recuperable","playerName":"Manu","color":"w"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Nexo-Client", "1")
	r.Header.Set("Authorization", "Bearer "+owner)
	lost := &lostResponseWriter{headers: make(http.Header)}
	a.ServeHTTP(lost, r)
	if lost.writes != 1 {
		t.Fatal("create response was not sent to the failing connection")
	}
	lobby := requestAPI(a, "GET", "/api/lobby", owner, "")
	if lobby.Code != 200 {
		t.Fatalf("lobby status %d", lobby.Code)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(lobby.Body.Bytes(), &fields); err != nil {
		t.Fatal(err)
	}
	active, ok := fields["activeRoom"]
	if !ok {
		t.Fatal("lobby cannot recover committed room: activeRoom missing")
	}
	var roomID string
	if err := json.Unmarshal(active, &roomID); err != nil || roomID == "" {
		t.Fatalf("invalid recovered room: %s", active)
	}
	var rooms []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(fields["rooms"], &rooms); err != nil || len(rooms) != 1 || rooms[0].ID != roomID {
		t.Fatal("recovery did not refer to the single committed room")
	}
	s := decodeSnapshot(t, requestAPI(a, "GET", "/api/state?room="+roomID, owner, ""))
	if s.You != "w" || s.White == nil || s.White.Name != "Manu" || s.Name != "Recuperable" {
		t.Fatal("recovery lost the creator's seat")
	}
	for _, token := range []string{other, sessionFor(t, a)} {
		w := requestAPI(a, "GET", "/api/lobby", token, "")
		var data map[string]json.RawMessage
		if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
			t.Fatal(err)
		}
		if value, ok := data["activeRoom"]; !ok || string(value) != "null" {
			t.Fatalf("unbound session sees another active room: %s", w.Body.String())
		}
	}
	anonymous := requestAPI(a, "GET", "/api/lobby", "", "")
	if anonymous.Code != 401 || strings.Contains(anonymous.Body.String(), "activeRoom") {
		t.Fatal("anonymous caller learned session membership")
	}
	decodeSnapshot(t, actionAPI(a, roomID, owner, "leave", ""))
	if w := requestAPI(a, "GET", "/api/lobby", owner, ""); !strings.Contains(w.Body.String(), `"activeRoom":null`) {
		t.Fatal("left room remains active")
	}
}

func TestEmptyArraysNamesPresenceAndSessionExpiry(t *testing.T) {
	a := NewApp()
	token := sessionFor(t, a)
	wr := requestAPI(a, "GET", "/api/lobby", token, "")
	if !strings.Contains(wr.Body.String(), `"rooms":[]`) {
		t.Fatal("rooms must be []")
	}
	s := decodeSnapshot(t, requestAPI(a, "POST", "/api/rooms", token, `{"name":"   Sala   ","playerName":"`+strings.Repeat("á", 40)+`\u0001","color":"w"}`))
	if len([]rune(s.White.Name)) != 32 || s.Name != "Sala" {
		t.Fatal("name sanitation")
	}
	if got := requestAPI(a, "GET", "/api/state?room="+s.ID, token, ""); !strings.Contains(got.Body.String(), `"history":[]`) {
		t.Fatal("history must be []")
	}
	a.mu.Lock()
	a.sessions[token].lastSeen = time.Now().Add(-20 * time.Second)
	a.mu.Unlock()
	observer := sessionFor(t, a)
	s = decodeSnapshot(t, requestAPI(a, "POST", "/api/join", observer, fmt.Sprintf(`{"room":%q,"playerName":"O","color":"spectator"}`, s.ID)))
	if s.White.Online {
		t.Fatal("presence did not expire")
	}
	s = decodeSnapshot(t, requestAPI(a, "GET", "/api/state?room="+s.ID, token, ""))
	if !s.White.Online {
		t.Fatal("reconnection not online")
	}
	a.mu.Lock()
	a.sessions[token].lastSeen = time.Now().Add(-25 * time.Hour)
	a.mu.Unlock()
	if a.HasToken(token) {
		t.Fatal("expired token accepted")
	}
}
