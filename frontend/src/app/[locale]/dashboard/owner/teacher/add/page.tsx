import { getMessages, setRequestLocale } from "next-intl/server";
import { redirect } from "next/navigation";
import { AppShell, type AppShellLabels } from "@/components/layout/app-shell";
import {
  AddTeacherForm,
  type AddTeacherFormLabels,
} from "@/components/owner/add-teacher-form";
import type { TeacherFinanceHrLabels } from "@/components/owner/teacher-finance-hr-view";
import { listBranches } from "@/lib/api/client";
import { requireAuthContext } from "@/lib/auth/session";
import { dashboardPathForRole } from "@/lib/navigation/roles";

type AddTeacherPageProps = {
  params: Promise<{
    locale: string;
  }>;
};

type AddTeacherMessages = AppShellLabels & {
  owner: {
    teacherFinanceHr: TeacherFinanceHrLabels & AddTeacherFormLabels;
  };
};

export const dynamic = "force-dynamic";

export default async function AddTeacherPage({ params }: AddTeacherPageProps) {
  const { locale } = await params;
  setRequestLocale(locale);
  const { session, token } = await requireAuthContext(locale);
  if (session.user.role !== "owner") {
    redirect(`/${locale}${dashboardPathForRole(session.user.role)}`);
  }

  const messages = (await getMessages()) as unknown as AddTeacherMessages;
  const branches = await listBranches(token);

  return (
    <AppShell
      labels={{ common: messages.common, shell: messages.shell }}
      locale={locale}
      session={session}
    >
      <AddTeacherForm
        branches={branches}
        labels={messages.owner.teacherFinanceHr}
        locale={locale}
      />
    </AppShell>
  );
}
