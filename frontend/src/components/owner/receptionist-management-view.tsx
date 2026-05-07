"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import type { ReactNode } from "react";
import { AnimatePresence, LayoutGroup, motion } from "framer-motion";
import {
  AlertTriangle,
  BriefcaseBusiness,
  CalendarDays,
  CheckCircle2,
  Edit3,
  Eye,
  EyeOff,
  ImagePlus,
  Mail,
  Phone,
  Plus,
  Save,
  Trash2,
  UserRound,
  X,
  XCircle,
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
import { Textarea } from "@/components/ui/textarea";
import type { Branch, StaffMember } from "@/lib/api/types";
import {
  checkStaffEmailAvailabilityAction,
  createStaffAction,
  deleteStaffAction,
  updateStaffAction,
  type StaffManagementState,
} from "@/lib/owner/actions";
import { cn } from "@/lib/utils";

type StaffGender = "" | "male" | "female" | "other";
type FormMode = "create" | "edit";
type StaffSharedLayoutIDs = {
  avatar: string;
  branch: string;
  gender: string;
  hours: string;
  name: string;
  salary: string;
  status: string;
};

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

type StaffFormValues = {
  address: string;
  birthDate: string;
  branchID: string;
  email: string;
  firstName: string;
  gender: StaffGender;
  hiredAt: string;
  isActive: boolean;
  lastName: string;
  password: string;
  phone: string;
  profilePhotoFileID: string;
  removePhoto: boolean;
  salary: string;
  staffID: string;
};

const initialActionState: StaffManagementState = {};
const MAX_STAFF_PHOTO_BYTES = 10 * 1024 * 1024;
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
const quickHideMotionTransition = {
  duration: 0.22,
  ease: [0.22, 1, 0.36, 1],
} as const;
const shellCollapseTransition = {
  duration: 0.56,
  ease: [0.22, 1, 0.36, 1],
} as const;
const emptyStaffForm: StaffFormValues = {
  address: "",
  birthDate: "",
  branchID: "",
  email: "",
  firstName: "",
  gender: "",
  hiredAt: "",
  isActive: true,
  lastName: "",
  password: "",
  phone: "",
  profilePhotoFileID: "",
  removePhoto: false,
  salary: "",
  staffID: "",
};

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
        <p className="mt-1 text-sm text-[#59667a] dark:text-[#a7b0bf]">
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
          <div className="rounded-xl border border-dashed border-[#cbd5e1] p-8 text-center text-sm font-bold text-[#59667a] dark:border-[#3a4658] dark:text-[#a7b0bf]">
            {labels.noReceptionists}
          </div>
        )}
      </section>

      <section className="space-y-4">
        {warning ? (
          <div className="rounded-lg border border-[#f87171] bg-[#3b1218] px-4 py-3 text-sm font-bold text-[#ffe4e6]">
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
            className="h-11 gap-2 rounded-lg bg-[#ef2334] px-6 font-black text-white transition hover:bg-[#d91f30] disabled:cursor-not-allowed disabled:opacity-50 dark:bg-[#ff3b4f] dark:hover:bg-[#ff5a69]"
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
    const result = await deleteStaffAction(initialActionState, formData);
    setDeleting(false);
    if (result.staff) {
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
          ref={shellRef}
          transition={shellTransition}
        >
          <Card className="overflow-hidden rounded-xl border-white/55 bg-white/75 shadow-[0_18px_50px_rgba(10,40,75,0.1)] backdrop-blur-xl transition-all duration-300 dark:border-white/10 dark:bg-[#1b2635]/72">
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
                        <Avatar className="size-12 border border-[#dce3ee] dark:border-[#3a4658]">
                          <AvatarImage
                            alt={fullName}
                            src={staff.profile_photo_url}
                          />
                          <AvatarFallback className="bg-[#edf2f7] font-black text-[#0a284b] dark:bg-[#2a3444] dark:text-[#f3f6fa]">
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
                        <p className="truncate text-xs font-semibold text-[#59667a] dark:text-[#a7b0bf]">
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
                        className="size-10 rounded-full border-[#ef2334] text-[#ef2334] hover:bg-[#ef2334]/10 hover:text-[#ef2334] dark:border-[#ff3b4f] dark:text-[#ff3b4f] dark:hover:bg-[#ff3b4f]/10"
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
              <p className="text-sm font-bold text-[#ef2334]">
                {labels.confirmNameMismatch}
              </p>
            ) : null}
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={closeDeleteDialogs}>
              {labels.cancel}
            </Button>
            <Button
              className="bg-[#ef2334] text-white hover:bg-[#d91f30]"
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
            <div className="flex items-center gap-3 text-[#ef2334]">
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
              className="bg-[#ef2334] text-white hover:bg-[#d91f30]"
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
        "inline-flex min-h-10 cursor-pointer items-center rounded-full border border-[#dce3ee] bg-white px-5 text-sm font-bold text-[#0a284b] shadow-sm transition-all hover:-translate-y-0.5 hover:border-[#ef2334]/40 hover:text-[#ef2334] dark:border-[#3a4658] dark:bg-[#17243a] dark:text-[#f3f6fa] dark:hover:border-[#ff3b4f]/50 dark:hover:text-[#ff5a69]",
        active &&
          "border-[#ef2334] bg-[#ef2334] text-white shadow-[0_0_22px_rgba(239,35,52,0.35)] hover:text-white dark:border-[#ff3b4f] dark:bg-[#ff3b4f] dark:text-white dark:shadow-[0_0_24px_rgba(255,59,79,0.35)] dark:hover:text-white",
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
        "flex min-w-0 items-center gap-2 text-sm font-bold text-[#59667a] dark:text-[#a7b0bf]",
        className,
      )}
      layoutId={layoutId}
      transition={sharedMotionTransition}
    >
      <span className="shrink-0 text-[#ef2334] dark:text-[#ff3b4f]">
        {icon}
      </span>
      <span className="truncate">{value}</span>
    </motion.div>
  );
}

function StaffEditor({
  branches,
  closing = false,
  embedded = false,
  initialStaff,
  labels,
  layoutIDs,
  mode,
  selectedBranch,
  onCancel,
  onSaved,
}: {
  branches: Branch[];
  closing?: boolean;
  embedded?: boolean;
  initialStaff?: StaffMember;
  labels: ReceptionistManagementLabels;
  layoutIDs?: StaffSharedLayoutIDs;
  mode: FormMode;
  selectedBranch?: Branch;
  onCancel: () => void;
  onSaved: (staff: StaffMember) => void;
}) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const objectURLRef = useRef<string | null>(null);
  const [photoPreview, setPhotoPreview] = useState(
    initialStaff?.profile_photo_url ?? "",
  );
  const [idempotencyKey, setIdempotencyKey] = useState(() =>
    crypto.randomUUID(),
  );
  const [showPassword, setShowPassword] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const submittingRef = useRef(false);
  const [state, setState] = useState<StaffManagementState>({});
  const [emailStatus, setEmailStatus] = useState<
    "idle" | "checking" | "available" | "taken" | "invalid"
  >(initialStaff ? "available" : "idle");
  const [values, setValues] = useState<StaffFormValues>(() =>
    initialStaff
      ? valuesFromStaff(initialStaff)
      : {
          ...emptyStaffForm,
          branchID: branches[0]?.id ?? "",
        },
  );
  const activeBranch = useMemo(
    () =>
      selectedBranch?.id === values.branchID
        ? selectedBranch
        : branches.find((branch) => branch.id === values.branchID),
    [branches, selectedBranch, values.branchID],
  );

  useEffect(() => {
    return () => {
      if (objectURLRef.current) {
        URL.revokeObjectURL(objectURLRef.current);
      }
    };
  }, []);

  useEffect(() => {
    if (emailStatus !== "checking" || !values.email.trim()) {
      return;
    }
    const timeout = window.setTimeout(async () => {
      const result = await checkStaffEmailAvailabilityAction(values.email);
      setEmailStatus(result.available ? "available" : "taken");
    }, 350);
    return () => window.clearTimeout(timeout);
  }, [emailStatus, values.email]);

  const hasInvalidHireDate =
    isValidDateString(values.birthDate) &&
    isValidDateString(values.hiredAt) &&
    !isHireDateOnOrAfterBirthDate(values.birthDate, values.hiredAt);
  const canSubmit =
    values.branchID &&
    values.firstName.trim() &&
    values.lastName.trim() &&
    isValidDateString(values.birthDate) &&
    isValidDateString(values.hiredAt) &&
    !hasInvalidHireDate &&
    values.phone.trim() &&
    values.salary.trim() &&
    emailStatus === "available" &&
    (mode === "edit"
      ? !values.password || values.password.length >= 8
      : values.password.length >= 8);

  function updateValue<K extends keyof StaffFormValues>(
    key: K,
    value: StaffFormValues[K],
  ) {
    setValues((current) => ({ ...current, [key]: value }));
  }

  function handleEmailChange(value: string) {
    updateValue("email", value);
    const normalized = value.trim();
    if (
      mode === "edit" &&
      normalized.toLocaleLowerCase("en-US") ===
        (initialStaff?.email ?? "").toLocaleLowerCase("en-US")
    ) {
      setEmailStatus("available");
      return;
    }
    if (!normalized) {
      setEmailStatus("idle");
      return;
    }
    setEmailStatus(/^\S+@\S+\.\S+$/.test(normalized) ? "checking" : "invalid");
  }

  function handlePhotoChange(file: File | null) {
    if (objectURLRef.current) {
      URL.revokeObjectURL(objectURLRef.current);
      objectURLRef.current = null;
    }
    if (!file) {
      setPhotoPreview("");
      return;
    }
    const nextURL = URL.createObjectURL(file);
    objectURLRef.current = nextURL;
    updateValue("removePhoto", false);
    setPhotoPreview(nextURL);
  }

  function clearPhoto() {
    if (fileInputRef.current) {
      fileInputRef.current.value = "";
    }
    if (objectURLRef.current) {
      URL.revokeObjectURL(objectURLRef.current);
      objectURLRef.current = null;
    }
    setPhotoPreview("");
    updateValue("profilePhotoFileID", "");
    updateValue("removePhoto", true);
  }

  async function submit() {
    if (!canSubmit || submitting || submittingRef.current) {
      return;
    }
    submittingRef.current = true;
    setSubmitting(true);
    setState({});

    const formData = new FormData();
    formData.set("address", values.address);
    formData.set("birth_date", values.birthDate);
    formData.set("branch_id", values.branchID);
    formData.set("email", values.email);
    formData.set("first_name", values.firstName);
    formData.set("gender", values.gender);
    formData.set("hired_at", values.hiredAt);
    formData.set("idempotency_key", idempotencyKey);
    formData.set("is_active", values.isActive ? "1" : "0");
    formData.set("last_name", values.lastName);
    formData.set("password", values.password);
    formData.set("phone", values.phone);
    formData.set("profile_photo_file_id", values.profilePhotoFileID);
    formData.set("remove_photo", values.removePhoto ? "1" : "0");
    formData.set("salary", values.salary);
    formData.set("staff_id", values.staffID);
    const file = fileInputRef.current?.files?.[0];
    if (file) {
      if (file.size > MAX_STAFF_PHOTO_BYTES) {
        submittingRef.current = false;
        setSubmitting(false);
        setState({ error: "photo_too_large" });
        return;
      }
      formData.set("photo", file);
    }

    try {
      const result =
        mode === "edit"
          ? await updateStaffAction(initialActionState, formData)
          : await createStaffAction(initialActionState, formData);
      submittingRef.current = false;
      setSubmitting(false);
      if (result.staff) {
        setIdempotencyKey(crypto.randomUUID());
        onSaved(result.staff);
        return;
      }
      setState(result);
    } catch {
      submittingRef.current = false;
      setSubmitting(false);
      setState({ error: "backend" });
    }
  }

  const content = (
    <div className="space-y-5">
      {!embedded ? (
        <h2 className="text-xl font-black">
          {mode === "edit" ? labels.editStaff : labels.addStaff}
        </h2>
      ) : null}

      {state.error && !(embedded && closing) ? (
        <StaffFormError labels={labels} state={state} />
      ) : null}

      <div className="flex justify-center">
        <motion.div
          className="relative size-36"
          layoutId={layoutIDs?.avatar}
          transition={sharedMotionTransition}
        >
          <label
            className="group flex size-36 cursor-pointer items-center justify-center overflow-hidden rounded-full border border-dashed border-[#b9c5d6] bg-[#f7f8fb] text-[#0a284b] shadow-inner transition hover:border-[#ef2334] hover:bg-[#f1f4f8] dark:border-[#3a4658] dark:bg-[#202b3a] dark:text-[#f3f6fa] dark:hover:border-[#ff3b4f] dark:hover:bg-[#263448]"
          >
            {photoPreview ? (
              // eslint-disable-next-line @next/next/no-img-element
              <img alt="" className="size-full object-cover" src={photoPreview} />
            ) : (
              <ImagePlus className="size-8 transition-transform group-hover:scale-110" />
            )}
            <input
              ref={fileInputRef}
              accept="image/jpeg,image/png,image/webp,.jpg,.jpeg,.png,.webp"
              className="sr-only"
              name="photo"
              type="file"
              onChange={(event) =>
                handlePhotoChange(event.currentTarget.files?.[0] ?? null)
              }
            />
          </label>
          {photoPreview && !(embedded && closing) ? (
            <button
              aria-label="Remove photo"
              className="absolute right-0 top-0 grid size-8 cursor-pointer place-items-center rounded-full bg-[#ef2334] text-white shadow-lg ring-4 ring-white transition hover:bg-[#d91f30] dark:bg-[#ff3b4f] dark:ring-[#1b2635] dark:hover:bg-[#ff5a69]"
              type="button"
              onClick={clearPhoto}
            >
              <X className="size-4" />
            </button>
          ) : null}
        </motion.div>
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <motion.div
          className="grid gap-4 lg:col-span-2 lg:grid-cols-2"
          layoutId={layoutIDs?.name}
          transition={sharedMotionTransition}
        >
          <Field label={labels.name} required reveal={embedded && !layoutIDs?.name}>
            <Input
              className="h-11"
              value={values.firstName}
              onChange={(event) => updateValue("firstName", event.target.value)}
            />
          </Field>
          <Field label={labels.surname} required reveal={embedded && !layoutIDs?.name}>
            <Input
              className="h-11"
              value={values.lastName}
              onChange={(event) => updateValue("lastName", event.target.value)}
            />
          </Field>
        </motion.div>
        <Field
          label={labels.assignedBranch}
          layoutId={layoutIDs?.branch}
          required
          reveal={embedded && !layoutIDs?.branch}
        >
          <select
            className={selectClassName}
            value={values.branchID}
            onChange={(event) => updateValue("branchID", event.target.value)}
          >
            {branches.map((branch) => (
              <option key={branch.id} value={branch.id}>
                {branch.name}
              </option>
            ))}
          </select>
        </Field>
        <Field
          closingHidden={embedded && closing}
          label={labels.birthDate}
          required
          reveal={embedded}
        >
          <Input
            className="h-11"
            placeholder="DD/MM/YYYY"
            value={values.birthDate}
            onChange={(event) =>
              updateValue("birthDate", formatDateInput(event.target.value))
            }
          />
        </Field>
        <Field
          label={labels.gender}
          layoutId={layoutIDs?.gender}
          reveal={embedded && !layoutIDs?.gender}
        >
          <select
            className={selectClassName}
            value={values.gender}
            onChange={(event) =>
              updateValue("gender", event.target.value as StaffGender)
            }
          >
            <option value="">{labels.genderPlaceholder}</option>
            <option value="male">{labels.male}</option>
            <option value="female">{labels.female}</option>
            <option value="other">{labels.other}</option>
          </select>
        </Field>
        <Field
          label={labels.status}
          layoutId={layoutIDs?.status}
          reveal={embedded && !layoutIDs?.status}
        >
          <select
            className={selectClassName}
            value={values.isActive ? "active" : "inactive"}
            onChange={(event) =>
              updateValue("isActive", event.target.value === "active")
            }
          >
            <option value="active">{labels.active}</option>
            <option value="inactive">{labels.inactive}</option>
          </select>
        </Field>
        <Field
          label={labels.workingHours}
          layoutId={layoutIDs?.hours}
          reveal={embedded && !layoutIDs?.hours}
        >
          <Input
            className="h-11"
            disabled
            value={`${activeBranch?.opening_time || "09:00"} - ${activeBranch?.closing_time || "18:00"}`}
          />
        </Field>
        <Field
          closingHidden={embedded && closing}
          label={labels.phone}
          required
          reveal={embedded}
        >
          <div className="relative">
            <Phone className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-[#7e8ea5]" />
            <Input
              className="h-11 pl-10"
              value={values.phone}
              onChange={(event) => updateValue("phone", event.target.value)}
            />
          </div>
        </Field>
        <Field
          closingHidden={embedded && closing}
          label={labels.address}
          reveal={embedded}
        >
          <Textarea
            className="h-11 min-h-11 resize-none py-2"
            value={values.address}
            onChange={(event) => updateValue("address", event.target.value)}
          />
        </Field>
        <Field
          label={labels.salary}
          layoutId={layoutIDs?.salary}
          required
          reveal={embedded && !layoutIDs?.salary}
        >
          <div className="relative">
            <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-base font-black text-[#7e8ea5]">
              {"\u20bc"}
            </span>
            <Input
              className="h-11 pl-10"
              inputMode="numeric"
              value={values.salary}
              onChange={(event) =>
                updateValue("salary", event.target.value.replace(/\D/g, ""))
              }
            />
          </div>
        </Field>
        <Field
          closingHidden={embedded && closing}
          label={labels.hiredAt}
          required
          reveal={embedded}
        >
          <Input
            className="h-11"
            placeholder="DD/MM/YYYY"
            value={values.hiredAt}
            onChange={(event) =>
              updateValue("hiredAt", formatDateInput(event.target.value))
            }
          />
          {hasInvalidHireDate ? (
            <p className="text-xs font-bold text-[#ef2334] dark:text-[#ff5a69]">
              {labels.hireDateBeforeBirthError}
            </p>
          ) : null}
        </Field>
        <Field
          closingHidden={embedded && closing}
          label={labels.email}
          required
          reveal={embedded}
        >
          <div className="relative">
            <Mail className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-[#7e8ea5]" />
            <Input
              className={cn(
                "h-11 pl-10 pr-10",
                emailStatus === "taken" &&
                  "border-[#ef2334] dark:border-[#ff3b4f]",
              )}
              value={values.email}
              onChange={(event) => handleEmailChange(event.target.value)}
            />
            <EmailStatusIcon status={emailStatus} />
          </div>
          <EmailStatusText labels={labels} status={emailStatus} />
        </Field>
        <Field
          closingHidden={embedded && closing}
          label={labels.password}
          required={mode === "create"}
          reveal={embedded}
        >
          <div className="relative">
            <Input
              className="h-11 pr-10"
              type={showPassword ? "text" : "password"}
              value={values.password}
              onChange={(event) => updateValue("password", event.target.value)}
            />
            <button
              aria-label={showPassword ? labels.hidePassword : labels.showPassword}
              className="absolute right-3 top-1/2 -translate-y-1/2 cursor-pointer text-[#7e8ea5] transition hover:text-[#ef2334] dark:hover:text-[#ff3b4f]"
              type="button"
              onClick={() => setShowPassword((current) => !current)}
            >
              {showPassword ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
            </button>
          </div>
        </Field>
      </div>

      <motion.div
        animate={
          embedded && closing ? { opacity: 0, y: 8 } : { opacity: 1, y: 0 }
        }
        className="flex justify-center gap-3"
        transition={quickHideMotionTransition}
      >
        <Button
          className="h-11 min-w-32 rounded-lg font-bold"
          type="button"
          variant="outline"
          onClick={onCancel}
        >
          {labels.cancel}
        </Button>
        <Button
          className="h-11 min-w-36 rounded-lg bg-[#079669] font-black text-white transition hover:bg-[#05865d] disabled:cursor-not-allowed disabled:opacity-50"
          disabled={!canSubmit || submitting}
          type="button"
          onClick={submit}
        >
          <Save className="size-4" />
          {submitting ? labels.saving : labels.save}
        </Button>
      </motion.div>
    </div>
  );

  if (embedded) {
    return content;
  }

  return (
    <Card className="rounded-xl border-white/55 bg-white/75 shadow-[0_24px_70px_rgba(10,40,75,0.12)] backdrop-blur-xl dark:border-white/10 dark:bg-[#1b2635]/72">
      <CardContent className="p-6">{content}</CardContent>
    </Card>
  );
}

function Field({
  children,
  closingHidden,
  label,
  layoutId,
  reveal,
  required,
}: {
  children: ReactNode;
  closingHidden?: boolean;
  label: string;
  layoutId?: string;
  reveal?: boolean;
  required?: boolean;
}) {
  const shouldHide = Boolean(closingHidden && !layoutId);

  return (
    <motion.div
      animate={
        shouldHide
          ? { opacity: 0, y: -8 }
          : reveal
            ? { opacity: 1, y: 0 }
            : undefined
      }
      className={cn(
        "space-y-2",
      )}
      initial={reveal && !shouldHide ? { opacity: 0, y: 16 } : false}
      layoutId={layoutId}
      transition={
        shouldHide
          ? quickHideMotionTransition
          : layoutId
            ? sharedMotionTransition
            : revealMotionTransition
      }
    >
      <Label className="font-black">
        {label}
        {required ? <span className="ml-1 text-[#ef2334]">*</span> : null}
      </Label>
      {children}
    </motion.div>
  );
}

function StaffFormError({
  labels,
  state,
}: {
  labels: ReceptionistManagementLabels;
  state: StaffManagementState;
}) {
  const message =
    state.error === "duplicate"
      ? labels.staffDuplicateError
      : state.error === "invalid_hire_date"
        ? labels.hireDateBeforeBirthError
      : state.error === "invalid_input"
        ? labels.staffInvalidInputError
        : state.error === "invalid_photo"
          ? labels.invalidPhotoError
          : state.error === "photo_too_large"
            ? labels.photoTooLargeError
            : labels.staffSaveError;

  return (
    <div className="rounded-lg border border-[#f87171] bg-[#3b1218] px-4 py-3 text-sm font-bold text-[#ffe4e6]">
      {message}
    </div>
  );
}

function EmailStatusIcon({
  status,
}: {
  status: "idle" | "checking" | "available" | "taken" | "invalid";
}) {
  if (status === "available") {
    return (
      <CheckCircle2 className="absolute right-3 top-1/2 size-5 -translate-y-1/2 text-[#0ca678]" />
    );
  }
  if (status === "taken" || status === "invalid") {
    return (
      <XCircle className="absolute right-3 top-1/2 size-5 -translate-y-1/2 text-[#ef2334] dark:text-[#ff3b4f]" />
    );
  }
  return null;
}

function EmailStatusText({
  labels,
  status,
}: {
  labels: ReceptionistManagementLabels;
  status: "idle" | "checking" | "available" | "taken" | "invalid";
}) {
  if (status === "checking") {
    return (
      <p className="text-xs font-bold text-[#7e8ea5]">
        {labels.emailChecking}
      </p>
    );
  }
  if (status === "available") {
    return (
      <p className="text-xs font-bold text-[#0ca678]">
        {labels.emailAvailable}
      </p>
    );
  }
  if (status === "taken") {
    return (
      <p className="text-xs font-bold text-[#ef2334]">{labels.emailTaken}</p>
    );
  }
  return null;
}

function valuesFromStaff(staff: StaffMember): StaffFormValues {
  return {
    address: staff.address ?? "",
    birthDate: staff.birth_date ?? "",
    branchID: staff.branch_id,
    email: staff.email,
    firstName: staff.first_name,
    gender: (staff.gender ?? "") as StaffGender,
    hiredAt: staff.hired_at ?? "",
    isActive: staff.is_active,
    lastName: staff.last_name,
    password: "",
    phone: staff.phone ?? "",
    profilePhotoFileID: staff.profile_photo_file_id ?? "",
    removePhoto: false,
    salary: String(staff.salary_amount_azn ?? 0),
    staffID: staff.id,
  };
}

function StatusBadge({
  isActive,
  labels,
}: {
  isActive: boolean;
  labels: ReceptionistManagementLabels;
}) {
  return (
    <span
      className={cn(
        "inline-flex h-8 items-center rounded-full border px-3 text-xs font-black",
        isActive
          ? "border-[#0ca678]/30 bg-[#0ca678] text-white"
          : "border-[#64748b]/30 bg-[#64748b] text-white",
      )}
    >
      {isActive ? labels.active : labels.inactive}
    </span>
  );
}

function genderLabel(value: string | undefined, labels: ReceptionistManagementLabels) {
  switch (value) {
    case "male":
      return labels.male;
    case "female":
      return labels.female;
    case "other":
      return labels.other;
    default:
      return "-";
  }
}

function initials(firstName: string, lastName: string) {
  return `${firstName[0] ?? ""}${lastName[0] ?? ""}`.toUpperCase();
}

function formatDateInput(value: string) {
  const digits = value.replace(/\D/g, "").slice(0, 8);
  if (digits.length <= 2) {
    return digits;
  }
  if (digits.length <= 4) {
    return `${digits.slice(0, 2)}/${digits.slice(2)}`;
  }
  return `${digits.slice(0, 2)}/${digits.slice(2, 4)}/${digits.slice(4)}`;
}

function isValidDateString(value: string) {
  const match = /^(\d{2})\/(\d{2})\/(\d{4})$/.exec(value);
  if (!match) {
    return false;
  }
  const day = Number(match[1]);
  const month = Number(match[2]);
  const year = Number(match[3]);
  const date = new Date(Date.UTC(year, month - 1, day));

  return (
    date.getUTCFullYear() === year &&
    date.getUTCMonth() === month - 1 &&
    date.getUTCDate() === day
  );
}

function isHireDateOnOrAfterBirthDate(birthDate: string, hiredAt: string) {
  const birth = parseDateString(birthDate);
  const hired = parseDateString(hiredAt);
  if (!birth || !hired) {
    return true;
  }

  return hired.getTime() >= birth.getTime();
}

function parseDateString(value: string) {
  const match = /^(\d{2})\/(\d{2})\/(\d{4})$/.exec(value);
  if (!match) {
    return null;
  }
  const day = Number(match[1]);
  const month = Number(match[2]);
  const year = Number(match[3]);
  const date = new Date(Date.UTC(year, month - 1, day));
  if (
    date.getUTCFullYear() !== year ||
    date.getUTCMonth() !== month - 1 ||
    date.getUTCDate() !== day
  ) {
    return null;
  }

  return date;
}

const selectClassName =
  "h-11 w-full rounded-md border border-input bg-transparent px-3 text-sm font-semibold text-[#0a284b] shadow-xs outline-none transition focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 dark:border-[#64748b] dark:bg-[#1b2635] dark:text-[#f3f6fa]";
