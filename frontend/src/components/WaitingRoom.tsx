import { useState } from "react";

interface Props {
  roomId: string;
}

export default function WaitingRoom({ roomId }: Props) {
  const [copied, setCopied] = useState(false);
  const shareUrl = `${window.location.origin}/game/${roomId}`;

  function handleCopy() {
    navigator.clipboard.writeText(shareUrl).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  }

  return (
    <div style={{ textAlign: "center", padding: "2rem" }}>
      <div style={{ fontSize: "2rem", marginBottom: "1rem" }}>⟳</div>
      <h2>Waiting for opponent...</h2>
      <p style={{ marginTop: "1.5rem" }}>Share this link:</p>
      <div style={{ display: "flex", gap: "0.5rem", justifyContent: "center", flexWrap: "wrap" }}>
        <input
          readOnly
          value={shareUrl}
          style={{ width: "18rem", padding: "0.4rem 0.6rem", borderRadius: 4 }}
        />
        <button onClick={handleCopy}>{copied ? "Copied!" : "Copy"}</button>
      </div>
      <p style={{ marginTop: "1rem", color: "#9ca3af" }}>Room code: {roomId}</p>
    </div>
  );
}
