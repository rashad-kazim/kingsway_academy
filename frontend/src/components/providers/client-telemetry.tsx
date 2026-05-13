"use client";

import { useEffect, useRef } from "react";
import {
  errorToTelemetryPayload,
  reportClientError,
} from "@/lib/telemetry/client-error";

const maxSeenFingerprints = 40;

export function ClientTelemetry() {
  const seenRef = useRef<string[]>([]);

  useEffect(() => {
    function shouldReport(source: string, message: string) {
      const fingerprint = `${source}:${message}`.slice(0, 220);
      if (seenRef.current.includes(fingerprint)) {
        return false;
      }
      seenRef.current = [...seenRef.current, fingerprint].slice(
        -maxSeenFingerprints,
      );
      return true;
    }

    function onError(event: ErrorEvent) {
      const message = event.error instanceof Error
        ? event.error.message
        : event.message;
      if (!shouldReport("window-error", message || "Unknown client error")) {
        return;
      }
      reportClientError(
        errorToTelemetryPayload(
          "window-error",
          event.error ?? event.message,
          event.message,
        ),
      );
    }

    function onUnhandledRejection(event: PromiseRejectionEvent) {
      const payload = errorToTelemetryPayload(
        "unhandled-rejection",
        event.reason,
        "Unhandled promise rejection",
      );
      if (!shouldReport(payload.source, payload.message)) {
        return;
      }
      reportClientError(payload);
    }

    window.addEventListener("error", onError);
    window.addEventListener("unhandledrejection", onUnhandledRejection);
    return () => {
      window.removeEventListener("error", onError);
      window.removeEventListener("unhandledrejection", onUnhandledRejection);
    };
  }, []);

  return null;
}
