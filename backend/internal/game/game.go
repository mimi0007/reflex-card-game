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

type ClickEvent struct {
	PlayerID string
}

type Game struct {
	mu       sync.Mutex
	deck     []Card
	cardIdx  int
	state    State
	players  [2]*ws.Client
	interval time.Duration
	clicks   chan ClickEvent
	done     chan struct{}
}

func New(players [2]*ws.Client, interval time.Duration) *Game {
	g := &Game{
		players:  players,
		interval: interval,
		clicks:   make(chan ClickEvent, 4),
		done:     make(chan struct{}),
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

func (g *Game) HandleMessage(playerID string, data []byte) {
	var msg struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}
	if msg.Type == "click" {
		select {
		case g.clicks <- ClickEvent{PlayerID: playerID}:
		default:
		}
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
		case ev := <-g.clicks:
			_ = ev
		}
	}
}

func (g *Game) revealNext() bool {
	g.mu.Lock()
	if g.cardIdx >= len(g.deck) {
		g.state = StateGameOver
		g.mu.Unlock()
		g.broadcast(map[string]any{
			"type":   "game_over",
			"winner": "",
			"scores": map[string]int{"p1": 0, "p2": 0},
		})
		return false
	}
	card := g.deck[g.cardIdx]
	g.cardIdx++
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
			select {
			case p.Send <- data:
			default:
			}
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
