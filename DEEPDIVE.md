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
  → opens WebSocket to wss://backend/ws?room={roomId}
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