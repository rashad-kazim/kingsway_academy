"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import type { ReactNode } from "react";
import Link from "next/link";
import {
  ArrowLeftRight,
  ChevronLeft,
  ChevronRight,
  Edit3,
  Loader2,
  Search,
  UserX,
  X,
  Plus,
} from "lucide-react";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type {
  Branch,
  StudentAssignmentHubFilters,
  StudentAssignmentHubPage,
  StudentAssignmentHubRecord,
  StudentStatus,
  TeacherFinanceRecord,
} from "@/lib/api/types";
import { fetchStudentAssignmentHubAction } from "@/lib/student-assignment/actions";
import { cn } from "@/lib/utils";

export type StudentAssignmentHubLabels = {
  title: string;
  description: string;
  addStudent: string;
  branch: string;
  status: string;
  teacher: string;
  search: string;
  allBranches: string;
  allStatuses: string;
  allTeachers: string;
  resetFilters: string;
  removeFilter: string;
  profile: string;
  fin: string;
  activeTeacher: string;
  notAssigned: string;
  registeredDate: string;
  actions: string;
  edit: string;
  quickAssignSwap: string;
  noStudents: string;
  loading: string;
  rowsPerPage: string;
  showing: string;
  of: string;
  statuses: Record<StudentStatus, string>;
};

type TeacherOption = {
  id: string;
  name: string;
};

type StudentAssignmentHubViewProps = {
  branches: Branch[];
  filters: StudentAssignmentHubFilters;
  initialPage: StudentAssignmentHubPage;
  labels: StudentAssignmentHubLabels;
  locale: string;
  teachers: TeacherFinanceRecord[];
};

const pageSizes = [20, 30, 50];

export function StudentAssignmentHubView({
  branches,
  filters,
  initialPage,
  labels,
  locale,
  teachers,
}: StudentAssignmentHubViewProps) {
  const requestIDRef = useRef(0);
  const [pageData, setPageData] = useState(initialPage);
  const [activeFilters, setActiveFilters] =
    useState<StudentAssignmentHubFilters>({
      branch_id: filters.branch_id,
      limit: filters.limit ?? 20,
      offset: filters.offset ?? 0,
      q: filters.q ?? "",
      status: filters.status,
      teacher_id: filters.teacher_id,
    });
  const [searchValue, setSearchValue] = useState(filters.q ?? "");
  const [loading, setLoading] = useState(false);
  const teacherOptions = useMemo(() => buildTeacherOptions(teachers), [teachers]);
  const currentPage =
    Math.floor((activeFilters.offset ?? 0) / (activeFilters.limit ?? 20)) + 1;
  const totalPages = Math.max(
    1,
    Math.ceil(pageData.total / (activeFilters.limit ?? 20)),
  );
  const chips = useMemo(
    () => buildFilterChips(activeFilters, branches, teacherOptions, labels),
    [activeFilters, branches, labels, teacherOptions],
  );

  useEffect(() => {
    const timeout = window.setTimeout(() => {
      if ((activeFilters.q ?? "") === searchValue) {
        return;
      }
      void updateFilters({ q: searchValue, offset: 0 });
    }, 250);

    return () => window.clearTimeout(timeout);
    // updateFilters intentionally reads the latest filter object from this render.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeFilters.q, searchValue]);

  async function load(nextFilters: StudentAssignmentHubFilters) {
    const requestID = requestIDRef.current + 1;
    requestIDRef.current = requestID;
    setLoading(true);
    const result = await fetchStudentAssignmentHubAction(nextFilters);
    if (requestIDRef.current !== requestID) {
      return;
    }
    if ("data" in result) {
      setPageData(result.data);
    }
    setLoading(false);
  }

  async function updateFilters(patch: StudentAssignmentHubFilters) {
    const nextFilters = {
      ...activeFilters,
      ...patch,
    };
    setActiveFilters(nextFilters);
    await load(nextFilters);
  }

  function resetFilters() {
    const nextFilters = {
      limit: activeFilters.limit,
      offset: 0,
      q: "",
    };
    setSearchValue("");
    setActiveFilters(nextFilters);
    void load(nextFilters);
  }

  function removeFilter(key: keyof StudentAssignmentHubFilters) {
    if (key === "q") {
      setSearchValue("");
    }
    void updateFilters({ [key]: "", offset: 0 });
  }

  return (
    <div className="space-y-6">
      <section className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-3xl font-black tracking-tight">{labels.title}</h1>
          <p className="mt-1 text-sm text-[#59667a] dark:text-[#a7b0bf]">
            {labels.description}
          </p>
        </div>
        <Button
          asChild
          className="h-[38px] gap-2 rounded-lg bg-[#ef2334] px-5 font-bold text-white shadow-[0_12px_26px_rgba(239,35,52,0.28)] transition hover:bg-[#d91f30] dark:bg-[#ff3b4f] dark:hover:bg-[#ff5a69]"
        >
          <Link href={`/${locale}/dashboard/owner?view=student-add`}>
            <Plus className="size-4" />
            {labels.addStudent}
          </Link>
        </Button>
      </section>

      <Card className="rounded-xl border-white/55 bg-white/65 shadow-[0_24px_70px_rgba(10,40,75,0.12)] backdrop-blur-xl dark:border-white/10 dark:bg-[#1b2635]/62 dark:shadow-[0_24px_80px_rgba(0,0,0,0.32)]">
        <CardContent className="space-y-4 p-4">
          <div className="grid gap-3 md:grid-cols-[repeat(3,minmax(0,1fr))_minmax(260px,1.4fr)_auto] md:items-end">
            <FilterSelect
              label={labels.branch}
              value={activeFilters.branch_id ?? ""}
              onChange={(value) =>
                void updateFilters({ branch_id: value, offset: 0 })
              }
            >
              <option value="">{labels.allBranches}</option>
              {branches.map((branch) => (
                <option key={branch.id} value={branch.id}>
                  {branch.name}
                </option>
              ))}
            </FilterSelect>
            <FilterSelect
              label={labels.status}
              value={activeFilters.status ?? ""}
              onChange={(value) =>
                void updateFilters({
                  offset: 0,
                  status: value ? (value as StudentStatus) : undefined,
                })
              }
            >
              <option value="">{labels.allStatuses}</option>
              {studentStatuses.map((status) => (
                <option key={status} value={status}>
                  {labels.statuses[status]}
                </option>
              ))}
            </FilterSelect>
            <FilterSelect
              label={labels.teacher}
              value={activeFilters.teacher_id ?? ""}
              onChange={(value) =>
                void updateFilters({ offset: 0, teacher_id: value })
              }
            >
              <option value="">{labels.allTeachers}</option>
              {teacherOptions.map((teacher) => (
                <option key={teacher.id} value={teacher.id}>
                  {teacher.name}
                </option>
              ))}
            </FilterSelect>
            <label className="space-y-2">
              <span className="text-xs font-black uppercase tracking-[0.08em] text-[#687386] dark:text-[#a7b0bf]">
                {labels.search}
              </span>
              <div className="relative">
                <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-[#687386] dark:text-[#a7b0bf]" />
                <Input
                  className="h-11 pl-10"
                  value={searchValue}
                  onChange={(event) => setSearchValue(event.target.value)}
                />
              </div>
            </label>
            <Button
              className="h-11 cursor-pointer rounded-lg px-4 font-bold"
              disabled={chips.length === 0}
              type="button"
              variant="outline"
              onClick={resetFilters}
            >
              {labels.resetFilters}
            </Button>
          </div>
          {chips.length > 0 ? (
            <div className="flex flex-wrap items-center gap-2">
              {chips.map((chip) => (
                <button
                  className="inline-flex cursor-pointer items-center gap-2 rounded-full border border-[#dce3ee] bg-white px-3 py-1.5 text-xs font-black text-[#0a284b] shadow-sm transition hover:border-[#ef2334] hover:text-[#ef2334] dark:border-[#3a4658] dark:bg-[#202b3a] dark:text-[#f3f6fa] dark:hover:border-[#ff3b4f] dark:hover:text-[#ff5a69]"
                  key={chip.key}
                  title={labels.removeFilter}
                  type="button"
                  onClick={() => removeFilter(chip.key)}
                >
                  <span>{chip.label}</span>
                  <X className="size-3.5" />
                </button>
              ))}
            </div>
          ) : null}
        </CardContent>
      </Card>

      <Card className="overflow-hidden rounded-xl border-white/55 bg-white/65 py-0 shadow-[0_24px_70px_rgba(10,40,75,0.14)] backdrop-blur-xl dark:border-white/10 dark:bg-[#1b2635]/62 dark:shadow-[0_24px_80px_rgba(0,0,0,0.34)]">
        <CardContent className="p-0">
          <div className="max-h-[560px] overflow-auto">
            <Table>
              <TableHeader className="sticky top-0 z-10">
                <TableRow className="h-14 border-white/40 bg-[#eef3f8] hover:bg-[#eef3f8] dark:border-white/10 dark:bg-[#202b3a] dark:hover:bg-[#202b3a]">
                  <TableHead className="px-5 py-4">{labels.profile}</TableHead>
                  <TableHead>{labels.fin}</TableHead>
                  <TableHead>{labels.activeTeacher}</TableHead>
                  <TableHead>{labels.branch}</TableHead>
                  <TableHead>{labels.registeredDate}</TableHead>
                  <TableHead>{labels.status}</TableHead>
                  <TableHead className="pr-5 text-right">
                    {labels.actions}
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading ? (
                  <TableRow>
                    <TableCell colSpan={7}>
                      <div className="flex items-center justify-center gap-3 px-4 py-16 text-[#687386] dark:text-[#a7b0bf]">
                        <Loader2 className="size-5 animate-spin" />
                        <span className="font-bold">{labels.loading}</span>
                      </div>
                    </TableCell>
                  </TableRow>
                ) : pageData.items.length > 0 ? (
                  pageData.items.map((student) => (
                    <TableRow
                      className="border-white/35 hover:bg-white/35 dark:border-white/10 dark:hover:bg-[#202b3a]/50"
                      key={student.id}
                    >
                      <TableCell className="px-5 py-5">
                        <div className="flex items-center gap-3">
                          <Avatar className="size-11 border border-white/50 shadow-sm dark:border-[#414d60]">
                            {student.profile_photo_url ? (
                              <AvatarImage
                                alt={studentFullName(student)}
                                src={student.profile_photo_url}
                              />
                            ) : null}
                            <AvatarFallback className="bg-[#e8edf5] text-sm font-black text-[#0a284b] dark:bg-[#2a3444] dark:text-[#f3f6fa]">
                              {studentInitials(student)}
                            </AvatarFallback>
                          </Avatar>
                          <div className="min-w-0">
                            <div className="truncate font-bold">
                              {studentFullName(student)}
                            </div>
                            <div className="mt-1 text-xs text-[#687386] dark:text-[#6f7a8a]">
                              {student.id.slice(0, 8)}
                            </div>
                          </div>
                        </div>
                      </TableCell>
                      <TableCell className="py-5 font-black tracking-[0.08em]">
                        {student.fin}
                      </TableCell>
                      <TableCell className="py-5">
                        <ActiveTeacherCell labels={labels} student={student} />
                      </TableCell>
                      <TableCell className="py-5 font-semibold">
                        {student.branch_name}
                      </TableCell>
                      <TableCell className="py-5 text-[#59667a] dark:text-[#a7b0bf]">
                        {formatDate(student.registered_at)}
                      </TableCell>
                      <TableCell className="py-5">
                        <StudentStatusBadge labels={labels} status={student.status} />
                      </TableCell>
                      <TableCell className="py-5 pr-5 text-right">
                        <div className="flex justify-end gap-1">
                          <Button
                            aria-label={labels.edit}
                            className="cursor-pointer"
                            size="icon"
                            type="button"
                            variant="ghost"
                          >
                            <Edit3 className="size-4" />
                          </Button>
                          <Button
                            aria-label={labels.quickAssignSwap}
                            className="cursor-pointer"
                            size="icon"
                            type="button"
                            variant="ghost"
                          >
                            <ArrowLeftRight className="size-4" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))
                ) : (
                  <TableRow>
                    <TableCell colSpan={7}>
                      <div className="flex flex-col items-center justify-center gap-3 px-4 py-14 text-center text-[#687386] dark:text-[#6f7a8a]">
                        <Search className="size-8" />
                        <div className="font-bold">{labels.noStudents}</div>
                      </div>
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>
          <div className="flex flex-wrap items-center justify-end gap-4 border-t border-[#dce3ee] px-5 py-4 dark:border-[#334155]">
            <div className="text-sm font-semibold text-[#687386] dark:text-[#a7b0bf]">
              {labels.showing} {pageData.items.length} {labels.of} {pageData.total}
            </div>
            <label className="flex items-center gap-2 text-sm font-bold text-[#59667a] dark:text-[#a7b0bf]">
              {labels.rowsPerPage}
              <select
                className="h-9 cursor-pointer rounded-lg border border-[#dce3ee] bg-white px-2 text-sm font-bold text-[#0a284b] outline-none dark:border-[#3a4658] dark:bg-[#202b3a] dark:text-[#f3f6fa]"
                value={activeFilters.limit ?? 20}
                onChange={(event) =>
                  void updateFilters({
                    limit: Number.parseInt(event.target.value, 10),
                    offset: 0,
                  })
                }
              >
                {pageSizes.map((size) => (
                  <option key={size} value={size}>
                    {size}
                  </option>
                ))}
              </select>
            </label>
            <div className="flex items-center gap-1">
              <PaginationButton
                disabled={currentPage <= 1}
                onClick={() =>
                  void updateFilters({
                    offset:
                      Math.max(1, currentPage - 1) *
                        (activeFilters.limit ?? 20) -
                      (activeFilters.limit ?? 20),
                  })
                }
              >
                <ChevronLeft className="size-4" />
              </PaginationButton>
              {visiblePages(currentPage, totalPages).map((page) => (
                <PaginationButton
                  active={page === currentPage}
                  key={page}
                  onClick={() =>
                    void updateFilters({
                      offset: (page - 1) * (activeFilters.limit ?? 20),
                    })
                  }
                >
                  {page}
                </PaginationButton>
              ))}
              <PaginationButton
                disabled={currentPage >= totalPages}
                onClick={() =>
                  void updateFilters({
                    offset:
                      Math.min(totalPages, currentPage + 1) *
                        (activeFilters.limit ?? 20) -
                      (activeFilters.limit ?? 20),
                  })
                }
              >
                <ChevronRight className="size-4" />
              </PaginationButton>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

function FilterSelect({
  children,
  label,
  onChange,
  value,
}: {
  children: ReactNode;
  label: string;
  onChange: (value: string) => void;
  value: string;
}) {
  return (
    <label className="space-y-2">
      <span className="text-xs font-black uppercase tracking-[0.08em] text-[#687386] dark:text-[#a7b0bf]">
        {label}
      </span>
      <select
        className="h-11 w-full cursor-pointer rounded-lg border border-[#dce3ee] bg-white px-3 text-sm font-semibold text-[#0a284b] outline-none transition focus:border-[#ef2334] dark:border-[#3a4658] dark:bg-[#202b3a] dark:text-[#f3f6fa] dark:focus:border-[#ff3b4f]"
        value={value}
        onChange={(event) => onChange(event.target.value)}
      >
        {children}
      </select>
    </label>
  );
}

function ActiveTeacherCell({
  labels,
  student,
}: {
  labels: StudentAssignmentHubLabels;
  student: StudentAssignmentHubRecord;
}) {
  const name = [student.active_teacher_first_name, student.active_teacher_last_name]
    .filter(Boolean)
    .join(" ");
  if (!student.active_teacher_id || !name) {
    return (
      <span className="inline-flex items-center gap-2 rounded-full border border-[#ef2334]/35 bg-[#ef2334]/10 px-3 py-1 text-xs font-black text-[#ef2334] dark:border-[#ff3b4f]/35 dark:bg-[#ff3b4f]/10 dark:text-[#ff6b7a]">
        <UserX className="size-3.5" />
        {labels.notAssigned}
      </span>
    );
  }

  return <span className="font-semibold">{name}</span>;
}

function StudentStatusBadge({
  labels,
  status,
}: {
  labels: StudentAssignmentHubLabels;
  status: StudentStatus;
}) {
  const className =
    status === "active"
      ? "bg-emerald-600 text-white hover:bg-emerald-600"
      : status === "graduated"
        ? "bg-sky-600 text-white hover:bg-sky-600"
        : "bg-[#64748b] text-white hover:bg-[#64748b]";

  return <Badge className={className}>{labels.statuses[status]}</Badge>;
}

function PaginationButton({
  active,
  children,
  disabled,
  onClick,
}: {
  active?: boolean;
  children: ReactNode;
  disabled?: boolean;
  onClick: () => void;
}) {
  return (
    <button
      className={cn(
        "grid size-9 cursor-pointer place-items-center rounded-lg border border-[#dce3ee] text-sm font-semibold text-[#0a284b] transition hover:border-[#ef2334] hover:text-[#ef2334] disabled:cursor-not-allowed disabled:opacity-40 dark:border-[#3a4658] dark:text-[#f3f6fa] dark:hover:border-[#ff3b4f] dark:hover:text-[#ff5a69]",
        active &&
          "border-[#ef2334] bg-[#ef2334] font-black text-white hover:text-white dark:border-[#ff3b4f] dark:bg-[#ff3b4f] dark:text-white",
      )}
      disabled={disabled}
      type="button"
      onClick={onClick}
    >
      {children}
    </button>
  );
}

function buildTeacherOptions(records: TeacherFinanceRecord[]): TeacherOption[] {
  const seen = new Set<string>();
  const options: TeacherOption[] = [];
  for (const record of records) {
    if (seen.has(record.id)) {
      continue;
    }
    seen.add(record.id);
    options.push({ id: record.id, name: `${record.first_name} ${record.last_name}`.trim() });
  }
  return options.sort((a, b) => a.name.localeCompare(b.name));
}

function buildFilterChips(
  filters: StudentAssignmentHubFilters,
  branches: Branch[],
  teachers: TeacherOption[],
  labels: StudentAssignmentHubLabels,
) {
  const chips: { key: keyof StudentAssignmentHubFilters; label: string }[] = [];
  if (filters.branch_id) {
    const branch = branches.find((item) => item.id === filters.branch_id);
    chips.push({ key: "branch_id", label: `${labels.branch}: ${branch?.name ?? labels.allBranches}` });
  }
  if (filters.status) {
    chips.push({ key: "status", label: `${labels.status}: ${labels.statuses[filters.status]}` });
  }
  if (filters.teacher_id) {
    const teacher = teachers.find((item) => item.id === filters.teacher_id);
    chips.push({ key: "teacher_id", label: `${labels.teacher}: ${teacher?.name ?? labels.allTeachers}` });
  }
  if (filters.q) {
    chips.push({ key: "q", label: `${labels.search}: ${filters.q}` });
  }
  return chips;
}

function visiblePages(currentPage: number, totalPages: number) {
  if (totalPages <= 3) {
    return Array.from({ length: totalPages }, (_, index) => index + 1);
  }
  if (currentPage >= totalPages - 1) {
    return [totalPages - 2, totalPages - 1, totalPages];
  }
  return [currentPage, currentPage + 1, currentPage + 2];
}

function studentFullName(student: StudentAssignmentHubRecord) {
  return `${student.first_name} ${student.last_name}`.trim();
}

function studentInitials(student: StudentAssignmentHubRecord) {
  return `${student.first_name[0] ?? ""}${student.last_name[0] ?? ""}`
    .trim()
    .toUpperCase();
}

function formatDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "-";
  }
  return new Intl.DateTimeFormat("en-GB", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
  }).format(date);
}

const studentStatuses: StudentStatus[] = ["active", "left", "graduated"];
