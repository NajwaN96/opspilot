import { cn } from "@/lib/utils";
import { statusLabel, statusTone, type Tone } from "@/lib/status";

const tones: Record<Tone, string> = {
  good: "border-status-success/20 bg-status-success-soft text-status-success",
  warn: "border-status-warning/25 bg-status-warning-soft text-status-warning",
  bad: "border-status-danger/20 bg-status-danger-soft text-status-danger",
  info: "border-status-info/20 bg-status-info-soft text-status-info",
  neutral: "border-border bg-status-neutral-soft text-status-neutral",
  sim: "border-status-sim/20 bg-status-sim-soft text-status-sim",
};

const dots: Record<Tone, string> = {
  good: "bg-[var(--status-dot-success)]",
  warn: "bg-[var(--status-dot-warning)]",
  bad: "bg-[var(--status-dot-danger)]",
  info: "bg-[var(--status-dot-info)]",
  neutral: "bg-[var(--status-dot-neutral)]",
  sim: "bg-[var(--status-dot-sim)]",
};

export function StatusBadge({ value, className }: { value: string; className?: string }) {
  const tone = statusTone(value);
  return (
    <span
      className={cn(
        "inline-flex h-6 items-center gap-1.5 rounded-md border px-2 text-[12px] font-medium leading-none transition-colors duration-150",
        tones[tone],
        className,
      )}
    >
      <span className={cn("size-1.5 shrink-0 rounded-full", dots[tone])} aria-hidden />
      {statusLabel(value)}
    </span>
  );
}
