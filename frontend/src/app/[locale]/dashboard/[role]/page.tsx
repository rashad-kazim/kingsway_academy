import { setRequestLocale } from "next-intl/server";
import { notFound, redirect } from "next/navigation";
import { RoleDashboardView } from "@/components/dashboard/role-dashboard-view";
import { getDashboard } from "@/lib/api/client";
import { requireAuthContext } from "@/lib/auth/session";
import { dashboardPathForRole, isRole } from "@/lib/navigation/roles";

type RoleDashboardPageProps = {
  params: Promise<{
    locale: string;
    role: string;
  }>;
};

export const dynamic = "force-dynamic";

export default async function RoleDashboardPage({
  params,
}: RoleDashboardPageProps) {
  const { locale, role } = await params;
  setRequestLocale(locale);
  if (!isRole(role)) {
    notFound();
  }

  const { token, session } = await requireAuthContext(locale);
  if (session.user.role !== role) {
    redirect(`/${locale}${dashboardPathForRole(session.user.role)}`);
  }

  const dashboard = await getDashboard(role, token);

  return (
    <RoleDashboardView dashboard={dashboard} role={role} session={session} />
  );
}
