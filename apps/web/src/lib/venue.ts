export function venueLabel(status?: { venueLabel?: string; mode?: string; cluster?: string } | null): string {
  if (!status) return "LOCAL — LIVE";
  if (status.venueLabel) return status.venueLabel;
  if (status.mode === "aws-portfolio-demo") return "AWS — PORTFOLIO DEMO";
  if (status?.cluster === "opspilot-aws-dev" || status?.mode === "eks") return "AWS — PLAN ONLY";
  if (status?.cluster === "opspilot-dev" || status?.mode === "local-kubernetes") return "LOCAL — LIVE";
  return "UNKNOWN";
}

export function clusterModeLabel(mode?: string): string {
  if (mode === "local-kubernetes") return "Local k3d";
  if (mode === "eks") return "EKS plan only";
  if (mode === "aws-portfolio-demo") return "AWS portfolio demo";
  return "Unavailable";
}
