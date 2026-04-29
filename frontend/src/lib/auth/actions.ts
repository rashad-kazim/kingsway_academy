"use server";

import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { z } from "zod";
import { ApiError, login } from "@/lib/api/client";
import type { Role } from "@/lib/api/types";
import { AUTH_COOKIE } from "@/lib/auth/constants";

export type LoginActionState = {
  error?: string;
  fieldErrors?: {
    email?: string[];
    password?: string[];
  };
};

const loginSchema = z.object({
  email: z.string().email(),
  password: z.string().min(8),
  locale: z.string().min(2).max(5),
  next: z.string().optional(),
});

export async function loginAction(
  _prevState: LoginActionState,
  formData: FormData,
): Promise<LoginActionState> {
  const parsed = loginSchema.safeParse({
    email: formData.get("email"),
    password: formData.get("password"),
    locale: formData.get("locale"),
    next: formData.get("next") || undefined,
  });

  if (!parsed.success) {
    const flattened = parsed.error.flatten();
    return {
      error: "invalid_input",
      fieldErrors: {
        email: flattened.fieldErrors.email,
        password: flattened.fieldErrors.password,
      },
    };
  }

  let result;
  try {
    result = await login(parsed.data.email, parsed.data.password);
  } catch (error) {
    if (error instanceof ApiError && error.status === 401) {
      return { error: "invalid_credentials" };
    }
    return { error: "backend_unavailable" };
  }

  const cookieStore = await cookies();
  cookieStore.set(AUTH_COOKIE, result.token, {
    httpOnly: true,
    sameSite: "lax",
    secure: shouldUseSecureAuthCookie(),
    path: "/",
    maxAge: 60 * 60 * 8,
  });

  redirect(safeRedirect(parsed.data.next, parsed.data.locale, result.user.role));
}

export async function logoutAction(formData: FormData) {
  const locale = String(formData.get("locale") ?? "en");
  const cookieStore = await cookies();
  cookieStore.delete(AUTH_COOKIE);
  redirect(`/${locale}/login`);
}

function safeRedirect(next: string | undefined, locale: string, role: Role) {
  if (next?.startsWith(`/${locale}/dashboard`)) {
    return next;
  }

  return `/${locale}${dashboardPathForRole(role)}`;
}

function shouldUseSecureAuthCookie() {
  const configured = process.env.KINGSWAY_AUTH_COOKIE_SECURE;
  if (configured !== undefined) {
    return configured === "true";
  }

  return process.env.NODE_ENV === "production";
}

function dashboardPathForRole(role: Role) {
  switch (role) {
    case "owner":
      return "/dashboard/owner";
    case "receptionist":
      return "/dashboard/receptionist";
    case "teacher":
      return "/dashboard/teacher";
    case "student":
      return "/dashboard/student";
  }
}
