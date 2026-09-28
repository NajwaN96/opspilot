export function formatPercent(value: number, digits = 2): string {
  const rendered = value.toFixed(digits).replace(/(\.\d*?)0+$/, "$1").replace(/\.$/, "");
  return `${rendered}%`;
}

export function formatLatency(ms: number): string {
  if (!Number.isFinite(ms)) return "—";
  if (ms >= 1000) {
    const seconds = ms / 1000;
    const digits = seconds >= 10 ? 0 : 1;
    return `${seconds.toFixed(digits)}s`;
  }
  return `${Math.round(ms)}ms`;
}

export function formatDuration(ms: number): string {
  const sec = Math.max(0, Math.round(ms / 1000));
  if (sec < 60) return `${sec}s`;
  const minutes = Math.floor(sec / 60);
  const seconds = sec % 60;
  if (minutes < 60) {
    return seconds ? `${minutes}m ${seconds}s` : `${minutes}m`;
  }
  const hours = Math.floor(minutes / 60);
  const remain = minutes % 60;
  return remain ? `${hours}h ${remain}m` : `${hours}h`;
}

export function formatAgo(iso: string, now = Date.now()): string {
  const then = new Date(iso).getTime();
  if (Number.isNaN(then)) return "—";
  return formatDuration(Math.max(0, now - then)) + " ago";
}

export function formatClock(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "—";
  return date.toISOString().slice(11, 16);
}

export function completedSteps(steps: { status: string }[] | undefined): number {
  if (!steps) return 0;
  return steps.filter((step) => step.status === "complete").length;
}
