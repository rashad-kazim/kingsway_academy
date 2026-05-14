import { getMessages, setRequestLocale } from "next-intl/server";
import { notFound, redirect } from "next/navigation";
import {
  RoleDashboardView,
  type DashboardViewLabels,
} from "@/components/dashboard/role-dashboard-view";
import { AppShell, type AppShellLabels } from "@/components/layout/app-shell";
import {
  AddTeacherForm,
  type AddTeacherFormLabels,
  type TeacherFormInitialData,
} from "@/components/owner/add-teacher-form";
import {
  BranchManagementView,
  type BranchManagementLabels,
} from "@/components/owner/branch-management-view";
import {
  TeacherFinanceHrView,
  type TeacherFinanceHrLabels,
} from "@/components/owner/teacher-finance-hr-view";
import {
  ReceptionistManagementView,
  type ReceptionistManagementLabels,
} from "@/components/owner/receptionist-management-view";
import {
  StudentAssignmentHubView,
  type StudentAssignmentHubLabels,
} from "@/components/owner/student-assignment-hub-view";
import {
  AddStudentForm,
  type AddStudentFormLabels,
} from "@/components/owner/add-student-form";
import type {
  BranchStats,
  OwnerOnboardingLabels,
} from "@/components/owner/owner-branch-workspace";
import {
  getDashboard,
  getTeacher,
  getFileDownloadURL,
  listBranchProfileFiles,
  listBranches,
  listBranchStaff,
  listSalaryModels,
  listCourses,
  listStudentAssignmentHub,
  listTeacherFinanceRecords,
  listRooms,
} from "@/lib/api/client";
import type {
  Branch,
  SalaryModelType,
  StaffMember,
  StudentAssignmentHubPage,
  StudentAssignmentHubRecord,
  StudentStatus,
  TeacherFinanceRecord,
  TeacherStatus,
} from "@/lib/api/types";
import { requireAuthContext } from "@/lib/auth/session";
import { dashboardPathForRole, isRole } from "@/lib/navigation/roles";

type RoleDashboardPageProps = {
  params: Promise<{
    locale: string;
    role: string;
  }>;
  searchParams: Promise<{
    branch_id?: string;
    limit?: string;
    offset?: string;
    q?: string;
    salary_model?: string;
    status?: string;
    subject?: string;
    teacher_id?: string;
    view?: string;
  }>;
};

type DashboardMessages = AppShellLabels & {
  dashboard: DashboardViewLabels;
  owner: {
    branchManagement: BranchManagementLabels;
    onboarding: OwnerOnboardingLabels;
    receptionistManagement: ReceptionistManagementLabels;
    studentAssignmentHub: StudentAssignmentHubLabels & AddStudentFormLabels;
    teacherFinanceHr: TeacherFinanceHrLabels & AddTeacherFormLabels;
  };
};

export const dynamic = "force-dynamic";

export default async function RoleDashboardPage({
  params,
  searchParams,
}: RoleDashboardPageProps) {
  const { locale, role } = await params;
  const {
    branch_id: branchID,
    limit,
    offset,
    q,
    salary_model: salaryModel,
    status,
    subject,
    teacher_id: teacherID,
    view,
  } = await searchParams;
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
    const branches = await listBranches(token);
    const branchManagementSelected = view === "branches";
    const addTeacherSelected = view === "teacher-add";
    const editTeacherSelected = view === "teacher-edit" && teacherID;
    const receptionistManagementSelected = view === "receptionists";
    const studentAssignmentHubSelected = view === "student-assignment";
    const addStudentSelected = view === "student-add";
    const teacherFinanceSelected = view === "teacher-finance";

    if (branchManagementSelected) {
      const [branchesWithPhotos, branchStats, rooms, staffByBranch] =
        await Promise.all([
          attachBranchPhotos(branches, token),
          loadBranchStats(branches, token),
          listRooms(token),
          Promise.all(
            branches.map((branch) => listBranchStaff(branch.id, token)),
          ),
        ]);
      const staff = staffByBranch.flat();
      const staffWithPhotos = await Promise.all(
        staff.map((member) => attachStaffPhoto(member, token)),
      );
      return (
        <AppShell labels={shellLabels} locale={locale} session={session}>
          <BranchManagementView
            branches={branchesWithPhotos}
            labels={messages.owner.branchManagement}
            rooms={rooms}
            staff={staffWithPhotos}
            stats={branchStats}
          />
        </AppShell>
      );
    }

    if (addTeacherSelected) {
      return (
        <AppShell labels={shellLabels} locale={locale} session={session}>
          <AddTeacherForm
            branches={branches}
            labels={messages.owner.teacherFinanceHr}
            locale={locale}
          />
        </AppShell>
      );
    }

    if (editTeacherSelected) {
      const initialTeacher = await buildTeacherInitialData(teacherID, token);
      return (
        <AppShell labels={shellLabels} locale={locale} session={session}>
          <AddTeacherForm
            branches={branches}
            initialTeacher={initialTeacher}
            labels={messages.owner.teacherFinanceHr}
            locale={locale}
          />
        </AppShell>
      );
    }

    if (teacherFinanceSelected) {
      const teacherBranchID =
        branchID && branchID !== "all" ? branchID : undefined;
      const filters = {
        branch_id: teacherBranchID,
        subject,
        status: parseTeacherStatus(status),
        salary_model: parseSalaryModel(salaryModel),
      };
      const records = await listTeacherFinanceRecords(filters, token);
      const recordsWithPhotos = await Promise.all(
        records.map((record) => attachTeacherFinancePhoto(record, token)),
      );
      return (
        <AppShell labels={shellLabels} locale={locale} session={session}>
          <TeacherFinanceHrView
            branches={branches}
            filters={filters}
            labels={messages.owner.teacherFinanceHr}
            locale={locale}
            records={recordsWithPhotos}
          />
        </AppShell>
      );
    }

    if (receptionistManagementSelected) {
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
          <ReceptionistManagementView
            branches={branches}
            labels={messages.owner.receptionistManagement}
            staff={staffWithPhotos}
          />
        </AppShell>
      );
    }

    if (addStudentSelected) {
      const [courses, teacherRecords] = await Promise.all([
        listCourses(token),
        listTeacherFinanceRecords({}, token),
      ]);
      return (
        <AppShell labels={shellLabels} locale={locale} session={session}>
          <AddStudentForm
            branches={branches}
            courses={courses}
            labels={messages.owner.studentAssignmentHub}
            locale={locale}
            teachers={teacherRecords}
          />
        </AppShell>
      );
    }

    if (studentAssignmentHubSelected) {
      const pageSize = parsePositiveInt(limit, 20);
      const pageOffset = parseNonNegativeInt(offset, 0);
      const filters = {
        branch_id: branchID && branchID !== "all" ? branchID : undefined,
        limit: pageSize,
        offset: pageOffset,
        q,
        status: parseStudentStatus(status),
        teacher_id: teacherID,
      };
      const [studentPage, teacherRecords] = await Promise.all([
        listStudentAssignmentHub(filters, token),
        listTeacherFinanceRecords({}, token),
      ]);
      const studentPageWithPhotos = await attachStudentHubPhotos(studentPage, token);
      return (
        <AppShell labels={shellLabels} locale={locale} session={session}>
          <StudentAssignmentHubView
            branches={branches}
            filters={filters}
            initialPage={studentPageWithPhotos}
            labels={messages.owner.studentAssignmentHub}
            locale={locale}
            teachers={teacherRecords}
          />
        </AppShell>
      );
    }

    const selectedBranchBase = branchID
      ? branches.find((branch) => branch.id === branchID)
      : undefined;
    const [branchesWithPhotos, dashboard] = await Promise.all([
      attachBranchPhotos(branches, token),
      getDashboard("owner", token, selectedBranchBase?.id),
    ]);
    const selectedBranch = selectedBranchBase
      ? branchesWithPhotos.find((branch) => branch.id === selectedBranchBase.id)
      : undefined;
    return (
      <AppShell
        labels={shellLabels}
        locale={locale}
        selectedBranch={selectedBranch}
        session={session}
      >
        <RoleDashboardView
          commonLabels={messages.common}
          activeBranchID={selectedBranch?.id ?? "all"}
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

function parseTeacherStatus(value?: string): TeacherStatus | undefined {
  if (
    value === "pending_owner_approval" ||
    value === "active" ||
    value === "terminated"
  ) {
    return value;
  }

  return undefined;
}

function parseStudentStatus(value?: string): StudentStatus | undefined {
  if (value === "active" || value === "left" || value === "graduated") {
    return value;
  }

  return undefined;
}

function parseSalaryModel(value?: string): SalaryModelType | undefined {
  if (value === "fixed" || value === "percent" || value === "hybrid") {
    return value;
  }

  return undefined;
}

function parsePositiveInt(value: string | undefined, fallback: number) {
  const parsed = Number.parseInt(value ?? "", 10);
  if (!Number.isFinite(parsed) || parsed < 1) {
    return fallback;
  }
  return parsed;
}

function parseNonNegativeInt(value: string | undefined, fallback: number) {
  const parsed = Number.parseInt(value ?? "", 10);
  if (!Number.isFinite(parsed) || parsed < 0) {
    return fallback;
  }
  return parsed;
}

async function buildTeacherInitialData(
  teacherID: string,
  token: string,
): Promise<TeacherFormInitialData> {
  const teacher = await getTeacher(teacherID, token);
  const [records, salaryModels] = await Promise.all([
    listTeacherFinanceRecords({ branch_id: teacher.branch_id }, token),
    listSalaryModels(token, teacher.branch_id),
  ]);
  const record = records.find((item) => item.id === teacher.id);
  if (!record) {
    notFound();
  }
  const salaryModel = salaryModels.find(
    (model) => model.teacher_id === teacher.id,
  );
  const photoURL = teacher.profile_photo_file_id
    ? await getTeacherPhotoURL(teacher.profile_photo_file_id, token)
    : "";

  return {
    address: teacher.address,
    birth_date: teacher.birth_date,
    branch_id: teacher.branch_id,
    email: record.email,
    first_name: record.first_name,
    gender: teacher.gender,
    id: teacher.id,
    last_name: record.last_name,
    percentage:
      salaryModel?.student_percent_basis_points !== undefined
        ? formatPercentForInput(salaryModel.student_percent_basis_points)
        : "",
    phone: teacher.phone,
    profile_photo_file_id: teacher.profile_photo_file_id,
    profile_photo_url: photoURL,
    salary_amount:
      salaryModel?.fixed_monthly_amount_cents !== undefined
        ? formatCentsForInput(salaryModel.fixed_monthly_amount_cents)
        : "",
    salary_model: salaryModel?.model_type ?? record.salary_type ?? "fixed",
    subjects: splitSubjects(record.subject),
  };
}

async function getTeacherPhotoURL(fileID: string, token: string) {
  try {
    const download = await getFileDownloadURL(fileID, token);
    return download.url;
  } catch {
    return "";
  }
}

function splitSubjects(value?: string) {
  return (value ?? "")
    .split(",")
    .map((subject) => subject.trim())
    .filter(Boolean);
}

function formatCentsForInput(value: number) {
  const amount = value / 100;
  return Number.isInteger(amount) ? String(amount) : amount.toFixed(2);
}

function formatPercentForInput(value: number) {
  const amount = value / 100;
  return Number.isInteger(amount) ? String(amount) : amount.toFixed(2);
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

async function attachTeacherFinancePhoto(
  record: TeacherFinanceRecord,
  token: string,
): Promise<TeacherFinanceRecord> {
  if (!record.profile_photo_file_id) {
    return record;
  }
  try {
    const download = await getFileDownloadURL(record.profile_photo_file_id, token);
    return { ...record, profile_photo_url: download.url };
  } catch {
    return record;
  }
}

async function attachStudentHubPhotos(
  page: StudentAssignmentHubPage,
  token: string,
): Promise<StudentAssignmentHubPage> {
  const items = await Promise.all(
    page.items.map((student) => attachStudentHubPhoto(student, token)),
  );
  return { ...page, items };
}

async function attachStudentHubPhoto(
  student: StudentAssignmentHubRecord,
  token: string,
): Promise<StudentAssignmentHubRecord> {
  if (!student.profile_photo_file_id) {
    return student;
  }
  try {
    const download = await getFileDownloadURL(student.profile_photo_file_id, token);
    return { ...student, profile_photo_url: download.url };
  } catch {
    return student;
  }
}

function attachBranchPhotos(branches: Branch[], token: string) {
  return Promise.all(branches.map((branch) => attachBranchPhoto(branch, token)));
}

function loadBranchStats(
  branches: Branch[],
  token: string,
): Promise<BranchStats[]> {
  return Promise.all(
    branches.map(async (branch): Promise<BranchStats> => {
      const branchDashboard = await getDashboard("owner", token, branch.id);
      return {
        branch_id: branch.id,
        active_students: branchDashboard.summary?.active_students ?? 0,
        active_teachers: branchDashboard.summary?.active_teachers ?? 0,
      };
    }),
  );
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
