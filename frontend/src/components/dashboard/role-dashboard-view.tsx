import {
  AlertTriangle,
  Banknote,
  BookOpen,
  CalendarDays,
  ClipboardList,
  FileText,
  GraduationCap,
  Users,
  UserRoundCheck,
  type LucideIcon,
} from "lucide-react";
import Link from "next/link";
import type { ReactNode } from "react";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { Branch, DashboardRecord, Role, Session } from "@/lib/api/types";
import type { ChromeLabels } from "@/components/layout/topbar-controls";
import { cn } from "@/lib/utils";
import { MetricCard } from "./metric-card";

export type DashboardViewLabels = {
  titleSuffix: string;
  capabilities: string;
  tableName: string;
  tableStatus: string;
  tableID: string;
  noRecords: string;
  fallbackRecord: string;
  metrics: {
    activeStudents: string;
    activeTeachers: string;
    activeClasses: string;
    upcomingSchedule: string;
    pendingPayments: string;
    writingFilesRetained: string;
    classes: string;
    schedule: string;
    assignments: string;
    exams: string;
    results: string;
  };
  queues: {
    branches: string;
    availableBranchRecords: string;
    schedule: string;
    upcomingBranchSchedule: string;
    payments: string;
    recentBranchPayments: string;
    classes: string;
    assignedClasses: string;
    assignments: string;
    classAssignments: string;
    assignedWork: string;
    examResults: string;
    recordedResults: string;
  };
  ownerGlobal: {
    allBranches: string;
    activeStudents: string;
    activeTeachers: string;
    activeClasses: string;
    debtTracker: string;
    totalTurnover: string;
    courseDistribution: string;
    courseDistributionDescription: string;
    noCourseData: string;
    urgentTasks: string;
    urgentTasksDescription: string;
    salaryApprovals: string;
    salaryApprovalsDescription: string;
    lowPerformance: string;
    lowPerformanceDescription: string;
    churnAnalysis: string;
    churnAnalysisDescription: string;
    clear: string;
    courses: {
      ielts: string;
      sat: string;
      physics: string;
      other: string;
    };
  };
};

type RoleDashboardViewProps = {
  activeBranchID?: string;
  ownerBranches?: Branch[];
  commonLabels: ChromeLabels;
  dashboard: DashboardRecord;
  labels: DashboardViewLabels;
  locale: string;
  role: Role;
  session: Session;
};

type Metric = {
  label: string;
  value: number | string;
  icon: LucideIcon;
};

export function RoleDashboardView({
  activeBranchID,
  commonLabels,
  labels,
  locale,
  ownerBranches = [],
  role,
  session,
  dashboard,
}: RoleDashboardViewProps) {
  const roleName = commonLabels.roles[role];

  if (role === "owner") {
    return (
      <OwnerGlobalDashboard
        activeBranchID={activeBranchID}
        branches={ownerBranches}
        dashboard={dashboard}
        labels={labels}
        locale={locale}
      />
    );
  }

  const metrics = metricsFor(role, dashboard, labels);
  const queues = queuesFor(role, dashboard, labels);

  return (
    <>
      <section className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div className="min-w-0">
          <div className="mb-2 flex flex-wrap items-center gap-2">
            <Badge variant="secondary">{roleName}</Badge>
            {session.branch?.name ? (
              <Badge variant="outline">{session.branch.name}</Badge>
            ) : null}
          </div>
          <h1 className="text-2xl font-semibold tracking-tight">
            {roleName} {labels.titleSuffix}
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            {session.user.first_name} {session.user.last_name}
          </p>
        </div>
        <div className="rounded-lg border px-3 py-2 text-sm">
          <span className="text-muted-foreground">{labels.capabilities}</span>
          <span className="ml-2 font-medium tabular-nums">
            {session.capabilities.length}
          </span>
        </div>
      </section>

      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
        {metrics.map((metric) => (
          <MetricCard
            key={metric.label}
            icon={metric.icon}
            label={metric.label}
            value={metric.value}
          />
        ))}
      </section>

      <section className="grid gap-4 xl:grid-cols-2">
        {queues.map((queue) => (
          <DataPreview
            key={queue.title}
            description={queue.description}
            items={queue.items}
            labels={labels}
            title={queue.title}
          />
        ))}
      </section>
    </>
  );
}

function OwnerGlobalDashboard({
  activeBranchID,
  branches,
  dashboard,
  labels,
  locale,
}: {
  activeBranchID?: string;
  branches: Branch[];
  dashboard: DashboardRecord;
  labels: DashboardViewLabels;
  locale: string;
}) {
  const metrics = ownerMetricsFor(dashboard, labels);

  return (
    <div className="space-y-6">
      <BranchScopeFilter
        activeBranchID={activeBranchID}
        branches={branches}
        labels={labels}
        locale={locale}
      />

      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
        {metrics.map((metric) => (
          <MetricCard
            key={metric.label}
            icon={metric.icon}
            label={metric.label}
            value={metric.value}
          />
        ))}
      </section>

      <CourseDistribution labels={labels} />

      <UrgentTasks labels={labels} />
    </div>
  );
}

function BranchScopeFilter({
  activeBranchID,
  branches,
  labels,
  locale,
}: {
  activeBranchID?: string;
  branches: Branch[];
  labels: DashboardViewLabels;
  locale: string;
}) {
  const allActive = activeBranchID === "all" || !activeBranchID;

  return (
    <section className="flex flex-wrap items-center gap-3">
      <ScopeButton
        active={allActive}
        href={`/${locale}/dashboard/owner?branch_id=all`}
      >
        {labels.ownerGlobal.allBranches}
      </ScopeButton>
      {branches.map((branch) => (
        <ScopeButton
          active={activeBranchID === branch.id}
          href={`/${locale}/dashboard/owner?branch_id=${branch.id}`}
          key={branch.id}
        >
          {branch.name}
        </ScopeButton>
      ))}
    </section>
  );
}

function ScopeButton({
  active,
  children,
  href,
}: {
  active: boolean;
  children: ReactNode;
  href: string;
}) {
  return (
    <Link
      className={cn(
        "inline-flex min-h-10 cursor-pointer items-center rounded-full border border-[#dce3ee] bg-white px-5 text-sm font-bold text-[#0a284b] shadow-sm transition-all hover:-translate-y-0.5 hover:border-[#ef2334]/40 hover:text-[#ef2334] dark:border-[#3a4658] dark:bg-[#17243a] dark:text-[#f3f6fa] dark:hover:border-[#ff3b4f]/50 dark:hover:text-[#ff5a69]",
        active &&
          "border-[#ef2334] bg-[#ef2334] text-white shadow-[0_0_22px_rgba(239,35,52,0.35)] hover:text-white dark:border-[#ff3b4f] dark:bg-[#ff3b4f] dark:text-white dark:shadow-[0_0_24px_rgba(255,59,79,0.35)] dark:hover:text-white",
      )}
      href={href}
    >
      {children}
    </Link>
  );
}

function ownerMetricsFor(
  dashboard: DashboardRecord,
  labels: DashboardViewLabels,
): Metric[] {
  const summary = dashboard.summary;

  return [
    {
      label: labels.ownerGlobal.activeStudents,
      value: summary?.active_students ?? 0,
      icon: Users,
    },
    {
      label: labels.ownerGlobal.activeTeachers,
      value: summary?.active_teachers ?? 0,
      icon: UserRoundCheck,
    },
    {
      label: labels.ownerGlobal.activeClasses,
      value: summary?.active_classes ?? 0,
      icon: GraduationCap,
    },
    {
      label: labels.ownerGlobal.debtTracker,
      value: formatManat(0),
      icon: AlertTriangle,
    },
    {
      label: labels.ownerGlobal.totalTurnover,
      value: formatManat(0),
      icon: Banknote,
    },
  ];
}

function CourseDistribution({ labels }: { labels: DashboardViewLabels }) {
  const courses = [
    { label: labels.ownerGlobal.courses.ielts, value: 0, color: "#ff3b4f" },
    { label: labels.ownerGlobal.courses.sat, value: 0, color: "#306186" },
    { label: labels.ownerGlobal.courses.physics, value: 0, color: "#22c55e" },
    { label: labels.ownerGlobal.courses.other, value: 0, color: "#f59e0b" },
  ];
  const total = courses.reduce((sum, course) => sum + course.value, 0);

  return (
    <Card className="rounded-lg dark:border-[#3a4658] dark:bg-[linear-gradient(180deg,#1e2a39_0%,#182331_100%)]">
      <CardHeader>
        <CardTitle>{labels.ownerGlobal.courseDistribution}</CardTitle>
        <CardDescription>
          {labels.ownerGlobal.courseDistributionDescription}
        </CardDescription>
      </CardHeader>
      <CardContent className="grid items-center gap-8 lg:grid-cols-[minmax(240px,0.8fr)_minmax(320px,1fr)]">
        <div className="flex justify-center">
          <div
            className="relative flex size-64 items-center justify-center rounded-full bg-[#eef2f7] shadow-inner dark:bg-[#2a3444]"
            style={{
              background:
                total > 0
                  ? "conic-gradient(#ff3b4f 0 45%, #306186 45% 75%, #22c55e 75% 90%, #f59e0b 90% 100%)"
                  : undefined,
            }}
          >
            <div className="flex size-32 items-center justify-center rounded-full bg-white text-center text-sm font-bold text-[#59667a] shadow-sm dark:bg-[#1b2635] dark:text-[#a7b0bf]">
              {total > 0 ? `${total}` : labels.ownerGlobal.noCourseData}
            </div>
          </div>
        </div>
        <div className="grid gap-3">
          {courses.map((course) => (
            <div
              className="flex items-center justify-between rounded-lg border border-[#dce3ee] bg-white/70 px-4 py-3 text-sm dark:border-[#3a4658] dark:bg-[#202b3a]/70"
              key={course.label}
            >
              <div className="flex items-center gap-3">
                <span
                  className="size-3 rounded-full"
                  style={{ backgroundColor: course.color }}
                />
                <span className="font-semibold">{course.label}</span>
              </div>
              <span className="font-bold tabular-nums text-[#59667a] dark:text-[#a7b0bf]">
                {course.value}%
              </span>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}

function UrgentTasks({ labels }: { labels: DashboardViewLabels }) {
  const tasks = [
    {
      title: labels.ownerGlobal.salaryApprovals,
      description: labels.ownerGlobal.salaryApprovalsDescription,
    },
    {
      title: labels.ownerGlobal.lowPerformance,
      description: labels.ownerGlobal.lowPerformanceDescription,
    },
    {
      title: labels.ownerGlobal.churnAnalysis,
      description: labels.ownerGlobal.churnAnalysisDescription,
    },
  ];

  return (
    <Card className="rounded-lg dark:border-[#3a4658] dark:bg-[linear-gradient(180deg,#1e2a39_0%,#182331_100%)]">
      <CardHeader>
        <CardTitle>{labels.ownerGlobal.urgentTasks}</CardTitle>
        <CardDescription>{labels.ownerGlobal.urgentTasksDescription}</CardDescription>
      </CardHeader>
      <CardContent className="grid gap-3">
        {tasks.map((task) => (
          <div
            className="flex items-center justify-between gap-4 rounded-lg border border-[#dce3ee] bg-white/70 px-4 py-4 dark:border-[#3a4658] dark:bg-[#202b3a]/70"
            key={task.title}
          >
            <div>
              <div className="font-bold">{task.title}</div>
              <p className="mt-1 text-sm text-[#59667a] dark:text-[#a7b0bf]">
                {task.description}
              </p>
            </div>
            <Badge className="shrink-0 bg-emerald-600 text-white hover:bg-emerald-600">
              {labels.ownerGlobal.clear}
            </Badge>
          </div>
        ))}
      </CardContent>
    </Card>
  );
}

function metricsFor(
  role: Role,
  dashboard: DashboardRecord,
  labels: DashboardViewLabels,
): Metric[] {
  if (dashboard.summary) {
    return [
      {
        label: labels.metrics.activeStudents,
        value: dashboard.summary.active_students,
        icon: Users,
      },
      {
        label: labels.metrics.activeTeachers,
        value: dashboard.summary.active_teachers,
        icon: UserRoundCheck,
      },
      {
        label: labels.metrics.activeClasses,
        value: dashboard.summary.active_classes,
        icon: GraduationCap,
      },
      {
        label: labels.metrics.upcomingSchedule,
        value: dashboard.summary.upcoming_schedule,
        icon: CalendarDays,
      },
      {
        label: labels.metrics.pendingPayments,
        value: dashboard.summary.pending_payments,
        icon: Banknote,
      },
      {
        label: labels.metrics.writingFilesRetained,
        value: dashboard.summary.writing_files_retained,
        icon: FileText,
      },
    ];
  }

  if (role === "teacher") {
    return [
      {
        label: labels.metrics.classes,
        value: count(dashboard.classes),
        icon: GraduationCap,
      },
      {
        label: labels.metrics.schedule,
        value: count(dashboard.schedule),
        icon: CalendarDays,
      },
      {
        label: labels.metrics.assignments,
        value: count(dashboard.assignments),
        icon: ClipboardList,
      },
    ];
  }

  return [
    {
      label: labels.metrics.classes,
      value: count(dashboard.classes),
      icon: GraduationCap,
    },
    {
      label: labels.metrics.assignments,
      value: count(dashboard.assignments),
      icon: ClipboardList,
    },
    { label: labels.metrics.exams, value: count(dashboard.exams), icon: BookOpen },
    {
      label: labels.metrics.results,
      value: count(dashboard.results),
      icon: FileText,
    },
  ];
}

function queuesFor(
  role: Role,
  dashboard: DashboardRecord,
  labels: DashboardViewLabels,
) {
  if (role === "owner") {
    return [
      {
        title: labels.queues.branches,
        description: labels.queues.availableBranchRecords,
        items: dashboard.branches ?? [],
      },
    ];
  }
  if (role === "receptionist") {
    return [
      {
        title: labels.queues.schedule,
        description: labels.queues.upcomingBranchSchedule,
        items: dashboard.schedule ?? [],
      },
      {
        title: labels.queues.payments,
        description: labels.queues.recentBranchPayments,
        items: dashboard.payments ?? [],
      },
    ];
  }
  if (role === "teacher") {
    return [
      {
        title: labels.queues.classes,
        description: labels.queues.assignedClasses,
        items: dashboard.classes ?? [],
      },
      {
        title: labels.queues.assignments,
        description: labels.queues.classAssignments,
        items: dashboard.assignments ?? [],
      },
    ];
  }

  return [
    {
      title: labels.queues.assignments,
      description: labels.queues.assignedWork,
      items: dashboard.assignments ?? [],
    },
    {
      title: labels.queues.examResults,
      description: labels.queues.recordedResults,
      items: dashboard.results ?? [],
    },
  ];
}

function DataPreview({
  description,
  items,
  labels,
  title,
}: {
  description: string;
  items: Record<string, unknown>[];
  labels: DashboardViewLabels;
  title: string;
}) {
  const visible = items.slice(0, 5);

  return (
    <Card className="rounded-lg">
      <CardHeader>
        <CardTitle>{title}</CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <CardContent>
        {visible.length > 0 ? (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{labels.tableName}</TableHead>
                <TableHead>{labels.tableStatus}</TableHead>
                <TableHead className="text-right">{labels.tableID}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {visible.map((item, index) => (
                <TableRow key={String(item.id ?? index)}>
                  <TableCell className="font-medium">
                    {displayName(item, labels)}
                  </TableCell>
                  <TableCell>{displayStatus(item)}</TableCell>
                  <TableCell className="text-right font-mono text-xs text-muted-foreground">
                    {shortID(item.id)}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        ) : (
          <div className="rounded-lg border border-dashed px-3 py-8 text-center text-sm text-muted-foreground">
            {labels.noRecords}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function count(items: unknown[] | undefined) {
  return items?.length ?? 0;
}

function displayName(item: Record<string, unknown>, labels: DashboardViewLabels) {
  for (const key of ["name", "title", "first_name", "type"]) {
    const value = item[key];
    if (typeof value === "string" && value.length > 0) {
      return value;
    }
  }

  return labels.fallbackRecord;
}

function displayStatus(item: Record<string, unknown>) {
  for (const key of ["status", "item_type", "role"]) {
    const value = item[key];
    if (typeof value === "string" && value.length > 0) {
      return value;
    }
  }

  return "-";
}

function shortID(value: unknown) {
  if (typeof value !== "string" || value.length === 0) {
    return "-";
  }

  return value.length > 8 ? value.slice(0, 8) : value;
}

function formatManat(value: number) {
  const formatted = new Intl.NumberFormat("en-US", {
    maximumFractionDigits: 0,
  }).format(value);
  return `₼ ${formatted}`;
}
