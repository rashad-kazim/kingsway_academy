import { cookies } from "next/headers";
import { NextResponse } from "next/server";
import { isLocale } from "@/i18n/routing";
import { AUTH_COOKIE } from "@/lib/auth/constants";

export async function GET(request: Request) {
  const url = new URL(request.url);
  const requestedLocale = url.searchParams.get("locale") ?? "en";
  const locale = isLocale(requestedLocale) ? requestedLocale : "en";

  const cookieStore = await cookies();
  cookieStore.delete(AUTH_COOKIE);

  return NextResponse.redirect(
    new URL(`/${locale}/login?session=expired`, url),
  );
}
