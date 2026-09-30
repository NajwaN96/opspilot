import type { Metadata } from "next";
import { RunbooksPage } from "@/components/runbooks-page";

export const metadata: Metadata = { title: "Runbooks" };

export default function Page() {
  return <RunbooksPage />;
}
