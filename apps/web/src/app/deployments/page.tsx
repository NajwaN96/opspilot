import type { Metadata } from "next";
import { DeploymentsPage } from "@/components/deployments-page";

export const metadata: Metadata = { title: "Deployments" };

export default function Page() {
  return <DeploymentsPage />;
}
