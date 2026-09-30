"use client";

import type { ReactNode } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState, useSyncExternalStore } from "react";
import {
  Activity,
  Boxes,
  FlaskConical,
  Gauge,
  BookOpen,
  Compass,
  LayoutDashboard,
  Menu,
  Network,
  PanelLeftClose,
  PanelLeftOpen,
  Shield,
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
import { isPortfolioRuntime } from "@/lib/runtime";
import { clusterModeLabel, venueLabel } from "@/lib/venue";
import { cn } from "@/lib/utils";

const groups = [
  {
    label: "Operate",
    items: [
      { href: "/", label: "Overview", icon: LayoutDashboard },
      { href: "/developer-portal", label: "Developer Portal", icon: Compass },
      { href: "/services", label: "Services", icon: Boxes },
      { href: "/incidents", label: "Incidents", icon: Siren },
      { href: "/deployments", label: "Deployments", icon: Rocket },
      { href: "/rollouts", label: "Rollouts", icon: Split },
      { href: "/slos", label: "SLOs", icon: Gauge },
    ],
  },
  {
    label: "Platform",
    items: [
      { href: "/infrastructure", label: "Infrastructure", icon: Server },
      { href: "/reliability-lab", label: "Reliability Lab", icon: FlaskConical },
      { href: "/runbooks", label: "Runbooks", icon: BookOpen },
    ],
  },
  {
    label: "System",
    items: [
      { href: "/architecture", label: "Architecture", icon: Network },
      { href: "/security", label: "Security", icon: Shield },
      { href: "/settings", label: "Settings", icon: Settings },
    ],
  },
];

let navCollapsed = false;
const navListeners = new Set<() => void>();

function subscribeNav(listener: () => void) {
  navListeners.add(listener);
  return () => navListeners.delete(listener);
}

function navCollapsedSnapshot() {
  return navCollapsed;
}

function navCollapsedServer() {
  return false;
}

function setNavCollapsed(next: boolean) {
  navCollapsed = next;
  window.localStorage.setItem("opspilot-nav", next ? "collapsed" : "open");
  navListeners.forEach((listener) => listener());
}

if (typeof window !== "undefined") {
  navCollapsed = window.localStorage.getItem("opspilot-nav") === "collapsed";
}

function isActive(pathname: string, href: string) {
  if (href === "/") return pathname === "/";
  return pathname === href || pathname.startsWith(`${href}/`);
}

function NavLinks({
  pathname,
  activeIncidents,
  collapsed,
  onNavigate,
}: {
  pathname: string;
  activeIncidents: number;
  collapsed?: boolean;
  onNavigate?: () => void;
}) {
  return (
    <nav aria-label="Primary" className="flex-1">
      {groups.map((group) => (
        <div key={group.label} className="mb-3">
          {collapsed ? null : (
            <div className="px-3 pb-1 text-[10px] font-semibold uppercase tracking-[0.08em] text-muted-foreground/80">{group.label}</div>
          )}
          <ul className="space-y-0.5">
            {group.items.map((item) => {
              const active = isActive(pathname, item.href);
              const Icon = item.icon;
              return (
                <li key={item.href}>
                  <Link
                    href={item.href}
                    aria-current={active ? "page" : undefined}
                    title={collapsed ? item.label : undefined}
                    onClick={onNavigate}
                    className={cn(
                      "relative flex h-9 items-center gap-2.5 rounded-[10px] px-2.5 text-[13px] text-muted-foreground transition-[background-color,color] duration-150 hover:bg-black/[0.04] hover:text-foreground",
                      collapsed && "justify-center px-0",
                      active && "bg-[var(--nav-active-bg)] font-medium text-[var(--nav-active)] hover:bg-[var(--nav-active-bg)] hover:text-[var(--nav-active)]",
                    )}
                  >
                    {active ? <span className="absolute top-1/2 left-0 h-4 w-0.5 -translate-y-1/2 rounded-full bg-[var(--nav-active)]" aria-hidden /> : null}
                    <Icon className="size-4 shrink-0" strokeWidth={1.75} aria-hidden />
                    {collapsed ? null : <span className="flex-1 truncate">{item.label}</span>}
                    {!collapsed && item.href === "/incidents" && activeIncidents > 0 ? (
                      <span className="inline-flex h-5 min-w-5 items-center justify-center rounded-md bg-status-danger-soft px-1 text-[11px] font-medium text-status-danger">
                        {activeIncidents}
                      </span>
                    ) : null}
                  </Link>
                </li>
              );
            })}
          </ul>
        </div>
      ))}
    </nav>
  );
}

function contextMeta(live?: KubernetesStatus | null, portfolio?: boolean) {
  const label = venueLabel(live);
  if (portfolio || label.startsWith("AWS — PORTFOLIO")) {
    return { platform: "AWS", environment: "Portfolio Demo", simulation: true };
  }
  if (label.startsWith("AWS — PLAN")) {
    return { platform: "AWS", environment: "Plan only", simulation: false };
  }
  if (label.startsWith("LOCAL")) {
    return { platform: "Local", environment: "Live lab", simulation: false };
  }
  return { platform: "Context", environment: label, simulation: false };
}

export function AppShell({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const [menuPath, setMenuPath] = useState<string | null>(null);
  const collapsed = useSyncExternalStore(subscribeNav, navCollapsedSnapshot, navCollapsedServer);
  const open = menuPath === pathname;
  const clusters = useApi<Cluster[]>("/api/v1/clusters", 5000);
  const incidents = useApi<Incident[]>("/api/v1/incidents", 5000);
  const kubernetes = useApi<KubernetesStatus>("/api/v1/kubernetes/status", 5000);
  const cluster = clusters.data?.find((item) => item.simulated) ?? clusters.data?.[0];
  const live = kubernetes.data;
  const portfolio = isPortfolioRuntime() || live?.mode === "aws-portfolio-demo";
  const activeIncidents = incidents.data?.filter((incident) => incident.status !== "resolved").length ?? 0;
  const meta = contextMeta(live, portfolio);

  function toggleCollapsed() {
    setNavCollapsed(!collapsed);
  }

  const footnote = portfolio
    ? "Public AWS portfolio. Not connected to k3d or EKS. Mutations stay refused."
    : venueLabel(live) === "AWS — PLAN ONLY"
      ? "EKS is plan-only. This console is not a cloud deployment."
      : "Local k3d is live. Kubernetes discovery is read-only. AWS EKS is plan-only.";

  return (
    <div className="flex h-dvh min-h-0">
      <aside
        className={cn(
          "glass-bar hidden shrink-0 flex-col border-r border-black/[0.06] md:flex",
          collapsed ? "w-[72px]" : "w-[236px]",
        )}
      >
        <div className={cn("flex items-center gap-2 px-3 pt-4 pb-3", collapsed && "flex-col px-2")}>
          {collapsed ? (
            <div className="text-[13px] font-semibold tracking-tight text-foreground">Op</div>
          ) : (
            <div className="min-w-0 flex-1">
              <div className="text-[15px] font-semibold tracking-tight">OpsPilot</div>
              <div className="truncate text-[11px] text-muted-foreground">Reliability control plane</div>
            </div>
          )}
          <Button variant="ghost" size="icon" onClick={toggleCollapsed} aria-pressed={collapsed} aria-label={collapsed ? "Expand navigation" : "Collapse navigation"}>
            {collapsed ? <PanelLeftOpen strokeWidth={1.75} /> : <PanelLeftClose strokeWidth={1.75} />}
          </Button>
        </div>
        <div className="flex min-h-0 flex-1 flex-col overflow-y-auto px-2 py-1">
          <NavLinks pathname={pathname} activeIncidents={activeIncidents} collapsed={collapsed} />
        </div>
        {collapsed ? null : (
          <div className="border-t border-black/[0.05] px-3 py-3">
            <p className="text-[11px] leading-snug text-muted-foreground">{footnote}</p>
          </div>
        )}
      </aside>
      <div className="flex min-w-0 flex-1 flex-col">
        <header className="glass-bar sticky top-0 z-30 flex min-h-14 shrink-0 items-center gap-3 border-b border-black/[0.06] px-3 md:px-5">
          <Button className="size-11 md:hidden" variant="ghost" size="icon" onClick={() => setMenuPath(pathname)} aria-label="Open navigation">
            <Menu strokeWidth={1.75} />
          </Button>
          <div className="flex min-w-0 flex-1 flex-wrap items-center gap-2">
            <span className="inline-flex h-6 items-center rounded-md border border-border bg-status-neutral-soft px-2 text-[11px] font-medium text-status-neutral">
              {meta.platform}
            </span>
            <span className="inline-flex h-6 items-center rounded-md border border-status-info/20 bg-status-info-soft px-2 text-[11px] font-medium text-status-info">
              {meta.environment}
            </span>
            {meta.simulation ? (
              <span className="inline-flex h-6 items-center rounded-md border border-status-sim/20 bg-status-sim-soft px-2 text-[11px] font-medium text-status-sim">
                Simulation data
              </span>
            ) : null}
            <span className="hidden text-muted-foreground/50 sm:inline" aria-hidden>
              /
            </span>
            <span className="truncate text-[13px] text-foreground">{live?.cluster ?? "opspilot-dev"}</span>
            <span className="text-muted-foreground/40" aria-hidden>
              /
            </span>
            <span className="truncate text-[13px] text-muted-foreground">{live?.namespace ?? "demo-shop"}</span>
            <StatusBadge value={live?.connectivity ?? "disconnected"} />
          </div>
          <div className="hidden min-w-0 items-center gap-2 text-[12px] text-muted-foreground lg:flex">
            <Activity className="size-3.5" strokeWidth={1.75} aria-hidden />
            <span>{clusterModeLabel(live?.mode)}</span>
            <span className="text-muted-foreground/40" aria-hidden>
              ·
            </span>
            <span className="truncate">
              {live?.lastSync ? `Last sync ${formatAgo(live.lastSync)}` : live?.message ?? "Kubernetes status pending"}
              {cluster ? ` · ${cluster.name}` : ""}
            </span>
          </div>
        </header>
        <main className="min-h-0 flex-1 overflow-auto">
          <div className="mx-auto w-full max-w-[1600px] px-4 py-5 md:px-6 md:py-6">{children}</div>
        </main>
      </div>
      {open ? (
        <div className="fixed inset-0 z-40 md:hidden">
          <button className="absolute inset-0 bg-slate-900/30" aria-label="Close navigation" onClick={() => setMenuPath(null)} />
          <div className="absolute inset-y-0 left-0 flex w-[280px] flex-col border-r border-black/[0.06] bg-white shadow-[var(--elevation-overlay)]">
            <div className="flex items-center justify-between px-3 py-3">
              <div>
                <div className="text-sm font-semibold">OpsPilot</div>
                <div className="text-[11px] text-muted-foreground">Reliability control plane</div>
              </div>
              <Button variant="ghost" size="icon" className="size-11" onClick={() => setMenuPath(null)} aria-label="Close menu">
                <X strokeWidth={1.75} />
              </Button>
            </div>
            <div className="overflow-y-auto px-2 py-1">
              <NavLinks pathname={pathname} activeIncidents={activeIncidents} onNavigate={() => setMenuPath(null)} />
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}
