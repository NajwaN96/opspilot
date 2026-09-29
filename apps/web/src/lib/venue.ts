export function venueLabel(status?: { venueLabel?: string; mode?: string; cluster?: string } | null): string {
  if (!status) return "LOCAL";
  if (status.venueLabel) return status.venueLabel;
  if (status?.cluster === "opspilot-aws-dev" || status?.mode === "eks") return "AWS DEV";
  if (status?.cluster === "opspilot-dev" || status?.mode === "local-kubernetes") return "LOCAL";
  return "UNKNOWN";
}

export function clusterModeLabel(mode?: string): string {
  if (mode === "local-kubernetes") return "Local Kubernetes";
  if (mode === "eks") return "Amazon EKS";
  return "Unavailable";
}
