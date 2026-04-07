package game

import (
	"os"
	"strconv"
	"time"
)

// HandleClick is called from a client's ReadPump goroutine when a "click"
// message arrives. Two goroutines may call this simultaneously (one per player),
// so ace resolution is protected by clickMu.
func (g *Game) HandleClick(playerID string) {
	g.mu.Lock()
	if g.state != StatePlaying || !g.cardRevealed {
		g.mu.Unlock()
		return
	}
	card := g.currentCard
	g.mu.Unlock()

	opp := opponent(playerID)

	if card.Rank != "A" {
		// False start — the clicking player loses; opponent wins.
		// No mutex needed: the state transitions to StateRoundEnd inside
		// resolveRound, so any subsequent click from the other player is
		// dropped by the StatePlaying guard above.
		g.resolveRound(opp, playerID, "false_start")
		return
	}

	// Ace click — first caller wins; protect with clickMu.
	g.clickMu.Lock()
	if g.roundResolved {
		g.clickMu.Unlock()
		return
	}
	g.roundResolved = true
	g.clickMu.Unlock()

	g.resolveRound(playerID, opp, "ace_clicked")
}

func (g *Game) resolveRound(winner, loser, reason string) {
	g.mu.Lock()
	g.state = StateRoundEnd
	g.scores[winner]++
	scores := map[string]int{"p1": g.scores["p1"], "p2": g.scores["p2"]}
	round := g.round
	g.mu.Unlock()

	g.broadcast(map[string]any{
		"type":   "round_result",
		"winner": winner,
		"loser":  loser,
		"reason": reason,
		"scores": scores,
		"round":  round,
	})

	if scores[winner] >= roundsToWin() {
		time.AfterFunc(2500*time.Millisecond, func() {
			g.broadcast(map[string]any{
				"type":   "game_over",
				"winner": winner,
				"scores": scores,
			})
			g.Stop()
		})
		return
	}

	time.AfterFunc(2500*time.Millisecond, g.newRound)
}

func (g *Game) newRound() {
	g.mu.Lock()
	g.round++
	g.deck = buildDeck()
	shuffle(g.deck)
	g.cardIdx = 0
	g.cardRevealed = false
	g.state = StatePlaying
	round := g.round
	g.mu.Unlock()

	g.clickMu.Lock()
	g.roundResolved = false
	g.clickMu.Unlock()

	g.broadcast(map[string]any{
		"type":  "round_start",
		"round": round,
	})
}

func roundsToWin() int {
	if v := os.Getenv("ROUNDS_TO_WIN"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 3
}

func opponent(playerID string) string {
	if playerID == "p1" {
		return "p2"
	}
	return "p1"
}
