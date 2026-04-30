import { setRequestLocale } from "next-intl/server";
import { AppShell } from "@/components/layout/app-shell";
import { requireAuthContext } from "@/lib/auth/session";

type DashboardLayoutProps = {
  children: React.ReactNode;
  params: Promise<{
    locale: string;
  }>;
};

export const dynamic = "force-dynamic";

export default async function DashboardLayout({
  children,
  params,
}: DashboardLayoutProps) {
  const { locale } = await params;
  setRequestLocale(locale);
  const { session } = await requireAuthContext(locale);

  return (
    <AppShell locale={locale} session={session}>
      {children}
    </AppShell>
  );
}
