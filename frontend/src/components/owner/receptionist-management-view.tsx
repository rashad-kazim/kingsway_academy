"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import type { ReactNode } from "react";
import { AnimatePresence, LayoutGroup, motion } from "framer-motion";
import {
  AlertTriangle,
  BriefcaseBusiness,
  CalendarDays,
  Edit3,
  Plus,
  Trash2,
  UserRound,
} from "lucide-react";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
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
  genderLabel,
  initials,
  StatusBadge,
} from "@/components/owner/receptionist-management-parts";
import {
  StaffEditor,
  type StaffSharedLayoutIDs,
} from "@/components/owner/receptionist-staff-editor";
import type { Branch, StaffMember } from "@/lib/api/types";
import {
  deleteStaffAction,
  type StaffManagementState,
} from "@/lib/owner/actions";
import { cn } from "@/lib/utils";

export type ReceptionistManagementLabels = {
  title: string;
  description: string;
  branch: string;
  allBranches: string;
  gender: string;
  allGenders: string;
  status: string;
  allStatuses: string;
  resetFilters: string;
  removeFilter: string;
  existingReceptionists: string;
  noReceptionists: string;
  addStaff: string;
  editStaff: string;
  finishNewStaffFirst: string;
  profile: string;
  assignedBranch: string;
  workingHours: string;
  salary: string;
  lastLogin: string;
  active: string;
  inactive: string;
  never: string;
  edit: string;
  delete: string;
  deleteStaffTitle: string;
  deleteStaffDescription: string;
  deleteStaffImpactTitle: string;
  deleteStaffImpactDescription: string;
  confirmNameLabel: string;
  confirmNamePlaceholder: string;
  confirmNameMismatch: string;
  continueDelete: string;
  deleting: string;
  cancel: string;
  save: string;
  saving: string;
  name: string;
  surname: string;
  birthDate: string;
  genderPlaceholder: string;
  male: string;
  female: string;
  other: string;
  phone: string;
  address: string;
  hiredAt: string;
  email: string;
  password: string;
  showPassword: string;
  hidePassword: string;
  emailAvailable: string;
  emailTaken: string;
  emailChecking: string;
  staffSaveError: string;
  staffDuplicateError: string;
  staffInvalidInputError: string;
  hireDateBeforeBirthError: string;
  invalidPhotoError: string;
  photoTooLargeError: string;
  receptionist: string;
};

type ReceptionistManagementViewProps = {
  branches: Branch[];
  labels: ReceptionistManagementLabels;
  staff: StaffMember[];
};

const initialActionState: StaffManagementState = {};
const sharedMotionTransition = {
  type: "spring",
  stiffness: 210,
  damping: 28,
  mass: 0.9,
} as const;
const revealMotionTransition = {
  duration: 0.34,
  ease: [0.22, 1, 0.36, 1],
} as const;
const shellCollapseTransition = {
  duration: 0.56,
  ease: [0.22, 1, 0.36, 1],
} as const;
export function ReceptionistManagementView({
  branches,
  labels,
  staff,
}: ReceptionistManagementViewProps) {
  const [activeBranchID, setActiveBranchID] = useState("");
  const [staffList, setStaffList] = useState(staff);
  const [createOpen, setCreateOpen] = useState(false);
  const [editingStaffID, setEditingStaffID] = useState("");
  const [warning, setWarning] = useState("");
  const branchesByID = useMemo(
    () => new Map(branches.map((branch) => [branch.id, branch])),
    [branches],
  );
  const visibleStaff = useMemo(
    () =>
      staffList.filter((member) =>
        activeBranchID ? member.branch_id === activeBranchID : true,
      ),
    [activeBranchID, staffList],
  );

  function upsertStaff(nextStaff: StaffMember) {
    setStaffList((current) => {
      const exists = current.some((member) => member.id === nextStaff.id);
      if (!exists) {
        return [nextStaff, ...current];
      }
      return current.map((member) =>
        member.id === nextStaff.id ? nextStaff : member,
      );
    });
  }

  function handleCreateSaved(nextStaff: StaffMember) {
    upsertStaff(nextStaff);
    setCreateOpen(false);
  }

  function handleEditSaved(nextStaff: StaffMember) {
    upsertStaff(nextStaff);
    setEditingStaffID("");
  }

  function removeStaff(staffID: string) {
    setStaffList((current) => current.filter((member) => member.id !== staffID));
  }

  function startCreate() {
    setWarning("");
    setEditingStaffID("");
    setCreateOpen(true);
  }

  function startEdit(member: StaffMember) {
    if (createOpen) {
      setWarning(labels.finishNewStaffFirst);
      return;
    }
    setWarning("");
    setEditingStaffID(member.id);
  }

  return (
    <div className="space-y-6">
      <section>
        <h1 className="text-3xl font-black tracking-tight">{labels.title}</h1>
        <p className="mt-1 text-sm text-kw-c-59667a dark:text-kw-c-a7b0bf">
          {labels.description}
        </p>
      </section>

      <BranchPills
        activeBranchID={activeBranchID}
        branches={branches}
        labels={labels}
        onChange={setActiveBranchID}
      />

      <section className="space-y-4">
        <h2 className="text-xl font-black">{labels.existingReceptionists}</h2>
        {visibleStaff.length > 0 ? (
          <div className="space-y-3">
            {visibleStaff.map((member) => (
              <ReceptionistLine
                branch={branchesByID.get(member.branch_id)}
                branches={branches}
                editing={editingStaffID === member.id}
                key={member.id}
                labels={labels}
                staff={member}
                onCancelEdit={() => setEditingStaffID("")}
                onDelete={removeStaff}
                onEdit={() => startEdit(member)}
                onSaved={handleEditSaved}
              />
            ))}
          </div>
        ) : (
          <div className="rounded-xl border border-dashed border-kw-c-cbd5e1 p-8 text-center text-sm font-bold text-kw-c-59667a dark:border-kw-c-3a4658 dark:text-kw-c-a7b0bf">
            {labels.noReceptionists}
          </div>
        )}
      </section>

      <section className="space-y-4">
        {warning ? (
          <div className="rounded-lg border border-kw-c-f87171 bg-kw-c-3b1218 px-4 py-3 text-sm font-bold text-kw-c-ffe4e6">
            {warning}
          </div>
        ) : null}
        {createOpen ? (
          <StaffEditor
            branches={branches}
            labels={labels}
            mode="create"
            onCancel={() => setCreateOpen(false)}
            onSaved={handleCreateSaved}
          />
        ) : null}
        <div className="flex justify-center">
          <Button
            className="h-11 gap-2 rounded-lg bg-kw-c-ef2334 px-6 font-black text-white transition hover:bg-kw-c-d91f30 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-kw-c-ff3b4f dark:hover:bg-kw-c-ff5a69"
            disabled={createOpen}
            type="button"
            onClick={startCreate}
          >
            <Plus className="size-4" />
            {labels.addStaff}
          </Button>
        </div>
      </section>
    </div>
  );
}

function ReceptionistLine({
  branch,
  branches,
  editing,
  labels,
  onCancelEdit,
  onDelete,
  onEdit,
  onSaved,
  staff,
}: {
  branch?: Branch;
  branches: Branch[];
  editing: boolean;
  labels: ReceptionistManagementLabels;
  onCancelEdit: () => void;
  onDelete: (staffID: string) => void;
  onEdit: () => void;
  onSaved: (staff: StaffMember) => void;
  staff: StaffMember;
}) {
  const [deleteStep, setDeleteStep] = useState<"confirm" | "impact" | null>(
    null,
  );
  const [confirmName, setConfirmName] = useState("");
  const [closing, setClosing] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [deleteIdempotencyKey, setDeleteIdempotencyKey] = useState(() =>
    crypto.randomUUID(),
  );
  const [editorLeaving, setEditorLeaving] = useState(false);
  const cardRef = useRef<HTMLDivElement>(null);
  const shellRef = useRef<HTMLDivElement>(null);
  const closePrepTimeoutRef = useRef<number | null>(null);
  const closeTimeoutRef = useRef<number | null>(null);
  const [lockedHeight, setLockedHeight] = useState<number | null>(null);
  const fullName = `${staff.first_name} ${staff.last_name}`.trim();
  const confirmMatches = confirmName.trim() === fullName;
  const showEditor = editing && !editorLeaving;
  const showSummary = !editing || editorLeaving;
  const shellMinHeight = lockedHeight
    ? editorLeaving
      ? 80
      : lockedHeight
    : undefined;
  const shellTransition =
    closing && shellMinHeight
      ? shellCollapseTransition
      : sharedMotionTransition;
  const layoutIDs = useMemo<StaffSharedLayoutIDs>(
    () => ({
      avatar: `receptionist-${staff.id}-avatar`,
      branch: `receptionist-${staff.id}-branch`,
      gender: `receptionist-${staff.id}-gender`,
      hours: `receptionist-${staff.id}-hours`,
      name: `receptionist-${staff.id}-name`,
      salary: `receptionist-${staff.id}-salary`,
      status: `receptionist-${staff.id}-status`,
    }),
    [staff.id],
  );

  useEffect(() => {
    return () => {
      if (closeTimeoutRef.current) {
        window.clearTimeout(closeTimeoutRef.current);
      }
      if (closePrepTimeoutRef.current) {
        window.clearTimeout(closePrepTimeoutRef.current);
      }
    };
  }, []);

  useEffect(() => {
    if (!editing) {
      return;
    }
    const timeout = window.setTimeout(() => {
      const card = cardRef.current;
      if (!card) {
        return;
      }
      const rect = card.getBoundingClientRect();
      const safeTop = 104;
      if (rect.top < safeTop) {
        card.scrollIntoView({
          behavior: "smooth",
          block: "start",
        });
      }
    }, 280);

    return () => window.clearTimeout(timeout);
  }, [editing]);

  function handleCancelEdit() {
    const height = shellRef.current?.getBoundingClientRect().height;
    setLockedHeight(height ?? null);
    setClosing(true);
    setEditorLeaving(false);
    if (closeTimeoutRef.current) {
      window.clearTimeout(closeTimeoutRef.current);
    }
    if (closePrepTimeoutRef.current) {
      window.clearTimeout(closePrepTimeoutRef.current);
    }
    closePrepTimeoutRef.current = window.setTimeout(() => {
      setEditorLeaving(true);
      closePrepTimeoutRef.current = null;
    }, 180);
    closeTimeoutRef.current = window.setTimeout(() => {
      onCancelEdit();
      setClosing(false);
      setEditorLeaving(false);
      setLockedHeight(null);
      closeTimeoutRef.current = null;
    }, 920);
  }

  async function handleDelete() {
    if (deleting) {
      return;
    }
    setDeleting(true);
    const formData = new FormData();
    formData.set("staff_id", staff.id);
    formData.set("idempotency_key", deleteIdempotencyKey);
    const result = await deleteStaffAction(initialActionState, formData);
    setDeleting(false);
    if (result.staff) {
      setDeleteIdempotencyKey(crypto.randomUUID());
      onDelete(result.staff.id);
    }
    closeDeleteDialogs();
  }

  function closeDeleteDialogs() {
    setDeleteStep(null);
    setConfirmName("");
  }

  return (
    <div ref={cardRef}>
      <LayoutGroup id={`receptionist-${staff.id}`}>
        <motion.div
          layout="size"
          animate={
            shellMinHeight ? { minHeight: shellMinHeight } : undefined
          }
          data-testid="receptionist-row"
          ref={shellRef}
          transition={shellTransition}
        >
          <Card className="overflow-hidden rounded-xl border-white/55 bg-white/75 shadow-kw-line-card backdrop-blur-xl transition-all duration-300 dark:border-white/10 dark:bg-kw-c-1b2635/72">
            <CardContent className="relative p-0">
              <AnimatePresence initial={false} mode="popLayout">
                {showSummary ? (
                  <motion.div
                    key="summary"
                    layout="size"
                    aria-label={fullName}
                    className="grid min-h-20 cursor-pointer items-center gap-4 px-5 py-4 text-sm transition-colors duration-300 lg:grid-cols-[minmax(0,1.35fr)_minmax(0,.85fr)_minmax(0,1fr)_minmax(0,1fr)_minmax(0,.82fr)_minmax(0,.78fr)_96px]"
                    initial={false}
                    role="button"
                    tabIndex={0}
                    transition={sharedMotionTransition}
                    onClick={onEdit}
                    onKeyDown={(event) => {
                      if (event.key === "Enter" || event.key === " ") {
                        event.preventDefault();
                        onEdit();
                      }
                    }}
                  >
                    <div className="flex min-w-0 items-center gap-3">
                      <motion.div
                        layoutId={layoutIDs.avatar}
                        transition={sharedMotionTransition}
                      >
                        <Avatar className="size-12 border border-kw-c-dce3ee dark:border-kw-c-3a4658">
                          <AvatarImage
                            alt={fullName}
                            src={staff.profile_photo_url}
                          />
                          <AvatarFallback className="bg-kw-c-edf2f7 font-black text-kw-c-0a284b dark:bg-kw-c-2a3444 dark:text-kw-c-f3f6fa">
                            {initials(staff.first_name, staff.last_name)}
                          </AvatarFallback>
                        </Avatar>
                      </motion.div>
                      <motion.div
                        className="min-w-0"
                        layoutId={layoutIDs.name}
                        transition={sharedMotionTransition}
                      >
                        <h3 className="truncate font-black">{fullName}</h3>
                        <p className="truncate text-xs font-semibold text-kw-c-59667a dark:text-kw-c-a7b0bf">
                          {labels.receptionist}
                        </p>
                      </motion.div>
                    </div>
                    <InlineMeta
                      icon={<UserRound className="size-4" />}
                      layoutId={layoutIDs.gender}
                      value={genderLabel(staff.gender, labels)}
                    />
                    <InlineMeta
                      icon={<BriefcaseBusiness className="size-4" />}
                      layoutId={layoutIDs.branch}
                      value={branch?.name ?? "-"}
                    />
                    <InlineMeta
                      icon={<CalendarDays className="size-4" />}
                      layoutId={layoutIDs.hours}
                      value={`${branch?.opening_time || "09:00"} - ${branch?.closing_time || "18:00"}`}
                    />
                    <InlineMeta
                      icon={
                        <span className="text-base font-black">
                          {"\u20bc"}
                        </span>
                      }
                      layoutId={layoutIDs.salary}
                      value={`${staff.salary_amount_azn || 0}`}
                    />
                    <motion.div
                      layoutId={layoutIDs.status}
                      transition={sharedMotionTransition}
                    >
                      <StatusBadge isActive={staff.is_active} labels={labels} />
                    </motion.div>
                    <motion.div
                      className="flex justify-end gap-2"
                      initial={false}
                      transition={revealMotionTransition}
                    >
                      <Button
                        aria-label={labels.edit}
                        className="size-10 rounded-full"
                        size="icon"
                        type="button"
                        variant="outline"
                        onClick={(event) => {
                          event.stopPropagation();
                          onEdit();
                        }}
                      >
                        <Edit3 className="size-4" />
                      </Button>
                      <Button
                        aria-label={labels.delete}
                        className="size-10 rounded-full border-kw-c-ef2334 text-kw-c-ef2334 hover:bg-kw-c-ef2334/10 hover:text-kw-c-ef2334 dark:border-kw-c-ff3b4f dark:text-kw-c-ff3b4f dark:hover:bg-kw-c-ff3b4f/10"
                        disabled={deleting}
                        size="icon"
                        type="button"
                        variant="outline"
                        onClick={(event) => {
                          event.stopPropagation();
                          setDeleteStep("confirm");
                        }}
                      >
                        <Trash2 className="size-4" />
                      </Button>
                    </motion.div>
                  </motion.div>
                ) : null}
                {showEditor ? (
                  <motion.div
                    key="editor"
                    layout="size"
                    className="overflow-hidden"
                    exit={
                      closing
                        ? {
                            opacity: 0,
                            transition: { duration: 0 },
                          }
                        : undefined
                    }
                    initial={false}
                    transition={sharedMotionTransition}
                  >
                    <div className="px-5 py-5">
                      <StaffEditor
                        embedded
                        branches={branches}
                        initialStaff={staff}
                        labels={labels}
                        closing={closing}
                        layoutIDs={layoutIDs}
                        mode="edit"
                        selectedBranch={branch}
                        onCancel={handleCancelEdit}
                        onSaved={onSaved}
                      />
                    </div>
                  </motion.div>
                ) : null}
              </AnimatePresence>
            </CardContent>
          </Card>
        </motion.div>
      </LayoutGroup>

      <Dialog open={deleteStep === "confirm"} onOpenChange={closeDeleteDialogs}>
        <DialogContent className="z-[1300]">
          <DialogHeader>
            <DialogTitle>{labels.deleteStaffTitle}</DialogTitle>
            <DialogDescription>
              {(labels.deleteStaffDescription || "").replace("{name}", fullName)}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label>{labels.confirmNameLabel}</Label>
            <Input
              placeholder={labels.confirmNamePlaceholder.replace(
                "{name}",
                fullName,
              )}
              value={confirmName}
              onChange={(event) => setConfirmName(event.target.value)}
            />
            {confirmName && !confirmMatches ? (
              <p className="text-sm font-bold text-kw-c-ef2334">
                {labels.confirmNameMismatch}
              </p>
            ) : null}
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={closeDeleteDialogs}>
              {labels.cancel}
            </Button>
            <Button
              className="bg-kw-c-ef2334 text-white hover:bg-kw-c-d91f30"
              disabled={!confirmMatches}
              type="button"
              onClick={() => setDeleteStep("impact")}
            >
              {labels.delete}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={deleteStep === "impact"} onOpenChange={closeDeleteDialogs}>
        <DialogContent className="z-[1300]">
          <DialogHeader>
            <div className="flex items-center gap-3 text-kw-c-ef2334">
              <AlertTriangle className="size-6" />
              <DialogTitle>{labels.deleteStaffImpactTitle}</DialogTitle>
            </div>
            <DialogDescription>
              {labels.deleteStaffImpactDescription}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={closeDeleteDialogs}>
              {labels.cancel}
            </Button>
            <Button
              className="bg-kw-c-ef2334 text-white hover:bg-kw-c-d91f30"
              disabled={deleting}
              type="button"
              onClick={handleDelete}
            >
              {deleting ? labels.deleting : labels.continueDelete}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function BranchPills({
  activeBranchID,
  branches,
  labels,
  onChange,
}: {
  activeBranchID: string;
  branches: Branch[];
  labels: ReceptionistManagementLabels;
  onChange: (branchID: string) => void;
}) {
  return (
    <section className="flex flex-wrap items-center gap-3">
      <BranchPill active={!activeBranchID} onClick={() => onChange("")}>
        {labels.allBranches}
      </BranchPill>
      {branches.map((branch) => (
        <BranchPill
          active={activeBranchID === branch.id}
          key={branch.id}
          onClick={() => onChange(branch.id)}
        >
          {branch.name}
        </BranchPill>
      ))}
    </section>
  );
}

function BranchPill({
  active,
  children,
  onClick,
}: {
  active: boolean;
  children: ReactNode;
  onClick: () => void;
}) {
  return (
    <button
      className={cn(
        "inline-flex min-h-10 cursor-pointer items-center rounded-full border border-kw-c-dce3ee bg-white px-5 text-sm font-bold text-kw-c-0a284b shadow-sm transition-all hover:-translate-y-0.5 hover:border-kw-c-ef2334/40 hover:text-kw-c-ef2334 dark:border-kw-c-3a4658 dark:bg-kw-c-17243a dark:text-kw-c-f3f6fa dark:hover:border-kw-c-ff3b4f/50 dark:hover:text-kw-c-ff5a69",
        active &&
          "border-kw-c-ef2334 bg-kw-c-ef2334 text-white shadow-kw-red-glow hover:text-white dark:border-kw-c-ff3b4f dark:bg-kw-c-ff3b4f dark:text-white kw-dark-shadow-red-glow dark:hover:text-white",
      )}
      type="button"
      onClick={onClick}
    >
      {children}
    </button>
  );
}

function InlineMeta({
  className,
  icon,
  layoutId,
  value,
}: {
  className?: string;
  icon: ReactNode;
  layoutId?: string;
  value: string;
}) {
  return (
    <motion.div
      className={cn(
        "flex min-w-0 items-center gap-2 text-sm font-bold text-kw-c-59667a dark:text-kw-c-a7b0bf",
        className,
      )}
      layoutId={layoutId}
      transition={sharedMotionTransition}
    >
      <span className="shrink-0 text-kw-c-ef2334 dark:text-kw-c-ff3b4f">
        {icon}
      </span>
      <span className="truncate">{value}</span>
    </motion.div>
  );
}
