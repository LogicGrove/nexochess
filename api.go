package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/corentings/chess/v2"
)

const (
	maxRooms        = 32
	maxSessions     = 512
	maxSpectators   = 64
	idleLifetime    = 24 * time.Hour
	onlineLifetime  = 15 * time.Second
	seatReservation = 2 * time.Minute
)

type PlayerSnapshot struct {
	Name   string `json:"name"`
	Online bool   `json:"online"`
}
type Proposal struct {
	Kind string `json:"kind"`
	By   string `json:"by"`
}
type Snapshot struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	You        string          `json:"you"`
	White      *PlayerSnapshot `json:"white"`
	Black      *PlayerSnapshot `json:"black"`
	FEN        string          `json:"fen"`
	Turn       string          `json:"turn"`
	Check      bool            `json:"check"`
	Outcome    string          `json:"outcome"`
	Reason     string          `json:"reason"`
	Legal      []LegalMove     `json:"legal"`
	History    []HistoryMove   `json:"history"`
	InitialFEN string          `json:"initialFen"`
	Version    uint64          `json:"version"`
	Pending    *Proposal       `json:"pending"`
	CanClaim   bool            `json:"canClaim"`
	PGN        string          `json:"pgn"`
}

type sessionState struct {
	lastSeen time.Time
	room     string
}
type seat struct{ token, name string }
type roomState struct {
	id, name     string
	white, black *seat
	spectators   map[string]struct{}
	match        *match
	pending      *Proposal
	version      uint64
	lastActive   time.Time
}

// App owns all authoritative state. The mutex covers both reads and writes:
// legal-move generation and PGN encoders may maintain caches in the engine.
type App struct {
	mu       sync.Mutex
	sessions map[string]*sessionState
	rooms    map[string]*roomState
	now      func() time.Time
}

func NewApp() *App {
	return &App{sessions: map[string]*sessionState{}, rooms: map[string]*roomState{}, now: time.Now}
}

func (a *App) HasToken(token string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.prune(a.now())
	_, ok := a.sessions[token]
	return ok && token != ""
}

func (a *App) prune(now time.Time) {
	for id, room := range a.rooms {
		if now.Sub(room.lastActive) > idleLifetime {
			delete(a.rooms, id)
			for _, session := range a.sessions {
				if session.room == id {
					session.room = ""
				}
			}
		}
	}
	for token, session := range a.sessions {
		if now.Sub(session.lastSeen) > idleLifetime {
			if room := a.rooms[session.room]; room != nil {
				a.removeMember(room, token, now)
			}
			delete(a.sessions, token)
		}
	}
}

func (r *roomState) role(token string) string {
	if r.white != nil && r.white.token == token {
		return "w"
	}
	if r.black != nil && r.black.token == token {
		return "b"
	}
	if _, ok := r.spectators[token]; ok {
		return "spectator"
	}
	return ""
}

func (a *App) removeMember(room *roomState, token string, now time.Time) {
	if room.white != nil && room.white.token == token {
		room.white = nil
		room.pending = nil
	}
	if room.black != nil && room.black.token == token {
		room.black = nil
		room.pending = nil
	}
	delete(room.spectators, token)
	if session := a.sessions[token]; session != nil {
		session.room = ""
	}
	room.version++
	room.lastActive = now
	if room.white == nil && room.black == nil && len(room.spectators) == 0 {
		delete(a.rooms, room.id)
	}
}

func (a *App) playerSnapshot(player *seat, now time.Time) *PlayerSnapshot {
	if player == nil {
		return nil
	}
	session := a.sessions[player.token]
	return &PlayerSnapshot{Name: player.name, Online: session != nil && now.Sub(session.lastSeen) <= onlineLifetime}
}

func (a *App) snapshot(room *roomState, token string, now time.Time) Snapshot {
	role := room.role(token)
	if role == "" {
		role = "spectator"
	}
	game := room.match.game
	legal := []LegalMove{}
	if role == game.Position().Turn().String() && room.white != nil && room.black != nil && game.Outcome() == chess.NoOutcome {
		for _, move := range game.ValidMoves() {
			promotion := ""
			if move.Promo() != chess.NoPieceType {
				promotion = move.Promo().String()
			}
			legal = append(legal, LegalMove{From: move.S1().String(), To: move.S2().String(), Promotion: promotion})
		}
	}
	history := append([]HistoryMove{}, room.match.history...)
	var pending *Proposal
	if room.pending != nil {
		copy := *room.pending
		pending = &copy
	}
	white, black := a.playerSnapshot(room.white, now), a.playerSnapshot(room.black, now)
	whiteName, blackName := "?", "?"
	if white != nil {
		whiteName = white.Name
	}
	if black != nil {
		blackName = black.Name
	}
	game.AddTagPair("Event", room.name)
	game.AddTagPair("Site", "Red local")
	game.AddTagPair("White", whiteName)
	game.AddTagPair("Black", blackName)
	game.AddTagPair("Result", game.Outcome().String())
	if room.match.initialFEN != chess.StartingPosition().String() {
		game.AddTagPair("SetUp", "1")
		game.AddTagPair("FEN", room.match.initialFEN)
	}
	reason := methodReason(game.Method())
	if game.Outcome() == chess.Draw && game.Method() == chess.Resignation {
		reason = "Tablas por rendición: rival sin posibilidad de mate"
	}
	return Snapshot{ID: room.id, Name: room.name, You: role, White: white, Black: black, FEN: game.FEN(), Turn: game.Position().Turn().String(), Check: inCheck(game.Position()), Outcome: game.Outcome().String(), Reason: reason, Legal: legal, History: history, InitialFEN: room.match.initialFEN, Version: room.version, Pending: pending, CanClaim: role == game.Position().Turn().String() && room.white != nil && room.black != nil && room.match.claimable(), PGN: game.String()}
}

func sanitizeName(name, fallback string) string {
	var clean []rune
	for _, r := range strings.TrimSpace(name) {
		if unicode.IsControl(r) {
			continue
		}
		if len(clean) == 32 {
			break
		}
		clean = append(clean, r)
	}
	name = strings.TrimSpace(string(clean))
	if name == "" {
		return fallback
	}
	return name
}

func randomToken() (string, error) {
	var entropy [32]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(entropy[:]), nil
}

func (a *App) roomID() (string, error) {
	for i := 0; i < 10; i++ {
		var entropy [4]byte
		if _, err := rand.Read(entropy[:]); err != nil {
			return "", err
		}
		id := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(entropy[:])[:6]
		if a.rooms[id] == nil {
			return id, nil
		}
	}
	return "", errors.New("No se pudo generar el código de sala")
}

func apiJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func apiError(w http.ResponseWriter, status int, message string) {
	apiJSON(w, status, map[string]string{"error": message})
}

func mutationAllowed(r *http.Request) bool {
	if r.Header.Get("X-Nexo-Client") != "1" {
		return false
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return false
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		u, err := url.Parse(origin)
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		if err != nil || u.Scheme != scheme || !strings.EqualFold(u.Host, r.Host) || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" {
			return false
		}
	}
	return true
}

func decodeBody(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	decoder := json.NewDecoder(r.Body)
	var raw json.RawMessage
	if err := decoder.Decode(&raw); err != nil {
		return err
	}
	if raw = bytes.TrimSpace(raw); len(raw) == 0 || raw[0] != '{' {
		return errors.New("Se requiere un objeto JSON")
	}
	strict := json.NewDecoder(bytes.NewReader(raw))
	strict.DisallowUnknownFields()
	if err := strict.Decode(destination); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("El cuerpo debe contener un único objeto JSON")
	}
	return nil
}

func bearerToken(r *http.Request) string {
	parts := strings.Split(r.Header.Get("Authorization"), " ")
	if len(parts) != 2 || parts[0] != "Bearer" || parts[1] == "" {
		return ""
	}
	return parts[1]
}

type roomRequest struct {
	Name       string `json:"name"`
	PlayerName string `json:"playerName"`
	Color      string `json:"color"`
}
type joinRequest struct {
	Room       string `json:"room"`
	PlayerName string `json:"playerName"`
	Color      string `json:"color"`
}
type actionRequest struct {
	Room      string  `json:"room"`
	Action    string  `json:"action"`
	From      string  `json:"from"`
	To        string  `json:"to"`
	Promotion string  `json:"promotion"`
	Accept    *bool   `json:"accept"`
	Version   *uint64 `json:"version"`
}

type responseBuffer struct {
	header http.Header
	body   bytes.Buffer
	status int
}

func (b *responseBuffer) Header() http.Header { return b.header }
func (b *responseBuffer) WriteHeader(status int) {
	if b.status == 0 {
		b.status = status
	}
}
func (b *responseBuffer) Write(body []byte) (int, error) {
	if b.status == 0 {
		b.status = 200
	}
	return b.body.Write(body)
}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Build the complete response in memory, so slow network writes never hold
	// the authoritative-state mutex or block players in unrelated rooms.
	buffer := &responseBuffer{header: make(http.Header)}
	a.serveHTTP(buffer, r)
	for key, values := range buffer.header {
		w.Header()[key] = values
	}
	w.WriteHeader(buffer.status)
	_, _ = w.Write(buffer.body.Bytes())
}

func (a *App) serveHTTP(w http.ResponseWriter, r *http.Request) {
	wantedMethod := ""
	switch r.URL.Path {
	case "/api/session", "/api/rooms", "/api/join", "/api/action":
		wantedMethod = http.MethodPost
	case "/api/lobby", "/api/state":
		wantedMethod = http.MethodGet
	default:
		apiError(w, 404, "Recurso no encontrado")
		return
	}
	if r.Method != wantedMethod {
		w.Header().Set("Allow", wantedMethod)
		apiError(w, 405, "Método no permitido")
		return
	}
	if r.Method == http.MethodPost && !mutationAllowed(r) {
		apiError(w, 403, "Solicitud rechazada: utiliza la aplicación desde su dirección local")
		return
	}
	// Body parsing does not hold the state lock while a slow client uploads.
	var create roomRequest
	var join joinRequest
	var action actionRequest
	if r.Method == http.MethodPost {
		var body any
		switch r.URL.Path {
		case "/api/session":
			body = &struct{}{}
		case "/api/rooms":
			body = &create
		case "/api/join":
			body = &join
		case "/api/action":
			body = &action
		}
		if err := decodeBody(w, r, body); err != nil {
			apiError(w, 400, "JSON inválido o solicitud demasiado grande")
			return
		}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	now := a.now()
	a.prune(now)
	if r.URL.Path == "/api/session" {
		if len(a.sessions) >= maxSessions {
			apiError(w, 503, "Hay demasiadas sesiones; vuelve a intentarlo más tarde")
			return
		}
		token, err := randomToken()
		if err != nil {
			apiError(w, 500, "No se pudo crear la sesión")
			return
		}
		a.sessions[token] = &sessionState{lastSeen: now}
		apiJSON(w, 200, map[string]string{"token": token})
		return
	}
	token := bearerToken(r)
	session := a.sessions[token]
	if session == nil {
		apiError(w, 401, "Tu sesión ha caducado; vuelve a entrar")
		return
	}
	session.lastSeen = now
	switch r.URL.Path {
	case "/api/lobby":
		type lobbyRoom struct {
			ID     string          `json:"id"`
			Name   string          `json:"name"`
			White  *PlayerSnapshot `json:"white"`
			Black  *PlayerSnapshot `json:"black"`
			Ply    int             `json:"ply"`
			Status string          `json:"status"`
		}
		rooms := []lobbyRoom{}
		for _, room := range a.rooms {
			status := "playing"
			if room.white == nil || room.black == nil {
				status = "waiting"
			}
			if room.match.game.Outcome() != chess.NoOutcome {
				status = "finished"
			}
			rooms = append(rooms, lobbyRoom{ID: room.id, Name: room.name, White: a.playerSnapshot(room.white, now), Black: a.playerSnapshot(room.black, now), Ply: len(room.match.history), Status: status})
		}
		sort.Slice(rooms, func(i, j int) bool { return rooms[i].ID < rooms[j].ID })
		var activeRoom *string
		if room := a.rooms[session.room]; room != nil && room.role(token) != "" {
			id := room.id
			activeRoom = &id
		}
		apiJSON(w, 200, struct {
			Rooms      []lobbyRoom `json:"rooms"`
			ActiveRoom *string     `json:"activeRoom"`
		}{Rooms: rooms, ActiveRoom: activeRoom})
	case "/api/rooms":
		if create.Color != "w" && create.Color != "b" {
			apiError(w, 400, "Elige blancas o negras")
			return
		}
		if session.room != "" {
			apiError(w, 409, "Sal de tu sala actual antes de crear otra")
			return
		}
		if len(a.rooms) >= maxRooms {
			apiError(w, 409, "Se ha alcanzado el límite de 32 salas")
			return
		}
		id, err := a.roomID()
		if err != nil {
			apiError(w, 500, "No se pudo crear la sala")
			return
		}
		m, err := newMatch("")
		if err != nil {
			apiError(w, 500, "No se pudo crear la partida")
			return
		}
		room := &roomState{id: id, name: sanitizeName(create.Name, "Partida"), spectators: map[string]struct{}{}, match: m, version: 1, lastActive: now}
		player := &seat{token: token, name: sanitizeName(create.PlayerName, "Jugador")}
		if create.Color == "w" {
			room.white = player
		} else {
			room.black = player
		}
		a.rooms[id] = room
		session.room = id
		apiJSON(w, 200, a.snapshot(room, token, now))
	case "/api/join":
		if join.Color != "w" && join.Color != "b" && join.Color != "spectator" {
			apiError(w, 400, "Elige blancas, negras o espectador")
			return
		}
		room := a.rooms[strings.ToUpper(strings.TrimSpace(join.Room))]
		if room == nil {
			apiError(w, 404, "La sala no existe o ha caducado")
			return
		}
		if session.room != "" && session.room != room.id {
			apiError(w, 409, "Sal de tu sala actual antes de entrar en otra")
			return
		}
		role := room.role(token)
		if role == "w" || role == "b" {
			if role != join.Color {
				apiError(w, 409, "Sal de la sala para cambiar de asiento")
				return
			}
			room.lastActive = now
			apiJSON(w, 200, a.snapshot(room, token, now))
			return
		}
		var occupied *seat
		if join.Color == "w" {
			occupied = room.white
		}
		if join.Color == "b" {
			occupied = room.black
		}
		if occupied != nil {
			owner := a.sessions[occupied.token]
			if owner != nil && now.Sub(owner.lastSeen) <= seatReservation {
				apiError(w, 409, "Asiento ocupado: se reserva 2 minutos tras una desconexión")
				return
			}
			if owner != nil && owner.room == room.id {
				owner.room = ""
			}
			if join.Color == "w" {
				room.white = nil
			} else {
				room.black = nil
			}
			room.pending = nil
		}
		if join.Color == "spectator" {
			if role == "" && len(room.spectators) >= maxSpectators {
				apiError(w, 409, "La sala tiene demasiados espectadores")
				return
			}
			room.spectators[token] = struct{}{}
		} else {
			player := &seat{token: token, name: sanitizeName(join.PlayerName, "Jugador")}
			if join.Color == "w" {
				room.white = player
			} else {
				room.black = player
			}
			delete(room.spectators, token)
		}
		session.room = room.id
		room.version++
		room.lastActive = now
		apiJSON(w, 200, a.snapshot(room, token, now))
	case "/api/state":
		room := a.rooms[strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("room")))]
		if room == nil {
			apiError(w, 404, "La sala no existe o ha caducado")
			return
		}
		if room.role(token) == "" {
			apiError(w, 403, "Entra en la sala antes de verla")
			return
		}
		room.lastActive = now
		apiJSON(w, 200, a.snapshot(room, token, now))
	case "/api/action":
		room := a.rooms[strings.ToUpper(strings.TrimSpace(action.Room))]
		if room == nil {
			apiError(w, 404, "La sala no existe o ha caducado")
			return
		}
		role := room.role(token)
		if role == "" {
			apiError(w, 403, "No perteneces a esta sala")
			return
		}
		if action.Action == "leave" {
			a.removeMember(room, token, now)
			apiJSON(w, 200, a.snapshot(room, token, now))
			return
		}
		if action.Version != nil && *action.Version != room.version {
			apiError(w, 409, "La partida ha cambiado; vuelve a intentarlo con la posición actual")
			return
		}
		if role == "spectator" {
			apiError(w, 403, "Los espectadores no pueden realizar esa acción")
			return
		}
		if room.white == nil || room.black == nil {
			apiError(w, 409, "Espera a que se una el otro jugador")
			return
		}
		if err := a.applyAction(room, role, action); err != nil {
			apiError(w, 409, err.Error())
			return
		}
		room.version++
		room.lastActive = now
		apiJSON(w, 200, a.snapshot(room, token, now))
	}
}

func (a *App) applyAction(room *roomState, role string, action actionRequest) error {
	game := room.match.game
	switch action.Action {
	case "move":
		if role != game.Position().Turn().String() {
			return errors.New("No es tu turno")
		}
		if !validSquare(action.From) || !validSquare(action.To) || !validPromotion(action.Promotion) {
			return errors.New("Movimiento inválido")
		}
		if err := room.match.playUCI(action.From + action.To + action.Promotion); err != nil {
			return err
		}
		room.pending = nil
	case "undo", "restart", "draw":
		if room.pending != nil {
			return errors.New("Responde primero a la propuesta pendiente")
		}
		if action.Action == "undo" && len(room.match.history) == 0 {
			return errors.New("No hay jugadas para deshacer")
		}
		if action.Action == "draw" && game.Outcome() != chess.NoOutcome {
			return errors.New("La partida ha terminado")
		}
		room.pending = &Proposal{Kind: action.Action, By: role}
	case "respond":
		if room.pending == nil {
			return errors.New("No hay una propuesta pendiente")
		}
		if room.pending.By == role {
			return errors.New("El otro jugador debe responder a tu propuesta")
		}
		if action.Accept == nil {
			return errors.New("Indica si aceptas la propuesta")
		}
		if *action.Accept {
			switch room.pending.Kind {
			case "undo":
				if err := room.match.undo(); err != nil {
					return err
				}
			case "restart":
				m, err := newMatch("")
				if err != nil {
					return err
				}
				room.match = m
			case "draw":
				if err := game.Draw(chess.DrawOffer); err != nil {
					return err
				}
			default:
				return fmt.Errorf("Propuesta desconocida")
			}
		}
		room.pending = nil
	case "resign":
		if game.Outcome() != chess.NoOutcome {
			return errors.New("La partida ha terminado")
		}
		color := chess.White
		if role == "b" {
			color = chess.Black
		}
		game.Resign(color)
		room.pending = nil
	case "claim":
		if role != game.Position().Turn().String() {
			return errors.New("Solo puedes reclamar tablas en tu turno")
		}
		if err := room.match.claim(); err != nil {
			return err
		}
		room.pending = nil
	default:
		return errors.New("Acción desconocida")
	}
	return nil
}
