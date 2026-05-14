import { NextResponse, type NextRequest } from "next/server";

export const runtime = "nodejs";

const allowedSources = new Set([
  "app-error-boundary",
  "window-error",
  "unhandled-rejection",
]);

export async function POST(request: NextRequest) {
  let payload: unknown;
  try {
    payload = await request.json();
  } catch {
    return NextResponse.json({ ok: false }, { status: 400 });
  }

  if (!isTelemetryPayload(payload)) {
    return NextResponse.json({ ok: false }, { status: 400 });
  }

  console.error("client_error", {
    digest: clean(payload.digest),
    message: clean(payload.message),
    name: clean(payload.name),
    path: clean(payload.path),
    source: payload.source,
    stack: clean(payload.stack),
    timestamp: clean(payload.timestamp),
  });

  return NextResponse.json({ ok: true });
}

function isTelemetryPayload(value: unknown): value is {
  digest?: string;
  message: string;
  name?: string;
  path?: string;
  source: string;
  stack?: string;
  timestamp?: string;
} {
  if (!value || typeof value !== "object") {
    return false;
  }

  const record = value as Record<string, unknown>;
  return (
    typeof record.message === "string" &&
    typeof record.source === "string" &&
    allowedSources.has(record.source)
  );
}

function clean(value: string | undefined) {
  if (!value) {
    return "";
  }
  return value.slice(0, 1200);
}
