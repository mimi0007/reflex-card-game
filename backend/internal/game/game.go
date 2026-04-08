package game

import (
	"encoding/json"
	"math/rand"
	"sync"
	"time"

	"github.com/mimi0007/reflex-card-game/backend/internal/ws"
)

type State int

const (
	StateWaiting  State = iota
	StatePlaying
	StateRoundEnd
	StateGameOver
)

type Card struct {
	Code string
	Rank string
	Suit string
}

type Game struct {
	mu            sync.Mutex
	clickMu       sync.Mutex
	deck          []Card
	cardIdx       int
	state         State
	players       [2]*ws.Client
	interval      time.Duration
	done          chan struct{}
	round         int
	scores        map[string]int
	roundResolved bool
	currentCard   Card
	cardRevealed  bool
	OnGameOver    func()
}

func New(players [2]*ws.Client, interval time.Duration) *Game {
	g := &Game{
		players:  players,
		interval: interval,
		done:     make(chan struct{}),
		round:    1,
		scores:   map[string]int{"p1": 0, "p2": 0},
	}
	g.deck = buildDeck()
	shuffle(g.deck)
	return g
}

func (g *Game) Start() {
	g.mu.Lock()
	g.state = StatePlaying
	g.mu.Unlock()
	go g.run()
}

func (g *Game) Stop() {
	select {
	case <-g.done:
	default:
		close(g.done)
	}
}

func (g *Game) IsRunning() bool {
	select {
	case <-g.done:
		return false
	default:
		return true
	}
}

func (g *Game) HandleMessage(playerID string, data []byte) {
	var msg struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}
	if msg.Type == "click" {
		g.HandleClick(playerID)
	}
}

func (g *Game) run() {
	ticker := time.NewTicker(g.interval)
	defer ticker.Stop()

	for {
		select {
		case <-g.done:
			return
		case <-ticker.C:
			if !g.revealNext() {
				return
			}
		}
	}
}

func (g *Game) revealNext() bool {
	g.mu.Lock()
	if g.state != StatePlaying {
		g.mu.Unlock()
		return true
	}
	if g.cardIdx >= len(g.deck) {
		g.state = StateGameOver
		scores := map[string]int{"p1": g.scores["p1"], "p2": g.scores["p2"]}
		onGameOver := g.OnGameOver
		g.mu.Unlock()

		msg := map[string]any{
			"type":   "game_over",
			"reason": "deck_exhausted",
			"scores": scores,
		}
		switch {
		case scores["p1"] > scores["p2"]:
			msg["winner"] = "p1"
		case scores["p2"] > scores["p1"]:
			msg["winner"] = "p2"
		default:
			msg["winner"] = ""
			msg["draw"] = true
		}
		g.broadcast(msg)
		if onGameOver != nil {
			onGameOver()
		}
		return false
	}
	card := g.deck[g.cardIdx]
	g.cardIdx++
	g.currentCard = card
	g.cardRevealed = true
	g.mu.Unlock()

	g.broadcast(map[string]any{
		"type": "card_reveal",
		"card": card.Code,
		"rank": card.Rank,
		"suit": card.Suit,
	})
	return true
}

func (g *Game) broadcast(v any) {
	data, _ := json.Marshal(v)
	for _, p := range g.players {
		if p != nil {
			p.SafeSend(data)
		}
	}
}

var ranks = []string{"A", "2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K"}

var suits = []struct {
	code string
	name string
}{
	{"S", "spades"},
	{"H", "hearts"},
	{"D", "diamonds"},
	{"C", "clubs"},
}

func buildDeck() []Card {
	cards := make([]Card, 0, 52)
	for _, suit := range suits {
		for _, rank := range ranks {
			cards = append(cards, Card{
				Code: rank + suit.code,
				Rank: rank,
				Suit: suit.name,
			})
		}
	}
	return cards
}

func shuffle(deck []Card) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})
}
