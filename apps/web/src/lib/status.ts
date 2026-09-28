export type Tone = "good" | "warn" | "bad" | "info" | "neutral";

const groups: Record<Tone, string[]> = {
  good: ["healthy", "resolved", "succeeded", "ready", "ok", "allowed", "complete", "completed"],
  warn: ["degraded", "investigating", "warning", "suspect", "running", "mitigating", "in-progress"],
  bad: ["critical", "error", "sev-1", "sev-2", "failed"],
  info: ["sev-3", "info", "simulated", "normal"],
  neutral: [],
};

export function statusTone(value: string): Tone {
  const key = value.trim().toLowerCase();
  for (const tone of ["good", "warn", "bad", "info"] as const) {
    if (groups[tone].includes(key)) return tone;
  }
  return "neutral";
}

export function statusLabel(value: string): string {
  if (value.toUpperCase().startsWith("SEV-")) return value.toUpperCase();
  if (!value) return "Unknown";
  return value
    .split(/[-_]/)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}
