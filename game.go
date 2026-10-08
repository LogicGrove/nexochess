package main

import (
	"errors"
	"strings"

	"github.com/corentings/chess/v2"
)

type LegalMove struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Promotion string `json:"promotion,omitempty"`
}

type HistoryMove struct {
	SAN  string `json:"san"`
	From string `json:"from"`
	To   string `json:"to"`
	FEN  string `json:"fen"`
}

// match keeps a canonical main line. Rebuilding it for undo removes variations
// and restores repetition counts, outcomes and PGN together.
type match struct {
	game       *chess.Game
	initialFEN string
	ucis       []string
	history    []HistoryMove
}

func newMatch(fen string) (*match, error) {
	g := chess.NewGame()
	if fen != "" {
		option, err := chess.FEN(fen)
		if err != nil {
			return nil, err
		}
		g = chess.NewGame(option)
	}
	return &match{game: g, initialFEN: g.FEN(), ucis: []string{}, history: []HistoryMove{}}, nil
}

func (m *match) playUCI(uci string) error {
	if m.game.Outcome() != chess.NoOutcome {
		return errors.New("La partida ha terminado")
	}
	var chosen *chess.Move
	pos := m.game.Position()
	for _, move := range m.game.ValidMoves() {
		if (chess.UCINotation{}).Encode(pos, &move) == uci {
			candidate := move
			chosen = &candidate
			break
		}
	}
	if chosen == nil {
		return errors.New("Movimiento ilegal")
	}
	san := (chess.AlgebraicNotation{}).Encode(pos, chosen)
	if err := m.game.Move(chosen, nil); err != nil {
		return errors.New("Movimiento ilegal")
	}
	m.ucis = append(m.ucis, uci)
	m.history = append(m.history, HistoryMove{SAN: san, From: chosen.S1().String(), To: chosen.S2().String(), FEN: m.game.FEN()})
	return nil
}

func (m *match) undo() error {
	if len(m.ucis) == 0 {
		return errors.New("No hay jugadas para deshacer")
	}
	rebuilt, err := newMatch(m.initialFEN)
	if err != nil {
		return err
	}
	for _, uci := range m.ucis[:len(m.ucis)-1] {
		if err := rebuilt.playUCI(uci); err != nil {
			return err
		}
	}
	*m = *rebuilt
	return nil
}

func (m *match) claimable() bool {
	if m.game.Outcome() != chess.NoOutcome {
		return false
	}
	for _, method := range m.game.EligibleDraws() {
		if method == chess.ThreefoldRepetition || method == chess.FiftyMoveRule {
			return true
		}
	}
	return false
}

func (m *match) claim() error {
	if m.game.Outcome() != chess.NoOutcome {
		return errors.New("La partida ha terminado")
	}
	for _, method := range m.game.EligibleDraws() {
		if method == chess.ThreefoldRepetition || method == chess.FiftyMoveRule {
			return m.game.Draw(method)
		}
	}
	return errors.New("Todavía no puedes reclamar tablas")
}

func inCheck(pos *chess.Position) bool {
	board := pos.Board()
	king := chess.NoSquare
	for square, piece := range board.SquareMap() {
		if piece.Color() == pos.Turn() && piece.Type() == chess.King {
			king = square
			break
		}
	}
	if king == chess.NoSquare {
		return false
	}
	for square, piece := range board.SquareMap() {
		if piece.Color() != pos.Turn().Other() {
			continue
		}
		for _, target := range board.AttacksFrom(square) {
			if target == king {
				return true
			}
		}
	}
	return false
}

func methodReason(method chess.Method) string {
	switch method {
	case chess.Checkmate:
		return "Jaque mate"
	case chess.Stalemate:
		return "Ahogado"
	case chess.Resignation:
		return "Rendición"
	case chess.DrawOffer:
		return "Tablas por acuerdo"
	case chess.ThreefoldRepetition:
		return "Triple repetición"
	case chess.FivefoldRepetition:
		return "Cinco repeticiones"
	case chess.FiftyMoveRule:
		return "Regla de 50 jugadas"
	case chess.SeventyFiveMoveRule:
		return "Regla de 75 jugadas"
	case chess.InsufficientMaterial:
		return "Material insuficiente"
	default:
		return ""
	}
}

func validSquare(square string) bool {
	return len(square) == 2 && square[0] >= 'a' && square[0] <= 'h' && square[1] >= '1' && square[1] <= '8'
}

func validPromotion(promotion string) bool {
	return promotion == "" || (len(promotion) == 1 && strings.Contains("qrbn", promotion))
}
