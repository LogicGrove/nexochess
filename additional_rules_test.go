package main

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/corentings/chess/v2"
)

func qaRulesMatch(t *testing.T, fen string) *match {
	t.Helper()
	m, err := newMatch(fen)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func qaRulesPlay(t *testing.T, m *match, moves ...string) {
	t.Helper()
	for _, uci := range moves {
		if err := m.playUCI(uci); err != nil {
			t.Fatalf("%s at %s: %v", uci, m.game.FEN(), err)
		}
	}
}

func qaRulesHasMove(m *match, uci string) bool {
	for _, move := range m.game.ValidMoves() {
		if (chess.UCINotation{}).Encode(m.game.Position(), &move) == uci {
			return true
		}
	}
	return false
}

// Check both advertised moves and actual execution. Illegal inputs must leave
// the position, outcome, main line and exported history unchanged.
func qaRulesAssertMove(t *testing.T, m *match, uci string, legal bool) {
	t.Helper()
	if got := qaRulesHasMove(m, uci); got != legal {
		t.Fatalf("%s advertised legal=%v, want %v; FEN=%s", uci, got, legal, m.game.FEN())
	}
	before, outcome, method := m.game.FEN(), m.game.Outcome(), m.game.Method()
	line, history := len(m.ucis), len(m.history)
	err := m.playUCI(uci)
	if (err == nil) != legal {
		t.Fatalf("%s error=%v, want legal=%v", uci, err, legal)
	}
	if !legal && (m.game.FEN() != before || m.game.Outcome() != outcome || m.game.Method() != method || len(m.ucis) != line || len(m.history) != history) {
		t.Fatalf("illegal %s changed match", uci)
	}
}

func TestAdditionalRulesPawnMovementAndPins(t *testing.T) {
	cases := []struct {
		name, fen, uci string
		legal          bool
	}{
		{"white single", "7k/8/8/8/8/8/4P3/7K w - - 0 1", "e2e3", true},
		{"white double", "7k/8/8/8/8/8/4P3/7K w - - 0 1", "e2e4", true},
		{"white empty diagonal", "7k/8/8/8/8/8/4P3/7K w - - 0 1", "e2d3", false},
		{"white backwards", "7k/8/8/8/8/8/4P3/7K w - - 0 1", "e2e1", false},
		{"white blocked single", "7k/8/8/8/8/4n3/4P3/7K w - - 0 1", "e2e3", false},
		{"white cannot jump", "7k/8/8/8/8/4n3/4P3/7K w - - 0 1", "e2e4", false},
		{"white double destination occupied", "7k/8/8/8/4n3/8/4P3/7K w - - 0 1", "e2e4", false},
		{"white captures left", "7k/8/8/8/8/3n1n2/4P3/7K w - - 0 1", "e2d3", true},
		{"white captures right", "7k/8/8/8/8/3n1n2/4P3/7K w - - 0 1", "e2f3", true},
		{"black single", "7k/4p3/8/8/8/8/8/7K b - - 0 1", "e7e6", true},
		{"black double", "7k/4p3/8/8/8/8/8/7K b - - 0 1", "e7e5", true},
		{"black empty diagonal", "7k/4p3/8/8/8/8/8/7K b - - 0 1", "e7d6", false},
		{"black captures left", "7k/4p3/3N1N2/8/8/8/8/7K b - - 0 1", "e7d6", true},
		{"black captures right", "7k/4p3/3N1N2/8/8/8/8/7K b - - 0 1", "e7f6", true},
		{"black cannot jump", "7k/4p3/4N3/8/8/8/8/7K b - - 0 1", "e7e5", false},
		{"later white double forbidden", "7k/8/8/8/8/4P3/8/7K w - - 0 1", "e3e5", false},
		{"pinned rook leaves file", "4r1k1/8/8/8/8/8/4R3/4K3 w - - 0 1", "e2a2", false},
		{"pinned rook stays between", "4r1k1/8/8/8/8/8/4R3/4K3 w - - 0 1", "e2e3", true},
		{"pinned rook captures attacker", "4r1k1/8/8/8/8/8/4R3/4K3 w - - 0 1", "e2e8", true},
		{"pinned knight", "4r1k1/8/8/8/8/8/4N3/4K3 w - - 0 1", "e2c3", false},
		{"king cannot approach king", "8/8/8/8/8/4k3/8/R3K3 w - - 0 1", "e1e2", false},
		{"king cannot enter pinned attack", "4k3/8/4n3/8/6K1/8/8/4R3 w - - 0 1", "g4f4", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			qaRulesAssertMove(t, qaRulesMatch(t, tc.fen), tc.uci, tc.legal)
		})
	}
}

func TestAdditionalRulesCheckUsesAttacks(t *testing.T) {
	// A pawn's forward push is never an attack. A pinned piece still attacks
	// squares for king movement/check purposes (FIDE 3.1.3 and 3.9.1).
	cases := []struct {
		name, fen string
		check     bool
	}{
		{"black pawn forward", "7k/8/8/4p3/4K3/8/8/8 w - - 0 1", false},
		{"black pawn diagonal", "7k/8/8/4p3/3K4/8/8/8 w - - 0 1", true},
		{"white pawn forward", "8/8/8/4k3/4P3/8/8/7K b - - 0 1", false},
		{"white pawn diagonal", "8/8/8/4k3/3P4/8/8/7K b - - 0 1", true},
		{"rook unobstructed", "4r2k/8/8/8/8/8/8/4K3 w - - 0 1", true},
		{"rook blocked", "4r2k/8/8/8/4P3/8/8/4K3 w - - 0 1", false},
		{"pinned knight still checks", "4k3/8/4n3/8/5K2/8/8/4R3 w - - 0 1", true},
		{"pinned rook still checks", "4k3/4r1K1/8/8/8/8/8/4R3 w - - 0 1", true},
		{"pawn a file no wrap", "7k/8/8/8/p7/8/7K/8 w - - 0 1", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := qaRulesMatch(t, tc.fen)
			if got := inCheck(m.game.Position()); got != tc.check {
				t.Fatalf("inCheck=%v want %v", got, tc.check)
			}
		})
	}
}

func TestAdditionalRulesCastlingAttackConditions(t *testing.T) {
	cases := []struct {
		name, fen, uci string
		legal          bool
	}{
		{"in check", "k3r3/8/8/8/8/8/8/4K2R w K - 0 1", "e1g1", false},
		{"through check", "k4r2/8/8/8/8/8/8/4K2R w K - 0 1", "e1g1", false},
		{"into check", "k5r1/8/8/8/8/8/8/4K2R w K - 0 1", "e1g1", false},
		{"rook attacked permitted", "k6r/8/8/8/8/8/8/4K2R w K - 0 1", "e1g1", true},
		{"queen side b1 attacked permitted", "1r5k/8/8/8/8/8/8/R3K3 w Q - 0 1", "e1c1", true},
		{"queen side through check", "3r3k/8/8/8/8/8/8/R3K3 w Q - 0 1", "e1c1", false},
		{"queen side into check", "2r4k/8/8/8/8/8/8/R3K3 w Q - 0 1", "e1c1", false},
		{"queen side b1 occupied", "7k/8/8/8/8/8/8/RN2K3 w Q - 0 1", "e1c1", false},
		{"no stored right", "k7/8/8/8/8/8/8/4K2R w - - 0 1", "e1g1", false},
		{"black through check", "4k2r/8/8/8/8/8/8/K4R2 b k - 0 1", "e8g8", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			qaRulesAssertMove(t, qaRulesMatch(t, tc.fen), tc.uci, tc.legal)
		})
	}
	for _, tc := range []struct {
		name  string
		moves []string
	}{
		{"rook returns", []string{"h1h2", "a8b8", "h2h1", "b8a8"}},
		{"king returns", []string{"e1f1", "a8b8", "f1e1", "b8a8"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := qaRulesMatch(t, "k7/8/8/8/8/8/8/4K2R w K - 0 1")
			qaRulesPlay(t, m, tc.moves...)
			qaRulesAssertMove(t, m, "e1g1", false)
		})
	}
}

func TestAdditionalRulesCastlingMovesBothPieces(t *testing.T) {
	for _, tc := range []struct {
		fen, uci   string
		king, rook chess.Square
		color      chess.Color
	}{
		{"4k3/8/8/8/8/8/8/R3K2R w KQ - 0 1", "e1g1", chess.G1, chess.F1, chess.White},
		{"4k3/8/8/8/8/8/8/R3K2R w KQ - 0 1", "e1c1", chess.C1, chess.D1, chess.White},
		{"r3k2r/8/8/8/8/8/8/4K3 b kq - 0 1", "e8g8", chess.G8, chess.F8, chess.Black},
		{"r3k2r/8/8/8/8/8/8/4K3 b kq - 0 1", "e8c8", chess.C8, chess.D8, chess.Black},
	} {
		t.Run(tc.uci, func(t *testing.T) {
			m := qaRulesMatch(t, tc.fen)
			qaRulesPlay(t, m, tc.uci)
			board := m.game.Position().Board()
			king, rook := board.Piece(tc.king), board.Piece(tc.rook)
			if king.Type() != chess.King || rook.Type() != chess.Rook || king.Color() != tc.color || rook.Color() != tc.color {
				t.Fatalf("castling did not move both pieces: %s", m.game.FEN())
			}
		})
	}
}

func TestAdditionalRulesEnPassantRemovalExpiryAndKingSafety(t *testing.T) {
	for _, tc := range []struct{ name, fen, uci string }{
		{"vertical exposure", "4r1k1/8/8/3pP3/8/8/8/4K3 w - d6 0 1", "e5d6"},
		{"horizontal exposure", "4k3/8/8/r4pPK/8/8/8/8 w - f6 0 1", "g5f6"},
	} {
		t.Run(tc.name, func(t *testing.T) { qaRulesAssertMove(t, qaRulesMatch(t, tc.fen), tc.uci, false) })
	}
	for _, tc := range []struct {
		fen                          string
		moves                        []string
		destination, removed, source chess.Square
		piece                        chess.Piece
	}{
		{"7k/3p4/8/4P3/8/8/8/7K b - - 0 1", []string{"d7d5", "e5d6"}, chess.D6, chess.D5, chess.E5, chess.WhitePawn},
		{"7k/8/8/8/4p3/8/3P4/7K w - - 0 1", []string{"d2d4", "e4d3"}, chess.D3, chess.D4, chess.E4, chess.BlackPawn},
	} {
		t.Run(tc.moves[1], func(t *testing.T) {
			m := qaRulesMatch(t, tc.fen)
			qaRulesPlay(t, m, tc.moves...)
			board := m.game.Position().Board()
			if board.Piece(tc.destination) != tc.piece || board.Piece(tc.removed) != chess.NoPiece || board.Piece(tc.source) != chess.NoPiece {
				t.Fatalf("wrong EP board: %s", m.game.FEN())
			}
		})
	}
	m := qaRulesMatch(t, "7k/3p4/8/4P3/8/8/8/7K b - - 0 1")
	qaRulesPlay(t, m, "d7d5")
	if !qaRulesHasMove(m, "e5d6") {
		t.Fatal("immediate EP absent")
	}
	qaRulesPlay(t, m, "h1h2", "h8h7")
	qaRulesAssertMove(t, m, "e5d6", false)
}

func TestAdditionalRulesAllPromotionChoicesBothColors(t *testing.T) {
	for _, tc := range []struct {
		name, fen, prefix string
		target            chess.Square
		color             chess.Color
	}{
		{"white quiet", "7k/P7/8/8/8/8/8/7K w - - 0 1", "a7a8", chess.A8, chess.White},
		{"black quiet", "7k/8/8/8/8/8/p7/7K b - - 0 1", "a2a1", chess.A1, chess.Black},
		{"white capture", "r6k/1P6/8/8/8/8/8/7K w - - 0 1", "b7a8", chess.A8, chess.White},
		{"black capture", "7k/8/8/8/8/8/1p6/R6K b - - 0 1", "b2a1", chess.A1, chess.Black},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, p := range []struct {
				suffix string
				kind   chess.PieceType
			}{{"q", chess.Queen}, {"r", chess.Rook}, {"b", chess.Bishop}, {"n", chess.Knight}} {
				t.Run(p.suffix, func(t *testing.T) {
					m := qaRulesMatch(t, tc.fen)
					qaRulesPlay(t, m, tc.prefix+p.suffix)
					piece := m.game.Position().Board().Piece(tc.target)
					if piece.Type() != p.kind || piece.Color() != tc.color {
						t.Fatalf("wrong promoted piece: %v", piece)
					}
				})
			}
			for _, bad := range []string{"", "k", "p"} {
				qaRulesAssertMove(t, qaRulesMatch(t, tc.fen), tc.prefix+bad, false)
			}
		})
	}
}

func TestAdditionalRulesCheckMateAndStalemateAreDistinct(t *testing.T) {
	for _, tc := range []struct {
		name, fen string
		check     bool
		outcome   chess.Outcome
		method    chess.Method
	}{
		{"black checkmate", "7k/6Q1/5K2/8/8/8/8/8 b - - 0 1", true, chess.WhiteWon, chess.Checkmate},
		{"white checkmate", "8/8/8/8/8/5k2/6q1/7K w - - 0 1", true, chess.BlackWon, chess.Checkmate},
		{"black stalemate", "7k/5Q2/6K1/8/8/8/8/8 b - - 0 1", false, chess.Draw, chess.Stalemate},
		{"white stalemate", "8/8/8/8/8/6k1/5q2/7K w - - 0 1", false, chess.Draw, chess.Stalemate},
		{"check with escape", "7k/8/8/8/8/8/8/K6R b - - 0 1", true, chess.NoOutcome, chess.NoMethod},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := qaRulesMatch(t, tc.fen)
			if inCheck(m.game.Position()) != tc.check || m.game.Outcome() != tc.outcome || m.game.Method() != tc.method {
				t.Fatalf("check=%v outcome=%v method=%v", inCheck(m.game.Position()), m.game.Outcome(), m.game.Method())
			}
			if tc.outcome != chess.NoOutcome && len(m.game.ValidMoves()) != 0 {
				t.Fatal("mate/stalemate has legal moves")
			}
		})
	}
	m := qaRulesMatch(t, "7k/5Q2/6K1/8/8/8/8/8 w - - 149 1")
	qaRulesPlay(t, m, "f7g7")
	if m.game.Method() != chess.Checkmate || m.game.Outcome() != chess.WhiteWon || m.game.Position().HalfMoveClock() != 150 {
		t.Fatal("checkmate must take priority over the 75-move rule")
	}
}

func TestAdditionalRulesMaterialOutcomes(t *testing.T) {
	for _, tc := range []struct {
		name, fen string
		dead      bool
	}{
		{"K versus K", "7k/8/8/8/8/8/8/7K w - - 0 1", true},
		{"KB versus K", "7k/8/8/8/8/8/8/1B5K w - - 0 1", true},
		{"KN versus K", "7k/8/8/8/8/8/8/N6K w - - 0 1", true},
		{"same color bishops", "2b4k/8/8/8/8/8/8/1B5K w - - 0 1", true},
		{"same color promoted bishops", "7k/8/8/8/8/8/8/1B1B3K w - - 0 1", true},
		{"opposite color bishops", "1b5k/8/8/8/8/8/8/1B5K w - - 0 1", false},
		{"KNN versus K can mate", "7k/8/8/8/8/8/8/NN5K w - - 0 1", false},
		{"KN versus KN can mate", "n6k/8/8/8/8/8/8/N6K w - - 0 1", false},
		{"KBN versus K", "7k/8/8/8/8/8/8/NB5K w - - 0 1", false},
		{"pawn remains", "7k/8/8/8/8/8/P7/7K w - - 0 1", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := qaRulesMatch(t, tc.fen)
			if tc.dead {
				if m.game.Outcome() != chess.Draw || m.game.Method() != chess.InsufficientMaterial {
					t.Fatalf("dead material outcome=%v method=%v", m.game.Outcome(), m.game.Method())
				}
			} else if m.game.Outcome() != chess.NoOutcome {
				t.Fatalf("checkmate is still possible; outcome=%v", m.game.Outcome())
			}
		})
	}
	// Two knights cannot force mate, but FIDE dead-position detection asks
	// whether any legal sequence can produce mate. This is an actual mate.
	m := qaRulesMatch(t, "7k/5N2/5NK1/8/8/8/8/8 b - - 0 1")
	if m.game.Method() != chess.Checkmate || m.game.Outcome() != chess.WhiteWon {
		t.Fatal("two-knight mate was treated as insufficient")
	}
}

func TestAdditionalRulesFiftyMovesAndResets(t *testing.T) {
	for _, tc := range []struct {
		name, fen, uci string
		clock          int
		claim          bool
	}{
		{"100th halfmove", "7k/8/8/8/8/8/8/R6K w - - 99 1", "a1a2", 100, true},
		{"pawn resets", "7k/8/8/8/8/8/4P3/R6K w - - 99 1", "e2e3", 0, false},
		{"capture resets", "7k/8/8/8/8/8/n7/R6K w - - 99 1", "a1a2", 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := qaRulesMatch(t, tc.fen)
			if m.claimable() {
				t.Fatal("claim offered before 100 halfmoves")
			}
			qaRulesPlay(t, m, tc.uci)
			if m.game.Position().HalfMoveClock() != tc.clock || m.claimable() != tc.claim || m.game.Outcome() != chess.NoOutcome {
				t.Fatalf("clock=%d claim=%v outcome=%v", m.game.Position().HalfMoveClock(), m.claimable(), m.game.Outcome())
			}
		})
	}
}

// FIDE 5.1.2: resignation is a draw if the opponent cannot possibly mate.
// A lone king cannot give check; the resigning player's rook does not help it.
func TestAdditionalRulesResignationBareKingException(t *testing.T) {
	for _, tc := range []struct {
		name, fen string
		resigning chess.Color
	}{
		{"white resigns", "7k/8/8/8/8/8/R7/K7 w - - 0 1", chess.White},
		{"black resigns", "7k/7r/8/8/8/8/8/K7 b - - 0 1", chess.Black},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := qaRulesMatch(t, tc.fen)
			if m.game.Outcome() != chess.NoOutcome {
				t.Fatal("fixture already terminal")
			}
			m.game.Resign(tc.resigning)
			if m.game.Outcome() != chess.Draw || m.game.Method() != chess.Resignation {
				t.Fatalf("bare opposing king: outcome=%v method=%v, want draw by resignation", m.game.Outcome(), m.game.Method())
			}
		})
	}
}

// FIDE 9.2.3 distinguishes positions only when en passant is a legal move.
// Here e5xd6 would expose the white king to the rook on e8, so the initial
// position with d6 in its FEN is identical to the later one with no EP square.
func TestAdditionalRulesPinnedEnPassantRepetition(t *testing.T) {
	m := qaRulesMatch(t, "1n2r1k1/8/8/3pP3/8/8/8/4K1N1 w - d6 0 1")
	for _, move := range m.game.ValidMoves() {
		if (chess.UCINotation{}).Encode(m.game.Position(), &move) == "e5d6" {
			t.Fatal("pinned en passant was a legal move")
		}
	}
	for i := 0; i < 2; i++ {
		qaRulesPlay(t, m, "g1f3", "b8c6", "f3g1", "c6b8")
	}
	if !m.claimable() {
		t.Fatalf("three occurrences with no legal EP must be claimable; eligible=%v FEN=%s", m.game.EligibleDraws(), m.game.FEN())
	}
	if err := m.claim(); err != nil {
		t.Fatal(err)
	}
	if m.game.Outcome() != chess.Draw || m.game.Method() != chess.ThreefoldRepetition || methodReason(m.game.Method()) != "Triple repetición" {
		t.Fatalf("wrong corrected claim outcome=%v method=%v", m.game.Outcome(), m.game.Method())
	}
	if err := m.undo(); err != nil {
		t.Fatal(err)
	}
	if m.game.Outcome() != chess.NoOutcome || m.claimable() {
		t.Fatal("undo retained removed repetition or draw outcome")
	}
	qaRulesPlay(t, m, "c6b8")
	if !m.claimable() {
		t.Fatal("replayed repetition lost initial pinned-EP occurrence")
	}
}

func TestAdditionalRulesEnPassantIdentityAndFivefold(t *testing.T) {
	cycle := []string{"g1f3", "b8c6", "f3g1", "c6b8"}
	for _, tc := range []struct {
		name, fen     string
		claimAfterTwo bool
	}{
		{"legal EP separates", "rn4k1/8/8/3pP3/8/8/8/4K1N1 w - d6 0 1", false},
		{"no adjacent pawn ignored", "rn4k1/8/8/3p4/8/8/8/4K1N1 w - d6 0 1", true},
		{"pinned EP ignored", "1n2r1k1/8/8/3pP3/8/8/8/4K1N1 w - d6 0 1", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := qaRulesMatch(t, tc.fen)
			for i := 0; i < 2; i++ {
				qaRulesPlay(t, m, cycle...)
			}
			if m.claimable() != tc.claimAfterTwo {
				t.Fatalf("claim after two cycles=%v", m.claimable())
			}
			qaRulesPlay(t, m, cycle...)
			if !m.claimable() || m.game.Outcome() != chess.NoOutcome {
				t.Fatal("threefold must be a claim, not automatic")
			}
		})
	}
	m := qaRulesMatch(t, "1n2r1k1/8/8/3pP3/8/8/8/4K1N1 w - d6 0 1")
	for i := 0; i < 4; i++ {
		qaRulesPlay(t, m, cycle...)
	}
	if m.game.Outcome() != chess.Draw || m.game.Method() != chess.FivefoldRepetition || methodReason(m.game.Method()) != "Cinco repeticiones" {
		t.Fatalf("wrong corrected fivefold outcome=%v method=%v", m.game.Outcome(), m.game.Method())
	}
	if m.claimable() {
		t.Fatal("terminal draw is claimable")
	}
	if err := m.playUCI("g1f3"); err == nil {
		t.Fatal("move after automatic draw accepted")
	}
	if err := m.undo(); err != nil {
		t.Fatal(err)
	}
	if m.game.Outcome() != chess.NoOutcome || m.game.Method() != chess.NoMethod {
		t.Fatal("undo retained automatic-draw method")
	}
}

func TestAdditionalRulesCastlingRightsSeparateRepetitions(t *testing.T) {
	m := qaRulesMatch(t, "r3k3/8/8/8/8/8/8/4K2R w K - 0 1")
	cycle := []string{"e1f1", "e8f8", "f1e1", "f8e8"}
	for i := 0; i < 2; i++ {
		qaRulesPlay(t, m, cycle...)
	}
	if m.claimable() {
		t.Fatal("initial castling-rights position counted as no-rights repetition")
	}
	qaRulesPlay(t, m, cycle...)
	if !m.claimable() {
		t.Fatal("three no-rights occurrences should be claimable")
	}
}

func qaRulesPerft(pos *chess.Position, depth int) uint64 {
	if depth == 0 {
		return 1
	}
	var n uint64
	for _, move := range pos.ValidMoves() {
		n += qaRulesPerft(pos.Update(&move), depth-1)
	}
	return n
}

func TestAdditionalRulesPerftIndependentFixtures(t *testing.T) {
	// Exact published perft fixtures 4–6 supplement the initial, Kiwipete and
	// rook/pawn fixtures already covered by game_test.go.
	for _, tc := range []struct {
		name, fen string
		counts    []uint64
	}{
		{"position 4 promotions castles pins", "r3k2r/Pppp1ppp/1b3nbN/nP6/BBP1P3/q4N2/Pp1P2PP/R2Q1RK1 w kq - 0 1", []uint64{6, 264, 9467}},
		{"position 5 promotion check", "rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8", []uint64{44, 1486, 62379}},
		{"position 6 middlegame", "r4rk1/1pp1qppp/p1np1n2/2b1p1B1/2B1P1b1/P1NP1N2/1PP1QPPP/R4RK1 w - - 0 10", []uint64{46, 2079, 89890}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := qaRulesMatch(t, tc.fen)
			for i, want := range tc.counts {
				if got := qaRulesPerft(m.game.Position(), i+1); got != want {
					t.Fatalf("depth=%d got=%d want=%d", i+1, got, want)
				}
			}
		})
	}
	// An independent depth-2 divide fixture catches offsetting perft errors
	// that a single grand total can hide.
	initial := qaRulesMatch(t, "")
	got := make([]string, 0, 20)
	for _, move := range initial.game.ValidMoves() {
		got = append(got, fmt.Sprintf("%s:%d", (chess.UCINotation{}).Encode(initial.game.Position(), &move), qaRulesPerft(initial.game.Position().Update(&move), 1)))
	}
	sort.Strings(got)
	want := []string{"a2a3:20", "a2a4:20", "b1a3:20", "b1c3:20", "b2b3:20", "b2b4:20", "c2c3:20", "c2c4:20", "d2d3:20", "d2d4:20", "e2e3:20", "e2e4:20", "f2f3:20", "f2f4:20", "g1f3:20", "g1h3:20", "g2g3:20", "g2g4:20", "h2h3:20", "h2h4:20"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("initial divide=%v want=%v", got, want)
	}
}
