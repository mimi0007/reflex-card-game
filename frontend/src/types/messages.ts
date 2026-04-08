export type Scores = {
  p1: number;
  p2: number;
};

export type ServerMessage =
  | { type: "waiting" }
  | { type: "game_start"; player_id: "p1" | "p2" }
  | { type: "card_reveal"; card: string; rank: string; suit: string }
  | { type: "round_result"; winner: string; reason: string; scores: Scores }
  | { type: "round_start"; round: number }
  | { type: "game_over"; winner: string; scores: Scores }
  | { type: "player_disconnected" };

export type ClientMessage = { type: "click" };
