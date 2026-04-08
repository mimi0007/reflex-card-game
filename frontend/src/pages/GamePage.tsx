import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { useWebSocket } from "../hooks/useWebSocket";
import WaitingRoom from "../components/WaitingRoom";
import ConnectionStatus from "../components/ConnectionStatus";
import GameScreen from "../components/GameScreen";
import type { Scores } from "../types/messages";

type Phase = "waiting" | "playing" | "gameover";

export default function GamePage() {
  const { roomId } = useParams<{ roomId: string }>();
  const wsUrl = roomId
    ? `${import.meta.env.VITE_WS_URL ?? "ws://localhost:8080"}/ws?room=${roomId}`
    : null;

  const { lastMessage, sendMessage, status } = useWebSocket(wsUrl);
  const [phase, setPhase] = useState<Phase>("waiting");
  const [playerId, setPlayerId] = useState<"p1" | "p2" | null>(null);
  const [gameOverData, setGameOverData] = useState<{ winner: string; scores: Scores } | null>(null);

  useEffect(() => {
    if (!lastMessage) return;
    if (lastMessage.type === "game_start") {
      setPlayerId(lastMessage.player_id);
      setPhase("playing");
    }
  }, [lastMessage]);

  function handleGameOver(winner: string, scores: Scores) {
    setGameOverData({ winner, scores });
    setPhase("gameover");
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", minHeight: "100svh" }}>
      <div style={{ display: "flex", justifyContent: "flex-end", padding: "0.75rem 1rem", borderBottom: "1px solid #e5e7eb" }}>
        <ConnectionStatus status={status} />
      </div>

      {phase === "waiting" && roomId && <WaitingRoom roomId={roomId} />}

      {phase === "playing" && playerId && (
        <GameScreen
          playerId={playerId}
          lastMessage={lastMessage}
          sendMessage={sendMessage}
          status={status}
          onGameOver={handleGameOver}
        />
      )}

      {phase === "gameover" && gameOverData && (
        <div style={{ textAlign: "center", padding: "3rem 1rem" }}>
          <h2>Game over — coming in F7</h2>
          <p>Winner: {gameOverData.winner}</p>
        </div>
      )}
    </div>
  );
}
