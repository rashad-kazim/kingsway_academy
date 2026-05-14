import { CheckCircle2, XCircle } from "lucide-react";

export type EmailAvailabilityStatus =
  | "idle"
  | "checking"
  | "available"
  | "taken"
  | "invalid";

type EmailAvailabilityLabels = {
  emailAvailable: string;
  emailChecking: string;
  emailTaken: string;
};

export function EmailAvailabilityIcon({
  status,
}: {
  status: EmailAvailabilityStatus;
}) {
  if (status === "available") {
    return (
      <CheckCircle2 className="absolute right-3 top-1/2 size-5 -translate-y-1/2 text-emerald-500" />
    );
  }

  if (status === "taken" || status === "invalid") {
    return (
      <XCircle className="absolute right-3 top-1/2 size-5 -translate-y-1/2 text-kw-c-ef2334 dark:text-kw-c-ff3b4f" />
    );
  }

  return null;
}

export function EmailAvailabilityText({
  labels,
  status,
}: {
  labels: EmailAvailabilityLabels;
  status: EmailAvailabilityStatus;
}) {
  const text =
    status === "checking"
      ? labels.emailChecking
      : status === "available"
        ? labels.emailAvailable
        : status === "taken"
          ? labels.emailTaken
          : "";

  if (!text) {
    return null;
  }

  return (
    <p
      className={`text-xs font-bold ${
        status === "available"
          ? "text-emerald-600"
          : status === "checking"
            ? "text-kw-c-687386"
            : "text-kw-c-ef2334 dark:text-kw-c-ff3b4f"
      }`}
    >
      {text}
    </p>
  );
}
