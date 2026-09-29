export type Tone = "good" | "warn" | "bad" | "info" | "neutral";

const groups: Record<Tone, string[]> = {
  good: ["healthy", "resolved", "succeeded", "ready", "ok", "allowed", "complete", "completed", "connected", "approved", "active"],
  warn: ["degraded", "investigating", "warning", "suspect", "running", "mitigating", "in-progress", "needs_attention"],
  bad: ["critical", "error", "sev-1", "sev-2", "failed", "disconnected"],
  info: ["sev-3", "info", "simulated", "normal", "recovering"],
  neutral: ["idle", "inactive", "aborted"],
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
