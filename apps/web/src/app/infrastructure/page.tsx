import type { Metadata } from "next";
import { InfrastructurePage } from "@/components/infrastructure-page";

export const metadata: Metadata = { title: "Infrastructure" };

export default function Page() {
  return <InfrastructurePage />;
}
