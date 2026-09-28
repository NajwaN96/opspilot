import type { Metadata } from "next";
import { SloPage } from "@/components/slo-page";

export const metadata: Metadata = { title: "SLOs" };

export default function Page() {
  return <SloPage />;
}
