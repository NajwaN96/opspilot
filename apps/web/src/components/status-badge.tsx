import { cn } from "@/lib/utils";
import { statusLabel, statusTone, type Tone } from "@/lib/status";

const tones: Record<Tone, string> = {
  good: "border-emerald-400/30 bg-emerald-400/10 text-emerald-300",
  warn: "border-amber-300/30 bg-amber-300/10 text-amber-200",
  bad: "border-red-400/35 bg-red-400/10 text-red-300",
  info: "border-sky-400/30 bg-sky-400/10 text-sky-200",
  neutral: "border-border bg-muted text-muted-foreground",
};

const dots: Record<Tone, string> = {
  good: "bg-emerald-400",
  warn: "bg-amber-300",
  bad: "bg-red-400",
  info: "bg-sky-300",
  neutral: "bg-muted-foreground",
};

export function StatusBadge({ value, className }: { value: string; className?: string }) {
  const tone = statusTone(value);
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 border px-1.5 py-0.5 text-[11px] font-medium tracking-wide",
        tones[tone],
        className,
      )}
    >
      <span className={cn("size-1.5 rounded-full", dots[tone])} aria-hidden />
      {statusLabel(value)}
    </span>
  );
}
