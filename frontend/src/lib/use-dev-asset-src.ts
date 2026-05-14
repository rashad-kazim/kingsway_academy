"use client";

import { useEffect, useMemo, useState } from "react";
import { appAsset } from "@/lib/assets";

export function useDevAssetSrc(path: string) {
  const baseSrc = useMemo(() => appAsset(path), [path]);
  const [src, setSrc] = useState(baseSrc);

  useEffect(() => {
    if (process.env.NODE_ENV !== "development") {
      return;
    }

    let cancelled = false;
    const update = () => {
      if (cancelled) {
        return;
      }
      const separator = baseSrc.includes("?") ? "&" : "?";
      setSrc(`${baseSrc}${separator}v=${Date.now()}`);
    };

    const firstUpdate = window.setTimeout(update, 0);
    const interval = window.setInterval(update, 2000);
    window.addEventListener("focus", update);

    return () => {
      cancelled = true;
      window.clearTimeout(firstUpdate);
      window.clearInterval(interval);
      window.removeEventListener("focus", update);
    };
  }, [baseSrc]);

  return src;
}
