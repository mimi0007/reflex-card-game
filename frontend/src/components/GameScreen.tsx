import { useEffect, useState } from "react";
import type { ClientMessage, Scores, ServerMessage } from "../types/messages";
import type { CardData } from "./Card";
import FlipCard from "./FlipCard";
import Scoreboard from "./Scoreboard";
import ClickArea from "./ClickArea";
import RoundResult from "./RoundResult";

interface Props {
  playerId: "p1" | "p2";
  lastMessage: ServerMessage | null;
  sendMessage: (msg: ClientMessage) => void;
  status: "connecting" | "open" | "closed" | "error";
  onGameOver: (winner: string, scores: Scores, reason?: string) => void;
}

export default function GameScreen({ playerId, lastMessage, sendMessage, status, onGameOver }: Props) {
  const [currentCard, setCurrentCard] = useState<CardData | null>(null);
  const [scores, setScores] = useState<Scores>({ p1: 0, p2: 0 });
  const [round, setRound] = useState(1);
  const [clickEnabled, setClickEnabled] = useState(false);
  const [roundResult, setRoundResult] = useState<{ winner: string; reason: string; scores: Scores } | null>(null);

  useEffect(() => {
    if (!lastMessage) return;

    switch (lastMessage.type) {
      case "card_reveal":
        setCurrentCard({ rank: lastMessage.rank, suit: lastMessage.suit });
        setClickEnabled(true);
        break;
      case "round_result":
        setScores(lastMessage.scores);
        setClickEnabled(false);
        setRoundResult({ winner: lastMessage.winner, reason: lastMessage.reason, scores: lastMessage.scores });
        break;
      case "round_start":
        setCurrentCard(null);
        setClickEnabled(false);
        setRound(lastMessage.round);
        setRoundResult(null);
        break;
      case "game_over":
        onGameOver(lastMessage.winner, lastMessage.scores, lastMessage.reason);
        break;
    }
  }, [lastMessage]);

  function handleClick() {
    if (!clickEnabled) return;
    setClickEnabled(false);
    sendMessage({ type: "click" });
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100%" }}>
      <Scoreboard scores={scores} round={round} playerId={playerId} />

      <div
        style={{
          flex: 1,
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          padding: "2rem",
          cursor: clickEnabled ? "pointer" : "default",
        }}
        onClick={handleClick}
      >
        <FlipCard card={currentCard} />
      </div>

      <div style={{ padding: "1rem" }}>
        <ClickArea enabled={clickEnabled} onClick={handleClick} />
      </div>

      {roundResult && (
        <RoundResult
          winner={roundResult.winner}
          reason={roundResult.reason}
          scores={roundResult.scores}
          playerId={playerId}
        />
      )}

      {status !== "open" && (
        <div style={{
          backgroundColor: "#fef3c7",
          color: "#92400e",
          padding: "0.5rem 1rem",
          textAlign: "center",
          fontSize: "0.875rem",
        }}>
          ⚠ Connection lost — return to lobby to rejoin
        </div>
      )}
    </div>
  );
}
