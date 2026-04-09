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
  | { type: "game_over"; winner: string; scores: Scores; reason?: string }
  | { type: "player_disconnected" }
  | { type: "play_again_request" }
  | { type: "play_again_declined" };

export type ClientMessage =
  | { type: "click" }
  | { type: "play_again" }
  | { type: "play_again_accept" }
  | { type: "play_again_decline" };
