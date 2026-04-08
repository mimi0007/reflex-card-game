import type { Scores } from "../types/messages";

interface Props {
  scores: Scores;
  round: number;
  playerId: "p1" | "p2";
}

export default function Scoreboard({ scores, round, playerId }: Props) {
  return (
    <div style={{
      display: "flex",
      justifyContent: "space-around",
      alignItems: "center",
      padding: "0.75rem 1rem",
      borderBottom: "1px solid #e5e7eb",
      fontWeight: 600,
    }}>
      <span style={{ color: playerId === "p1" ? "#2563eb" : "#6b7280" }}>
        {playerId === "p1" ? "You" : "P1"}: {scores.p1}
      </span>
      <span style={{ color: "#6b7280", fontSize: "0.85rem" }}>Round {round}</span>
      <span style={{ color: playerId === "p2" ? "#2563eb" : "#6b7280" }}>
        {playerId === "p2" ? "You" : "P2"}: {scores.p2}
      </span>
    </div>
  );
}
