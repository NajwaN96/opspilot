import type { ReactNode } from "react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

export function PageHeader({
  kicker,
  title,
  description,
  actions,
}: {
  kicker?: string;
  title: string;
  description?: string;
  actions?: ReactNode;
}) {
  return (
    <div className="mb-5 flex flex-wrap items-end justify-between gap-4">
      <div className="min-w-0">
        {kicker ? (
          <div className="text-[11px] font-semibold uppercase tracking-[0.08em] text-muted-foreground">{kicker}</div>
        ) : null}
        <h1 className={cn("text-[26px] font-semibold tracking-[-0.02em] text-foreground md:text-[30px]", kicker && "mt-1")}>{title}</h1>
        {description ? <p className="mt-1.5 max-w-3xl text-[14px] leading-relaxed text-muted-foreground">{description}</p> : null}
      </div>
      {actions}
    </div>
  );
}

export function Panel({
  title,
  action,
  children,
  padded = true,
  prominence = "default",
  className,
}: {
  title: string;
  action?: ReactNode;
  children: ReactNode;
  padded?: boolean;
  prominence?: "default" | "primary";
  className?: string;
}) {
  return (
    <section className={cn("surface-card overflow-hidden rounded-xl", className)}>
      <header className="flex items-center justify-between gap-3 border-b border-border px-4 py-2.5">
        <h2
          className={
            prominence === "primary"
              ? "text-[16px] font-semibold tracking-tight text-foreground"
              : "text-[11px] font-semibold uppercase tracking-[0.06em] text-muted-foreground"
          }
        >
          {title}
        </h2>
        {action}
      </header>
      <div className={padded ? "p-4" : undefined}>{children}</div>
    </section>
  );
}

export function LoadingBlock({ label = "Loading operational data" }: { label?: string }) {
  return (
    <div className="space-y-3" role="status" aria-live="polite">
      <span className="sr-only">{label}</span>
      <div className="h-10 animate-pulse rounded-lg bg-white/80" />
      <div className="h-28 animate-pulse rounded-xl bg-white/80" />
      <div className="h-48 animate-pulse rounded-xl bg-white/80" />
    </div>
  );
}

export function ErrorBlock({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <div className="rounded-xl border border-status-danger/20 bg-status-danger-soft px-4 py-3 text-sm" role="alert">
      <p className="font-medium text-status-danger">The control plane could not load this view.</p>
      <p className="mt-1 text-status-danger/80">{message}</p>
      {onRetry ? (
        <Button className="mt-3" variant="outline" size="sm" onClick={onRetry}>
          Retry
        </Button>
      ) : null}
    </div>
  );
}

export function EmptyBlock({ title, detail }: { title: string; detail: string }) {
  return (
    <div className="px-4 py-8 text-sm">
      <p className="font-medium">{title}</p>
      <p className="mt-1 text-muted-foreground">{detail}</p>
    </div>
  );
}
