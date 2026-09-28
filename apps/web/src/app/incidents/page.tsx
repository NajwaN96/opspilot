import type { Metadata } from "next";
import { IncidentList } from "@/components/incident-list";

export const metadata: Metadata = { title: "Incidents" };

export default function Page() {
  return <IncidentList />;
}
