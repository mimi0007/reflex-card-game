const CHARS = "ABCDEFGHJKMNPQRSTUVWXYZ23456789";

export function generateRoomCode(): string {
  return Array.from({ length: 5 }, () =>
    CHARS[Math.floor(Math.random() * CHARS.length)]
  ).join("");
}

export function sanitizeRoomCode(input: string): string {
  return input
    .toUpperCase()
    .split("")
    .filter((c) => CHARS.includes(c))
    .slice(0, 5)
    .join("");
}
