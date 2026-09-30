import type { Metadata } from "next";
import { RunbookDetail } from "@/components/runbooks-page";

export const metadata: Metadata = { title: "Runbook" };

export function generateStaticParams() {
  return [
    "payment-api-high-error-rate",
    "payment-api-latency",
    "bad-release",
    "canary-failure",
    "crashloop",
    "dependency-failure",
  ].map((id) => ({ id }));
}

export const dynamicParams = false;

export default async function Page({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <RunbookDetail id={id} />;
}
