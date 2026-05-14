"use client";

import { useCallback, useEffect, useRef, useState } from "react";

export function useSubmitLock(pending: boolean) {
  const [locked, setLocked] = useState(false);
  const lockedRef = useRef(false);
  const pendingRef = useRef(pending);
  const sawPending = useRef(false);
  const unlockTimerRef = useRef<number | null>(null);

  useEffect(() => {
    pendingRef.current = pending;
    if (pending) {
      sawPending.current = true;
      if (unlockTimerRef.current) {
        window.clearTimeout(unlockTimerRef.current);
        unlockTimerRef.current = null;
      }
      return;
    }

    if (sawPending.current) {
      sawPending.current = false;
      lockedRef.current = false;
      setLocked(false);
    }
  }, [pending]);

  useEffect(() => {
    return () => {
      if (unlockTimerRef.current) {
        window.clearTimeout(unlockTimerRef.current);
      }
    };
  }, []);

  const onClick = useCallback(
    (event: { preventDefault: () => void }) => {
      if (pendingRef.current || lockedRef.current) {
        event.preventDefault();
        return;
      }

      lockedRef.current = true;
      window.setTimeout(() => setLocked(true), 0);
      unlockTimerRef.current = window.setTimeout(() => {
        if (!pendingRef.current && !sawPending.current) {
          lockedRef.current = false;
          setLocked(false);
        }
        unlockTimerRef.current = null;
      }, 1500);
    },
    [],
  );

  return { locked, onClick };
}
