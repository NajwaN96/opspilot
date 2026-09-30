"use client";

import { CartesianGrid, Legend, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import type { MetricPoint } from "@/lib/types";

const axis = { fontSize: 11, fill: "#606775" };

export function MetricChart({ data }: { data: MetricPoint[] }) {
  if (data.length === 0) {
    return <p className="text-sm text-muted-foreground">No metric samples in this window.</p>;
  }
  return (
    <div className="h-56 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={data} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
          <CartesianGrid stroke="rgba(15,23,42,0.08)" vertical={false} />
          <XAxis dataKey="clock" tick={axis} stroke="rgba(15,23,42,0.08)" minTickGap={24} />
          <YAxis yAxisId="latency" tick={axis} stroke="rgba(15,23,42,0.08)" width={40} />
          <YAxis yAxisId="errors" orientation="right" tick={axis} stroke="rgba(15,23,42,0.08)" width={32} />
          <Tooltip
            contentStyle={{
              background: "#ffffff",
              border: "1px solid rgba(15,23,42,0.08)",
              borderRadius: 10,
              fontSize: 12,
              color: "#111318",
            }}
          />
          <Legend wrapperStyle={{ fontSize: 12, color: "#606775" }} />
          <Line yAxisId="latency" type="monotone" dataKey="p95LatencyMs" name="p95 ms" stroke="#c78300" dot={false} strokeWidth={1.75} isAnimationActive={false} />
          <Line yAxisId="errors" type="monotone" dataKey="errorRate" name="error %" stroke="#d64545" dot={false} strokeWidth={1.75} isAnimationActive={false} />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
