import { setRequestLocale } from "next-intl/server";
import { redirect } from "next/navigation";
import { requireAuthContext } from "@/lib/auth/session";

type DashboardIndexPageProps = {
  params: Promise<{
    locale: string;
  }>;
};

export const dynamic = "force-dynamic";

export default async function DashboardIndexPage({
  params,
}: DashboardIndexPageProps) {
  const { locale } = await params;
  setRequestLocale(locale);
  const { session } = await requireAuthContext(locale);

  redirect(`/${locale}${session.dashboard_path}`);
}
