"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import type { ReactNode } from "react";
import { useActionState, useEffect, useMemo, useState } from "react";
import { useFormStatus } from "react-dom";
import {
  AlertTriangle,
  Edit3,
  Plus,
  Search,
  Trash2,
  Users,
  X,
} from "lucide-react";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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
  SalaryModelType,
  Teacher,
  TeacherFinanceFilters,
  TeacherFinanceRecord,
  TeacherStatus,
} from "@/lib/api/types";
import {
  deleteTeacherAction,
  type DeleteTeacherState,
} from "@/lib/teacher-finance/actions";

export type TeacherFinanceHrLabels = {
  title: string;
  description: string;
  addTeacher: string;
  branch: string;
  subject: string;
  status: string;
  salaryModel: string;
  allBranches: string;
  allSubjects: string;
  allStatuses: string;
  allSalaryModels: string;
  resetFilters: string;
  removeFilter: string;
  profile: string;
  assignedStudents: string;
  calculatedSalary: string;
  actions: string;
  edit: string;
  delete: string;
  cancel: string;
  noTeachers: string;
  unassigned: string;
  deleteBlockedTitle: string;
  deleteBlockedDescription: string;
  deleteTeacherTitle: string;
  deleteTeacherDescription: string;
  deleteTeacherImpactTitle: string;
  deleteTeacherImpactDescription: string;
  confirmNameLabel: string;
  confirmNamePlaceholder: string;
  confirmNameMismatch: string;
  continueDelete: string;
  deleting: string;
  deleteBackendError: string;
  deleteNotFoundError: string;
  deleteUnauthorizedError: string;
  addTeacherPlaceholderTitle: string;
  addTeacherPlaceholderDescription: string;
  addTeacherDescription: string;
  profilePhoto: string;
  editPhoto: string;
  removePhoto: string;
  name: string;
  surname: string;
  birthDate: string;
  gender: string;
  genderPlaceholder: string;
  male: string;
  female: string;
  other: string;
  branchAssignment: string;
  salaryAmount: string;
  percentage: string;
  phoneNumber: string;
  address: string;
  email: string;
  password: string;
  save: string;
  saving: string;
  emailAvailable: string;
  emailTaken: string;
  emailChecking: string;
  requiredFieldsError: string;
  createBackendError: string;
  createUnauthorizedError: string;
  invalidPhotoError: string;
  photoTooLargeError: string;
  statuses: Record<TeacherStatus, string>;
  salaryTypes: Record<SalaryModelType, string>;
  defaultSubjects: string[];
};

type TeacherFinanceHrViewProps = {
  branches: Branch[];
  filters: TeacherFinanceFilters;
  labels: TeacherFinanceHrLabels;
  locale: string;
  records: TeacherFinanceRecord[];
};

const initialDeleteState: DeleteTeacherState = {};

export function TeacherFinanceHrView({
  branches,
  filters,
  labels,
  locale,
  records,
}: TeacherFinanceHrViewProps) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [deletedTeacherIDs, setDeletedTeacherIDs] = useState<Set<string>>(
    () => new Set(),
  );
  const subjectOptions = useMemo(
    () => buildSubjectOptions(records, labels.defaultSubjects),
    [labels.defaultSubjects, records],
  );
  const activeFilters = useMemo(
    () => buildActiveFilterChips(filters, branches, labels),
    [branches, filters, labels],
  );
  const visibleRecords = useMemo(
    () =>
      records
        .filter((record) => !deletedTeacherIDs.has(record.id))
        .filter((record) => matchesFilters(record, filters)),
    [deletedTeacherIDs, filters, records],
  );

  function updateFilter(key: keyof TeacherFinanceFilters, value: string) {
    const params = new URLSearchParams(searchParams.toString());
    params.set("view", "teacher-finance");
    if (value) {
      params.set(key, value);
    } else {
      params.delete(key);
    }
    router.push(`/${locale}/dashboard/owner?${params.toString()}`);
  }

  function resetFilters() {
    const params = new URLSearchParams(searchParams.toString());
    params.set("view", "teacher-finance");
    for (const key of teacherFilterKeys) {
      params.delete(key);
    }
    router.push(`/${locale}/dashboard/owner?${params.toString()}`);
  }

  function handleDeleted(teacher: Teacher) {
    setDeletedTeacherIDs((current) => {
      const next = new Set(current);
      next.add(teacher.id);
      return next;
    });
    router.refresh();
  }

  return (
    <div className="space-y-6">
      <section className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-3xl font-black tracking-tight">{labels.title}</h1>
          <p className="mt-1 text-sm text-kw-c-59667a dark:text-kw-c-a7b0bf">
            {labels.description}
          </p>
        </div>
        <Button
          asChild
          className="h-[38px] gap-2 rounded-lg bg-kw-c-ef2334 px-5 font-bold text-white shadow-kw-action transition hover:bg-kw-c-d91f30 dark:bg-kw-c-ff3b4f dark:hover:bg-kw-c-ff5a69"
        >
          <Link href={`/${locale}/dashboard/owner?view=teacher-add`}>
            <Plus className="size-4" />
            {labels.addTeacher}
          </Link>
        </Button>
      </section>

      <Card className="rounded-xl border-white/55 bg-white/65 shadow-kw-panel backdrop-blur-xl dark:border-white/10 dark:bg-kw-c-1b2635/62 kw-dark-shadow-panel">
        <CardContent className="p-4">
          <div className="grid gap-3 md:grid-cols-[repeat(4,minmax(0,1fr))_auto] md:items-end">
            <FilterSelect
              label={labels.branch}
              value={filters.branch_id ?? ""}
              onChange={(value) => updateFilter("branch_id", value)}
            >
              <option value="">{labels.allBranches}</option>
              {branches.map((branch) => (
                <option key={branch.id} value={branch.id}>
                  {branch.name}
                </option>
              ))}
            </FilterSelect>
            <FilterSelect
              label={labels.subject}
              value={filters.subject ?? ""}
              onChange={(value) => updateFilter("subject", value)}
            >
              <option value="">{labels.allSubjects}</option>
              {subjectOptions.map((subject) => (
                <option key={subject} value={subject}>
                  {subject}
                </option>
              ))}
            </FilterSelect>
            <FilterSelect
              label={labels.status}
              value={filters.status ?? ""}
              onChange={(value) => updateFilter("status", value)}
            >
              <option value="">{labels.allStatuses}</option>
              {teacherStatuses.map((status) => (
                <option key={status} value={status}>
                  {labels.statuses[status]}
                </option>
              ))}
            </FilterSelect>
            <FilterSelect
              label={labels.salaryModel}
              value={filters.salary_model ?? ""}
              onChange={(value) => updateFilter("salary_model", value)}
            >
              <option value="">{labels.allSalaryModels}</option>
              {salaryTypes.map((salaryType) => (
                <option key={salaryType} value={salaryType}>
                  {labels.salaryTypes[salaryType]}
                </option>
              ))}
            </FilterSelect>
            <Button
              className="h-11 cursor-pointer rounded-lg px-4 font-bold"
              disabled={activeFilters.length === 0}
              type="button"
              variant="outline"
              onClick={resetFilters}
            >
              {labels.resetFilters}
            </Button>
          </div>
          {activeFilters.length > 0 ? (
            <div className="mt-4 flex flex-wrap items-center gap-2">
              {activeFilters.map((filter) => (
                <button
                  className="inline-flex cursor-pointer items-center gap-2 rounded-full border border-kw-c-dce3ee bg-white px-3 py-1.5 text-xs font-black text-kw-c-0a284b shadow-sm transition hover:border-kw-c-ef2334 hover:text-kw-c-ef2334 dark:border-kw-c-3a4658 dark:bg-kw-c-202b3a dark:text-kw-c-f3f6fa dark:hover:border-kw-c-ff3b4f dark:hover:text-kw-c-ff5a69"
                  key={filter.key}
                  title={labels.removeFilter}
                  type="button"
                  onClick={() => updateFilter(filter.key, "")}
                >
                  <span>{filter.label}</span>
                  <X className="size-3.5" />
                </button>
              ))}
            </div>
          ) : null}
        </CardContent>
      </Card>

      <Card className="overflow-hidden rounded-xl border-white/55 bg-white/65 py-0 shadow-kw-panel-strong backdrop-blur-xl dark:border-white/10 dark:bg-kw-c-1b2635/62 kw-dark-shadow-panel-strong">
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow className="h-14 border-white/40 bg-kw-c-eef3f8 hover:bg-kw-c-eef3f8 dark:border-white/10 dark:bg-kw-c-202b3a dark:hover:bg-kw-c-202b3a">
                <TableHead className="px-5 py-4">{labels.profile}</TableHead>
                <TableHead>{labels.branch}</TableHead>
                <TableHead>{labels.subject}</TableHead>
                <TableHead>{labels.status}</TableHead>
                <TableHead>{labels.salaryModel}</TableHead>
                <TableHead>{labels.assignedStudents}</TableHead>
                <TableHead>{labels.calculatedSalary}</TableHead>
                <TableHead className="pr-5 text-right">
                  {labels.actions}
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {visibleRecords.length > 0 ? (
                visibleRecords.map((record) => (
                  <TableRow
                    className="border-white/35 hover:bg-white/35 dark:border-white/10 dark:hover:bg-kw-c-202b3a/50"
                    key={record.id}
                  >
                    <TableCell className="px-5 py-5">
                      <div className="flex items-center gap-3">
                        <Avatar className="size-11 border border-white/50 shadow-sm dark:border-kw-c-414d60">
                          <AvatarImage
                            alt={teacherFullName(record)}
                            src={record.profile_photo_url}
                          />
                          <AvatarFallback className="bg-kw-c-e8edf5 text-sm font-black text-kw-c-0a284b dark:bg-kw-c-2a3444 dark:text-kw-c-f3f6fa">
                            {teacherInitials(record)}
                          </AvatarFallback>
                        </Avatar>
                        <div className="min-w-0">
                          <div className="truncate font-bold">
                            {teacherFullName(record)}
                          </div>
                          <div className="mt-1 text-xs text-kw-c-687386 dark:text-kw-c-6f7a8a">
                            {record.email}
                          </div>
                        </div>
                      </div>
                    </TableCell>
                    <TableCell className="py-5 font-semibold">
                      {record.branch_name}
                    </TableCell>
                    <TableCell className="py-5">
                      <span className="text-kw-c-59667a dark:text-kw-c-a7b0bf">
                        {record.subject || labels.unassigned}
                      </span>
                    </TableCell>
                    <TableCell className="py-5">
                      <StatusBadge labels={labels} status={record.status} />
                    </TableCell>
                    <TableCell className="py-5">
                      {record.salary_type ? (
                        labels.salaryTypes[record.salary_type]
                      ) : (
                        <span className="text-kw-c-687386 dark:text-kw-c-6f7a8a">
                          {labels.unassigned}
                        </span>
                      )}
                    </TableCell>
                    <TableCell className="py-5">
                      <div className="inline-flex items-center gap-2 rounded-full border border-kw-c-ef2334/25 bg-kw-c-ef2334/8 px-3 py-1 text-sm font-black text-kw-c-0a284b dark:border-kw-c-ff3b4f/35 dark:bg-kw-c-ff3b4f/10 dark:text-kw-c-f3f6fa">
                        <Users className="size-4 text-kw-c-ef2334 dark:text-kw-c-ff3b4f" />
                        {record.assigned_students}
                      </div>
                    </TableCell>
                    <TableCell className="py-5 font-black tabular-nums">
                      {formatManat(record.calculated_salary_amount_cents)}
                    </TableCell>
                    <TableCell className="py-5 pr-5 text-right">
                      <div className="flex justify-end gap-1">
                        <Button
                          asChild
                          className="cursor-pointer"
                          size="icon"
                          type="button"
                          variant="ghost"
                        >
                          <Link
                            aria-label={labels.edit}
                            href={`/${locale}/dashboard/owner?view=teacher-edit&teacher_id=${record.id}`}
                          >
                            <Edit3 className="size-4" />
                          </Link>
                        </Button>
                        <TeacherDeleteDialog
                          labels={labels}
                          onDeleted={handleDeleted}
                          record={record}
                        />
                      </div>
                    </TableCell>
                  </TableRow>
                ))
              ) : (
                <TableRow>
                  <TableCell colSpan={8}>
                    <div className="flex flex-col items-center justify-center gap-3 px-4 py-14 text-center text-kw-c-687386 dark:text-kw-c-6f7a8a">
                      <Search className="size-8" />
                      <div className="font-bold">{labels.noTeachers}</div>
                    </div>
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
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
      <span className="text-xs font-black uppercase tracking-[0.08em] text-kw-c-687386 dark:text-kw-c-a7b0bf">
        {label}
      </span>
      <select
        className="h-11 w-full cursor-pointer rounded-lg border border-kw-c-dce3ee bg-white px-3 text-sm font-semibold text-kw-c-0a284b outline-none transition focus:border-kw-c-ef2334 dark:border-kw-c-3a4658 dark:bg-kw-c-202b3a dark:text-kw-c-f3f6fa dark:focus:border-kw-c-ff3b4f"
        value={value}
        onChange={(event) => onChange(event.target.value)}
      >
        {children}
      </select>
    </label>
  );
}

function StatusBadge({
  labels,
  status,
}: {
  labels: TeacherFinanceHrLabels;
  status: TeacherStatus;
}) {
  const className =
    status === "active"
      ? "bg-emerald-600 text-white hover:bg-emerald-600"
      : status === "pending_owner_approval"
        ? "bg-amber-500 text-white hover:bg-amber-500"
        : "bg-kw-c-64748b text-white hover:bg-kw-c-64748b";

  return <Badge className={className}>{labels.statuses[status]}</Badge>;
}

function TeacherDeleteDialog({
  labels,
  onDeleted,
  record,
}: {
  labels: TeacherFinanceHrLabels;
  onDeleted: (teacher: Teacher) => void;
  record: TeacherFinanceRecord;
}) {
  const [open, setOpen] = useState(false);
  const [confirmName, setConfirmName] = useState("");
  const [confirmStep, setConfirmStep] = useState<"name" | "impact">("name");
  const [hasSubmittedDelete, setHasSubmittedDelete] = useState(false);
  const [deleteIdempotencyKey, setDeleteIdempotencyKey] = useState(() =>
    crypto.randomUUID(),
  );
  const [state, formAction] = useActionState(
    deleteTeacherAction,
    initialDeleteState,
  );
  const fullName = teacherFullName(record);
  const nameMatches = confirmName.trim() === fullName;
  const blocked = record.assigned_students > 0;

  useEffect(() => {
    if (!state.teacher) {
      return;
    }
    window.queueMicrotask(() => {
      onDeleted(state.teacher as Teacher);
      setDeleteIdempotencyKey(crypto.randomUUID());
      setOpen(false);
      setConfirmName("");
      setConfirmStep("name");
      setHasSubmittedDelete(false);
    });
  }, [onDeleted, state.teacher]);

  function handleOpenChange(nextOpen: boolean) {
    setOpen(nextOpen);
    if (!nextOpen) {
      setConfirmName("");
      setConfirmStep("name");
      setHasSubmittedDelete(false);
    }
  }

  return (
    <>
      <Button
        aria-label={labels.delete}
        className="cursor-pointer text-kw-c-ef2334 hover:text-kw-c-ef2334 dark:text-kw-c-ff3b4f dark:hover:text-kw-c-ff3b4f"
        size="icon"
        type="button"
        variant="ghost"
        onClick={() => setOpen(true)}
      >
        <Trash2 className="size-4" />
      </Button>
      <Dialog open={open} onOpenChange={handleOpenChange}>
        <DialogContent className="z-[1200] border-kw-c-dce3ee dark:border-kw-c-3a4658 dark:bg-kw-c-1b2635 dark:text-kw-c-f3f6fa">
          {blocked ? (
            <>
              <DialogHeader>
                <DialogTitle className="text-xl font-black">
                  {labels.deleteBlockedTitle}
                </DialogTitle>
                <DialogDescription>
                  {labels.deleteBlockedDescription}
                </DialogDescription>
              </DialogHeader>
              <DialogFooter className="bg-transparent px-0 pb-0">
                <Button type="button" onClick={() => handleOpenChange(false)}>
                  {labels.cancel}
                </Button>
              </DialogFooter>
            </>
          ) : confirmStep === "impact" ? (
            <form
              action={formAction}
              onSubmit={() => setHasSubmittedDelete(true)}
            >
              <input name="teacher_id" type="hidden" value={record.id} />
              <input
                name="idempotency_key"
                type="hidden"
                value={deleteIdempotencyKey}
              />
              <DialogHeader>
                <DialogTitle className="flex items-center gap-2 text-xl font-black">
                  <AlertTriangle className="size-5 text-kw-c-ef2334 dark:text-kw-c-ff5a69" />
                  {labels.deleteTeacherImpactTitle}
                </DialogTitle>
                <DialogDescription>
                  {labels.deleteTeacherImpactDescription}
                </DialogDescription>
              </DialogHeader>
              <DeleteTeacherError
                labels={labels}
                show={hasSubmittedDelete}
                state={state}
              />
              <DialogFooter className="mt-5 bg-transparent px-0 pb-0">
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => handleOpenChange(false)}
                >
                  {labels.cancel}
                </Button>
                <DeleteTeacherSubmit
                  disabled={false}
                  label={labels.continueDelete}
                  pending={labels.deleting}
                />
              </DialogFooter>
            </form>
          ) : (
            <>
              <DialogHeader>
                <DialogTitle className="text-xl font-black">
                  {labels.deleteTeacherTitle}
                </DialogTitle>
                <DialogDescription>
                  {labels.deleteTeacherDescription}
                </DialogDescription>
              </DialogHeader>
              <div className="mt-5 space-y-2">
                <Label htmlFor={`delete-teacher-${record.id}`}>
                  {labels.confirmNameLabel}
                </Label>
                <Input
                  id={`delete-teacher-${record.id}`}
                  placeholder={labels.confirmNamePlaceholder.replace(
                    "{name}",
                    fullName,
                  )}
                  value={confirmName}
                  onChange={(event) => setConfirmName(event.target.value)}
                />
                {confirmName && !nameMatches ? (
                  <p className="text-sm font-semibold text-kw-c-ef2334 dark:text-kw-c-ff6b7a">
                  {labels.confirmNameMismatch}
                </p>
              ) : null}
              </div>
              <DialogFooter className="mt-5 bg-transparent px-0 pb-0">
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => handleOpenChange(false)}
                >
                  {labels.cancel}
                </Button>
                <Button
                  className="bg-kw-c-8f1020 text-white hover:bg-kw-c-74101d"
                  disabled={!nameMatches}
                  type="button"
                  onClick={() => setConfirmStep("impact")}
                >
                  {labels.delete}
                </Button>
              </DialogFooter>
            </>
          )}
        </DialogContent>
      </Dialog>
    </>
  );
}

function DeleteTeacherSubmit({
  disabled,
  label,
  pending,
}: {
  disabled: boolean;
  label: string;
  pending: string;
}) {
  const status = useFormStatus();

  return (
    <Button
      className="bg-kw-c-8f1020 text-white hover:bg-kw-c-74101d"
      disabled={disabled || status.pending}
      type="submit"
    >
      {status.pending ? pending : label}
    </Button>
  );
}

function DeleteTeacherError({
  labels,
  show,
  state,
}: {
  labels: TeacherFinanceHrLabels;
  show: boolean;
  state: DeleteTeacherState;
}) {
  if (!show) {
    return null;
  }

  const message =
    state.error === "assigned_students"
      ? labels.deleteBlockedDescription
      : state.error === "not_found"
        ? labels.deleteNotFoundError
        : state.error === "unauthorized"
          ? labels.deleteUnauthorizedError
          : state.error === "backend"
            ? labels.deleteBackendError
            : null;

  if (!message) {
    return null;
  }

  return (
    <div className="mt-4 rounded-lg border border-kw-c-f87171 bg-kw-c-fff1f2 px-4 py-3 text-sm font-semibold text-kw-c-991b1b dark:bg-kw-c-3b1218 dark:text-kw-c-ffe4e6">
      {message}
    </div>
  );
}

function buildSubjectOptions(
  records: TeacherFinanceRecord[],
  defaults: string[],
) {
  const subjects = new Set(defaults);
  for (const record of records) {
    for (const subject of (record.subject ?? "").split(",")) {
      const normalized = subject.trim();
      if (normalized) {
        subjects.add(normalized);
      }
    }
  }

  return [...subjects].sort((a, b) => a.localeCompare(b));
}

function buildActiveFilterChips(
  filters: TeacherFinanceFilters,
  branches: Branch[],
  labels: TeacherFinanceHrLabels,
) {
  const chips: { key: keyof TeacherFinanceFilters; label: string }[] = [];

  if (filters.branch_id) {
    const branch = branches.find((item) => item.id === filters.branch_id);
    chips.push({
      key: "branch_id",
      label: `${labels.branch}: ${branch?.name ?? labels.allBranches}`,
    });
  }
  if (filters.subject) {
    chips.push({
      key: "subject",
      label: `${labels.subject}: ${filters.subject}`,
    });
  }
  if (filters.status) {
    chips.push({
      key: "status",
      label: `${labels.status}: ${labels.statuses[filters.status]}`,
    });
  }
  if (filters.salary_model) {
    chips.push({
      key: "salary_model",
      label: `${labels.salaryModel}: ${labels.salaryTypes[filters.salary_model]}`,
    });
  }

  return chips;
}

function matchesFilters(
  record: TeacherFinanceRecord,
  filters: TeacherFinanceFilters,
) {
  if (filters.branch_id && record.branch_id !== filters.branch_id) {
    return false;
  }
  if (
    filters.subject &&
    !(record.subject ?? "")
      .toLocaleLowerCase("en-US")
      .includes(filters.subject.toLocaleLowerCase("en-US"))
  ) {
    return false;
  }
  if (filters.status && record.status !== filters.status) {
    return false;
  }
  if (filters.salary_model && record.salary_type !== filters.salary_model) {
    return false;
  }

  return true;
}

function teacherFullName(record: TeacherFinanceRecord) {
  return `${record.first_name} ${record.last_name}`.trim();
}

function teacherInitials(record: TeacherFinanceRecord) {
  return `${record.first_name[0] ?? ""}${record.last_name[0] ?? ""}`
    .trim()
    .toUpperCase();
}

function formatManat(valueCents: number) {
  const value = valueCents / 100;
  const formatted = new Intl.NumberFormat("en-US", {
    maximumFractionDigits: 0,
  }).format(value);
  return `₼ ${formatted}`;
}

const teacherStatuses: TeacherStatus[] = [
  "pending_owner_approval",
  "active",
  "terminated",
];

const salaryTypes: SalaryModelType[] = ["fixed", "percent", "hybrid"];

const teacherFilterKeys: (keyof TeacherFinanceFilters)[] = [
  "branch_id",
  "subject",
  "status",
  "salary_model",
];
