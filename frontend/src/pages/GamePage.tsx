import { useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useWebSocket } from "../hooks/useWebSocket";
import WaitingRoom from "../components/WaitingRoom";
import ConnectionStatus from "../components/ConnectionStatus";
import GameScreen from "../components/GameScreen";
import GameOver from "../components/GameOver";
import type { Scores } from "../types/messages";

type Phase = "waiting" | "playing" | "gameover";
export type PlayAgainState = "idle" | "pending" | "requested";

export default function GamePage() {
  const { roomId } = useParams<{ roomId: string }>();
  const navigate = useNavigate();
  const wsUrl = roomId
    ? `${import.meta.env.VITE_WS_URL ?? "ws://localhost:8080"}/ws?room=${roomId}`
    : null;

  const { lastMessage, sendMessage, status } = useWebSocket(wsUrl);
  const [phase, setPhase] = useState<Phase>("waiting");
  const [playerId, setPlayerId] = useState<"p1" | "p2" | null>(null);
  const [gameOverData, setGameOverData] = useState<{ winner: string; scores: Scores; reason?: string } | null>(null);
  const [playAgainState, setPlayAgainState] = useState<PlayAgainState>("idle");

  useEffect(() => {
    setPhase("waiting");
    setPlayerId(null);
    setGameOverData(null);
    setPlayAgainState("idle");
  }, [roomId]);

  useEffect(() => {
    if (!lastMessage) return;
    if (lastMessage.type === "game_start") {
      setPlayerId(lastMessage.player_id);
      setPhase("playing");
      setPlayAgainState("idle");
    }
    if (lastMessage.type === "play_again_request") {
      setPlayAgainState("requested");
    }
    if (lastMessage.type === "play_again_declined") {
      navigate("/");
    }
  }, [lastMessage, navigate]);

  function handleGameOver(winner: string, scores: Scores, reason?: string) {
    setGameOverData({ winner, scores, reason });
    setPhase("gameover");
  }

  function handlePlayAgain() {
    sendMessage({ type: "play_again" });
    setPlayAgainState("pending");
  }

  function handlePlayAgainAccept() {
    sendMessage({ type: "play_again_accept" });
  }

  function handlePlayAgainDecline() {
    sendMessage({ type: "play_again_decline" });
    navigate("/");
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

      {phase === "gameover" && gameOverData && playerId && (
        <GameOver
          winner={gameOverData.winner}
          scores={gameOverData.scores}
          reason={gameOverData.reason}
          playerId={playerId}
          playAgainState={playAgainState}
          onPlayAgain={handlePlayAgain}
          onPlayAgainAccept={handlePlayAgainAccept}
          onPlayAgainDecline={handlePlayAgainDecline}
        />
      )}
    </div>
  );
}
