"use client";

import type { ReactNode } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState } from "react";
import {
  Activity,
  Boxes,
  FlaskConical,
  Gauge,
  LayoutDashboard,
  Menu,
  Rocket,
  Server,
  Split,
  Settings,
  Siren,
  X,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { StatusBadge } from "@/components/status-badge";
import { useApi } from "@/lib/use-api";
import { formatAgo } from "@/lib/format";
import type { Cluster, Incident, KubernetesStatus } from "@/lib/types";
import { clusterModeLabel, venueLabel } from "@/lib/venue";
import { cn } from "@/lib/utils";

const items = [
  { href: "/", label: "Overview", icon: LayoutDashboard },
  { href: "/services", label: "Services", icon: Boxes },
  { href: "/incidents", label: "Incidents", icon: Siren },
  { href: "/deployments", label: "Deployments", icon: Rocket },
  { href: "/rollouts", label: "Rollouts", icon: Split },
  { href: "/slos", label: "SLOs", icon: Gauge },
  { href: "/infrastructure", label: "Infrastructure", icon: Server },
  { href: "/reliability-lab", label: "Reliability Lab", icon: FlaskConical },
  { href: "/settings", label: "Settings", icon: Settings },
];

function isActive(pathname: string, href: string) {
  if (href === "/") return pathname === "/";
  return pathname === href || pathname.startsWith(`${href}/`);
}

function NavLinks({ pathname, activeIncidents, onNavigate }: { pathname: string; activeIncidents: number; onNavigate?: () => void }) {
  return (
    <nav aria-label="Primary" className="flex-1">
      <ul className="space-y-0.5">
        {items.map((item) => {
          const active = isActive(pathname, item.href);
          const Icon = item.icon;
          return (
            <li key={item.href}>
              <Link
                href={item.href}
                aria-current={active ? "page" : undefined}
                onClick={onNavigate}
                className={cn(
                  "flex items-center gap-2 border-l-2 px-3 py-1.5 text-sm text-muted-foreground hover:bg-muted hover:text-foreground",
                  active && "border-l-primary bg-muted text-foreground",
                  !active && "border-l-transparent",
                )}
              >
                <Icon className="size-4 shrink-0" aria-hidden />
                <span className="flex-1">{item.label}</span>
                {item.href === "/incidents" && activeIncidents > 0 ? (
                  <span className="font-mono text-[11px] text-red-300">{activeIncidents}</span>
                ) : null}
              </Link>
            </li>
          );
        })}
      </ul>
    </nav>
  );
}

export function AppShell({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const [menuPath, setMenuPath] = useState<string | null>(null);
  const open = menuPath === pathname;
  const clusters = useApi<Cluster[]>("/api/v1/clusters", 5000);
  const incidents = useApi<Incident[]>("/api/v1/incidents", 5000);
  const kubernetes = useApi<KubernetesStatus>("/api/v1/kubernetes/status", 5000);
  const cluster = clusters.data?.find((item) => item.simulated) ?? clusters.data?.[0];
  const live = kubernetes.data;
  const activeIncidents = incidents.data?.filter((incident) => incident.status !== "resolved").length ?? 0;

  return (
    <div className="flex h-dvh min-h-0 bg-background">
      <aside className="hidden w-56 shrink-0 flex-col border-r border-border bg-sidebar md:flex">
        <div className="border-b border-border px-3 py-3">
          <div className="text-sm font-semibold tracking-tight">OpsPilot</div>
          <div className="text-[11px] text-muted-foreground">Reliability control plane</div>
        </div>
        <div className="flex min-h-0 flex-1 flex-col py-2">
          <NavLinks pathname={pathname} activeIncidents={activeIncidents} />
        </div>
        <div className="border-t border-border px-3 py-2 text-[11px] text-muted-foreground">
          {venueLabel(live) === "AWS — PLAN ONLY"
            ? "EKS is plan-only. This console is not a cloud deployment."
            : "Local k3d is live. Kubernetes discovery is read-only. AWS is plan-only."}
        </div>
      </aside>
      <div className="flex min-w-0 flex-1 flex-col">
        <header className="flex h-12 shrink-0 items-center gap-3 border-b border-border px-3">
          <Button className="md:hidden" variant="ghost" size="icon" onClick={() => setMenuPath(pathname)} aria-label="Open navigation">
            <Menu />
          </Button>
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2">
              <span className="font-mono text-[11px] uppercase tracking-wide text-muted-foreground">{venueLabel(live)}</span>
              <span className="truncate font-mono text-sm">{live?.cluster ?? "opspilot-dev"}</span>
              <StatusBadge value={live?.connectivity ?? "disconnected"} />
              <span className="hidden font-mono text-[11px] text-muted-foreground sm:inline">{live?.namespace ?? "demo-shop"}</span>
            </div>
            <div className="truncate text-[11px] text-muted-foreground">
              {live?.lastSync ? `Last sync ${formatAgo(live.lastSync)}` : live?.message ?? "Kubernetes status pending"}
              {cluster ? ` · Simulated incident plane ${cluster.name}` : ""}
            </div>
          </div>
          <div className="ml-auto hidden items-center gap-2 text-[11px] text-muted-foreground md:flex">
            <Activity className="size-3.5" aria-hidden />
            <span className="font-mono uppercase">{clusterModeLabel(live?.mode)}</span>
          </div>
        </header>
        <main className="min-h-0 flex-1 overflow-auto">
          <div className="mx-auto w-full max-w-[1440px] p-3 md:p-4">{children}</div>
        </main>
      </div>
      {open ? (
        <div className="fixed inset-0 z-40 md:hidden">
          <button className="absolute inset-0 bg-black/60" aria-label="Close navigation" onClick={() => setMenuPath(null)} />
          <div className="absolute inset-y-0 left-0 flex w-64 flex-col border-r border-border bg-sidebar">
            <div className="flex items-center justify-between border-b border-border px-3 py-3">
              <div className="text-sm font-semibold">OpsPilot</div>
              <Button variant="ghost" size="icon" onClick={() => setMenuPath(null)} aria-label="Close menu">
                <X />
              </Button>
            </div>
            <div className="py-2">
              <NavLinks pathname={pathname} activeIncidents={activeIncidents} onNavigate={() => setMenuPath(null)} />
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}
