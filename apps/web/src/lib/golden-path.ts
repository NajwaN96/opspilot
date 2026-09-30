export type GoldenRuntime = "go" | "node" | "python";
export type GoldenStrategy = "rolling" | "canary";

export interface GoldenInput {
  name: string;
  runtime: GoldenRuntime;
  owner: string;
  slo: string;
  strategy: GoldenStrategy;
}

export interface PreviewFile {
  path: string;
  body: string;
}

const NAME = /^[a-z][a-z0-9-]{1,40}$/;

export function previewFiles(input: GoldenInput): PreviewFile[] {
  const name = input.name.trim();
  const owner = input.owner.trim() || "unowned";
  const slo = input.slo.trim() || "99.9";
  const strategy = input.strategy;
  const source = input.runtime === "go" ? "main.go" : input.runtime === "node" ? "server.js" : "app.py";
  return [
    {
      path: "Dockerfile",
      body: `FROM ${input.runtime === "go" ? "golang:1.27.1-alpine" : input.runtime === "node" ? "node:22-alpine" : "python:3.12-alpine"}\n# Build ${name}. Run as uid 65532. See templates/services/${input.runtime}/Dockerfile.\nUSER 65532:65532\nEXPOSE 8080\n`,
    },
    {
      path: source,
      body: `# ${name} serves GET /health, GET /ready, and GET /metrics.\n# Set OTEL_EXPORTER_OTLP_ENDPOINT to export traces. Do not accept PromQL from the browser.\n`,
    },
    {
      path: "deploy/deployment.yaml",
      body: [
        "apiVersion: apps/v1",
        "kind: Deployment",
        `metadata: { name: ${name} }`,
        "spec.template.spec.securityContext.runAsNonRoot: true",
        "containers:",
        `- name: ${name}`,
        "  readinessProbe: /ready",
        "  livenessProbe: /health",
        "  resources.requests: cpu 50m, memory 64Mi",
        "  resources.limits: cpu 200m, memory 128Mi",
        "  env: OTEL_EXPORTER_OTLP_ENDPOINT",
        "  securityContext.readOnlyRootFilesystem: true",
      ].join("\n"),
    },
    {
      path: "deploy/service.yaml",
      body: `kind: Service\nmetadata:\n  name: ${name}\n`,
    },
    {
      path: "kustomization.yaml",
      body: `resources:\n  - deploy/deployment.yaml\n  - deploy/service.yaml\n`,
    },
    {
      path: "slo.yaml",
      body: `availability: ${slo}\nlatencyMs: 300\nerrorRate: 1.0\nstrategy: ${strategy}\n`,
    },
    {
      path: "service.yaml",
      body: `apiVersion: opspilot.io/v1\nkind: Service\nmetadata:\n  name: ${name}\nspec:\n  owner: ${owner}\n  runtime: ${input.runtime}\n  deploymentStrategy: ${strategy}\n`,
    },
    {
      path: "ci.yml",
      body: `name: ${name}\non: pull_request\njobs:\n  test:\n    steps:\n      - run: echo test ${name}\n`,
    },
    {
      path: "runbook.md",
      body: `# ${name}\n\nSymptoms, detection, evidence, allowed remediation, verification, escalation.\nThe public site does not execute this runbook.\n`,
    },
  ];
}

export function previewError(input: GoldenInput): string | null {
  if (!NAME.test(input.name.trim())) return "Use a short lowercase name, for example billing-api.";
  if (!["go", "node", "python"].includes(input.runtime)) return "Runtime must be Go, Node.js, or Python.";
  if (!input.owner.trim()) return "Owner is required.";
  if (!/^[0-9]{2,3}(\.[0-9])?$/.test(input.slo.trim())) return "SLO target looks like 99.9.";
  if (input.strategy !== "rolling" && input.strategy !== "canary") return "Strategy must be rolling or canary.";
  return null;
}
