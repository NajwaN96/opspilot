import type { Metadata } from "next";
import { GoldenPathPage } from "@/components/golden-path-page";

export const metadata: Metadata = { title: "Golden path" };

export default function Page() {
  return <GoldenPathPage />;
}
