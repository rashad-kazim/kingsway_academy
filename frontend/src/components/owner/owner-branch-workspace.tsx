"use client";

import Image from "next/image";
import Link from "next/link";
import { type ReactNode, useEffect, useMemo, useRef, useState } from "react";
import { useActionState } from "react";
import { useFormStatus } from "react-dom";
import {
  AlertCircle,
  Banknote,
  CalendarDays,
  CheckCircle2,
  GraduationCap,
  ImagePlus,
  MapPin,
  Plus,
  School,
  Settings,
  Users,
  XCircle,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  TopbarControls,
  type ChromeLabels,
} from "@/components/layout/topbar-controls";
import type { Branch } from "@/lib/api/types";
import {
  createBranchAction,
  type CreateBranchState,
} from "@/lib/owner/actions";
import { cn } from "@/lib/utils";
import { useDevAssetSrc } from "@/lib/use-dev-asset-src";

export type BranchStats = {
  branch_id: string;
  active_students: number;
  active_teachers: number;
};

export type OwnerOnboardingLabels = {
  initialBranchSetup: string;
  title: string;
  description: string;
  branchPhoto: string;
  uploadBranchPhoto: string;
  branchName: string;
  address: string;
  duplicateBranchName: string;
  branchNameRequired: string;
  addressRequired: string;
  duplicateError: string;
  backendError: string;
  unauthorizedError: string;
  invalidPhotoError: string;
  photoTooLargeError: string;
  cancel: string;
  save: string;
  saving: string;
  branchesTitle: string;
  branchesDescription: string;
  branchAddress: string;
  branchCardSuffix: string;
  quickFinance: string;
  quickSchedule: string;
  quickSettings: string;
  student: string;
  teacher: string;
  addBranch: string;
};

export type OwnerBranchWorkspaceLabels = {
  common: ChromeLabels;
  onboarding: OwnerOnboardingLabels;
};

type OwnerBranchWorkspaceProps = {
  branches: Branch[];
  labels: OwnerBranchWorkspaceLabels;
  locale: string;
  stats: BranchStats[];
  userName: string;
};

const initialState: CreateBranchState = {};
type WorkspaceMode = "create" | "list";
const MAX_BRANCH_PHOTO_BYTES = 10 * 1024 * 1024;
const ALLOWED_BRANCH_PHOTO_TYPES = new Set([
  "image/jpeg",
  "image/png",
  "image/webp",
]);

export function OwnerBranchWorkspace({
  branches: initialBranches,
  labels,
  locale,
  stats,
  userName,
}: OwnerBranchWorkspaceProps) {
  const [branches, setBranches] = useState(initialBranches);
  const [mode, setMode] = useState<WorkspaceMode>(
    initialBranches.length > 0 ? "list" : "create",
  );
  const [name, setName] = useState("");
  const [address, setAddress] = useState("");
  const [photoPreview, setPhotoPreview] = useState<string | null>(null);
  const [photoError, setPhotoError] = useState<PhotoError>(null);
  const [branchPhotos, setBranchPhotos] = useState<Record<string, string>>({});
  const logoSrc = useDevAssetSrc("/images/kingsway-mark.png");
  const [createState, createFormAction] = useActionState(
    createBranchAction,
    initialState,
  );
  const handledCreateBranchID = useRef<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    const nextPhotos: Record<string, string> = {};
    for (const branch of branches) {
      const stored = window.localStorage.getItem(photoKey(branch.id));
      if (stored) {
        nextPhotos[branch.id] = stored;
      }
    }
    window.queueMicrotask(() => {
      if (!cancelled) {
        setBranchPhotos(nextPhotos);
      }
    });
    return () => {
      cancelled = true;
    };
  }, [branches]);

  useEffect(() => {
    if (
      !createState.branch ||
      handledCreateBranchID.current === createState.branch.id
    ) {
      return;
    }

    const createdBranch = createState.branch;
    handledCreateBranchID.current = createdBranch.id;

    window.queueMicrotask(() => {
      setBranches((current) =>
        current.some((branch) => branch.id === createdBranch.id)
          ? current
          : [...current, createdBranch],
      );

      if (createdBranch.photo_url) {
        setBranchPhotos((current) => ({
          ...current,
          [createdBranch.id]: createdBranch.photo_url ?? current[createdBranch.id],
        }));
      }

      setName("");
      setAddress("");
      setPhotoPreview(null);
      setPhotoError(null);
      setMode("list");
    });
  }, [createState.branch]);

  const normalizedName = normalizeName(name);
  const generatedSlug = slugFromName(name);
  const duplicateName = useMemo(
    () =>
      normalizedName.length > 0 &&
      branches.some(
        (branch) =>
          normalizeName(branch.name) === normalizedName ||
          branch.slug === generatedSlug,
      ),
    [branches, generatedSlug, normalizedName],
  );
  const nameAccepted = normalizedName.length >= 2 && !duplicateName;
  const statsByBranch = useMemo(
    () => new Map(stats.map((item) => [item.branch_id, item])),
    [stats],
  );

  function resetForm() {
    setName("");
    setAddress("");
    setPhotoPreview(null);
    setPhotoError(null);
  }

  function startCreate() {
    resetForm();
    setMode("create");
  }

  return (
    <main className="fixed inset-0 overflow-hidden bg-[#f7f8fb] text-[#0a284b] transition-colors dark:bg-[#0b1622] dark:text-[#f3f6fa]">
      <header className="relative z-20 flex h-20 shrink-0 items-center justify-between border-b border-transparent bg-[#0a284b] px-10 text-white shadow-sm dark:border-[#293445] dark:bg-[#121f2d] dark:text-[#f3f6fa]">
        <Link
          className="flex cursor-pointer items-center gap-4 dark:text-[#f3f6fa]"
          href={`/${locale}/dashboard/owner`}
        >
          <Image
            src={logoSrc}
            alt={labels.common.brand}
            width={501}
            height={499}
            priority
            unoptimized={process.env.NODE_ENV === "development"}
            className="size-14 object-contain drop-shadow-[0_1px_3px_rgba(255,255,255,0.45)]"
          />
          <div>
            <div className="text-sm font-semibold uppercase tracking-[0.22em] text-[#ff4b55] dark:text-[#ff3b4f]">
              {labels.common.brand}
            </div>
            <div className="text-xl font-black leading-none">
              {labels.common.commandCenter}
            </div>
          </div>
        </Link>

        <TopbarControls
          labels={labels.common}
          locale={locale}
          role="owner"
          userName={userName}
        />
      </header>

      <section
        className={cn(
          "h-[calc(100svh-5rem)] overflow-hidden",
          mode === "list" ? "w-full" : "mx-auto max-w-[1420px] px-10",
        )}
      >
        {mode !== "list" ? (
          <BranchSetupForm
            address={address}
            duplicateName={duplicateName}
            labels={labels.onboarding}
            name={name}
            nameAccepted={nameAccepted}
            photoPreview={photoPreview}
            photoError={photoError}
            serverState={createState}
            setAddress={setAddress}
            setMode={setMode}
            setName={setName}
            setPhotoError={setPhotoError}
            setPhotoPreview={setPhotoPreview}
            showCancel={branches.length > 0}
            formAction={createFormAction}
          />
        ) : (
          <BranchCards
            branchPhotos={branchPhotos}
            branches={branches}
            labels={labels.onboarding}
            locale={locale}
            onAddBranch={startCreate}
            statsByBranch={statsByBranch}
          />
        )}
      </section>
    </main>
  );
}

function BranchSetupForm({
  address,
  duplicateName,
  formAction,
  labels,
  name,
  nameAccepted,
  photoPreview,
  photoError,
  serverState,
  setAddress,
  setMode,
  setName,
  setPhotoError,
  setPhotoPreview,
  showCancel,
}: {
  address: string;
  duplicateName: boolean;
  formAction: (payload: FormData) => void;
  labels: OwnerOnboardingLabels;
  name: string;
  nameAccepted: boolean;
  photoPreview: string | null;
  photoError: PhotoError;
  serverState: CreateBranchState;
  setAddress: (value: string) => void;
  setMode: (value: WorkspaceMode) => void;
  setName: (value: string) => void;
  setPhotoError: (value: PhotoError) => void;
  setPhotoPreview: (value: string | null) => void;
  showCancel: boolean;
}) {
  const nameHasError =
    duplicateName || Boolean(serverState.fieldErrors?.name?.length);
  const addressHasError = Boolean(serverState.fieldErrors?.address?.length);

  return (
    <div className="grid h-full items-center gap-[3.6rem] lg:grid-cols-2">
      <div className="animate-[owner-copy-in_650ms_ease-out_both] justify-self-center space-y-7">
        <div
          className="inline-flex items-center gap-3 rounded-full border border-[#dce3ee] bg-white px-6 py-3 text-lg font-bold text-[#0a284b] shadow-sm dark:border-[#3a4658] dark:bg-[#202b3a] dark:text-[#f3f6fa]"
        >
          <School className="size-5 text-[#ef2334] dark:text-[#ff3b4f]" />
          {labels.initialBranchSetup}
        </div>
        <div className="space-y-5">
          <h1 className="max-w-[620px] text-[clamp(46px,4.2vw,68px)] font-black leading-[1.05] tracking-tight">
            <HighlightedTitle title={labels.title} />
          </h1>
          <p className="max-w-xl text-[18px] leading-8 text-[#59667a] dark:text-[#a7b0bf]">
            {labels.description}
          </p>
        </div>
      </div>

      <Card className="w-full max-w-[720px] justify-self-center rounded-lg border-[#dce3ee] bg-white shadow-[0_24px_80px_rgba(10,40,75,0.10)] dark:border-[#3a4658] dark:bg-[linear-gradient(180deg,#1e2a39_0%,#182331_100%)] dark:text-[#f3f6fa] dark:shadow-[0_20px_50px_rgba(0,0,0,0.28)]">
        <CardContent className="p-7">
          <form action={formAction} className="space-y-5">
            <div>
              <Label className="mb-3 block text-sm font-bold text-[#0a284b] dark:text-[#f3f6fa]">
                {labels.branchPhoto}
              </Label>
              <label className="group flex h-36 cursor-pointer flex-col items-center justify-center gap-3 rounded-lg bg-[#f7f8fb] text-center transition-colors hover:bg-[#f1f4f8] dark:bg-[#202b3a] dark:hover:bg-[#263448]">
                {photoPreview ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={photoPreview}
                    alt=""
                    className="size-28 rounded-full border border-dashed border-[#b9c5d6] object-cover shadow-md dark:border-[#414d60]"
                  />
                ) : (
                  <span className="flex size-28 items-center justify-center rounded-full border border-dashed border-[#b9c5d6] bg-white text-[#0a284b] shadow-inner transition-colors group-hover:border-[#0a284b] dark:border-[#414d60] dark:bg-[#2a3444] dark:text-[#f3f6fa] dark:group-hover:border-[#54657c]">
                    <ImagePlus className="size-8" />
                  </span>
                )}
                <input
                  accept="image/jpeg,image/png,image/webp,.jpg,.jpeg,.png,.webp"
                  className="sr-only"
                  name="photo"
                  type="file"
                  onChange={(event) => {
                    const file = event.target.files?.[0];
                    if (!file) {
                      setPhotoPreview(null);
                      setPhotoError(null);
                      return;
                    }
                    const nextPhotoError = validateSelectedBranchPhoto(file);
                    if (nextPhotoError) {
                      event.currentTarget.value = "";
                      setPhotoPreview(null);
                      setPhotoError(nextPhotoError);
                      return;
                    }
                    setPhotoError(null);
                    const reader = new FileReader();
                    reader.onload = () =>
                      setPhotoPreview(
                        typeof reader.result === "string" ? reader.result : null,
                      );
                    reader.readAsDataURL(file);
                  }}
                />
              </label>
              {photoError === "invalid_photo" ? (
                <InlineError>{labels.invalidPhotoError}</InlineError>
              ) : null}
              {photoError === "photo_too_large" ? (
                <InlineError>{labels.photoTooLargeError}</InlineError>
              ) : null}
            </div>

            <div className="space-y-2">
              <Label htmlFor="branch-name" className="font-bold text-[#0a284b] dark:text-[#f3f6fa]">
                {labels.branchName}
              </Label>
              <div className="relative">
                <Input
                  id="branch-name"
                  name="name"
                  value={name}
                  onChange={(event) => setName(event.target.value)}
                  className={cn(
                    "h-11 rounded-md border-[#0a284b] bg-white pr-11 text-black focus-visible:ring-[#0a284b]/20 dark:border-[#3a4658] dark:bg-[#0b1622] dark:text-[#f3f6fa] dark:focus-visible:border-[#54657c] dark:focus-visible:ring-[#54657c]/30",
                    nameHasError &&
                      "border-[#ef2334] focus-visible:ring-[#ef2334]/20 dark:border-[#ff5a69] dark:focus-visible:border-[#ff5a69] dark:focus-visible:ring-[#ff5a69]/30",
                  )}
                  required
                />
                <div className="absolute right-3 top-1/2 -translate-y-1/2">
                  {nameAccepted ? (
                    <CheckCircle2 className="size-5 text-emerald-600" />
                  ) : name.trim().length > 0 ? (
                    <XCircle className="size-5 text-[#ef2334] dark:text-[#ff5a69]" />
                  ) : null}
                </div>
              </div>
              {duplicateName ? (
                <InlineError>{labels.duplicateBranchName}</InlineError>
              ) : null}
              {serverState.fieldErrors?.name ? (
                <InlineError>{labels.branchNameRequired}</InlineError>
              ) : null}
            </div>

            <div className="space-y-2">
              <Label htmlFor="branch-address" className="font-bold text-[#0a284b] dark:text-[#f3f6fa]">
                {labels.address}
              </Label>
              <Textarea
                id="branch-address"
                name="address"
                value={address}
                onChange={(event) => setAddress(event.target.value)}
                className={cn(
                  "min-h-24 resize-none rounded-md border-[#0a284b] bg-white text-black focus-visible:ring-[#0a284b]/20 dark:border-[#3a4658] dark:bg-[#0b1622] dark:text-[#f3f6fa] dark:focus-visible:border-[#54657c] dark:focus-visible:ring-[#54657c]/30",
                  addressHasError &&
                    "border-[#ef2334] focus-visible:ring-[#ef2334]/20 dark:border-[#ff5a69] dark:focus-visible:border-[#ff5a69] dark:focus-visible:ring-[#ff5a69]/30",
                )}
                required
              />
              {serverState.fieldErrors?.address ? (
                <InlineError>{labels.addressRequired}</InlineError>
              ) : null}
            </div>

            {serverState.error === "duplicate" ? (
              <FormError>{labels.duplicateError}</FormError>
            ) : null}
            {serverState.error === "backend" ? (
              <FormError>{labels.backendError}</FormError>
            ) : null}
            {serverState.error === "unauthorized" ? (
              <FormError>{labels.unauthorizedError}</FormError>
            ) : null}
            {serverState.error === "invalid_photo" ? (
              <FormError>{labels.invalidPhotoError}</FormError>
            ) : null}
            {serverState.error === "photo_too_large" ? (
              <FormError>{labels.photoTooLargeError}</FormError>
            ) : null}

            <div className="flex items-center justify-end gap-3 pt-1">
              {showCancel ? (
                <Button type="button" variant="outline" onClick={() => setMode("list")}>
                  {labels.cancel}
                </Button>
              ) : null}
              <SaveBranchButton
                disabled={
                  !nameAccepted ||
                  address.trim().length < 3 ||
                  photoError !== null
                }
                labels={labels}
              />
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}

type PhotoError = "invalid_photo" | "photo_too_large" | null;

function validateSelectedBranchPhoto(file: File): PhotoError {
  if (!ALLOWED_BRANCH_PHOTO_TYPES.has(file.type)) {
    return "invalid_photo";
  }
  if (file.size > MAX_BRANCH_PHOTO_BYTES) {
    return "photo_too_large";
  }

  return null;
}

function HighlightedTitle({ title }: { title: string }) {
  const [before, after] = title.split("Kingsway");
  if (after === undefined) {
    return title;
  }

  return (
    <>
      {before}
      <span className="text-[#ef2334] dark:text-[#ff3b4f]">Kingsway</span>
      {after}
    </>
  );
}

function InlineError({ children }: { children: ReactNode }) {
  return (
    <p className="mt-2 flex items-center gap-1.5 text-sm font-semibold text-[#ef2334] dark:text-[#ff5a69]">
      <AlertCircle className="size-4 shrink-0 text-[#ef2334] dark:text-[#ff3b4f]" />
      <span>{children}</span>
    </p>
  );
}

function FormError({ children }: { children: ReactNode }) {
  return (
    <div className="flex items-start gap-2 rounded-md border border-[#ef2334]/25 bg-[#ef2334]/10 px-3 py-2 text-sm font-semibold text-[#ef2334] dark:border-[#ff3b4f]/45 dark:bg-[#3a1e2a] dark:text-[#f3f6fa]">
      <AlertCircle className="mt-0.5 size-4 shrink-0 text-[#ef2334] dark:text-[#ff5a69]" />
      <span>{children}</span>
    </div>
  );
}

function SaveBranchButton({
  disabled,
  labels,
}: {
  disabled: boolean;
  labels: OwnerOnboardingLabels;
}) {
  const { pending } = useFormStatus();
  return (
    <Button
      disabled={disabled || pending}
      className="h-11 min-w-36 bg-[#15803d] font-bold text-white transition-colors hover:bg-[#166534] dark:bg-[#22c55e] dark:text-[#052e16] dark:hover:bg-[#4ade80]"
      type="submit"
    >
      {pending ? labels.saving : labels.save}
    </Button>
  );
}

function BranchCards({
  branchPhotos,
  branches,
  labels,
  locale,
  onAddBranch,
  statsByBranch,
}: {
  branchPhotos: Record<string, string>;
  branches: Branch[];
  labels: OwnerOnboardingLabels;
  locale: string;
  onAddBranch: () => void;
  statsByBranch: Map<string, BranchStats>;
}) {
  const hasWrappedCards = branches.length + 1 > 3;

  return (
    <div className="h-full w-full overflow-y-auto overscroll-contain">
      <div
        className={cn(
          "mx-auto flex min-h-full w-full flex-col items-center gap-8 px-10",
          hasWrappedCards ? "justify-start py-10" : "justify-center pb-6",
        )}
      >
        <div className="text-center">
          <h1 className="text-[clamp(30px,2.8vw,44px)] font-black leading-tight tracking-tight text-[#0a284b] dark:text-[#f3f6fa]">
            {labels.branchesDescription}
          </h1>
        </div>

        <div className="grid w-full max-w-[1340px] grid-cols-1 items-start justify-items-center gap-7 md:grid-cols-2 xl:grid-cols-3">
          {branches.map((branch) => {
            const branchStats = statsByBranch.get(branch.id);
            return (
              <div
                className="flex w-full max-w-[420px] flex-col items-center gap-[10px]"
                key={branch.id}
              >
                <Card className="group h-[420px] w-full gap-0 overflow-hidden rounded-lg border border-[#e1e8f2] bg-white/80 py-0 shadow-[0_18px_55px_rgba(10,40,75,0.12)] backdrop-blur-xl transition duration-300 hover:-translate-y-0.5 hover:bg-white/95 hover:shadow-[0_22px_65px_rgba(10,40,75,0.18)] dark:border-[#3a4658] dark:bg-[linear-gradient(180deg,#1e2a39_0%,#182331_100%)] dark:shadow-[0_20px_50px_rgba(0,0,0,0.28)] dark:hover:border-[#4a5a70] dark:hover:bg-[linear-gradient(180deg,#223044_0%,#1b2635_100%)]">
                  <CardContent className="flex h-full flex-col items-center p-7 text-center">
                    <Link
                      className="flex w-full cursor-pointer flex-col items-center"
                      href={`/${locale}/dashboard/owner?branch_id=${branch.id}`}
                    >
                      <BranchPhoto
                        name={branch.name}
                        src={branch.photo_url ?? branchPhotos[branch.id]}
                      />
                      <h2 className="mt-7 line-clamp-1 max-w-full text-center text-2xl font-black leading-tight text-[#0a284b] dark:text-[#f3f6fa]">
                        {branchTitle(branch.name, labels.branchCardSuffix)}
                      </h2>
                      <div className="mt-3 flex min-h-12 w-full items-start justify-center gap-2 text-sm font-semibold leading-6 text-[#687386] dark:text-[#a7b0bf]">
                        <MapPin className="mt-1 size-4 shrink-0 text-[#ef2334] dark:text-[#ff3b4f]" />
                        <span className="line-clamp-2 text-left">
                          {branch.address || "-"}
                        </span>
                      </div>
                    </Link>
                    <BranchQuickActions
                      branchID={branch.id}
                      labels={labels}
                      locale={locale}
                    />
                    <div className="mt-auto flex flex-wrap items-center justify-center gap-3">
                      <BranchStatBadge
                        icon={<Users className="size-4" />}
                        label={labels.student}
                        value={branchStats?.active_students ?? 0}
                      />
                      <BranchStatBadge
                        icon={<GraduationCap className="size-4" />}
                        label={labels.teacher}
                        value={branchStats?.active_teachers ?? 0}
                      />
                    </div>
                  </CardContent>
                </Card>
              </div>
            );
          })}

          <div className="flex w-full max-w-[420px] flex-col items-center gap-[10px]">
            <button
              aria-label={labels.addBranch}
              className="group/add flex h-[420px] w-full cursor-pointer flex-col items-center justify-center rounded-lg border border-transparent bg-white/45 text-[#0a284b] shadow-[0_18px_55px_rgba(10,40,75,0.08)] backdrop-blur-xl transition duration-300 hover:-translate-y-0.5 hover:bg-white/75 hover:shadow-[0_22px_65px_rgba(10,40,75,0.14)] dark:border-[#3a4658]/70 dark:bg-[#1b2635]/70 dark:text-[#f3f6fa] dark:shadow-[0_20px_50px_rgba(0,0,0,0.18)] dark:hover:border-[#4a5a70] dark:hover:bg-[#202d3e]"
              type="button"
              onClick={onAddBranch}
            >
              <span className="flex size-20 items-center justify-center rounded-full bg-[#ef2334]/[0.08] text-[#0a284b] transition duration-300 group-hover/add:scale-110 group-hover/add:bg-[#ef2334]/[0.14] dark:bg-[#3a1e2a] dark:text-[#f3f6fa] dark:group-hover/add:bg-[#ff3b4f]/[0.14]">
                <Plus className="size-12 stroke-[2.4] transition duration-300 group-hover/add:rotate-90 group-hover/add:scale-110" />
              </span>
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

function BranchPhoto({ name, src }: { name: string; src?: string }) {
  if (src) {
    return (
      // eslint-disable-next-line @next/next/no-img-element
      <img
        alt=""
        className="size-28 rounded-full object-cover shadow-md ring-4 ring-[#f1f4f8] dark:border dark:border-[#414d60] dark:ring-[#414d60]/40"
        src={src}
      />
    );
  }

  return (
    <div className="flex size-28 items-center justify-center rounded-full bg-[#e3e3e8] text-3xl font-black text-[#0a284b] shadow-inner ring-4 ring-[#f1f4f8] dark:border dark:border-[#414d60] dark:bg-[radial-gradient(circle_at_top,#354154_0%,#263141_100%)] dark:text-[#f3f6fa] dark:ring-[#414d60]/40">
      {initials(name)}
    </div>
  );
}

function BranchQuickActions({
  branchID,
  labels,
  locale,
}: {
  branchID: string;
  labels: OwnerOnboardingLabels;
  locale: string;
}) {
  const actions = [
    {
      href: `/${locale}/dashboard/owner?branch_id=${branchID}&quick=salary`,
      icon: <Banknote className="size-5" />,
      label: labels.quickFinance,
    },
    {
      href: `/${locale}/dashboard/owner?branch_id=${branchID}&quick=schedule`,
      icon: <CalendarDays className="size-5" />,
      label: labels.quickSchedule,
    },
    {
      href: `/${locale}/dashboard/owner?branch_id=${branchID}&quick=settings`,
      icon: <Settings className="size-5" />,
      label: labels.quickSettings,
    },
  ];

  return (
    <div className="mt-5 mb-6 flex w-full justify-center gap-4">
      {actions.map((action) => (
        <Link
          aria-label={action.label}
          className="group/action flex w-20 cursor-pointer flex-col items-center gap-2"
          href={action.href}
          key={action.href}
        >
          <span className="flex size-11 items-center justify-center rounded-full border border-[#ef2334]/15 bg-[#ef2334]/[0.07] text-[#0a284b] transition duration-300 group-hover/action:-translate-y-0.5 group-hover/action:border-[#ef2334]/30 group-hover/action:bg-[#ef2334]/[0.14] group-hover/action:text-[#ef2334] dark:border-[#3b4658] dark:bg-[#202b3a] dark:text-[#e7ecf3] dark:group-hover/action:border-[#54657c] dark:group-hover/action:bg-[#263448] dark:group-hover/action:text-white">
            {action.icon}
          </span>
          <span className="h-4 translate-y-1 text-center text-[11px] font-black leading-none text-[#0a284b] opacity-0 transition duration-300 group-hover/action:translate-y-0 group-hover/action:opacity-100 dark:text-[#a7b0bf]">
            {action.label}
          </span>
        </Link>
      ))}
    </div>
  );
}

function BranchStatBadge({
  icon,
  label,
  value,
}: {
  icon: ReactNode;
  label: string;
  value: number;
}) {
  return (
    <Badge className="h-9 rounded-full border border-[#ef2334]/15 bg-[#ef2334]/[0.08] px-3 text-sm font-black text-[#0a284b] shadow-none dark:border-[#ff3b4f]/35 dark:bg-[#ff3b4f]/[0.08] dark:text-[#f0f3f8] dark:[&_svg]:text-[#ff3b4f]">
      {icon}
      <span>
        {value} {label}
      </span>
    </Badge>
  );
}

function branchTitle(name: string, suffix: string) {
  const trimmed = name.trim();
  if (!trimmed) {
    return suffix;
  }
  if (new RegExp(`\\b${escapeRegExp(suffix)}\\b`, "i").test(trimmed)) {
    return trimmed;
  }

  return `${trimmed} ${suffix}`;
}

function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function photoKey(branchID: string) {
  return `kingsway.branchPhoto.${branchID}`;
}

function initials(name: string) {
  const parts = name.trim().split(/\s+/).slice(0, 2);
  return parts.map((part) => part[0]?.toUpperCase()).join("") || "K";
}

function normalizeName(value: string) {
  return value.trim().toLowerCase().replace(/\s+/g, " ");
}

function slugFromName(value: string) {
  return value
    .trim()
    .toLowerCase()
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .replace(/\u0259/g, "e")
    .replace(/\u0131/g, "i")
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 80);
}
