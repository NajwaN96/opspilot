import type { Metadata } from "next";
import { ContractDetail } from "@/components/contract-detail";

export const metadata: Metadata = { title: "Service contract" };

export function generateStaticParams() {
  return ["storefront", "checkout-api", "payment-api", "orders-api", "inventory-api"].map((name) => ({ name }));
}

export const dynamicParams = false;

export default async function Page({ params }: { params: Promise<{ name: string }> }) {
  const { name } = await params;
  return <ContractDetail name={name} />;
}
