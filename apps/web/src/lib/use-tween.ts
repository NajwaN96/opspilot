"use client";

import { useEffect, useRef, useState } from "react";

export function useTween(value: number, duration = 700): number {
  const [display, setDisplay] = useState(value);
  const from = useRef(value);

  useEffect(() => {
    const startValue = from.current;
    if (startValue === value) return;
    const started = performance.now();
    let frame = 0;
    const tick = (now: number) => {
      const t = Math.min(1, (now - started) / duration);
      const eased = 1 - (1 - t) ** 3;
      const next = startValue + (value - startValue) * eased;
      from.current = next;
      setDisplay(next);
      if (t < 1) frame = requestAnimationFrame(tick);
    };
    frame = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(frame);
  }, [value, duration]);

  return display;
}
