package main

import (
	"strings"
	"testing"

	"github.com/corentings/chess/v2"
)

func perft(pos *chess.Position, depth int) int {
	if depth == 0 {
		return 1
	}
	n := 0
	for _, move := range pos.ValidMoves() {
		n += perft(pos.Update(&move), depth-1)
	}
	return n
}

func TestKnownPositionPerft(t *testing.T) {
	for _, c := range []struct {
		fen         string
		depth, want int
	}{
		{"", 3, 8902},
		{"r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1", 3, 97862},
		{"8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1", 3, 2812},
	} {
		m, err := newMatch(c.fen)
		if err != nil {
			t.Fatal(err)
		}
		if got := perft(m.game.Position(), c.depth); got != c.want {
			t.Fatalf("perft depth %d: %d want %d", c.depth, got, c.want)
		}
	}
}

func TestMoveHistoryCheckMateAndUndo(t *testing.T) {
	m, _ := newMatch("")
	for _, move := range []string{"f2f3", "e7e5", "g2g4", "d8h4"} {
		if err := m.playUCI(move); err != nil {
			t.Fatal(err)
		}
	}
	if m.game.Outcome().String() != "0-1" || !inCheck(m.game.Position()) || m.history[3].SAN != "Qh4#" {
		t.Fatal("fools mate missing")
	}
	if err := m.playUCI("e2e3"); err == nil {
		t.Fatal("move after mate accepted")
	}
	if err := m.undo(); err != nil {
		t.Fatal(err)
	}
	if m.game.Outcome().String() != "*" || len(m.history) != 3 || strings.Contains(m.game.String(), "Qh4") {
		t.Fatal("undo retained terminal variation")
	}
}

func TestRuleEdgeCasesAndPromotions(t *testing.T) {
	for _, c := range []struct{ fen, uci string }{
		{"4kr2/8/8/8/8/8/8/4K2R w K - 0 1", "e1g1"},
		{"4r1k1/8/8/3pP3/8/8/8/4K3 w - d6 0 1", "e5d6"},
	} {
		m, err := newMatch(c.fen)
		if err != nil {
			t.Fatal(err)
		}
		before := m.game.FEN()
		if err := m.playUCI(c.uci); err == nil || m.game.FEN() != before {
			t.Fatalf("illegal move accepted: %s", c.uci)
		}
	}
	for _, promo := range []string{"q", "r", "b", "n"} {
		m, _ := newMatch("7k/P7/8/8/8/8/8/7K w - - 0 1")
		if err := m.playUCI("a7a8" + promo); err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(m.game.FEN(), strings.ToUpper(promo)) {
			t.Fatalf("wrong promotion %s", promo)
		}
	}
	m, _ := newMatch("7k/5Q2/6K1/8/8/8/8/8 b - - 0 1")
	if m.game.Method() != chess.Stalemate || m.game.Outcome() != chess.Draw {
		t.Fatal("stalemate not detected")
	}
	m, _ = newMatch("7k/8/8/8/8/8/8/7K w - - 0 1")
	if m.game.Method() != chess.InsufficientMaterial {
		t.Fatal("insufficient material not detected")
	}
}

func TestRepetitionUndoAndAutomaticDraws(t *testing.T) {
	m, _ := newMatch("")
	cycle := []string{"g1f3", "g8f6", "f3g1", "f6g8"}
	for i := 0; i < 2; i++ {
		for _, move := range cycle {
			if err := m.playUCI(move); err != nil {
				t.Fatal(err)
			}
		}
	}
	if !m.claimable() {
		t.Fatal("threefold claim missing")
	}
	if err := m.undo(); err != nil {
		t.Fatal(err)
	}
	if m.claimable() {
		t.Fatal("undo preserved removed repetition")
	}
	if err := m.playUCI("f6g8"); err != nil {
		t.Fatal(err)
	}
	if !m.claimable() {
		t.Fatal("replay lost repetition")
	}
	if err := m.claim(); err != nil || m.game.Outcome() != chess.Draw {
		t.Fatal("draw claim failed")
	}
	m, _ = newMatch("")
	for i := 0; i < 4; i++ {
		for _, move := range cycle {
			if err := m.playUCI(move); err != nil {
				t.Fatal(err)
			}
		}
	}
	if m.game.Method() != chess.FivefoldRepetition {
		t.Fatal("fivefold automatic draw missing")
	}
	m, _ = newMatch("7k/8/8/8/8/8/8/R6K w - - 100 1")
	if !m.claimable() {
		t.Fatal("fifty move claim missing")
	}
	m, _ = newMatch("7k/8/8/8/8/8/8/R6K w - - 149 1")
	if err := m.playUCI("a1a2"); err != nil {
		t.Fatal(err)
	}
	if m.game.Method() != chess.SeventyFiveMoveRule {
		t.Fatal("75 move automatic draw missing")
	}
}
