export type ClientErrorSource =
  | "app-error-boundary"
  | "window-error"
  | "unhandled-rejection";

type ReportClientErrorInput = {
  message: string;
  name?: string;
  source: ClientErrorSource;
  stack?: string;
  digest?: string;
  path?: string;
};

const maxFieldLength = 1200;

export function reportClientError(input: ReportClientErrorInput) {
  if (typeof window === "undefined") {
    return;
  }

  const payload = {
    digest: sanitize(input.digest),
    message: sanitize(input.message),
    name: sanitize(input.name),
    path: sanitize(input.path ?? window.location.pathname),
    source: input.source,
    stack: sanitize(input.stack),
    timestamp: new Date().toISOString(),
  };

  const body = JSON.stringify(payload);
  const endpoint = "/api/telemetry/client-error";

  if (navigator.sendBeacon) {
    const blob = new Blob([body], { type: "application/json" });
    if (navigator.sendBeacon(endpoint, blob)) {
      return;
    }
  }

  void fetch(endpoint, {
    body,
    cache: "no-store",
    headers: { "Content-Type": "application/json" },
    keepalive: true,
    method: "POST",
  }).catch(() => {
    // Telemetry must never break the user flow.
  });
}

export function errorToTelemetryPayload(
  source: ClientErrorSource,
  value: unknown,
  fallbackMessage = "Unknown client error",
): ReportClientErrorInput {
  if (value instanceof Error) {
    return {
      digest: "digest" in value ? String(value.digest ?? "") : undefined,
      message: value.message || fallbackMessage,
      name: value.name,
      source,
      stack: value.stack,
    };
  }

  return {
    message: typeof value === "string" ? value : fallbackMessage,
    source,
  };
}

function sanitize(value: string | undefined) {
  if (!value) {
    return "";
  }

  return value
    .replace(/[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}/gi, "[redacted-email]")
    .replace(/Bearer\s+[A-Za-z0-9._~+/=-]+/gi, "Bearer [redacted]")
    .replace(/token[=:]\s*[A-Za-z0-9._~+/=-]+/gi, "token=[redacted]")
    .slice(0, maxFieldLength);
}
