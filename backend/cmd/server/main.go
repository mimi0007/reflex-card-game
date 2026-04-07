package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

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

	hub := ws.NewHub()
	go hub.Run()

	rm := room.NewManager()

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/ws", wsHandler(hub, rm))

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
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

		// Register with Hub so lifecycle (ping/unregister) is tracked.
		hub.Register <- client

		go client.WritePump()
		go client.ReadPump()

		// Slot the client into the requested room (or a new one).
		ro := rm.GetOrCreate(roomID)
		idx, err := ro.Join(client)
		if err != nil {
			// Room is full — close the connection; ReadPump will unregister from Hub.
			log.Printf("room %s full, rejecting %s", ro.ID, client.ID)
			conn.Close()
			return
		}

		// Persist room/player info on the client for downstream use (B4/B5).
		client.RoomID = ro.ID
		client.PlayerID = playerLabel(idx)

		log.Printf("client %s joined room %s as %s", client.ID, ro.ID, client.PlayerID)
	}
}

func playerLabel(idx int) string {
	if idx == 0 {
		return "p1"
	}
	return "p2"
}
