import type { ReactNode } from "react";
import { Button } from "@/components/ui/button";

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
    <div className="mb-3 flex flex-wrap items-end justify-between gap-3">
      <div className="min-w-0">
        {kicker ? (
          <div className="font-mono text-[11px] uppercase tracking-wide text-muted-foreground">{kicker}</div>
        ) : null}
        <h1 className="text-lg font-semibold tracking-tight">{title}</h1>
        {description ? <p className="mt-1 max-w-3xl text-sm text-muted-foreground">{description}</p> : null}
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
  className,
}: {
  title: string;
  action?: ReactNode;
  children: ReactNode;
  padded?: boolean;
  className?: string;
}) {
  return (
    <section className={`border border-border bg-card ${className ?? ""}`}>
      <header className="flex items-center justify-between gap-2 border-b border-border px-3 py-2">
        <h2 className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">{title}</h2>
        {action}
      </header>
      <div className={padded ? "p-3" : undefined}>{children}</div>
    </section>
  );
}

export function LoadingBlock({ label = "Loading operational data" }: { label?: string }) {
  return (
    <div className="space-y-2" role="status" aria-live="polite">
      <span className="sr-only">{label}</span>
      <div className="h-8 animate-pulse bg-muted" />
      <div className="h-24 animate-pulse bg-muted" />
      <div className="h-40 animate-pulse bg-muted" />
    </div>
  );
}

export function ErrorBlock({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <div className="border border-red-400/40 bg-red-400/10 px-3 py-3 text-sm" role="alert">
      <p className="font-medium text-red-200">The control plane could not load this view.</p>
      <p className="mt-1 text-red-100/80">{message}</p>
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
    <div className="px-3 py-6 text-sm">
      <p className="font-medium">{title}</p>
      <p className="mt-1 text-muted-foreground">{detail}</p>
    </div>
  );
}
