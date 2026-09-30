import type { Metadata } from "next";
import { DeveloperPortal } from "@/components/developer-portal";

export const metadata: Metadata = { title: "Developer Portal" };

export default function Page() {
  return <DeveloperPortal />;
}
