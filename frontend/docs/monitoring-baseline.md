# Frontend Monitoring Baseline

Status: pre-live baseline, 2026-05-13.

This is intentionally provider-neutral. It gives the frontend a stable internal telemetry path now, and can later forward the same sanitized events to Sentry, Vercel Observability, or another provider.

## Current Implementation

- `ClientTelemetry` is mounted inside `AppProviders`.
- Browser `error` and `unhandledrejection` events are captured.
- Dashboard route error boundary reports App Router runtime errors.
- Events are sent to `POST /api/telemetry/client-error`.
- The route logs a sanitized `client_error` event to server logs.

## Privacy/Safety Rules

- Telemetry must never block user flow.
- Emails and bearer/token-like values are redacted before sending.
- Payload fields are truncated.
- No auth token, password, request body, uploaded file, or raw headers are sent.

## Live Provider Hook

Before production, replace or extend the internal route handler so it forwards sanitized events to the chosen provider.

Keep this internal route as the app-facing contract:

```http
POST /api/telemetry/client-error
```

That keeps frontend components independent from the monitoring vendor.
