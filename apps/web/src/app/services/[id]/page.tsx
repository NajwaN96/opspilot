import type { Metadata } from "next";
import { ServiceDetail } from "@/components/service-detail";

export const metadata: Metadata = { title: "Service" };

export function generateStaticParams() {
  return [
    { id: "k8s_demo-shop_payment-api" },
    { id: "k8s_demo-shop_checkout-api" },
    { id: "k8s_demo-shop_inventory" },
  ];
}

export default async function Page({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <ServiceDetail id={id} />;
}
