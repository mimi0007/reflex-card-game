type Status = "connecting" | "open" | "closed" | "error";

const config: Record<Status, { color: string; label: string }> = {
  connecting: { color: "#f59e0b", label: "Connecting..." },
  open:       { color: "#22c55e", label: "Connected" },
  closed:     { color: "#9ca3af", label: "Disconnected" },
  error:      { color: "#ef4444", label: "Connection error" },
};

export default function ConnectionStatus({ status }: { status: Status }) {
  const { color, label } = config[status];
  return (
    <span style={{ display: "inline-flex", alignItems: "center", gap: 6 }}>
      <span
        style={{
          width: 10,
          height: 10,
          borderRadius: "50%",
          backgroundColor: color,
          display: "inline-block",
        }}
      />
      {label}
    </span>
  );
}
