import createMiddleware from "next-intl/middleware";
import { NextResponse, type NextRequest } from "next/server";
import { routing } from "@/i18n/routing";
import { AUTH_COOKIE } from "@/lib/auth/constants";

const intlProxy = createMiddleware(routing);

export default function proxy(request: NextRequest) {
  const pathname = request.nextUrl.pathname;
  const localized = localizedPath(pathname);

  if (!localized) {
    return intlProxy(request);
  }

  const token = request.cookies.get(AUTH_COOKIE)?.value;
  const isDashboard = localized.path === "/dashboard" || localized.path.startsWith("/dashboard/");
  const isLogin = localized.path === "/login";

  if (isDashboard && !token) {
    const url = request.nextUrl.clone();
    url.pathname = `/${localized.locale}/login`;
    url.searchParams.set("next", pathname);
    return NextResponse.redirect(url);
  }

  if (isLogin && token) {
    const url = request.nextUrl.clone();
    url.pathname = `/${localized.locale}/dashboard`;
    url.search = "";
    return NextResponse.redirect(url);
  }

  return intlProxy(request);
}

export const config = {
  matcher: ["/((?!api|_next|_vercel|.*\\..*).*)"],
};

function localizedPath(pathname: string) {
  const [, maybeLocale, ...rest] = pathname.split("/");
  if (!routing.locales.includes(maybeLocale as (typeof routing.locales)[number])) {
    return null;
  }

  const path = `/${rest.join("/")}`;
  return {
    locale: maybeLocale,
    path: path === "/" ? "/" : path.replace(/\/$/, ""),
  };
}
