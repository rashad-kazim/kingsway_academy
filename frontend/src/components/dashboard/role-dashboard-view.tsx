import {
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
import type { DashboardRecord, Role, Session } from "@/lib/api/types";
import { roleLabel } from "@/lib/navigation/roles";
import { MetricCard } from "./metric-card";

type RoleDashboardViewProps = {
  role: Role;
  session: Session;
  dashboard: DashboardRecord;
};

type Metric = {
  label: string;
  value: number | string;
  icon: LucideIcon;
};

export function RoleDashboardView({
  role,
  session,
  dashboard,
}: RoleDashboardViewProps) {
  const metrics = metricsFor(role, dashboard);
  const queues = queuesFor(role, dashboard);

  return (
    <>
      <section className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div className="min-w-0">
          <div className="mb-2 flex flex-wrap items-center gap-2">
            <Badge variant="secondary">{roleLabel(role)}</Badge>
            {session.branch?.name ? (
              <Badge variant="outline">{session.branch.name}</Badge>
            ) : null}
          </div>
          <h1 className="text-2xl font-semibold tracking-tight">
            {roleLabel(role)} dashboard
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            {session.user.first_name} {session.user.last_name}
          </p>
        </div>
        <div className="rounded-lg border px-3 py-2 text-sm">
          <span className="text-muted-foreground">Capabilities</span>
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
            title={queue.title}
          />
        ))}
      </section>
    </>
  );
}

function metricsFor(role: Role, dashboard: DashboardRecord): Metric[] {
  if (dashboard.summary) {
    return [
      {
        label: "Active students",
        value: dashboard.summary.active_students,
        icon: Users,
      },
      {
        label: "Active teachers",
        value: dashboard.summary.active_teachers,
        icon: UserRoundCheck,
      },
      {
        label: "Active classes",
        value: dashboard.summary.active_classes,
        icon: GraduationCap,
      },
      {
        label: "Upcoming schedule",
        value: dashboard.summary.upcoming_schedule,
        icon: CalendarDays,
      },
      {
        label: "Pending payments",
        value: dashboard.summary.pending_payments,
        icon: Banknote,
      },
      {
        label: "Writing files retained",
        value: dashboard.summary.writing_files_retained,
        icon: FileText,
      },
    ];
  }

  if (role === "teacher") {
    return [
      { label: "Classes", value: count(dashboard.classes), icon: GraduationCap },
      { label: "Schedule", value: count(dashboard.schedule), icon: CalendarDays },
      {
        label: "Assignments",
        value: count(dashboard.assignments),
        icon: ClipboardList,
      },
    ];
  }

  return [
    { label: "Classes", value: count(dashboard.classes), icon: GraduationCap },
    {
      label: "Assignments",
      value: count(dashboard.assignments),
      icon: ClipboardList,
    },
    { label: "Exams", value: count(dashboard.exams), icon: BookOpen },
    { label: "Results", value: count(dashboard.results), icon: FileText },
  ];
}

function queuesFor(role: Role, dashboard: DashboardRecord) {
  if (role === "owner") {
    return [
      {
        title: "Branches",
        description: "Available branch records",
        items: dashboard.branches ?? [],
      },
    ];
  }
  if (role === "receptionist") {
    return [
      {
        title: "Schedule",
        description: "Upcoming branch schedule",
        items: dashboard.schedule ?? [],
      },
      {
        title: "Payments",
        description: "Recent branch payments",
        items: dashboard.payments ?? [],
      },
    ];
  }
  if (role === "teacher") {
    return [
      {
        title: "Classes",
        description: "Assigned classes",
        items: dashboard.classes ?? [],
      },
      {
        title: "Assignments",
        description: "Class assignments",
        items: dashboard.assignments ?? [],
      },
    ];
  }

  return [
    {
      title: "Assignments",
      description: "Assigned work",
      items: dashboard.assignments ?? [],
    },
    {
      title: "Exam results",
      description: "Recorded results",
      items: dashboard.results ?? [],
    },
  ];
}

function DataPreview({
  description,
  items,
  title,
}: {
  description: string;
  items: Record<string, unknown>[];
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
                <TableHead>Name</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">ID</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {visible.map((item, index) => (
                <TableRow key={String(item.id ?? index)}>
                  <TableCell className="font-medium">
                    {displayName(item)}
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
            No records
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function count(items: unknown[] | undefined) {
  return items?.length ?? 0;
}

function displayName(item: Record<string, unknown>) {
  for (const key of ["name", "title", "first_name", "type"]) {
    const value = item[key];
    if (typeof value === "string" && value.length > 0) {
      return value;
    }
  }

  return "Record";
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
