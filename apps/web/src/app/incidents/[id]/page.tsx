import type { Metadata } from "next";
import { Investigation } from "@/components/incident/investigation";

export const metadata: Metadata = { title: "Incident" };

export function generateStaticParams() {
  return [{ id: "INC-142" }];
}

export default async function Page({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <Investigation id={id} />;
}
