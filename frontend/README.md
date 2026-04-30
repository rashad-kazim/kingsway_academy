# Kingsway Frontend

Next.js App Router frontend for the Kingsway Operation Center.

## Stack

- Next.js 16.2, React 19.2, TypeScript
- Tailwind CSS 4 and shadcn/ui
- next-intl with `en`, `tr`, and `az`
- next-themes for dark/light mode
- TanStack Query for client server-state where interactive screens need it
- Zod and React Hook Form ready for validated forms
- Lucide icons

## Runtime Contract

Set the backend base URL:

```bash
KINGSWAY_API_BASE_URL=http://127.0.0.1:8080
KINGSWAY_AUTH_COOKIE_SECURE=false
```

The frontend stores backend JWTs in an HTTP-only `kingsway_token` cookie. Server Components and Server Actions call the Go backend directly; the token is not exposed to browser JavaScript.

`KINGSWAY_AUTH_COOKIE_SECURE=false` is for local HTTP development. Production HTTPS deployments should set it to `true` or omit it and rely on the production default.

## Architecture

- `src/proxy.ts`: Next 16 proxy for locale routing and optimistic auth redirects.
- `src/lib/api`: backend API types and `apiFetch` wrapper.
- `src/lib/auth`: login/logout server actions and session helpers.
- `src/components/auth`: login form.
- `src/components/layout`: authenticated app shell.
- `src/components/dashboard`: reusable role dashboard presentation.
- `src/app/[locale]/login`: localized login screen.
- `src/app/[locale]/dashboard`: authenticated role dashboard routes.

## Development

Preferred full-stack local run from the repository root:

```bash
docker compose up -d --build
```

On Windows, `start-kingsway.cmd` in the repository root starts the same stack; `stop-kingsway.cmd` stops it without deleting data.

Open:

```text
http://127.0.0.1:3000/en/login
```

For frontend-only development:

```bash
npm run dev -- --hostname 127.0.0.1 --port 3000
```

## Verification

```bash
npm run lint
npm run build
```

Both pass after the initial frontend architecture pass.

Docker full-stack smoke also passes from the repository root:

```bash
docker compose up -d --build
```

The API health endpoint, login page, and authenticated owner dashboard return `200 OK`.
