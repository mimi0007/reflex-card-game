import type { Scores } from "../types/messages";
import "../styles/results.css";

type Outcome = "win" | "lose" | "false_start_lose" | "false_start_win";

const OUTCOME_CONTENT: Record<Outcome, { emoji: string; title: string; subtitle: string }> = {
  win:              { emoji: "⚡", title: "You got it!",           subtitle: "You clicked the Ace first" },
  lose:             { emoji: "😅", title: "Too slow!",             subtitle: "Opponent was faster" },
  false_start_lose: { emoji: "❌", title: "False start!",          subtitle: "You clicked before the Ace appeared" },
  false_start_win:  { emoji: "🎯", title: "Opponent jumped early!", subtitle: "They clicked before the Ace" },
};

interface Props {
  winner: string;
  reason: string;
  scores: Scores;
  playerId: "p1" | "p2";
}

function getOutcome(winner: string, reason: string, playerId: string): Outcome {
  if (reason === "false_start") {
    return winner === playerId ? "false_start_win" : "false_start_lose";
  }
  return winner === playerId ? "win" : "lose";
}

export default function RoundResult({ winner, reason, scores, playerId }: Props) {
  const outcome = getOutcome(winner, reason, playerId);
  const { emoji, title, subtitle } = OUTCOME_CONTENT[outcome];
  const isWin = outcome === "win" || outcome === "false_start_win";

  const p1IsWinner = winner === "p1";

  return (
    <div className={`round-result-overlay ${isWin ? "round-result-overlay--win" : "round-result-overlay--lose"}`}>
      <div className="round-result-card">
        <div className="round-result-emoji">{emoji}</div>
        <p className="round-result-title">{title}</p>
        <p className="round-result-subtitle">{subtitle}</p>
        <div className="round-result-scores">
          <span className={p1IsWinner ? "round-result-scores__winner" : "round-result-scores__loser"}>
            {playerId === "p1" ? "You" : "Player1"}: {scores.p1}
          </span>
          <span className={!p1IsWinner ? "round-result-scores__winner" : "round-result-scores__loser"}>
            {playerId === "p2" ? "You" : "Player2"}: {scores.p2}
          </span>
        </div>
        <p className="round-result-hint">Next round...</p>
      </div>
    </div>
  );
}
