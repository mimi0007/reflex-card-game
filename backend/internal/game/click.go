package game

import (
	"log"
	"os"
	"strconv"
	"time"
)

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
		g.resolveRound(opp, playerID, "false_start")
		return
	}

	g.clickMu.Lock()
	if g.roundResolved {
		g.clickMu.Unlock()
		log.Printf("player %s clicked ace but round already resolved — discarding", playerID)
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
			if g.OnGameOver != nil {
				g.OnGameOver()
			}
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
