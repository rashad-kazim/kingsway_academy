"use client";

import { useState } from "react";
import { useActionState } from "react";
import { useFormStatus } from "react-dom";
import { Eye, EyeOff } from "lucide-react";
import Image from "next/image";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { loginAction, type LoginActionState } from "@/lib/auth/actions";
import { useSubmitLock } from "@/lib/forms/use-submit-lock";
import { useDevAssetSrc } from "@/lib/use-dev-asset-src";

type LoginFormLabels = {
  title: string;
  description: string;
  email: string;
  password: string;
  forgotPassword: string;
  submit: string;
  pending: string;
  invalidCredentials: string;
  backendUnavailable: string;
  invalidInput: string;
};

type LoginFormProps = {
  locale: string;
  nextPath?: string;
  labels: LoginFormLabels;
};

const initialState: LoginActionState = {};

export function LoginForm({ locale, nextPath, labels }: LoginFormProps) {
  const [state, formAction] = useActionState(loginAction, initialState);
  const [showPassword, setShowPassword] = useState(false);
  const logoSrc = useDevAssetSrc("/images/kingsway-mark.png");
  const message = errorMessage(state.error, labels);
  const hasCredentialError =
    state.error === "invalid_credentials" || state.error === "invalid_input";
  const emailInvalid = Boolean(state.fieldErrors?.email) || hasCredentialError;
  const passwordInvalid =
    Boolean(state.fieldErrors?.password) || hasCredentialError;

  return (
    <div className="relative z-30 w-[80vw] max-w-[296px] sm:w-full sm:max-w-[360px]">
      <div className="mb-8 flex justify-center">
        <Image
          src={logoSrc}
          width={501}
          height={499}
          priority
          unoptimized={process.env.NODE_ENV === "development"}
          alt="Kingsway"
          className="h-auto w-[120px] sm:w-[138px]"
        />
      </div>

      <div className="mb-7 space-y-3">
        <h1 className="text-kw-30 font-black leading-none text-kw-c-0a284b">
          {labels.title}
        </h1>
        <p className="text-kw-15 font-medium text-kw-c-687386">
          {labels.description}
        </p>
      </div>

      <form action={formAction} className="space-y-6">
        <input name="locale" type="hidden" value={locale} />
        <input name="next" type="hidden" value={nextPath ?? ""} />

        {message ? (
          <Alert
            variant="destructive"
            className="border-kw-c-ef2334/25 bg-kw-c-ef2334/10 text-kw-c-ef2334"
          >
            <AlertDescription>{message}</AlertDescription>
          </Alert>
        ) : null}

        <div className="space-y-2">
          <Label htmlFor="email" className="text-kw-14 font-semibold text-kw-c-0a284b">
            {labels.email}
          </Label>
          <Input
            id="email"
            name="email"
            type="email"
            autoComplete="email"
            aria-invalid={emailInvalid}
            className="h-[44px] rounded-[5px] border-kw-c-0a284b bg-kw-c-fdfdfd px-3 text-kw-15 text-black shadow-none focus-visible:border-kw-c-0a284b focus-visible:ring-2 focus-visible:ring-kw-c-0a284b/20 aria-invalid:border-kw-c-ef2334 aria-invalid:focus-visible:border-kw-c-ef2334 aria-invalid:focus-visible:ring-kw-c-ef2334/20"
            required
          />
        </div>

        <div className="space-y-2">
          <Label
            htmlFor="password"
            className="text-kw-14 font-semibold text-kw-c-0a284b"
          >
            {labels.password}
          </Label>
          <div className="relative">
            <Input
              id="password"
              name="password"
              type={showPassword ? "text" : "password"}
              autoComplete="current-password"
              aria-invalid={passwordInvalid}
              className="h-[44px] rounded-[5px] border-kw-c-0a284b bg-kw-c-fdfdfd px-3 pr-11 text-kw-15 text-black shadow-none focus-visible:border-kw-c-0a284b focus-visible:ring-2 focus-visible:ring-kw-c-0a284b/20 aria-invalid:border-kw-c-ef2334 aria-invalid:focus-visible:border-kw-c-ef2334 aria-invalid:focus-visible:ring-kw-c-ef2334/20"
              required
            />
            <button
              type="button"
              aria-label={showPassword ? "Hide password" : "Show password"}
              onClick={() => setShowPassword((current) => !current)}
              className="absolute right-3 top-1/2 flex size-8 -translate-y-1/2 items-center justify-center rounded-full text-kw-c-0a284b/60 transition-colors hover:bg-kw-c-eef2f7 hover:text-kw-c-0a284b focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-kw-c-0a284b/25"
            >
              {showPassword ? (
                <EyeOff className="size-4" />
              ) : (
                <Eye className="size-4" />
              )}
            </button>
          </div>
        </div>

        <button
          type="button"
          className="text-kw-14 font-semibold text-kw-c-0a284b transition-colors hover:text-kw-c-0f3764 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-kw-c-0a284b/25"
        >
          {labels.forgotPassword}
        </button>

        <SubmitButton labels={labels} />
      </form>
    </div>
  );
}

function SubmitButton({ labels }: { labels: LoginFormLabels }) {
  const { pending } = useFormStatus();
  const submitLock = useSubmitLock(pending);

  return (
    <Button
      className="mt-4 h-[48px] w-full rounded-[9px] bg-kw-c-0a284b text-kw-15 font-bold text-white shadow-kw-login-button transition-colors hover:bg-kw-c-0f3764"
      type="submit"
      disabled={pending || submitLock.locked}
      onClick={submitLock.onClick}
    >
      {pending ? labels.pending : labels.submit}
    </Button>
  );
}

function errorMessage(error: string | undefined, labels: LoginFormLabels) {
  switch (error) {
    case "invalid_credentials":
      return labels.invalidCredentials;
    case "backend_unavailable":
      return labels.backendUnavailable;
    case "invalid_input":
      return labels.invalidInput;
    default:
      return null;
  }
}
