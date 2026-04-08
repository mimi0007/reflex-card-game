import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { useWebSocket } from "../hooks/useWebSocket";
import WaitingRoom from "../components/WaitingRoom";
import ConnectionStatus from "../components/ConnectionStatus";

type Phase = "waiting" | "playing";

export default function GamePage() {
  const { roomId } = useParams<{ roomId: string }>();
  const wsUrl = roomId
    ? `${import.meta.env.VITE_WS_URL ?? "ws://localhost:8080"}/ws?room=${roomId}`
    : null;

  const { lastMessage, status } = useWebSocket(wsUrl);
  const [phase, setPhase] = useState<Phase>("waiting");
  const [playerId, setPlayerId] = useState<"p1" | "p2" | null>(null);

  useEffect(() => {
    if (!lastMessage) return;
    if (lastMessage.type === "game_start") {
      setPlayerId(lastMessage.player_id);
      setPhase("playing");
    }
  }, [lastMessage]);

  return (
    <div style={{ padding: "1rem" }}>
      <div style={{ display: "flex", justifyContent: "flex-end", marginBottom: "1rem" }}>
        <ConnectionStatus status={status} />
      </div>

      {phase === "waiting" && roomId && <WaitingRoom roomId={roomId} />}
      {phase === "playing" && (
        <div>Game screen — coming in F5 (you are {playerId})</div>
      )}
    </div>
  );
}
