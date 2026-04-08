import "../styles/card.css";

const SUIT_SYMBOLS: Record<string, string> = {
  spades: "♠",
  hearts: "♥",
  diamonds: "♦",
  clubs: "♣",
};

const RED_SUITS = new Set(["hearts", "diamonds"]);

export interface CardData {
  rank: string;
  suit: string;
}

interface Props {
  card: CardData | null;
}

export default function Card({ card }: Props) {
  if (!card) {
    return <div className="card card--back" />;
  }

  const { rank, suit } = card;
  const isAce = rank === "A";
  const isRed = RED_SUITS.has(suit);
  const symbol = SUIT_SYMBOLS[suit] ?? suit;

  const colorClass = isRed ? "card--red" : "card--black";
  const aceClass = isAce ? "card--ace" : "";

  return (
    <div className={`card ${colorClass} ${aceClass}`}>
      <span className="card__corner">{rank}</span>
      <span className="card__suit">{symbol}</span>
      <span className="card__corner card__corner--bottom">{rank}</span>
    </div>
  );
}
