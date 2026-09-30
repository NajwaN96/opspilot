import type { NextConfig } from "next";

const apiOrigin = process.env.API_PROXY_URL ?? "http://127.0.0.1:8094";
const portfolioExport = process.env.OPSPILOT_AWS_PORTFOLIO === "1";

const nextConfig: NextConfig = {
  // The console is opened at 127.0.0.1. Next blocks dev assets for that host unless it is listed.
  allowedDevOrigins: ["127.0.0.1", "localhost"],
  ...(portfolioExport
    ? {
        output: "export" as const,
        trailingSlash: true,
        images: { unoptimized: true },
      }
    : {
        async rewrites() {
          return [
            { source: "/api/:path*", destination: `${apiOrigin}/api/:path*` },
            { source: "/health", destination: `${apiOrigin}/health` },
            { source: "/ready", destination: `${apiOrigin}/ready` },
          ];
        },
      }),
};

export default nextConfig;
