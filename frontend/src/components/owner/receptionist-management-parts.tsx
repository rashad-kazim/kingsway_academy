import { CheckCircle2, XCircle } from "lucide-react";
import type { StaffManagementState } from "@/lib/owner/actions";
import { cn } from "@/lib/utils";
import type { ReceptionistManagementLabels } from "./receptionist-management-view";

type EmailStatus = "idle" | "checking" | "available" | "taken" | "invalid";

export function StaffFormError({
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
    <div className="rounded-lg border border-kw-c-f87171 bg-kw-c-3b1218 px-4 py-3 text-sm font-bold text-kw-c-ffe4e6">
      {message}
    </div>
  );
}

export function EmailStatusIcon({ status }: { status: EmailStatus }) {
  if (status === "available") {
    return (
      <CheckCircle2 className="absolute right-3 top-1/2 size-5 -translate-y-1/2 text-kw-c-0ca678" />
    );
  }
  if (status === "taken" || status === "invalid") {
    return (
      <XCircle className="absolute right-3 top-1/2 size-5 -translate-y-1/2 text-kw-c-ef2334 dark:text-kw-c-ff3b4f" />
    );
  }
  return null;
}

export function EmailStatusText({
  labels,
  status,
}: {
  labels: ReceptionistManagementLabels;
  status: EmailStatus;
}) {
  if (status === "checking") {
    return (
      <p className="text-xs font-bold text-kw-c-7e8ea5">
        {labels.emailChecking}
      </p>
    );
  }
  if (status === "available") {
    return (
      <p className="text-xs font-bold text-kw-c-0ca678">
        {labels.emailAvailable}
      </p>
    );
  }
  if (status === "taken") {
    return (
      <p className="text-xs font-bold text-kw-c-ef2334">{labels.emailTaken}</p>
    );
  }
  return null;
}

export function StatusBadge({
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
          ? "border-kw-c-0ca678/30 bg-kw-c-0ca678 text-white"
          : "border-kw-c-64748b/30 bg-kw-c-64748b text-white",
      )}
    >
      {isActive ? labels.active : labels.inactive}
    </span>
  );
}

export function genderLabel(
  value: string | undefined,
  labels: ReceptionistManagementLabels,
) {
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

export function initials(firstName: string, lastName: string) {
  return `${firstName[0] ?? ""}${lastName[0] ?? ""}`.toUpperCase();
}
