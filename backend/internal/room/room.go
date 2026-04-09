package room

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/mimi0007/reflex-card-game/backend/internal/game"
	"github.com/mimi0007/reflex-card-game/backend/internal/ws"
)

const (
	codeChars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	codeLen   = 6
)

func newRoomID() string {
	b := make([]byte, codeLen)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(codeChars))))
		b[i] = codeChars[n.Int64()]
	}
	return string(b)
}

func sendJSON(client *ws.Client, v any) {
	data, _ := json.Marshal(v)
	client.SafeSend(data)
}

type Room struct {
	ID                 string
	players            [2]*ws.Client
	mu                 sync.Mutex
	count              int
	game               *game.Game
	interval           time.Duration
	onGameOver         func()
	playAgainRequester int
	removalTimer       *time.Timer
	playAgainTimer     *time.Timer
}

func (r *Room) Join(client *ws.Client) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.count >= 2 {
		return 0, fmt.Errorf("room %s is full", r.ID)
	}

	idx := r.count
	r.players[idx] = client
	r.count++

	if idx == 0 {
		sendJSON(r.players[0], map[string]string{
			"type":    "waiting",
			"room_id": r.ID,
		})
	} else {
		sendJSON(r.players[0], map[string]any{"type": "game_start", "player_id": "p1"})
		sendJSON(r.players[1], map[string]any{"type": "game_start", "player_id": "p2"})

		r.game = game.New(r.players, r.interval)
		r.game.OnGameOver = r.onGameOver

		r.players[0].SetOnMessage(func(data []byte) { r.HandleMessage(0, data) })
		r.players[1].SetOnMessage(func(data []byte) { r.HandleMessage(1, data) })

		r.game.Start()
	}

	return idx, nil
}

func (r *Room) HandleMessage(playerIdx int, data []byte) {
	var msg struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}

	playerID := "p1"
	if playerIdx == 1 {
		playerID = "p2"
	}

	switch msg.Type {
	case "play_again":
		r.handlePlayAgainRequest(playerIdx)
	case "play_again_accept":
		r.handlePlayAgainAccept()
	case "play_again_decline":
		r.handlePlayAgainDecline()
	default:
		r.mu.Lock()
		g := r.game
		r.mu.Unlock()
		if g != nil {
			g.HandleMessage(playerID, data)
		}
	}
}

func (r *Room) handlePlayAgainRequest(playerIdx int) {
	r.mu.Lock()
	r.playAgainRequester = playerIdx
	other := r.players[1-playerIdx]
	requesterClient := r.players[playerIdx]
	r.mu.Unlock()

	if other != nil {
		sendJSON(other, map[string]string{"type": "play_again_request"})
	}

	t := time.AfterFunc(30*time.Second, func() {
		r.mu.Lock()
		if r.playAgainRequester == -1 {
			r.mu.Unlock()
			return
		}
		r.playAgainRequester = -1
		otherClient := r.players[1-playerIdx]
		r.mu.Unlock()

		if requesterClient != nil {
			sendJSON(requesterClient, map[string]string{"type": "play_again_declined"})
		}
		if otherClient != nil {
			sendJSON(otherClient, map[string]string{"type": "play_again_declined"})
		}
	})

	r.mu.Lock()
	r.playAgainTimer = t
	r.mu.Unlock()
}

func (r *Room) handlePlayAgainAccept() {
	r.mu.Lock()
	requester := r.playAgainRequester
	timer := r.removalTimer
	paTimer := r.playAgainTimer
	players := r.players
	r.playAgainRequester = -1
	r.mu.Unlock()

	if paTimer != nil {
		paTimer.Stop()
	}

	if requester == -1 {
		return
	}

	if timer != nil {
		timer.Stop()
	}

	r.mu.Lock()
	r.game = game.New(players, r.interval)
	g := r.game
	r.mu.Unlock()

	g.OnGameOver = r.onGameOver
	r.players[0].SetOnMessage(func(data []byte) { r.HandleMessage(0, data) })
	r.players[1].SetOnMessage(func(data []byte) { r.HandleMessage(1, data) })

	sendJSON(r.players[0], map[string]any{"type": "game_start", "player_id": "p1"})
	sendJSON(r.players[1], map[string]any{"type": "game_start", "player_id": "p2"})

	g.Start()
}

func (r *Room) handlePlayAgainDecline() {
	r.mu.Lock()
	requester := r.playAgainRequester
	paTimer := r.playAgainTimer
	r.playAgainRequester = -1
	r.mu.Unlock()

	if paTimer != nil {
		paTimer.Stop()
	}

	if requester == -1 {
		return
	}

	sendJSON(r.players[requester], map[string]string{"type": "play_again_declined"})
}

func (r *Room) StopGame() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	g := r.game
	if g == nil || !g.IsRunning() {
		return false
	}
	g.Stop()
	return true
}

func (r *Room) BroadcastExcept(exclude *ws.Client, v any) {
	data, _ := json.Marshal(v)
	r.mu.Lock()
	players := r.players
	r.mu.Unlock()
	for _, p := range players {
		if p != nil && p != exclude {
			p.SafeSend(data)
		}
	}
}

type Manager struct {
	mu       sync.Mutex
	rooms    map[string]*Room
	interval time.Duration
}

func NewManager(interval time.Duration) *Manager {
	return &Manager{
		rooms:    make(map[string]*Room),
		interval: interval,
	}
}

func (m *Manager) newRoom(id string) *Room {
	r := &Room{ID: id, interval: m.interval, playAgainRequester: -1}
	r.onGameOver = func() {
		t := time.AfterFunc(30*time.Second, func() {
			m.Remove(id)
		})
		r.mu.Lock()
		r.removalTimer = t
		r.mu.Unlock()
	}
	return r
}

func (m *Manager) GetOrCreate(roomID string) *Room {
	m.mu.Lock()
	defer m.mu.Unlock()

	if roomID == "" {
		return m.createLocked()
	}

	if r, ok := m.rooms[roomID]; ok {
		return r
	}

	r := m.newRoom(roomID)
	m.rooms[roomID] = r
	return r
}

func (m *Manager) createLocked() *Room {
	for {
		id := newRoomID()
		if _, exists := m.rooms[id]; !exists {
			r := m.newRoom(id)
			m.rooms[id] = r
			return r
		}
	}
}

func (m *Manager) Remove(roomID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.rooms, roomID)
}
