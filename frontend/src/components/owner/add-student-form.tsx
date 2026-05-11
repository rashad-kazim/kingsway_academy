"use client";

import { useActionState, useEffect, useMemo, useRef, useState } from "react";
import type { ReactNode } from "react";
import { useFormStatus } from "react-dom";
import {
  BookOpen,
  CalendarDays,
  CheckCircle2,
  ChevronDown,
  Eye,
  EyeOff,
  ImagePlus,
  KeyRound,
  MapPin,
  Phone,
  Plus,
  Save,
  UserRound,
  X,
  XCircle,
} from "lucide-react";
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
import type { Branch, Course, TeacherFinanceRecord } from "@/lib/api/types";
import {
  checkStudentEmailAvailabilityAction,
  createStudentAction,
  type CreateStudentState,
} from "@/lib/student-assignment/actions";
import { useSubmitLock } from "@/lib/forms/use-submit-lock";

export type AddStudentFormLabels = {
  addStudent: string;
  addStudentDescription: string;
  personalInfo: string;
  academicInfo: string;
  accountAccess: string;
  personalDetails: string;
  academicDetails: string;
  accountDetails: string;
  uploadStudentPhoto: string;
  next: string;
  back: string;
  profilePhoto: string;
  removePhoto: string;
  name: string;
  surname: string;
  fin: string;
  birthDate: string;
  gender: string;
  genderPlaceholder: string;
  male: string;
  female: string;
  other: string;
  address: string;
  finLengthError: string;
  phoneNumber: string;
  addParentInfo: string;
  parentRelation: string;
  parentName: string;
  parentPhoneNumber: string;
  addParent: string;
  branchAssignment: string;
  chooseCourse: string;
  courseDropdownPlaceholder: string;
  teacher: string;
  monthlyPayment: string;
  startDate: string;
  email: string;
  password: string;
  save: string;
  saving: string;
  cancel: string;
  emailAvailable: string;
  emailTaken: string;
  emailChecking: string;
  requiredFieldsError: string;
  duplicateFinError: string;
  duplicateEmailError: string;
  createBackendError: string;
  createUnauthorizedError: string;
  invalidPhotoError: string;
  photoTooLargeError: string;
  noTeacher: string;
  parentRelations: Record<"father" | "mother" | "sister" | "brother" | "other", string>;
};

type ParentDraft = {
  id: string;
  name: string;
  phones: string[];
  relation: "father" | "mother" | "sister" | "brother" | "other";
};

type AddStudentFormProps = {
  branches: Branch[];
  courses: Course[];
  labels: AddStudentFormLabels;
  locale: string;
  teachers: TeacherFinanceRecord[];
};

const initialState: CreateStudentState = {};

export function AddStudentForm({
  branches,
  courses,
  labels,
  locale,
  teachers,
}: AddStudentFormProps) {
  const [state, formAction] = useActionState(createStudentAction, initialState);
  const idempotencyKey = useMemo(() => crypto.randomUUID(), []);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const objectURLRef = useRef<string | null>(null);
  const [photoPreview, setPhotoPreview] = useState("");
  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const [fin, setFin] = useState("");
  const [finTouched, setFinTouched] = useState(false);
  const [birthDate, setBirthDate] = useState("");
  const [phone, setPhone] = useState("");
  const [branchID, setBranchID] = useState(branches[0]?.id ?? "");
  const [selectedCourses, setSelectedCourses] = useState<string[]>([]);
  const [startDate, setStartDate] = useState("");
  const [defaultMonthlyAmount, setDefaultMonthlyAmount] = useState("");
  const [amounts, setAmounts] = useState<Record<string, string>>({});
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [step, setStep] = useState(0);
  const [parentEnabled, setParentEnabled] = useState(false);
  const [parents, setParents] = useState<ParentDraft[]>([
    { id: "0", name: "", phones: [""], relation: "father" },
  ]);
  const [emailStatus, setEmailStatus] = useState<
    "idle" | "checking" | "available" | "taken" | "invalid"
  >("idle");
  const branchCourses = useMemo(
    () => courses.filter((course) => course.branch_id === branchID && course.is_active),
    [branchID, courses],
  );
  const selectedCourseRecords = useMemo(
    () =>
      selectedCourses
        .map((courseID) => branchCourses.find((course) => course.id === courseID))
        .filter(Boolean) as Course[],
    [branchCourses, selectedCourses],
  );
  const finInvalid = finTouched && fin.length > 0 && fin.length !== 7;

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
      const result = await checkStudentEmailAvailabilityAction(normalized);
      setEmailStatus(result.available ? "available" : "taken");
    }, 350);

    return () => window.clearTimeout(timeout);
  }, [email, emailStatus]);

  const parentsReady =
    !parentEnabled ||
    parents.every((parent) =>
      Boolean(parent.name.trim() && parent.phones.some((item) => item.trim())),
    );

  const personalReady = Boolean(
    firstName.trim() &&
      lastName.trim() &&
      fin.length === 7 &&
      birthDate.length === 10 &&
      phone.trim() &&
      parentsReady,
  );

  const academicReady = Boolean(
    branchID &&
      selectedCourses.length > 0 &&
      startDate.length === 10 &&
      selectedCourses.every((courseID) => Number(amounts[courseID]) > 0),
  );

  const accountReady = Boolean(
      emailStatus === "available" &&
      password.length >= 8,
  );

  const canSubmit = Boolean(
    personalReady &&
      academicReady &&
      accountReady,
  );

  const currentStepReady =
    step === 0 ? personalReady : step === 1 ? academicReady : accountReady;

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
  }

  function handleEmailChange(value: string) {
    setEmail(value);
    const normalized = value.trim();
    if (!normalized) {
      setEmailStatus("idle");
      return;
    }
    setEmailStatus(/^\S+@\S+\.\S+$/.test(normalized) ? "checking" : "invalid");
  }

  function handleDefaultMonthlyAmountChange(value: string) {
    const normalized = value.replace(/[^\d]/g, "");
    setDefaultMonthlyAmount(normalized);
    setAmounts((current) => {
      const next = { ...current };
      selectedCourses.forEach((courseID) => {
        next[courseID] = normalized;
      });
      return next;
    });
  }

  function toggleCourse(courseID: string) {
    setSelectedCourses((current) => {
      if (current.includes(courseID)) {
        setAmounts((amountCurrent) => {
          const next = { ...amountCurrent };
          delete next[courseID];
          return next;
        });
        return current.filter((item) => item !== courseID);
      }

      setAmounts((amountCurrent) => ({
        ...amountCurrent,
        [courseID]: amountCurrent[courseID] ?? defaultMonthlyAmount,
      }));
      return [...current, courseID];
    });
  }

  function handleBranchChange(nextBranchID: string) {
    const allowed = new Set(
      courses
        .filter((course) => course.branch_id === nextBranchID && course.is_active)
        .map((course) => course.id),
    );
    setBranchID(nextBranchID);
    setSelectedCourses((current) =>
      current.filter((courseID) => allowed.has(courseID)),
    );
  }

  function addParent() {
    setParents((current) => [
      ...current,
      { id: crypto.randomUUID(), name: "", phones: [""], relation: "father" },
    ]);
  }

  function setParentName(parentID: string, value: string) {
    setParents((current) =>
      current.map((parent) =>
        parent.id === parentID ? { ...parent, name: value } : parent,
      ),
    );
  }

  function setParentRelation(
    parentID: string,
    value: ParentDraft["relation"],
  ) {
    setParents((current) =>
      current.map((parent) =>
        parent.id === parentID ? { ...parent, relation: value } : parent,
      ),
    );
  }

  function addParentPhone(parentID: string) {
    setParents((current) =>
      current.map((parent) =>
        parent.id === parentID
          ? { ...parent, phones: [...parent.phones, ""] }
          : parent,
      ),
    );
  }

  function setParentPhone(parentID: string, index: number, value: string) {
    setParents((current) =>
      current.map((parent) =>
        parent.id === parentID
          ? {
              ...parent,
              phones: parent.phones.map((phoneValue, phoneIndex) =>
                phoneIndex === index ? value : phoneValue,
              ),
            }
          : parent,
      ),
    );
  }

  return (
    <form action={formAction} autoComplete="off" className="mx-auto max-w-7xl space-y-4">
      <input name="locale" type="hidden" value={locale} />
      <input name="idempotency_key" type="hidden" value={idempotencyKey} />
      {selectedCourses.map((courseID) => (
        <input key={courseID} name="course_ids" type="hidden" value={courseID} />
      ))}

      <section className="student-wizard rounded-2xl border border-[#dce3ee] bg-white/75 p-5 pt-8 text-[#0a284b] shadow-[0_24px_70px_rgba(10,40,75,0.12)] backdrop-blur-xl dark:border-white/10 dark:bg-[#152334]/95 dark:text-[#f3f6fa] dark:shadow-[0_28px_80px_rgba(0,0,0,0.34)]">
        <div className="mb-5 flex justify-center">
          <div className="relative size-36">
            <label className="group flex size-36 cursor-pointer items-center justify-center overflow-hidden rounded-full border border-dashed border-[#b9c5d6] bg-[#f7f8fb] text-[#0a284b] shadow-inner transition hover:border-[#ef2334] hover:bg-[#f1f4f8] dark:border-[#3a4658] dark:bg-[#202b3a] dark:text-[#f3f6fa] dark:hover:border-[#ff3b4f] dark:hover:bg-[#263448]">
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
                name="profile_photo"
                onChange={(event) =>
                  handlePhotoChange(event.currentTarget.files?.[0] ?? null)
                }
                type="file"
              />
            </label>
            {photoPreview ? (
              <button
                aria-label={labels.removePhoto}
                className="absolute right-0 top-0 grid size-8 cursor-pointer place-items-center rounded-full bg-[#ef2334] text-white shadow-lg ring-4 ring-white transition hover:bg-[#d91f30] dark:bg-[#ff3b4f] dark:ring-[#152334] dark:hover:bg-[#ff5a69]"
                type="button"
                onClick={clearPhoto}
              >
                <X className="size-4" />
              </button>
            ) : null}
          </div>
        </div>

        <StepRail labels={labels} step={step} setStep={setStep} readiness={[personalReady, academicReady]} />

        <div className="mt-4 grid gap-4 xl:grid-cols-3">
          <WizardCard
            active={step === 0}
            icon={<UserRound className="size-4" />}
            title={labels.personalDetails}
          >
            <div className="grid grid-cols-2 gap-3">
              <Field compact label={labels.name} required icon={<UserRound className="size-4" />}>
                <Input autoComplete="off" className="student-wizard-input" name="first_name" value={firstName} onChange={(event) => setFirstName(event.target.value)} />
              </Field>
              <Field compact label={labels.surname} required icon={<UserRound className="size-4" />}>
                <Input autoComplete="off" className="student-wizard-input" name="last_name" value={lastName} onChange={(event) => setLastName(event.target.value)} />
              </Field>
              <Field compact label={labels.fin} required icon={<KeyRound className="size-4" />}>
                <Input
                  aria-invalid={finInvalid}
                  autoComplete="off"
                  className={[
                    "student-wizard-input uppercase",
                    finInvalid ? "!border-[#ef2334] !ring-3 !ring-[#ef2334]/15 dark:!border-[#ff5a69] dark:!ring-[#ff5a69]/15" : "",
                  ].join(" ")}
                  maxLength={7}
                  name="fin"
                  value={fin}
                  onBlur={() => setFinTouched(true)}
                  onChange={(event) => setFin(event.target.value.replace(/[^a-zA-Z0-9]/g, "").toUpperCase().slice(0, 7))}
                />
                {finInvalid ? (
                  <p className="text-xs font-bold text-[#ef2334] dark:text-[#ff5a69]">
                    {labels.finLengthError}
                  </p>
                ) : null}
              </Field>
              <Field compact label={labels.birthDate} required icon={<CalendarDays className="size-4" />}>
                <Input autoComplete="off" className="student-wizard-input" inputMode="numeric" maxLength={10} name="birth_date" placeholder="DD/MM/YYYY" value={birthDate} onChange={(event) => setBirthDate(formatDateInput(event.target.value))} />
              </Field>
              <Field compact label={labels.gender}>
                <select className="student-wizard-input cursor-pointer" name="gender" defaultValue="">
                  <option value="">{labels.genderPlaceholder}</option>
                  <option value="male">{labels.male}</option>
                  <option value="female">{labels.female}</option>
                  <option value="other">{labels.other}</option>
                </select>
              </Field>
              <Field compact label={labels.address} icon={<MapPin className="size-4" />}>
                <Textarea autoComplete="off" className="student-wizard-input h-11 min-h-11 max-h-11 resize-none overflow-hidden py-3" name="address" />
              </Field>
              <Field compact label={labels.phoneNumber} required icon={<Phone className="size-4" />}>
                <Input autoComplete="off" className="student-wizard-input" name="phone" value={phone} onChange={(event) => setPhone(event.target.value)} />
              </Field>
              <label className="mt-7 flex cursor-pointer items-center gap-2 text-xs font-black text-[#0a284b] transition hover:text-[#ef2334] dark:text-[#f3f6fa] dark:hover:text-[#ff5a69]">
                <input
                  checked={parentEnabled}
                  className="size-4 cursor-pointer accent-[#ef2334]"
                  onChange={(event) => setParentEnabled(event.target.checked)}
                  type="checkbox"
                />
                {labels.addParentInfo}
              </label>
            </div>

            {parentEnabled ? (
              <div className="mt-3 max-h-52 space-y-4 overflow-y-auto rounded-xl border border-[#d6dfec] bg-white/65 p-3 dark:border-white/10 dark:bg-white/10">
                {parents.map((parent, parentIndex) => (
                  <div className="space-y-3" key={parent.id}>
                    <input name="parent_indexes" type="hidden" value={parent.id} />
                    <div className="grid grid-cols-2 gap-3">
                      <Field compact label={labels.parentRelation} required>
                        <select className="student-wizard-input cursor-pointer" name={`parent_relation_${parent.id}`} value={parent.relation} onChange={(event) => setParentRelation(parent.id, event.target.value as ParentDraft["relation"])}>
                          {parentRelationKeys.map((relation) => (
                            <option key={relation} value={relation}>
                              {labels.parentRelations[relation]}
                            </option>
                          ))}
                        </select>
                      </Field>
                      <Field compact label={labels.parentName} required>
                        <Input className="student-wizard-input" name={`parent_name_${parent.id}`} value={parent.name} onChange={(event) => setParentName(parent.id, event.target.value)} />
                      </Field>
                    </div>
                    {parent.phones.map((value, phoneIndex) => (
                      <div className="flex gap-2" key={`${parent.id}-${phoneIndex}`}>
                        <Field compact label={labels.parentPhoneNumber} required>
                          <Input
                            className="student-wizard-input"
                            name={`parent_phone_${parent.id}`}
                            value={value}
                            onChange={(event) =>
                              setParentPhone(parent.id, phoneIndex, event.target.value)
                            }
                          />
                        </Field>
                        {phoneIndex === parent.phones.length - 1 ? (
                          <Button
                            className="mt-6 size-9 shrink-0 cursor-pointer rounded-full"
                            type="button"
                            variant="outline"
                            onClick={() => addParentPhone(parent.id)}
                          >
                            <Plus className="size-4" />
                          </Button>
                        ) : null}
                      </div>
                    ))}
                    {parentIndex === parents.length - 1 ? (
                      <Button className="h-8 gap-2 rounded-lg text-xs" type="button" variant="outline" onClick={addParent}>
                        <Plus className="size-3" />
                        {labels.addParent}
                      </Button>
                    ) : null}
                  </div>
                ))}
              </div>
            ) : null}
          </WizardCard>

          <WizardCard
            active={step === 1}
            icon={<BookOpen className="size-4" />}
            title={labels.academicDetails}
          >
            <div className="grid grid-cols-2 gap-3">
              <Field compact label={labels.branchAssignment} required>
                <select className="student-wizard-input cursor-pointer" name="branch_id" value={branchID} onChange={(event) => handleBranchChange(event.target.value)}>
                  {branches.map((branch) => (
                    <option key={branch.id} value={branch.id}>{branch.name}</option>
                  ))}
                </select>
              </Field>
              <Field compact label={labels.chooseCourse} required icon={<BookOpen className="size-4" />}>
                <CourseMultiSelect
                  courses={branchCourses}
                  labels={labels}
                  onToggle={toggleCourse}
                  selected={selectedCourses}
                />
              </Field>
              <Field compact label={labels.startDate} required>
                <Input autoComplete="off" className="student-wizard-input" inputMode="numeric" maxLength={10} name="start_date" placeholder="DD/MM/YYYY" value={startDate} onChange={(event) => setStartDate(formatDateInput(event.target.value))} />
              </Field>
              <Field compact label={labels.monthlyPayment} required>
                <Input
                  autoComplete="off"
                  className="student-wizard-input"
                  inputMode="numeric"
                  value={defaultMonthlyAmount}
                  onChange={(event) =>
                    handleDefaultMonthlyAmountChange(event.target.value)
                  }
                />
              </Field>
            </div>

            {selectedCourseRecords.length > 0 ? (
              <div className="mt-3 max-h-56 space-y-3 overflow-y-auto">
                {selectedCourseRecords.map((course) => (
                  <div className="grid gap-3 rounded-xl border border-[#d6dfec] bg-white/65 p-3 dark:border-white/10 dark:bg-white/10 lg:grid-cols-2" key={course.id}>
                    <Field compact label={`${course.name} ${labels.teacher}`}>
                      <select className="student-wizard-input cursor-pointer" name={`teacher_${course.id}`}>
                        <option value="">{labels.noTeacher}</option>
                        {teachersForCourse(teachers, branchID, course).map((teacher) => (
                          <option key={teacher.id} value={teacher.id}>
                            {teacher.first_name} {teacher.last_name}
                          </option>
                        ))}
                      </select>
                    </Field>
                    <Field compact label={`${course.name} ${labels.monthlyPayment}`} required>
                      <Input
                        autoComplete="off"
                        className="student-wizard-input"
                        inputMode="numeric"
                        name={`amount_${course.id}`}
                        value={amounts[course.id] ?? ""}
                        onChange={(event) =>
                          setAmounts((current) => ({
                            ...current,
                            [course.id]: event.target.value.replace(/[^\d]/g, ""),
                          }))
                        }
                      />
                    </Field>
                  </div>
                ))}
              </div>
            ) : null}
          </WizardCard>

          <WizardCard
            active={step === 2}
            icon={<KeyRound className="size-4" />}
            title={labels.accountDetails}
          >
            <div className="grid grid-cols-2 gap-3">
              <Field compact label={labels.email} required>
                <div className="relative">
                  <Input autoComplete="off" className="student-wizard-input" name="email" type="email" value={email} onChange={(event) => handleEmailChange(event.target.value)} />
                  <EmailStatusIcon status={emailStatus} />
                </div>
                <EmailStatusText labels={labels} status={emailStatus} />
              </Field>
              <Field compact label={labels.password} required>
                <div className="relative">
                  <Input autoComplete="new-password" className="student-wizard-input pr-10" name="password" type={showPassword ? "text" : "password"} value={password} onChange={(event) => setPassword(event.target.value)} />
                  <button className="absolute right-3 top-1/2 -translate-y-1/2 cursor-pointer text-[#a7b0bf] transition hover:text-white" type="button" onClick={() => setShowPassword((current) => !current)}>
                    {showPassword ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
                  </button>
                </div>
              </Field>
            </div>
          </WizardCard>
        </div>

        <CreateStudentError labels={labels} state={state} />

        <div className="mt-5 flex justify-center gap-3">
          <Button className="h-10 rounded-lg border-white/70 bg-white px-9 font-bold text-[#203247] hover:bg-[#e6edf6]" type="button" variant="outline" onClick={() => window.history.back()}>
            {labels.cancel}
          </Button>
          <Button
            className="h-10 rounded-lg border-white/10 bg-[#5c6672] px-9 font-bold text-white hover:bg-[#687482] disabled:bg-[#6a7480] disabled:text-[#d8e1ee] disabled:opacity-55"
            disabled={step === 0}
            type="button"
            variant="outline"
            onClick={() => setStep((current) => Math.max(0, current - 1))}
          >
            {labels.back}
          </Button>
          {step < 2 ? (
            <Button
              className="h-10 rounded-lg bg-emerald-600 px-10 font-bold text-white hover:bg-emerald-700 disabled:cursor-not-allowed disabled:bg-emerald-700/65 disabled:text-[#d8f3e5]"
              disabled={!currentStepReady}
              type="button"
              onClick={() => setStep((current) => Math.min(2, current + 1))}
            >
              {labels.next}
            </Button>
          ) : (
            <SubmitButton disabled={!canSubmit} label={labels.save} pending={labels.saving} />
          )}
        </div>
      </section>
    </form>
  );
}

const parentRelationKeys = ["father", "mother", "sister", "brother", "other"] as const;

function Field({
  children,
  compact = false,
  icon,
  label,
  required = false,
}: {
  children: ReactNode;
  compact?: boolean;
  icon?: ReactNode;
  label: string;
  required?: boolean;
}) {
  return (
    <div className={compact ? "w-full space-y-1.5" : "space-y-2"}>
      <Label className={compact ? "flex items-center gap-1 text-[11px] font-black text-[#0a284b] dark:text-[#dce7f5]" : "font-black"}>
        {icon ? <span className="text-[#64748b] dark:text-[#91a0b4]">{icon}</span> : null}
        <span>{label}</span>
        {required ? <span className="text-[#ef2334] dark:text-[#ff3b4f]"> *</span> : null}
      </Label>
      {children}
    </div>
  );
}

function StepRail({
  labels,
  readiness,
  setStep,
  step,
}: {
  labels: AddStudentFormLabels;
  readiness: boolean[];
  setStep: (step: number) => void;
  step: number;
}) {
  const items = [
    { icon: <UserRound className="size-4" />, label: labels.personalInfo },
    { icon: <BookOpen className="size-4" />, label: labels.academicInfo },
    { icon: <KeyRound className="size-4" />, label: labels.accountAccess },
  ];

  function canOpen(index: number) {
    if (index === 0) {
      return true;
    }
    if (index === 1) {
      return readiness[0];
    }
    return readiness[0] && readiness[1];
  }

  return (
    <div className="flex gap-[5px] overflow-hidden rounded-full bg-[#d9e0ea] p-0.5 shadow-inner dark:bg-[#5f6b78]/70">
      {items.map((item, index) => (
        <button
          className={[
            "relative flex h-10 flex-1 items-center justify-center gap-2 text-sm font-black transition",
            index > 0 ? "-ml-[22px] pl-7" : "rounded-l-full",
            index === items.length - 1 ? "rounded-r-full pr-4" : "pr-7",
            step === index
              ? "z-20 bg-[#ef2334] text-white shadow-[0_12px_30px_rgba(239,35,52,0.28)]"
              : "z-10 bg-[#eef2f6] text-[#526074] hover:bg-[#e3e9f1] dark:bg-[#66717f] dark:text-[#dbe7f5] dark:hover:bg-[#758292]",
            !canOpen(index) ? "cursor-not-allowed opacity-45" : "cursor-pointer",
          ].join(" ")}
          disabled={!canOpen(index)}
          key={item.label}
          style={{ clipPath: stepClipPath(index, items.length) }}
          type="button"
          onClick={() => setStep(index)}
        >
          {item.icon}
          <span>{index + 1}. {item.label}</span>
        </button>
      ))}
    </div>
  );
}

function stepClipPath(index: number, total: number) {
  if (total <= 1) {
    return "none";
  }
  if (index === 0) {
    return "polygon(0 0, calc(100% - 22px) 0, 100% 50%, calc(100% - 22px) 100%, 0 100%)";
  }
  if (index === total - 1) {
    return "polygon(0 0, 100% 0, 100% 100%, 0 100%, 22px 50%)";
  }
  return "polygon(0 0, calc(100% - 22px) 0, 100% 50%, calc(100% - 22px) 100%, 0 100%, 22px 50%)";
}

function WizardCard({
  active,
  children,
  icon,
  title,
}: {
  active: boolean;
  children: ReactNode;
  icon: ReactNode;
  title: string;
}) {
  return (
    <section
      data-active={active}
      className={[
        "min-h-[280px] rounded-xl border p-4 shadow-[inset_0_1px_0_rgba(255,255,255,0.06)] transition duration-300",
        active
          ? "border-[#d6dfec] bg-white/80 text-[#0a284b] shadow-[0_12px_35px_rgba(10,40,75,0.08)] dark:border-[#526276] dark:bg-[#1b2a3c] dark:text-[#f3f6fa]"
          : "pointer-events-none border-[#dce3ee] bg-[#eef2f6]/75 text-[#7e8ea5] opacity-65 grayscale-[0.15] dark:border-[#334155] dark:bg-[#2b3644]/72 dark:text-[#8794a5]",
      ].join(" ")}
    >
      <h2 className="mb-4 flex items-center gap-2 text-lg font-black">
        <span className="text-[#ef2334] dark:text-[#ff5a69]">{icon}</span>
        {title}
      </h2>
      {children}
    </section>
  );
}

function CourseMultiSelect({
  courses,
  labels,
  onToggle,
  selected,
}: {
  courses: Course[];
  labels: AddStudentFormLabels;
  onToggle: (courseID: string) => void;
  selected: string[];
}) {
  const selectedNames = courses
    .filter((course) => selected.includes(course.id))
    .map((course) => course.name);

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button className="flex h-11 w-full cursor-pointer items-center justify-between gap-3 rounded-md border border-input bg-background px-3 text-left text-sm font-semibold outline-none transition hover:border-[#ef2334] focus:border-[#ef2334] dark:hover:border-[#ff3b4f] dark:focus:border-[#ff3b4f]" type="button">
          <span className={selectedNames.length ? "truncate" : "truncate text-[#687386] dark:text-[#a7b0bf]"}>
            {selectedNames.length ? selectedNames.join(", ") : labels.courseDropdownPlaceholder}
          </span>
          <ChevronDown className="size-4 shrink-0 text-[#687386] dark:text-[#a7b0bf]" />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent className="z-[1200] border-[#dce3ee] dark:border-[#3a4658] dark:bg-[#1b2635]">
        {courses.map((course) => (
          <DropdownMenuCheckboxItem
            checked={selected.includes(course.id)}
            className="cursor-pointer py-2 font-semibold"
            key={course.id}
            onCheckedChange={() => onToggle(course.id)}
            onSelect={(event) => event.preventDefault()}
          >
            {course.name}
          </DropdownMenuCheckboxItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function SubmitButton({
  disabled,
  label,
  pending,
}: {
  disabled: boolean;
  label: string;
  pending: string;
}) {
  const status = useFormStatus();
  const submitLock = useSubmitLock(status.pending);

  const blocked = disabled || status.pending || submitLock.locked;

  return (
    <Button
      className="gap-2 rounded-lg bg-emerald-600 px-7 font-bold text-white hover:bg-emerald-700 disabled:cursor-not-allowed disabled:opacity-50"
      disabled={blocked}
      type="submit"
      onClick={submitLock.onClick}
    >
      <Save className="size-4" />
      {status.pending ? pending : label}
    </Button>
  );
}

function EmailStatusIcon({
  status,
}: {
  status: "idle" | "checking" | "available" | "taken" | "invalid";
}) {
  if (status === "available") {
    return <CheckCircle2 className="absolute right-3 top-1/2 size-5 -translate-y-1/2 text-emerald-500" />;
  }
  if (status === "taken" || status === "invalid") {
    return <XCircle className="absolute right-3 top-1/2 size-5 -translate-y-1/2 text-[#ef2334] dark:text-[#ff3b4f]" />;
  }

  return null;
}

function EmailStatusText({
  labels,
  status,
}: {
  labels: AddStudentFormLabels;
  status: "idle" | "checking" | "available" | "taken" | "invalid";
}) {
  if (status === "checking") {
    return <p className="text-xs font-bold text-[#687386]">{labels.emailChecking}</p>;
  }
  if (status === "available") {
    return <p className="text-xs font-bold text-emerald-600">{labels.emailAvailable}</p>;
  }
  if (status === "taken") {
    return <p className="text-xs font-bold text-[#ef2334] dark:text-[#ff3b4f]">{labels.emailTaken}</p>;
  }

  return null;
}

function CreateStudentError({
  labels,
  state,
}: {
  labels: AddStudentFormLabels;
  state: CreateStudentState;
}) {
  const message =
    state.error === "invalid"
      ? labels.requiredFieldsError
      : state.error === "duplicate_fin"
        ? labels.duplicateFinError
      : state.error === "duplicate_email"
        ? labels.duplicateEmailError
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
    <div className="mt-6 rounded-lg border border-[#f87171] bg-[#fff1f2] px-4 py-3 text-sm font-black text-[#991b1b] dark:bg-[#3b1218] dark:text-[#ffe4e6]">
      {message}
    </div>
  );
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

function teachersForCourse(
  teachers: TeacherFinanceRecord[],
  branchID: string,
  course: Course,
) {
  return teachers.filter((teacher) => {
    if (teacher.branch_id !== branchID || teacher.status !== "active") {
      return false;
    }
    const subjects = (teacher.subject ?? "")
      .split(",")
      .map((subject) => subject.trim().toLocaleLowerCase("en-US"));
    return subjects.includes(course.name.trim().toLocaleLowerCase("en-US"));
  });
}
