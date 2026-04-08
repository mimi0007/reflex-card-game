import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { generateRoomCode, sanitizeRoomCode } from "../utils/roomCode";

export default function LobbyPage() {
  const navigate = useNavigate();
  const [code, setCode] = useState("");

  function handleCreate() {
    navigate(`/game/${generateRoomCode()}`);
  }

  function handleJoin() {
    if (code.length === 5) {
      navigate(`/game/${code}`);
    }
  }

  function handleCodeChange(e: React.ChangeEvent<HTMLInputElement>) {
    setCode(sanitizeRoomCode(e.target.value));
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === "Enter") handleJoin();
  }

  return (
    <div style={{ textAlign: "center", padding: "3rem 1rem" }}>
      <h1>Reflex Card Game</h1>
      <p style={{ color: "#9ca3af", marginBottom: "2rem" }}>Two players. One Ace.</p>

      <button onClick={handleCreate} style={{ fontSize: "1rem", padding: "0.6rem 1.4rem" }}>
        Create Game
      </button>

      <p style={{ margin: "1.5rem 0", color: "#6b7280" }}>— or —</p>

      <div style={{ display: "flex", gap: "0.5rem", justifyContent: "center" }}>
        <input
          value={code}
          onChange={handleCodeChange}
          onKeyDown={handleKeyDown}
          placeholder="Enter room code"
          maxLength={5}
          style={{ padding: "0.5rem 0.75rem", fontSize: "1rem", borderRadius: 4, width: "12rem", textTransform: "uppercase" }}
        />
        <button
          onClick={handleJoin}
          disabled={code.length !== 5}
          style={{ padding: "0.5rem 1rem", fontSize: "1rem" }}
        >
          Join Game
        </button>
      </div>
    </div>
  );
}
