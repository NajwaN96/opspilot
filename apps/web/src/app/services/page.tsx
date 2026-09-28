import type { Metadata } from "next";
import { ServicesCatalog } from "@/components/services-catalog";

export const metadata: Metadata = { title: "Services" };

export default function Page() {
  return <ServicesCatalog />;
}
