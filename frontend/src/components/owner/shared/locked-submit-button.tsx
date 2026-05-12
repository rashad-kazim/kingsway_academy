"use client";

import { Save } from "lucide-react";
import { useFormStatus } from "react-dom";
import { Button } from "@/components/ui/button";
import { useSubmitLock } from "@/lib/forms/use-submit-lock";

type LockedSubmitButtonProps = {
  disabled: boolean;
  label: string;
  pending: string;
};

export function LockedSubmitButton({
  disabled,
  label,
  pending,
}: LockedSubmitButtonProps) {
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
