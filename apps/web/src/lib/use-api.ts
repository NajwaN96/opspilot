"use client";

import { useCallback, useEffect, useState } from "react";
import { apiGet } from "@/lib/api";

export function useApi<T>(path: string, refreshMs?: number) {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const reload = useCallback(async () => {
    try {
      const next = await apiGet<T>(path);
      setData(next);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Request failed");
    } finally {
      setLoading(false);
    }
  }, [path]);

  useEffect(() => {
    let active = true;
    const run = async () => {
      try {
        const next = await apiGet<T>(path);
        if (!active) return;
        setData(next);
        setError(null);
      } catch (err) {
        if (!active) return;
        setError(err instanceof Error ? err.message : "Request failed");
      } finally {
        if (active) setLoading(false);
      }
    };
    void run();
    if (!refreshMs) return () => {
      active = false;
    };
    const id = setInterval(() => void run(), refreshMs);
    return () => {
      active = false;
      clearInterval(id);
    };
  }, [path, refreshMs]);

  return { data, error, loading, reload };
}
