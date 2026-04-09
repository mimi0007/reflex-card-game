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

## Backend Deep Dive

### WebSocket Hub (`internal/ws/hub.go`)

The hub is a central connection registry using Go's channel-based concurrency model. A single `Run()` goroutine serializes all register/unregister/broadcast operations, avoiding lock contention:

```
Register chan   → adds client to map
Unregister chan → removes client, closes its Send channel
Broadcast chan  → fans out a message to all live clients
```

Each client has its own `Send chan []byte`. The hub writes to this channel; the client's `WritePump` goroutine drains it. This means the hub never blocks waiting for a slow client — it drops messages to disconnected ones instead.

### WebSocket Client (`internal/ws/client.go`)

Each connection gets a `Client` struct with independent `ReadPump` and `WritePump` goroutines. They share no state with each other.

- **ReadPump** reads incoming frames, calls `onMessage` callback (set by the room), and unregisters the client on disconnect.
- **WritePump** drains the `Send` channel with a 10s write timeout, and sends WebSocket ping frames every 54s to detect dead connections (60s pong deadline on the read side).

### Room Manager (`internal/room/room.go`)

A `RoomManager` holds a mutex-protected map of active rooms. When a WebSocket connects with `?room=ABC123`, `GetOrCreate` either returns the existing room or makes a new one.

Each `Room` has two fixed player slots. On `Join`:

1. Player 0 (first to join) receives `{ type: "waiting" }`
2. Player 1 (second to join) triggers `{ type: "game_start", player_id: "p1|p2" }` to both players, then immediately starts the game loop

Each player's `onMessage` callback routes all incoming messages through `room.HandleMessage`, which dispatches `click` to the game engine and handles `play_again`, `play_again_accept`, and `play_again_decline` directly in the room.

After game over, the room stays alive for 30 seconds to allow a rematch. If Player A sends `play_again`, the room forwards a `play_again_request` to Player B and starts a 30-second response timer. If Player B accepts, the room resets the game and sends `game_start` to both players. If Player B declines, or the timer expires, both players receive `play_again_declined` and the room is removed.

### Game Engine (`internal/game/game.go`)

The game runs a state machine:

```
StateWaiting → StatePlaying → StateRoundEnd → StateGameOver
                    ↑___________↓ (loop until winner)
```

A ticker fires every `CARD_INTERVAL_MS` (default 3000ms) to reveal the next card. On each tick, `revealNext` broadcasts a `card_reveal` message to both players. The deck is a standard 52-card shuffle — when exhausted, the game ends in a draw.

### Click Resolution (`internal/game/click.go`)

This is where correctness matters most. Two players may click within milliseconds of each other. The resolution strategy:

1. `HandleClick` acquires `game.mu` to read the current card and game state atomically.
2. If the card is **not an Ace**: the clicking player loses immediately (`false_start`). No lock needed — this is unambiguous.
3. If the card **is an Ace**: acquire `clickMu` and check `roundResolved`. The first click to acquire `clickMu` wins; subsequent clicks in the same round are ignored.

**Why two separate mutexes?**  
`revealNext` holds `game.mu` while updating card state. If `HandleClick` tried to acquire `game.mu` then `clickMu` in sequence, and `revealNext` also needed both, you'd have a deadlock risk. Keeping `clickMu` scoped only to the "first click wins" check sidesteps this entirely.

After resolution, a 2500ms delay fires `newRound` (to allow the overlay to display), which rebuilds the deck, resets `roundResolved`, and broadcasts `round_start`.

---

## Frontend Deep Dive

### State Flow

```
LobbyPage
  → generates/validates room code
  → navigates to /game/:roomId

GamePage
  → opens WebSocket to ws://backend/ws?room={roomId}
  → routes on phase: "waiting" | "playing" | "gameover"
  → passes lastMessage + sendMessage down as props

GameScreen (phase = "playing")
  → card_reveal      → set currentCard, enable ClickArea
  → click            → sendMessage({ type: "click" })
  → round_result     → update scores, show RoundResult overlay
  → round_start      → clear overlay, clear card
  → game_over        → call onGameOver → GamePage routes to GameOver

GameOver (phase = "gameover")
  → idle             → Play Again / Home buttons
  → Play Again click → sendMessage({ type: "play_again" }) → state = "pending" (30s countdown)
  → play_again_request received → state = "requested" (30s countdown, Accept / Decline)
  → Accept click     → sendMessage({ type: "play_again_accept" }) → awaits game_start
  → Decline click    → sendMessage({ type: "play_again_decline" }) → navigate to /
  → play_again_declined received → navigate to /
  → game_start received → phase = "playing" (rematch starts)
```

### `useWebSocket` Hook (`src/hooks/useWebSocket.ts`)

Wraps the WebSocket lifecycle. Exposes `lastMessage`, `status`, and `sendMessage`. `lastMessage` is set on each incoming frame; components react via `useEffect([lastMessage])`. The hook closes the connection on unmount and resets `lastMessage` on reconnect.

### Component Tree

```
GamePage
├── WaitingRoom       (phase = "waiting") — shareable link, spinner
└── GameScreen        (phase = "playing")
    ├── Scoreboard    — live scores + round number
    ├── FlipCard      — 3D CSS flip animation, swaps card at peak (150ms)
    │   └── Card      — rank/suit display, ace-glow animation
    ├── ClickArea     — large button, pulses when active
    └── RoundResult   — overlay: win/loss/false-start with scores
```

`GameOver` is rendered by `GamePage` when phase is "gameover" — final scores, emoji, and a play-again flow with three states: `idle` (Play Again / Home), `pending` (waiting for opponent with 30s countdown), and `requested` (opponent wants a rematch — Accept / Decline with 30s countdown).

### Message Types (`src/types/messages.ts`)

All WebSocket messages are typed as discriminated unions. This means the TypeScript compiler enforces exhaustive handling — you can't accidentally miss a message type.

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
{ "type": "player_disconnected" }
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
| `FRONTEND_ORIGIN` | `http://localhost:5173` | Allowed CORS origin |
| `CARD_INTERVAL_MS` | `3000` | Milliseconds between card reveals |
| `ROUNDS_TO_WIN` | `3` | Rounds needed to win the game |

**Frontend**

| Variable | Description |
|----------|-------------|
| `VITE_WS_URL` | WebSocket URL of the backend (e.g. `wss://your-app.onrender.com`) |
