import type { DashboardRecord, LoginResult, Role, Session } from "./types";

type ApiFetchOptions = Omit<RequestInit, "body"> & {
  body?: unknown;
  token?: string;
};

export class ApiError extends Error {
  status: number;
  payload: unknown;

  constructor(status: number, payload: unknown) {
    super(extractErrorMessage(payload) ?? `Backend request failed: ${status}`);
    this.name = "ApiError";
    this.status = status;
    this.payload = payload;
  }
}

export async function apiFetch<T>(
  path: string,
  options: ApiFetchOptions = {},
): Promise<T> {
  const headers = new Headers(options.headers);
  if (options.token) {
    headers.set("Authorization", `Bearer ${options.token}`);
  }
  if (options.body !== undefined && !(options.body instanceof FormData)) {
    headers.set("Content-Type", "application/json");
  }
  if (!headers.has("X-Request-ID")) {
    headers.set("X-Request-ID", crypto.randomUUID());
  }

  const response = await fetch(backendURL(path), {
    ...options,
    headers,
    cache: "no-store",
    body:
      options.body === undefined
        ? undefined
        : options.body instanceof FormData
          ? options.body
          : JSON.stringify(options.body),
  });

  const payload = await readPayload(response);
  if (!response.ok) {
    throw new ApiError(response.status, payload);
  }

  return payload as T;
}

export function login(email: string, password: string) {
  return apiFetch<LoginResult>("/v1/auth/login", {
    method: "POST",
    body: { email, password },
  });
}

export function getSession(token: string) {
  return apiFetch<Session>("/v1/session", { token });
}

export function getDashboard(role: Role, token: string) {
  return apiFetch<DashboardRecord>(`/v1/dashboard/${role}`, { token });
}

function backendURL(path: string) {
  const base = process.env.KINGSWAY_API_BASE_URL ?? "http://127.0.0.1:8080";
  return new URL(path, base).toString();
}

async function readPayload(response: Response) {
  const contentType = response.headers.get("content-type") ?? "";
  if (contentType.includes("application/json")) {
    return response.json();
  }

  const text = await response.text();
  return text.length > 0 ? text : null;
}

function extractErrorMessage(payload: unknown) {
  if (
    payload &&
    typeof payload === "object" &&
    "error" in payload &&
    typeof payload.error === "string"
  ) {
    return payload.error;
  }

  return null;
}
