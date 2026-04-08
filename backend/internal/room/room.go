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
	ID         string
	players    [2]*ws.Client
	mu         sync.Mutex
	count      int
	game       *game.Game
	interval   time.Duration
	onGameOver func()
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

		r.players[0].SetOnMessage(func(data []byte) { r.game.HandleMessage("p1", data) })
		r.players[1].SetOnMessage(func(data []byte) { r.game.HandleMessage("p2", data) })

		r.game.Start()
	}

	return idx, nil
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
	r := &Room{ID: id, interval: m.interval}
	r.onGameOver = func() {
		time.AfterFunc(5*time.Second, func() {
			m.Remove(id)
		})
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
