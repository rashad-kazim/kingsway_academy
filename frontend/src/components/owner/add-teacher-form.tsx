"use client";

import { useActionState, useEffect, useMemo, useRef, useState } from "react";
import type { ReactNode } from "react";
import { ChevronDown, Eye, EyeOff } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  EmailAvailabilityIcon,
  EmailAvailabilityText,
  type EmailAvailabilityStatus,
} from "@/components/owner/shared/email-availability";
import { LockedSubmitButton } from "@/components/owner/shared/locked-submit-button";
import { PhotoUploadAvatar } from "@/components/owner/shared/photo-upload-avatar";
import type { Branch, SalaryModelType } from "@/lib/api/types";
import {
  checkTeacherEmailAvailabilityAction,
  createTeacherAction,
  type CreateTeacherState,
  updateTeacherAction,
} from "@/lib/teacher-finance/actions";

export type AddTeacherFormLabels = {
  addTeacher: string;
  addTeacherDescription: string;
  edit: string;
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
  subject: string;
  subjectSelection: string;
  subjectDropdownPlaceholder: string;
  salaryModel: string;
  salaryAmount: string;
  percentage: string;
  phoneNumber: string;
  address: string;
  email: string;
  password: string;
  save: string;
  saving: string;
  cancel: string;
  emailAvailable: string;
  emailTaken: string;
  emailChecking: string;
  requiredFieldsError: string;
  createBackendError: string;
  createUnauthorizedError: string;
  invalidPhotoError: string;
  photoTooLargeError: string;
  salaryTypes: Record<SalaryModelType, string>;
  defaultSubjects: string[];
};

type AddTeacherFormProps = {
  branches: Branch[];
  initialTeacher?: TeacherFormInitialData;
  labels: AddTeacherFormLabels;
  locale: string;
};

export type TeacherFormInitialData = {
  id: string;
  address?: string;
  birth_date?: string;
  branch_id: string;
  email: string;
  first_name: string;
  gender?: string;
  last_name: string;
  percentage?: string;
  phone?: string;
  profile_photo_file_id?: string;
  profile_photo_url?: string;
  salary_amount?: string;
  salary_model?: SalaryModelType;
  subjects: string[];
};

const initialState: CreateTeacherState = {};

export function AddTeacherForm({
  branches,
  initialTeacher,
  labels,
  locale,
}: AddTeacherFormProps) {
  const editing = Boolean(initialTeacher);
  const [state, formAction] = useActionState(
    editing ? updateTeacherAction : createTeacherAction,
    initialState,
  );
  const idempotencyKey = useMemo(() => crypto.randomUUID(), []);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const objectURLRef = useRef<string | null>(null);
  const [photoPreview, setPhotoPreview] = useState(
    initialTeacher?.profile_photo_url ?? "",
  );
  const [photoRemoved, setPhotoRemoved] = useState(false);
  const [firstName, setFirstName] = useState(initialTeacher?.first_name ?? "");
  const [lastName, setLastName] = useState(initialTeacher?.last_name ?? "");
  const [birthDate, setBirthDate] = useState(initialTeacher?.birth_date ?? "");
  const [branchID, setBranchID] = useState(
    initialTeacher?.branch_id ?? branches[0]?.id ?? "",
  );
  const [subjects, setSubjects] = useState<string[]>(
    initialTeacher?.subjects ?? [],
  );
  const [salaryModel, setSalaryModel] = useState<SalaryModelType>(
    initialTeacher?.salary_model ?? "fixed",
  );
  const [salaryAmount, setSalaryAmount] = useState(
    initialTeacher?.salary_amount ?? "",
  );
  const [percentage, setPercentage] = useState(initialTeacher?.percentage ?? "");
  const [phone, setPhone] = useState(initialTeacher?.phone ?? "");
  const [email, setEmail] = useState(initialTeacher?.email ?? "");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [emailStatus, setEmailStatus] = useState<EmailAvailabilityStatus>(
    editing ? "available" : "idle",
  );
  const subjectOptions = useMemo(
    () => [
      ...new Set([...(labels.defaultSubjects ?? []), ...subjects].filter(Boolean)),
    ],
    [labels.defaultSubjects, subjects],
  );

  useEffect(() => {
    return () => {
      if (objectURLRef.current) {
        URL.revokeObjectURL(objectURLRef.current);
      }
    };
  }, []);

  useEffect(() => {
    const normalized = email.trim();
    if (emailStatus !== "checking" || !normalized) {
      return;
    }
    const timeout = window.setTimeout(async () => {
      const result = await checkTeacherEmailAvailabilityAction(normalized);
      setEmailStatus(result.available ? "available" : "taken");
    }, 350);

    return () => window.clearTimeout(timeout);
  }, [email, emailStatus]);

  const canSubmit = useMemo(() => {
    const amountNeeded = salaryModel === "fixed" || salaryModel === "hybrid";
    const percentNeeded = salaryModel === "percent" || salaryModel === "hybrid";

    return Boolean(
      firstName.trim() &&
      lastName.trim() &&
      birthDate.length === 10 &&
      branchID &&
      subjects.length > 0 &&
      phone.trim() &&
      emailStatus === "available" &&
      (!editing ? password.length >= 8 : !password || password.length >= 8) &&
      (!amountNeeded || (salaryAmount.trim() && Number(salaryAmount) > 0)) &&
      (!percentNeeded || (percentage.trim() && Number(percentage) >= 0))
    );
  }, [
    birthDate,
    branchID,
    emailStatus,
    editing,
    firstName,
    lastName,
    password,
    percentage,
    phone,
    salaryAmount,
    salaryModel,
    subjects.length,
  ]);

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
    setPhotoRemoved(false);
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
    setPhotoRemoved(true);
  }

  function toggleSubject(subject: string) {
    setSubjects((current) =>
      current.includes(subject)
        ? current.filter((item) => item !== subject)
        : [...current, subject],
    );
  }

  function handleEmailChange(value: string) {
    setEmail(value);
    const normalized = value.trim();
    if (
      editing &&
      normalized.toLocaleLowerCase("en-US") ===
        (initialTeacher?.email ?? "").toLocaleLowerCase("en-US")
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

  return (
    <form action={formAction} className="mx-auto max-w-5xl space-y-7">
      <input name="locale" type="hidden" value={locale} />
      <input name="idempotency_key" type="hidden" value={idempotencyKey} />
      {initialTeacher ? (
        <>
          <input name="teacher_id" type="hidden" value={initialTeacher.id} />
          <input
            name="profile_photo_file_id"
            type="hidden"
            value={initialTeacher.profile_photo_file_id ?? ""}
          />
          <input
            name="remove_photo"
            type="hidden"
            value={photoRemoved ? "1" : "0"}
          />
        </>
      ) : null}
      {subjects.map((subject) => (
        <input key={subject} name="subjects" type="hidden" value={subject} />
      ))}

      <section className="text-center">
        <h1 className="text-3xl font-black tracking-tight">
          {editing ? labels.edit : labels.addTeacher}
        </h1>
        <p className="mt-2 text-sm text-kw-c-59667a dark:text-kw-c-a7b0bf">
          {labels.addTeacherDescription}
        </p>
      </section>

      <section className="rounded-2xl border border-white/55 bg-white/70 p-7 shadow-kw-panel backdrop-blur-xl dark:border-white/10 dark:bg-kw-c-1b2635/70">
        <div className="flex justify-center">
          <PhotoUploadAvatar
            inputRef={fileInputRef}
            name="profile_photo"
            onChange={handlePhotoChange}
            onClear={clearPhoto}
            previewURL={photoPreview}
            removeLabel={labels.removePhoto}
          />
        </div>

        <div className="mt-8 grid gap-5 lg:grid-cols-2">
          <Field label={labels.name} required>
            <Input
              className="h-11"
              name="first_name"
              onChange={(event) => setFirstName(event.target.value)}
              value={firstName}
            />
          </Field>
          <Field label={labels.surname} required>
            <Input
              className="h-11"
              name="last_name"
              onChange={(event) => setLastName(event.target.value)}
              value={lastName}
            />
          </Field>
          <Field label={labels.birthDate} required>
            <Input
              className="h-11"
              inputMode="numeric"
              maxLength={10}
              name="birth_date"
              onChange={(event) => setBirthDate(formatBirthDate(event.target.value))}
              placeholder="DD/MM/YYYY"
              value={birthDate}
            />
          </Field>
          <Field label={labels.gender}>
            <select
              className="h-11 w-full cursor-pointer rounded-md border border-input bg-background px-3 text-sm outline-none"
              name="gender"
              defaultValue={initialTeacher?.gender ?? ""}
            >
              <option value="">{labels.genderPlaceholder}</option>
              <option value="male">{labels.male}</option>
              <option value="female">{labels.female}</option>
              <option value="other">{labels.other}</option>
            </select>
          </Field>
          <Field label={labels.phoneNumber} required>
            <Input
              className="h-11"
              name="phone"
              onChange={(event) => setPhone(event.target.value)}
              value={phone}
            />
          </Field>
          <Field label={labels.address}>
            <Textarea
              className="h-11 min-h-11 resize-none py-2"
              defaultValue={initialTeacher?.address ?? ""}
              name="address"
            />
          </Field>
          <Field label={labels.branchAssignment} required>
            <select
              className="h-11 w-full cursor-pointer rounded-md border border-input bg-background px-3 text-sm outline-none"
              name="branch_id"
              onChange={(event) => setBranchID(event.target.value)}
              value={branchID}
            >
              {branches.map((branch) => (
                <option key={branch.id} value={branch.id}>
                  {branch.name}
                </option>
              ))}
            </select>
          </Field>
          <Field label={labels.subjectSelection} required>
            <SubjectMultiSelect
              labels={labels}
              onToggle={toggleSubject}
              selected={subjects}
              subjects={subjectOptions}
            />
          </Field>
          <Field label={labels.salaryModel} required>
            <select
              className="h-11 w-full cursor-pointer rounded-md border border-input bg-background px-3 text-sm outline-none"
              name="salary_model"
              onChange={(event) =>
                setSalaryModel(event.target.value as SalaryModelType)
              }
              value={salaryModel}
            >
              <option value="fixed">{labels.salaryTypes.fixed}</option>
              <option value="percent">{labels.salaryTypes.percent}</option>
              <option value="hybrid">{labels.salaryTypes.hybrid}</option>
            </select>
          </Field>
          <div className="grid gap-3 sm:grid-cols-2">
            {(salaryModel === "percent" || salaryModel === "hybrid") && (
              <Field label={labels.percentage} required>
                <Input
                  className="h-11"
                  inputMode="decimal"
                  name="percentage"
                  onChange={(event) =>
                    setPercentage(event.target.value.replace(/[^\d.]/g, ""))
                  }
                  value={percentage}
                />
              </Field>
            )}
            <Field
              label={labels.salaryAmount}
              required={salaryModel === "fixed" || salaryModel === "hybrid"}
            >
              <Input
                className="h-11"
                disabled={salaryModel === "percent"}
                inputMode="decimal"
                name="salary_amount"
                onChange={(event) =>
                  setSalaryAmount(event.target.value.replace(/[^\d.]/g, ""))
                }
                value={salaryAmount}
              />
            </Field>
          </div>
          <Field label={labels.email} required>
            <div className="relative">
              <Input
                className="h-11"
                name="email"
                onChange={(event) => handleEmailChange(event.target.value)}
                type="email"
                value={email}
              />
              <EmailAvailabilityIcon status={emailStatus} />
            </div>
            <EmailAvailabilityText labels={labels} status={emailStatus} />
          </Field>
          <Field label={labels.password} required={!editing}>
            <div className="relative">
              <Input
                className="h-11"
                name="password"
                onChange={(event) => setPassword(event.target.value)}
                type={showPassword ? "text" : "password"}
                value={password}
              />
              <button
                className="absolute right-3 top-1/2 -translate-y-1/2 cursor-pointer text-kw-c-687386 transition hover:text-kw-c-0a284b dark:text-kw-c-a7b0bf dark:hover:text-white"
                onClick={() => setShowPassword((current) => !current)}
                type="button"
              >
                {showPassword ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
              </button>
            </div>
          </Field>
        </div>

        <CreateTeacherError labels={labels} state={state} />

        <div className="mt-8 flex justify-center gap-3">
          <Button
            className="rounded-lg px-6 font-bold"
            onClick={() => window.history.back()}
            type="button"
            variant="outline"
          >
            {labels.cancel}
          </Button>
          <LockedSubmitButton
            disabled={!canSubmit}
            label={labels.save}
            pending={labels.saving}
          />
        </div>
      </section>
    </form>
  );
}

function Field({
  children,
  label,
  required = false,
}: {
  children: ReactNode;
  label: string;
  required?: boolean;
}) {
  return (
    <div className="space-y-2">
      <Label className="font-black">
        {label}
        {required ? <span className="text-kw-c-ef2334 dark:text-kw-c-ff3b4f"> *</span> : null}
      </Label>
      {children}
    </div>
  );
}

function SubjectMultiSelect({
  labels,
  onToggle,
  selected,
  subjects,
}: {
  labels: AddTeacherFormLabels;
  onToggle: (subject: string) => void;
  selected: string[];
  subjects: string[];
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          className="flex h-11 w-full cursor-pointer items-center justify-between gap-3 rounded-md border border-input bg-background px-3 text-left text-sm font-semibold outline-none transition hover:border-kw-c-ef2334 focus:border-kw-c-ef2334 dark:hover:border-kw-c-ff3b4f dark:focus:border-kw-c-ff3b4f"
          type="button"
        >
          <span
            className={`truncate ${
              selected.length > 0
                ? "text-kw-c-0a284b dark:text-kw-c-f3f6fa"
                : "text-kw-c-687386 dark:text-kw-c-a7b0bf"
            }`}
          >
            {selected.length > 0
              ? selected.join(", ")
              : labels.subjectDropdownPlaceholder}
          </span>
          <ChevronDown className="size-4 shrink-0 text-kw-c-687386 dark:text-kw-c-a7b0bf" />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent className="z-[1200] border-kw-c-dce3ee dark:border-kw-c-3a4658 dark:bg-kw-c-1b2635">
        {subjects.map((subject) => (
          <DropdownMenuCheckboxItem
            checked={selected.includes(subject)}
            className="cursor-pointer py-2 font-semibold"
            key={subject}
            onCheckedChange={() => onToggle(subject)}
            onSelect={(event) => event.preventDefault()}
          >
            {subject}
          </DropdownMenuCheckboxItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function CreateTeacherError({
  labels,
  state,
}: {
  labels: AddTeacherFormLabels;
  state: CreateTeacherState;
}) {
  const message =
    state.error === "invalid"
      ? labels.requiredFieldsError
      : state.error === "duplicate_email"
        ? labels.emailTaken
        : state.error === "invalid_photo"
          ? labels.invalidPhotoError
          : state.error === "photo_too_large"
            ? labels.photoTooLargeError
            : state.error === "unauthorized"
              ? labels.createUnauthorizedError
              : state.error === "backend"
                ? labels.createBackendError
                : "";

  if (!message) {
    return null;
  }

  return (
    <div className="mt-6 rounded-lg border border-kw-c-f87171 bg-kw-c-fff1f2 px-4 py-3 text-sm font-semibold text-kw-c-991b1b dark:bg-kw-c-3b1218 dark:text-kw-c-ffe4e6">
      {message}
    </div>
  );
}

function formatBirthDate(value: string) {
  const digits = value.replace(/\D/g, "").slice(0, 8);
  if (digits.length <= 2) {
    return digits;
  }
  if (digits.length <= 4) {
    return `${digits.slice(0, 2)}/${digits.slice(2)}`;
  }

  return `${digits.slice(0, 2)}/${digits.slice(2, 4)}/${digits.slice(4)}`;
}
