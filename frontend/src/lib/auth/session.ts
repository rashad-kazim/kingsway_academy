import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { ApiError, getSession } from "@/lib/api/client";
import type { Session } from "@/lib/api/types";
import { AUTH_COOKIE } from "@/lib/auth/constants";

export type AuthContext = {
  token: string;
  session: Session;
};

export async function getAuthToken() {
  const cookieStore = await cookies();
  return cookieStore.get(AUTH_COOKIE)?.value ?? null;
}

export async function currentSession() {
  const token = await getAuthToken();
  if (!token) {
    return null;
  }

  try {
    return await getSession(token);
  } catch (error) {
    if (error instanceof ApiError && error.status === 401) {
      return null;
    }
    return null;
  }
}

export async function requireAuthContext(locale: string): Promise<AuthContext> {
  const token = await getAuthToken();
  if (!token) {
    redirect(`/${locale}/login`);
  }

  const session = await currentSession();
  if (!session) {
    redirect(`/${locale}/login`);
  }

  return { token, session };
}
