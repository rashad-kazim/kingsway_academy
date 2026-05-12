"use client";

import { useActionState, useEffect, useMemo, useRef, useState } from "react";
import {
  BookOpen,
  CalendarDays,
  Eye,
  EyeOff,
  KeyRound,
  MapPin,
  Phone,
  Plus,
  UserRound,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
  CourseMultiSelect,
  CreateStudentError,
  Field,
  parentRelationKeys,
  StepRail,
  WizardCard,
  type AddStudentFormLabels,
} from "@/components/owner/add-student-form-parts";
import {
  EmailAvailabilityIcon,
  EmailAvailabilityText,
  type EmailAvailabilityStatus,
} from "@/components/owner/shared/email-availability";
import { LockedSubmitButton } from "@/components/owner/shared/locked-submit-button";
import { PhotoUploadAvatar } from "@/components/owner/shared/photo-upload-avatar";
import type { Branch, Course, TeacherFinanceRecord } from "@/lib/api/types";
import {
  checkStudentEmailAvailabilityAction,
  createStudentAction,
  type CreateStudentState,
} from "@/lib/student-assignment/actions";
import { formatShortDateInput } from "@/lib/forms/date-format";

export type { AddStudentFormLabels };

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
  const [emailStatus, setEmailStatus] =
    useState<EmailAvailabilityStatus>("idle");
  const branchCourses = useMemo(
    () =>
      courses.filter(
        (course) => course.branch_id === branchID && course.is_active,
      ),
    [branchID, courses],
  );
  const branchCoursesByID = useMemo(
    () => new Map(branchCourses.map((course) => [course.id, course])),
    [branchCourses],
  );
  const selectedCourseRecords = useMemo(
    () =>
      selectedCourses.flatMap(
        (courseID) => branchCoursesByID.get(courseID) ?? [],
      ),
    [branchCoursesByID, selectedCourses],
  );
  const teachersByCourseName = useMemo(
    () => groupActiveTeachersBySubject(teachers, branchID),
    [branchID, teachers],
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

      <section className="student-wizard rounded-2xl border border-kw-c-dce3ee bg-white/75 p-5 pt-8 text-kw-c-0a284b shadow-kw-panel backdrop-blur-xl dark:border-white/10 dark:bg-kw-c-152334/95 dark:text-kw-c-f3f6fa kw-dark-shadow-form">
        <div className="mb-5 flex justify-center">
          <PhotoUploadAvatar
            inputRef={fileInputRef}
            name="profile_photo"
            onChange={handlePhotoChange}
            onClear={clearPhoto}
            previewURL={photoPreview}
            removeLabel={labels.removePhoto}
            removeRingClassName="dark:ring-kw-c-152334"
          />
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
                    finInvalid ? "!border-kw-c-ef2334 !ring-3 !ring-kw-c-ef2334/15 dark:!border-kw-c-ff5a69 dark:!ring-kw-c-ff5a69/15" : "",
                  ].join(" ")}
                  maxLength={7}
                  name="fin"
                  value={fin}
                  onBlur={() => setFinTouched(true)}
                  onChange={(event) => setFin(event.target.value.replace(/[^a-zA-Z0-9]/g, "").toUpperCase().slice(0, 7))}
                />
                {finInvalid ? (
                  <p className="text-xs font-bold text-kw-c-ef2334 dark:text-kw-c-ff5a69">
                    {labels.finLengthError}
                  </p>
                ) : null}
              </Field>
              <Field compact label={labels.birthDate} required icon={<CalendarDays className="size-4" />}>
                <Input autoComplete="off" className="student-wizard-input" inputMode="numeric" maxLength={10} name="birth_date" placeholder="DD/MM/YYYY" value={birthDate} onChange={(event) => setBirthDate(formatShortDateInput(event.target.value))} />
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
              <label className="mt-7 flex cursor-pointer items-center gap-2 text-xs font-black text-kw-c-0a284b transition hover:text-kw-c-ef2334 dark:text-kw-c-f3f6fa dark:hover:text-kw-c-ff5a69">
                <input
                  checked={parentEnabled}
                  className="size-4 cursor-pointer accent-kw-c-ef2334"
                  onChange={(event) => setParentEnabled(event.target.checked)}
                  type="checkbox"
                />
                {labels.addParentInfo}
              </label>
            </div>

            {parentEnabled ? (
              <div className="mt-3 max-h-52 space-y-4 overflow-y-auto rounded-xl border border-kw-c-d6dfec bg-white/65 p-3 dark:border-white/10 dark:bg-white/10">
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
                <Input autoComplete="off" className="student-wizard-input" inputMode="numeric" maxLength={10} name="start_date" placeholder="DD/MM/YYYY" value={startDate} onChange={(event) => setStartDate(formatShortDateInput(event.target.value))} />
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
                  <div
                    className="grid gap-3 rounded-xl border border-kw-c-d6dfec bg-white/65 p-3 dark:border-white/10 dark:bg-white/10 lg:grid-cols-2"
                    key={course.id}
                  >
                    <Field compact label={`${course.name} ${labels.teacher}`}>
                      <select
                        className="student-wizard-input cursor-pointer"
                        name={`teacher_${course.id}`}
                      >
                        <option value="">{labels.noTeacher}</option>
                        {(
                          teachersByCourseName.get(
                            normalizeCourseName(course.name),
                          ) ?? []
                        ).map((teacher) => (
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
                  <EmailAvailabilityIcon status={emailStatus} />
                </div>
                <EmailAvailabilityText labels={labels} status={emailStatus} />
              </Field>
              <Field compact label={labels.password} required>
                <div className="relative">
                  <Input autoComplete="new-password" className="student-wizard-input pr-10" name="password" type={showPassword ? "text" : "password"} value={password} onChange={(event) => setPassword(event.target.value)} />
                  <button className="absolute right-3 top-1/2 -translate-y-1/2 cursor-pointer text-kw-c-a7b0bf transition hover:text-white" type="button" onClick={() => setShowPassword((current) => !current)}>
                    {showPassword ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
                  </button>
                </div>
              </Field>
            </div>
          </WizardCard>
        </div>

        <CreateStudentError labels={labels} state={state} />

        <div className="mt-5 flex justify-center gap-3">
          <Button className="h-10 rounded-lg border-white/70 bg-white px-9 font-bold text-kw-c-203247 hover:bg-kw-c-e6edf6" type="button" variant="outline" onClick={() => window.history.back()}>
            {labels.cancel}
          </Button>
          <Button
            className="h-10 rounded-lg border-white/10 bg-kw-c-5c6672 px-9 font-bold text-white hover:bg-kw-c-687482 disabled:bg-kw-c-6a7480 disabled:text-kw-c-d8e1ee disabled:opacity-55"
            disabled={step === 0}
            type="button"
            variant="outline"
            onClick={() => setStep((current) => Math.max(0, current - 1))}
          >
            {labels.back}
          </Button>
          {step < 2 ? (
            <Button
              className="h-10 rounded-lg bg-emerald-600 px-10 font-bold text-white hover:bg-emerald-700 disabled:cursor-not-allowed disabled:bg-emerald-700/65 disabled:text-kw-c-d8f3e5"
              disabled={!currentStepReady}
              type="button"
              onClick={() => setStep((current) => Math.min(2, current + 1))}
            >
              {labels.next}
            </Button>
          ) : (
            <LockedSubmitButton
              disabled={!canSubmit}
              label={labels.save}
              pending={labels.saving}
            />
          )}
        </div>
      </section>
    </form>
  );
}

function groupActiveTeachersBySubject(
  teachers: TeacherFinanceRecord[],
  branchID: string,
) {
  const bySubject = new Map<string, TeacherFinanceRecord[]>();
  for (const teacher of teachers) {
    if (teacher.branch_id !== branchID || teacher.status !== "active") {
      continue;
    }
    for (const subject of (teacher.subject ?? "")
      .split(",")
      .map(normalizeCourseName)
      .filter(Boolean)) {
      const subjectTeachers = bySubject.get(subject);
      if (subjectTeachers) {
        subjectTeachers.push(teacher);
      } else {
        bySubject.set(subject, [teacher]);
      }
    }
  }
  return bySubject;
}

function normalizeCourseName(value: string) {
  return value.trim().toLocaleLowerCase("en-US");
}
