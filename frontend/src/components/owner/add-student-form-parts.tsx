import type { ReactNode } from "react";
import { BookOpen, ChevronDown, KeyRound, UserRound } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Label } from "@/components/ui/label";
import type { Course } from "@/lib/api/types";
import type { CreateStudentState } from "@/lib/student-assignment/actions";

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
  parentRelations: Record<
    "father" | "mother" | "sister" | "brother" | "other",
    string
  >;
};

export const parentRelationKeys = [
  "father",
  "mother",
  "sister",
  "brother",
  "other",
] as const;

export function Field({
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
      <Label
        className={
          compact
            ? "flex items-center gap-1 text-kw-11 font-black text-kw-c-0a284b dark:text-kw-c-dce7f5"
            : "font-black"
        }
      >
        {icon ? (
          <span className="text-kw-c-64748b dark:text-kw-c-91a0b4">{icon}</span>
        ) : null}
        <span>{label}</span>
        {required ? (
          <span className="text-kw-c-ef2334 dark:text-kw-c-ff3b4f"> *</span>
        ) : null}
      </Label>
      {children}
    </div>
  );
}

export function StepRail({
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
    <div className="flex gap-[5px] overflow-hidden rounded-full bg-kw-c-d9e0ea p-0.5 shadow-inner dark:bg-kw-c-5f6b78/70">
      {items.map((item, index) => (
        <button
          className={[
            "relative flex h-10 flex-1 items-center justify-center gap-2 text-sm font-black transition",
            index > 0 ? "-ml-[22px] pl-7" : "rounded-l-full",
            index === items.length - 1 ? "rounded-r-full pr-4" : "pr-7",
            step === index
              ? "z-20 bg-kw-c-ef2334 text-white shadow-kw-step"
              : "z-10 bg-kw-c-eef2f6 text-kw-c-526074 hover:bg-kw-c-e3e9f1 dark:bg-kw-c-66717f dark:text-kw-c-dbe7f5 dark:hover:bg-kw-c-758292",
            !canOpen(index) ? "cursor-not-allowed opacity-45" : "cursor-pointer",
          ].join(" ")}
          disabled={!canOpen(index)}
          key={item.label}
          style={{ clipPath: stepClipPath(index, items.length) }}
          type="button"
          onClick={() => setStep(index)}
        >
          {item.icon}
          <span>
            {index + 1}. {item.label}
          </span>
        </button>
      ))}
    </div>
  );
}

export function WizardCard({
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
      className={[
        "min-h-[280px] rounded-xl border p-4 shadow-kw-inset-top transition duration-300",
        active
          ? "border-kw-c-d6dfec bg-white/80 text-kw-c-0a284b shadow-kw-wizard dark:border-kw-c-526276 dark:bg-kw-c-1b2a3c dark:text-kw-c-f3f6fa"
          : "pointer-events-none border-kw-c-dce3ee bg-kw-c-eef2f6/75 text-kw-c-7e8ea5 opacity-65 grayscale-[0.15] dark:border-kw-c-334155 dark:bg-kw-c-2b3644/72 dark:text-kw-c-8794a5",
      ].join(" ")}
      data-active={active}
    >
      <h2 className="mb-4 flex items-center gap-2 text-lg font-black">
        <span className="text-kw-c-ef2334 dark:text-kw-c-ff5a69">{icon}</span>
        {title}
      </h2>
      {children}
    </section>
  );
}

export function CourseMultiSelect({
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
        <button
          className="flex h-11 w-full cursor-pointer items-center justify-between gap-3 rounded-md border border-input bg-background px-3 text-left text-sm font-semibold outline-none transition hover:border-kw-c-ef2334 focus:border-kw-c-ef2334 dark:hover:border-kw-c-ff3b4f dark:focus:border-kw-c-ff3b4f"
          type="button"
        >
          <span
            className={
              selectedNames.length
                ? "truncate"
                : "truncate text-kw-c-687386 dark:text-kw-c-a7b0bf"
            }
          >
            {selectedNames.length
              ? selectedNames.join(", ")
              : labels.courseDropdownPlaceholder}
          </span>
          <ChevronDown className="size-4 shrink-0 text-kw-c-687386 dark:text-kw-c-a7b0bf" />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent className="z-[1200] border-kw-c-dce3ee dark:border-kw-c-3a4658 dark:bg-kw-c-1b2635">
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

export function CreateStudentError({
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
    <div className="mt-6 rounded-lg border border-kw-c-f87171 bg-kw-c-fff1f2 px-4 py-3 text-sm font-black text-kw-c-991b1b dark:bg-kw-c-3b1218 dark:text-kw-c-ffe4e6">
      {message}
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
