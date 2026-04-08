interface Props {
  enabled: boolean;
  onClick: () => void;
}

export default function ClickArea({ enabled, onClick }: Props) {
  return (
    <button
      onClick={onClick}
      disabled={!enabled}
      style={{
        width: "100%",
        padding: "1.5rem",
        fontSize: "1.5rem",
        fontWeight: 700,
        letterSpacing: "0.1em",
        border: "none",
        borderRadius: 8,
        cursor: enabled ? "pointer" : "not-allowed",
        backgroundColor: enabled ? "#dc2626" : "#9ca3af",
        color: "#fff",
        animation: enabled ? "pulse 1s ease-in-out infinite alternate" : "none",
        transition: "background-color 0.2s",
      }}
    >
      CLICK!
    </button>
  );
}
