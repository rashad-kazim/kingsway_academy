"use client";

import { useState } from "react";
import { useActionState } from "react";
import { useFormStatus } from "react-dom";
import { Eye, EyeOff } from "lucide-react";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { loginAction, type LoginActionState } from "@/lib/auth/actions";

type LoginFormLabels = {
  title: string;
  description: string;
  email: string;
  password: string;
  forgotPassword: string;
  submit: string;
  pending: string;
  signUpPrompt: string;
  signUp: string;
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
  const message = errorMessage(state.error, labels);

  return (
    <div className="w-full max-w-[300px]">
      <div className="mb-8 space-y-4">
        <h2 className="text-[34px] font-black leading-none text-white">
          {labels.title}
        </h2>
        <p className="text-[13px] font-medium text-white/62">
          {labels.description}
        </p>
      </div>

      <form action={formAction} className="space-y-5">
          <input name="locale" type="hidden" value={locale} />
          <input name="next" type="hidden" value={nextPath ?? ""} />

          {message ? (
            <Alert
              variant="destructive"
              className="border-red-400/30 bg-red-500/10 text-red-100"
            >
              <AlertDescription>{message}</AlertDescription>
            </Alert>
          ) : null}

          <div className="space-y-1">
            <Label htmlFor="email" className="text-xs font-medium text-white/46">
              {labels.email}
            </Label>
            <Input
              id="email"
              name="email"
              type="email"
              autoComplete="email"
              aria-invalid={Boolean(state.fieldErrors?.email)}
              className="h-8 rounded-none border-0 border-b border-white/22 bg-transparent px-0 text-sm text-white shadow-none placeholder:text-white/35 focus-visible:border-[#9b68e8] focus-visible:ring-0"
              required
            />
          </div>

          <div className="space-y-1">
            <Label
              htmlFor="password"
              className="text-xs font-medium text-white/46"
            >
              {labels.password}
            </Label>
            <div className="relative">
              <Input
                id="password"
                name="password"
                type={showPassword ? "text" : "password"}
                autoComplete="current-password"
                aria-invalid={Boolean(state.fieldErrors?.password)}
                className="h-8 rounded-none border-0 border-b border-white/22 bg-transparent px-0 pr-9 text-sm text-white shadow-none placeholder:text-white/35 focus-visible:border-[#9b68e8] focus-visible:ring-0"
                required
              />
              <button
                type="button"
                aria-label={showPassword ? "Hide password" : "Show password"}
                onClick={() => setShowPassword((current) => !current)}
                className="absolute right-0 top-1/2 flex size-7 -translate-y-1/2 items-center justify-center text-white/42 transition-colors hover:text-white/80 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#9b68e8]/70"
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
            className="text-xs font-medium text-white/42 transition-colors hover:text-white/75 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#9b68e8]/70"
          >
            {labels.forgotPassword}
          </button>

          <SubmitButton labels={labels} />
        </form>

      <div className="mt-28 flex items-center justify-between gap-4 text-xs text-white/38">
        <span>{labels.signUpPrompt}</span>
        <button
          type="button"
          className="h-9 rounded-[7px] bg-white/10 px-5 text-xs font-semibold text-white/76 transition-colors hover:bg-white/16 hover:text-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#9b68e8]/70"
        >
          {labels.signUp}
        </button>
      </div>
    </div>
  );
}

function SubmitButton({ labels }: { labels: LoginFormLabels }) {
  const { pending } = useFormStatus();

  return (
    <Button
      className="mt-6 h-9 w-full rounded-[8px] bg-[#9b68e8] text-xs font-semibold text-white shadow-none transition-colors hover:bg-[#ab79f0]"
      type="submit"
      disabled={pending}
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
