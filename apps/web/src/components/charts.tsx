"use client";

import { CartesianGrid, Legend, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import type { MetricPoint } from "@/lib/types";

const axis = { fontSize: 11, fill: "#9aa3b2" };

export function MetricChart({ data }: { data: MetricPoint[] }) {
  if (data.length === 0) {
    return <p className="text-sm text-muted-foreground">No metric samples in this window.</p>;
  }
  return (
    <div className="h-56 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={data} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
          <CartesianGrid stroke="#2a3142" vertical={false} />
          <XAxis dataKey="clock" tick={axis} stroke="#2a3142" minTickGap={24} />
          <YAxis yAxisId="latency" tick={axis} stroke="#2a3142" width={40} />
          <YAxis yAxisId="errors" orientation="right" tick={axis} stroke="#2a3142" width={32} />
          <Tooltip
            contentStyle={{
              background: "#161b24",
              border: "1px solid #2a3142",
              borderRadius: 2,
              fontSize: 12,
            }}
          />
          <Legend wrapperStyle={{ fontSize: 12 }} />
          <Line yAxisId="latency" type="monotone" dataKey="p95LatencyMs" name="p95 ms" stroke="#e6b450" dot={false} strokeWidth={1.5} isAnimationActive={false} />
          <Line yAxisId="errors" type="monotone" dataKey="errorRate" name="error %" stroke="#f07178" dot={false} strokeWidth={1.5} isAnimationActive={false} />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
