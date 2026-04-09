# Reflex Card Game

A real-time two-player reflex game. Cards flip one by one — the first player to click when an Ace appears wins the round. Click on a non-Ace and you lose the round immediately. First to win 3 rounds wins the game.

## Live Demo

| Service | URL |
|---------|-----|
| Frontend | https://reflex-card-game.vercel.app |
| Backend | https://reflex-card-game-backend.fly.dev |

## How to Play

1. Open the game and click **Create Game** — you get a shareable room link
2. Share the link with a friend — they join the same room
3. Cards are revealed one by one on screen
4. **Wait for an Ace** — then click as fast as you can
5. First to click an Ace wins the round. Click any other card and you lose the round immediately
6. First player to win 3 rounds wins the game

---

## Architecture

### Overview

```
Player 1 Browser          Go Server              Player 2 Browser
     |                        |                        |
     |--- WS Connect -------->|                        |
     |                        |<------- WS Connect ----|
     |    [Both connected → game starts]               |
     |<-- card_reveal --------|-------- card_reveal -->|
     |--- click ------------->|                        |
     |<-- round_result -------|-------- round_result ->|
     |<-- round_start --------|-------- round_start -->|
     |         ...repeat until winner...               |
     |<-- game_over ----------|-------- game_over ---->|
```

The server is the single source of truth. It controls card timing, validates clicks, and broadcasts results. Neither client ever knows what the next card will be.

### Project Structure

```
reflex-card-game/
├── backend/
│   ├── cmd/server/main.go          # Entry point, HTTP server, WebSocket upgrade
│   └── internal/
│       ├── game/
│       │   ├── game.go             # Game engine: state machine, deck, card reveal loop
│       │   └── click.go            # Click handling and round resolution
│       ├── ws/
│       │   ├── hub.go              # WebSocket hub (connection registry)
│       │   └── client.go           # Per-connection read/write pumps
│       └── room/
│           └── room.go             # Room management, player slots, game lifecycle
└── frontend/
    └── src/
        ├── pages/                  # LobbyPage, GamePage
        ├── components/             # GameScreen, Card, FlipCard, Scoreboard, etc.
        ├── hooks/useWebSocket.ts   # WebSocket lifecycle hook
        └── types/messages.ts      # TypeScript discriminated union message types
```

### Tech Stack

| Layer | Technology |
|-------|------------|
| Backend | Go 1.22 |
| WebSocket | gorilla/websocket |
| Frontend | React 19 + TypeScript + Vite |
| Backend deploy | Fly.io |
| Frontend deploy | Vercel |

---

## Running Locally

### Docker (recommended)

```bash
docker compose up
```

Frontend at `http://localhost:5173`, backend at `http://localhost:8080`. Open two browser tabs to test both players.

For live reloading while editing:

```bash
docker compose watch
```

Source changes under `frontend/src` sync into the container instantly. Backend changes trigger a rebuild.

### Without Docker

**Prerequisites:** Go 1.21+, Node.js 18+

**Backend**

```bash
cd backend
go mod download
go run cmd/server/main.go
# Server starts on http://localhost:8080
```

**Frontend**

```bash
cd frontend
npm install
cp .env.example .env        # VITE_WS_URL=ws://localhost:8080
npm run dev
# App starts on http://localhost:5173
```

Open two browser tabs to `http://localhost:5173` to test both players locally.

### Environment Variables

**Backend**

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `FRONTEND_ORIGIN` | `https://reflex-card-game.vercel.app` | Allowed CORS origin — omit to allow all origins (local dev) |
| `CARD_INTERVAL_MS` | `3000` | Milliseconds between card reveals |
| `ROUNDS_TO_WIN` | `3` | Rounds needed to win the game |

**Frontend**

| Variable | Description |
|----------|-------------|
| `VITE_WS_URL` | WebSocket URL of the backend (e.g. `wss://reflex-card-game-backend.fly.dev`) |

---

## WebSocket Protocol

### Server → Client

```json
{ "type": "waiting" }
{ "type": "game_start", "player_id": "p1" }
{ "type": "card_reveal", "card": "AS", "rank": "A", "suit": "spades" }
{ "type": "round_result", "winner": "p1", "loser": "p2", "reason": "ace_clicked", "scores": { "p1": 1, "p2": 0 }, "round": 1 }
{ "type": "round_start", "round": 2 }
{ "type": "game_over", "winner": "p1", "scores": { "p1": 3, "p2": 1 } }
{ "type": "game_over", "reason": "deck_exhausted", "draw": true, "scores": { "p1": 2, "p2": 2 } }
{ "type": "player_disconnected", "player_id": "p1", "message": "p1 disconnected" }
{ "type": "play_again_request" }
{ "type": "play_again_declined" }
```

### Client → Server

```json
{ "type": "click" }
{ "type": "play_again" }
{ "type": "play_again_accept" }
{ "type": "play_again_decline" }
```

---

## Key Design Decisions & Tradeoffs

### Server-authoritative game state

**Decision:** The server owns all game state — deck order, click timing, scores.  
**Why it matters:** If clients controlled card timing, a player could inspect the deck in memory before Aces appear. If click resolution happened client-side, simultaneous clicks couldn't be fairly adjudicated.  
**Tradeoff:** Every interaction requires a round-trip to the server. On a slow connection (~200ms RTT), a player might click an Ace but lose because the server's next card reveal crossed in flight. This is acceptable for a reflex game where network latency is a known variable for all players.

### In-memory state, no database

**Decision:** All rooms and game state live in the Go process's memory.  
**Why it matters:** Games are ephemeral (5–10 minutes). Persisting to a database would add latency to every card reveal and every click resolution — the two operations that need to be fast.  
**Tradeoff:** A server restart kills all active games. Horizontal scaling is not possible without a shared state layer (Redis pub/sub, etc.). For a single-server deployment this is fine; at scale it would need rethinking.

### Two mutexes for click resolution

**Decision:** `game.mu` protects card state; `clickMu` protects the `roundResolved` flag separately.  
**Why it matters:** `revealNext` holds `game.mu` while updating the current card. If `HandleClick` needed both locks in sequence, and `revealNext` ever needed `clickMu`, you'd have a deadlock window. Keeping the "first click wins" check in its own `clickMu` scope removes this risk entirely.  
**Tradeoff:** Two mutexes are harder to reason about than one. The invariant that must hold: `clickMu` is only ever acquired while `game.mu` is **not** held (or the reverse — never both at once).

### No reconnection support

**Decision:** If a player disconnects, the game ends immediately (opponent sees `player_disconnected`).  
**Why it matters:** Reconnection requires associating a new WebSocket to an existing player slot, buffering missed messages, and deciding how long to wait. For a short reflex game, this complexity isn't worth it.  
**Tradeoff:** A brief network blip ends the game. Players on mobile or flaky Wi-Fi will find this frustrating.

### No accounts or lobbies

**Decision:** Room codes are generated client-side and used once. No login, no matchmaking.  
**Why it matters:** Zero-friction entry — open the link and play. No signup wall.  
**Tradeoff:** No way to re-find a past game, no leaderboards, no play history. Also, room codes are not secret — anyone who guesses one can join as the second player.

### Props-down state management (no global store)

**Decision:** WebSocket state flows from `GamePage` down through props. No Redux, Zustand, or Context.  
**Why it matters:** The component tree is shallow (3 levels max). Adding a global store would be indirection with no benefit at this scale.  
**Tradeoff:** `GamePage` props become the chokepoint. If the tree grew — say, adding a chat panel that also needs WebSocket access — you'd need to either prop-drill or add a context layer.

### CSS design tokens, no CSS-in-JS

**Decision:** Global CSS variables for colors and spacing; component styles in separate `.css` files.  
**Why it matters:** No runtime overhead, no build complexity. The design is simple enough that scoped styles add no real value.  
**Tradeoff:** No style isolation — a class name collision would silently break styles. Manageable at this scale; would need CSS Modules or similar at larger scale.

---

### Deep Dive

To deep dive the backend and frontend click [here](DEEPDIVE.md)
