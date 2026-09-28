import type { Metadata } from "next";
import { ReliabilityLab } from "@/components/reliability-lab";

export const metadata: Metadata = { title: "Reliability Lab" };

export default function Page() {
  return <ReliabilityLab />;
}
