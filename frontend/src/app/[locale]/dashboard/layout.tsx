import { setRequestLocale } from "next-intl/server";
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
  await requireAuthContext(locale);

  return children;
}
