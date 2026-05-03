import { getMessages, setRequestLocale } from "next-intl/server";
import { notFound, redirect } from "next/navigation";
import {
  RoleDashboardView,
  type DashboardViewLabels,
} from "@/components/dashboard/role-dashboard-view";
import { AppShell, type AppShellLabels } from "@/components/layout/app-shell";
import {
  BranchManagementView,
  type BranchManagementLabels,
} from "@/components/owner/branch-management-view";
import {
  OwnerBranchWorkspace,
  type BranchStats,
  type OwnerOnboardingLabels,
} from "@/components/owner/owner-branch-workspace";
import {
  getDashboard,
  getFileDownloadURL,
  listBranchProfileFiles,
  listBranches,
  listBranchStaff,
  listRooms,
} from "@/lib/api/client";
import type { Branch, StaffMember } from "@/lib/api/types";
import { requireAuthContext } from "@/lib/auth/session";
import { dashboardPathForRole, isRole } from "@/lib/navigation/roles";

type RoleDashboardPageProps = {
  params: Promise<{
    locale: string;
    role: string;
  }>;
  searchParams: Promise<{
    branch_id?: string;
    view?: string;
  }>;
};

type DashboardMessages = AppShellLabels & {
  dashboard: DashboardViewLabels;
  owner: {
    branchManagement: BranchManagementLabels;
    onboarding: OwnerOnboardingLabels;
  };
};

export const dynamic = "force-dynamic";

export default async function RoleDashboardPage({
  params,
  searchParams,
}: RoleDashboardPageProps) {
  const { locale, role } = await params;
  const { branch_id: branchID, view } = await searchParams;
  setRequestLocale(locale);
  if (!isRole(role)) {
    notFound();
  }

  const { token, session } = await requireAuthContext(locale);
  if (session.user.role !== role) {
    redirect(`/${locale}${dashboardPathForRole(session.user.role)}`);
  }

  const messages = (await getMessages()) as unknown as DashboardMessages;
  const shellLabels: AppShellLabels = {
    common: messages.common,
    shell: messages.shell,
  };

  if (role === "owner") {
    const branches = await Promise.all(
      (await listBranches(token)).map((branch) =>
        attachBranchPhoto(branch, token),
      ),
    );
    const branchStats = await Promise.all(
      branches.map(async (branch): Promise<BranchStats> => {
        const branchDashboard = await getDashboard("owner", token, branch.id);
        return {
          branch_id: branch.id,
          active_students: branchDashboard.summary?.active_students ?? 0,
          active_teachers: branchDashboard.summary?.active_teachers ?? 0,
        };
      }),
    );
    const selectedBranch = branchID
      ? branches.find((branch) => branch.id === branchID)
      : undefined;
    const allBranchesSelected = branchID === "all";
    const branchManagementSelected = view === "branches";

    if (branchManagementSelected) {
      const rooms = await listRooms(token);
      const staff = (
        await Promise.all(
          branches.map((branch) => listBranchStaff(branch.id, token)),
        )
      ).flat();
      const staffWithPhotos = await Promise.all(
        staff.map((member) => attachStaffPhoto(member, token)),
      );
      return (
        <AppShell labels={shellLabels} locale={locale} session={session}>
          <BranchManagementView
            branches={branches}
            labels={messages.owner.branchManagement}
            rooms={rooms}
            staff={staffWithPhotos}
            stats={branchStats}
          />
        </AppShell>
      );
    }

    if (!selectedBranch && !allBranchesSelected) {
      return (
        <OwnerBranchWorkspace
          branches={branches}
          labels={{
            common: messages.common,
            onboarding: messages.owner.onboarding,
          }}
          locale={locale}
          stats={branchStats}
          userName={`${session.user.first_name} ${session.user.last_name}`}
        />
      );
    }

    const dashboard = await getDashboard(
      "owner",
      token,
      selectedBranch?.id,
    );
    return (
      <AppShell
        labels={shellLabels}
        locale={locale}
        selectedBranch={selectedBranch}
        session={session}
      >
        <RoleDashboardView
          commonLabels={messages.common}
          activeBranchID={allBranchesSelected ? "all" : selectedBranch?.id}
          dashboard={dashboard}
          labels={messages.dashboard}
          locale={locale}
          ownerBranches={branches}
          role={role}
          session={
            selectedBranch ? { ...session, branch: selectedBranch } : session
          }
        />
      </AppShell>
    );
  }

  const dashboard = await getDashboard(role, token);

  return (
    <AppShell labels={shellLabels} locale={locale} session={session}>
      <RoleDashboardView
        commonLabels={messages.common}
        dashboard={dashboard}
        labels={messages.dashboard}
        locale={locale}
        role={role}
        session={session}
      />
    </AppShell>
  );
}

async function attachStaffPhoto(
  staff: StaffMember,
  token: string,
): Promise<StaffMember> {
  if (!staff.profile_photo_file_id) {
    return staff;
  }
  try {
    const download = await getFileDownloadURL(staff.profile_photo_file_id, token);
    return { ...staff, profile_photo_url: download.url };
  } catch {
    return staff;
  }
}

async function attachBranchPhoto(
  branch: Branch,
  token: string,
): Promise<Branch> {
  try {
    const files = await listBranchProfileFiles(branch.id, token);
    const file = files[0];
    if (!file) {
      return branch;
    }

    const download = await getFileDownloadURL(file.id, token);
    return { ...branch, photo_file_id: file.id, photo_url: download.url };
  } catch {
    return branch;
  }
}
