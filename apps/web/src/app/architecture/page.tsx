import type { Metadata } from "next";
import { ArchitecturePage } from "@/components/architecture-page";

export const metadata: Metadata = { title: "Architecture" };

export default function Page() {
  return <ArchitecturePage />;
}
