package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mimi0007/reflex-card-game/backend/internal/room"
	"github.com/mimi0007/reflex-card-game/backend/internal/ws"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	interval := cardInterval()

	hub := ws.NewHub()
	go hub.Run()

	rm := room.NewManager(interval)

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/ws", wsHandler(hub, rm))

	log.Printf("Server starting on port %s (card interval: %v)", port, interval)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func cardInterval() time.Duration {
	if v := os.Getenv("CARD_INTERVAL_MS"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms > 0 {
			return time.Duration(ms) * time.Millisecond
		}
	}
	return 3 * time.Second
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func wsHandler(hub *ws.Hub, rm *room.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roomID := r.URL.Query().Get("room")

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("upgrade error: %v", err)
			return
		}

		clientID := fmt.Sprintf("%s-%s", roomID, r.RemoteAddr)
		client := ws.NewClient(clientID, hub, conn)

		hub.Register <- client
		go client.WritePump()

		ro := rm.GetOrCreate(roomID)
		idx, err := ro.Join(client)
		if err != nil {
			log.Printf("room %s full, rejecting %s", ro.ID, client.ID)
			conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "room full"))
			conn.Close()
			hub.Unregister <- client
			return
		}

		client.RoomID = ro.ID
		client.PlayerID = playerLabel(idx)

		log.Printf("client %s joined room %s as %s", client.ID, ro.ID, client.PlayerID)

		client.ReadPump()

		wasRunning := ro.StopGame()
		if wasRunning {
			ro.BroadcastExcept(client, map[string]any{
				"type":      "player_disconnected",
				"player_id": client.PlayerID,
				"message":   client.PlayerID + " disconnected",
			})
		}

		rm.Remove(ro.ID)
		log.Printf("client %s disconnected from room %s, room removed", client.ID, ro.ID)
	}
}

func playerLabel(idx int) string {
	if idx == 0 {
		return "p1"
	}
	return "p2"
}
