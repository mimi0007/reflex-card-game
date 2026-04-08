import { useEffect, useRef, useState } from "react";
import Card, { type CardData } from "./Card";
import "../styles/card.css";

interface Props {
  card: CardData | null;
}

export default function FlipCard({ card }: Props) {
  const [displayed, setDisplayed] = useState<CardData | null>(card);
  const [flipping, setFlipping] = useState(false);
  const pendingCard = useRef<CardData | null>(card);

  useEffect(() => {
    if (card === displayed) return;

    pendingCard.current = card;
    setFlipping(true);

    const swapTimer = setTimeout(() => {
      setDisplayed(pendingCard.current);
      setFlipping(false);
    }, 150);

    return () => clearTimeout(swapTimer);
  }, [card]);

  return (
    <div className="flip-card-wrapper">
      <div className={`flip-card-inner ${flipping ? "flip-card-inner--flipping" : ""}`}>
        <Card card={displayed} />
      </div>
    </div>
  );
}
