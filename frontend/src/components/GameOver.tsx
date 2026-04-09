import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import type { Scores } from "../types/messages";
import type { PlayAgainState } from "../pages/GamePage";
import "../styles/game-over.css";

interface Props {
  winner: string;
  scores: Scores;
  reason?: string;
  playerId: "p1" | "p2";
  playAgainState: PlayAgainState;
  onPlayAgain: () => void;
  onPlayAgainAccept: () => void;
  onPlayAgainDecline: () => void;
}

function getSubtitle(reason: string | undefined, isDraw: boolean): string {
  if (isDraw) return "Nobody reached 3 rounds. A draw!";
  if (reason === "deck_exhausted") return "The deck ran out of cards.";
  if (reason === "player_disconnected") return "Your opponent disconnected.";
  return "First to 3 rounds wins.";
}

export default function GameOver({ winner, scores, reason, playerId, playAgainState, onPlayAgain, onPlayAgainAccept, onPlayAgainDecline }: Props) {
  const navigate = useNavigate();
  const [countdown, setCountdown] = useState(30);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    if (playAgainState === "pending") {
      setCountdown(30);
      intervalRef.current = setInterval(() => {
        setCountdown(prev => {
          if (prev <= 1) {
            clearInterval(intervalRef.current!);
            return 0;
          }
          return prev - 1;
        });
      }, 1000);
    } else {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
      setCountdown(30);
    }
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, [playAgainState]);

  const isDraw = scores.p1 === scores.p2;
  const isWinner = !isDraw && winner === playerId;

  const emoji = isDraw ? "🤝" : isWinner ? "🏆" : "😔";
  const title = isDraw ? "It's a draw!" : isWinner ? "You win!" : "You lose!";
  const subtitle = getSubtitle(reason, isDraw);

  const youScore = playerId === "p1" ? scores.p1 : scores.p2;
  const themScore = playerId === "p1" ? scores.p2 : scores.p1;
  const youLabel = "You";
  const themLabel = playerId === "p1" ? "Player 2" : "Player 1";

  return (
    <div className="game-over">
      <div className="game-over__emoji">{emoji}</div>
      <h2 className={`game-over__title${isWinner ? " game-over__title--win" : !isDraw ? " game-over__title--lose" : ""}`}>{title}</h2>
      <p className="game-over__subtitle">{subtitle}</p>

      <div className="game-over__scores">
        <span className="game-over__scores__label">
          {youLabel}: <span className={isWinner || isDraw ? "game-over__scores__you" : "game-over__scores__them"}>{youScore}</span>
        </span>
        <span style={{ color: "#9ca3af", fontWeight: 400 }}>VS</span>
        <span className="game-over__scores__label">
          {themLabel}: <span className={!isWinner && !isDraw ? "game-over__scores__you" : "game-over__scores__them"}>{themScore}</span>
        </span>
      </div>

      <div className="game-over__actions">
        {playAgainState === "idle" && (
          <>
            <button className="game-over__btn game-over__btn--primary" onClick={onPlayAgain}>
              Play Again
            </button>
            <button className="game-over__btn game-over__btn--secondary" onClick={() => navigate("/")}>
              Home
            </button>
          </>
        )}

        {playAgainState === "pending" && (
          <p className="game-over__waiting">Waiting for opponent... {countdown}s</p>
        )}

        {playAgainState === "requested" && (
          <>
            <p className="game-over__waiting">Opponent wants a rematch!</p>
            <button className="game-over__btn game-over__btn--primary" onClick={onPlayAgainAccept}>
              Accept
            </button>
            <button className="game-over__btn game-over__btn--secondary" onClick={onPlayAgainDecline}>
              Decline
            </button>
          </>
        )}
      </div>
    </div>
  );
}
