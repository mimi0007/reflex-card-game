import { useParams } from "react-router-dom";

export default function GamePage() {
  const { roomId } = useParams<{ roomId: string }>();
  return <div>Game room: {roomId} — coming in F5</div>;
}
