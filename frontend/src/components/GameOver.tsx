import { useState } from "react";
import { useNavigate } from "react-router-dom";
import type { Scores } from "../types/messages";
import { generateRoomCode } from "../utils/roomCode";
import "../styles/game-over.css";

interface Props {
  winner: string;
  scores: Scores;
  reason?: string;
  playerId: "p1" | "p2";
}

function getSubtitle(reason: string | undefined, isDraw: boolean): string {
  if (isDraw) return "Nobody reached 3 rounds. A draw!";
  if (reason === "deck_exhausted") return "The deck ran out of cards.";
  if (reason === "player_disconnected") return "Your opponent disconnected.";
  return "First to 3 rounds wins.";
}

export default function GameOver({ winner, scores, reason, playerId }: Props) {
  const navigate = useNavigate();
  const [copied, setCopied] = useState(false);

  const isDraw = scores.p1 === scores.p2;
  const isWinner = !isDraw && winner === playerId;

  const emoji = isDraw ? "🤝" : isWinner ? "🏆" : "😔";
  const title = isDraw ? "It's a draw!" : isWinner ? "You win!" : "You lose!";
  const subtitle = getSubtitle(reason, isDraw);

  function handlePlayAgain() {
    navigate(`/game/${generateRoomCode()}`);
  }

  function handleShare() {
    const code = generateRoomCode();
    const url = `${window.location.origin}/game/${code}`;
    navigator.clipboard.writeText(url).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  }

  const youScore = playerId === "p1" ? scores.p1 : scores.p2;
  const themScore = playerId === "p1" ? scores.p2 : scores.p1;
  const youLabel = "You";
  const themLabel = playerId === "p1" ? "P2" : "P1";

  return (
    <div className="game-over">
      <div className="game-over__emoji">{emoji}</div>
      <h2 className="game-over__title">{title}</h2>
      <p className="game-over__subtitle">{subtitle}</p>

      <div className="game-over__scores">
        <span className={isWinner || isDraw ? "game-over__scores__you" : "game-over__scores__them"}>
          {youLabel}: {youScore}
        </span>
        <span style={{ color: "#9ca3af", fontWeight: 400 }}>VS</span>
        <span className={!isWinner && !isDraw ? "game-over__scores__you" : "game-over__scores__them"}>
          {themLabel}: {themScore}
        </span>
      </div>

      <div className="game-over__actions">
        <button className="game-over__btn game-over__btn--primary" onClick={handlePlayAgain}>
          Play Again
        </button>
        <button className="game-over__btn game-over__btn--secondary" onClick={handleShare}>
          {copied ? "Link copied!" : "Share New Game"}
        </button>
      </div>
    </div>
  );
}
