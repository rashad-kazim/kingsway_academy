import { useEffect, useMemo, useRef, useState } from "react";
import type { ReactNode } from "react";
import { motion } from "framer-motion";
import {
  Eye,
  EyeOff,
  ImagePlus,
  Mail,
  Phone,
  Save,
  X,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { ObjectCoverImage } from "@/components/owner/shared/object-cover-image";
import {
  EmailStatusIcon,
  EmailStatusText,
  StaffFormError,
} from "@/components/owner/receptionist-management-parts";
import type { Branch, StaffMember } from "@/lib/api/types";
import {
  checkStaffEmailAvailabilityAction,
  createStaffAction,
  updateStaffAction,
  type StaffManagementState,
} from "@/lib/owner/actions";
import { formatShortDateInput } from "@/lib/forms/date-format";
import { cn } from "@/lib/utils";
import type { ReceptionistManagementLabels } from "./receptionist-management-view";

type StaffGender = "" | "male" | "female" | "other";
export type FormMode = "create" | "edit";
export type StaffSharedLayoutIDs = {
  avatar: string;
  branch: string;
  gender: string;
  hours: string;
  name: string;
  salary: string;
  status: string;
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
const MAX_STAFF_PHOTO_BYTES = 15 * 1024 * 1024;
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
export function StaffEditor({
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
    <div className="space-y-5" data-testid={`staff-editor-${mode}`}>
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
            className="group relative flex size-36 cursor-pointer items-center justify-center overflow-hidden rounded-full border border-dashed border-kw-c-b9c5d6 bg-kw-c-f7f8fb text-kw-c-0a284b shadow-inner transition hover:border-kw-c-ef2334 hover:bg-kw-c-f1f4f8 dark:border-kw-c-3a4658 dark:bg-kw-c-202b3a dark:text-kw-c-f3f6fa dark:hover:border-kw-c-ff3b4f dark:hover:bg-kw-c-263448"
          >
            {photoPreview ? (
              <ObjectCoverImage src={photoPreview} />
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
              className="absolute right-0 top-0 grid size-8 cursor-pointer place-items-center rounded-full bg-kw-c-ef2334 text-white shadow-lg ring-4 ring-white transition hover:bg-kw-c-d91f30 dark:bg-kw-c-ff3b4f dark:ring-kw-c-1b2635 dark:hover:bg-kw-c-ff5a69"
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
              data-testid="staff-first-name"
              value={values.firstName}
              onChange={(event) => updateValue("firstName", event.target.value)}
            />
          </Field>
          <Field label={labels.surname} required reveal={embedded && !layoutIDs?.name}>
            <Input
              className="h-11"
              data-testid="staff-last-name"
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
            data-testid="staff-branch-id"
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
            data-testid="staff-birth-date"
            placeholder="DD/MM/YYYY"
            value={values.birthDate}
            onChange={(event) =>
              updateValue("birthDate", formatShortDateInput(event.target.value))
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
            data-testid="staff-gender"
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
            data-testid="staff-status"
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
            <Phone className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-kw-c-7e8ea5" />
            <Input
              className="h-11 pl-10"
              data-testid="staff-phone"
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
            data-testid="staff-address"
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
            <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-base font-black text-kw-c-7e8ea5">
              {"\u20bc"}
            </span>
            <Input
              className="h-11 pl-10"
              data-testid="staff-salary"
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
            data-testid="staff-hired-at"
            placeholder="DD/MM/YYYY"
            value={values.hiredAt}
            onChange={(event) =>
              updateValue("hiredAt", formatShortDateInput(event.target.value))
            }
          />
          {hasInvalidHireDate ? (
            <p className="text-xs font-bold text-kw-c-ef2334 dark:text-kw-c-ff5a69">
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
            <Mail className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-kw-c-7e8ea5" />
            <Input
              className={cn(
                "h-11 pl-10 pr-10",
                emailStatus === "taken" &&
                  "border-kw-c-ef2334 dark:border-kw-c-ff3b4f",
              )}
              data-testid="staff-email"
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
              data-testid="staff-password"
              type={showPassword ? "text" : "password"}
              value={values.password}
              onChange={(event) => updateValue("password", event.target.value)}
            />
            <button
              aria-label={showPassword ? labels.hidePassword : labels.showPassword}
              className="absolute right-3 top-1/2 -translate-y-1/2 cursor-pointer text-kw-c-7e8ea5 transition hover:text-kw-c-ef2334 dark:hover:text-kw-c-ff3b4f"
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
          className="h-11 min-w-36 rounded-lg bg-kw-c-079669 font-black text-white transition hover:bg-kw-c-05865d disabled:cursor-not-allowed disabled:opacity-50"
          data-testid="staff-save"
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
    <Card className="rounded-xl border-white/55 bg-white/75 shadow-kw-panel backdrop-blur-xl dark:border-white/10 dark:bg-kw-c-1b2635/72">
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
        {required ? <span className="ml-1 text-kw-c-ef2334">*</span> : null}
      </Label>
      {children}
    </motion.div>
  );
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
  "h-11 w-full rounded-md border border-input bg-transparent px-3 text-sm font-semibold text-kw-c-0a284b shadow-xs outline-none transition focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 dark:border-kw-c-64748b dark:bg-kw-c-1b2635 dark:text-kw-c-f3f6fa";
