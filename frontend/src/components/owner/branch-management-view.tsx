"use client";

import { useRouter } from "next/navigation";
import {
  useActionState,
  type ChangeEvent,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useFormStatus } from "react-dom";
import {
  Archive,
  Building2,
  DoorOpen,
  ImagePlus,
  MapPin,
  MoreHorizontal,
  Pencil,
  Plus,
  Trash2,
  Users,
  X,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Progress } from "@/components/ui/progress";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Textarea } from "@/components/ui/textarea";
import {
  BranchProfileAvatar,
  MiniStat,
  StaffCell,
} from "@/components/owner/branch-management-parts";
import {
  AvailabilityIcon,
  BranchDetailPage,
  getNameAvailability,
} from "@/components/owner/branch-detail-page";
import { ObjectCoverImage } from "@/components/owner/shared/object-cover-image";
import type { Branch, Room } from "@/lib/api/types";
import {
  createBranchAction,
  deleteBranchAction,
  type CreateBranchState,
  type DeleteBranchState,
} from "@/lib/owner/actions";
import type { StaffMember } from "@/lib/api/types";
import { useSubmitLock } from "@/lib/forms/use-submit-lock";
import type { BranchStats } from "./owner-branch-workspace";

export type BranchManagementLabels = {
  title: string;
  description: string;
  addNewBranch: string;
  totalBranches: string;
  active: string;
  archived: string;
  totalClassrooms: string;
  capacityStatus: string;
  branchProfile: string;
  branchName: string;
  branchID: string;
  address: string;
  rooms: string;
  staff: string;
  status: string;
  actions: string;
  inactive: string;
  unassigned: string;
  edit: string;
  archive: string;
  delete: string;
  manageRooms: string;
  deleteBranchTitle: string;
  deleteBranchIrreversibleWarning: string;
  deleteBranchDescription: string;
  deleteBranchNameLabel: string;
  deleteBranchNamePlaceholder: string;
  deleteBranchNameMismatch: string;
  deleteBranchWarningTitle: string;
  deleteBranchWarningDescription: string;
  deleteBranchBackendError: string;
  deleteBranchNotFoundError: string;
  deleteBranchUnauthorizedError: string;
  deleting: string;
  addBranchDescription: string;
  branchPhoto: string;
  branchNamePlaceholder: string;
  branchNameRequired: string;
  branchNameAvailable: string;
  branchNameUnavailable: string;
  addressPlaceholder: string;
  addressRequired: string;
  saveBranch: string;
  saving: string;
  cancel: string;
  saveChanges: string;
  savingChanges: string;
  detailsTitle: string;
  detailsDescription: string;
  roomManagement: string;
  addRoom: string;
  roomPlaceholder: string;
  roomCapacity: string;
  noRooms: string;
  existingRooms: string;
  pendingRooms: string;
  removeRoom: string;
  deleteRoomTitle: string;
  deleteRoomWarningDescription: string;
  currentBranchName: string;
  currentAddress: string;
  operationalHours: string;
  openTime: string;
  closeTime: string;
  branchAssets: string;
  editPhoto: string;
  removePhoto: string;
  replacePhoto: string;
  changesPending: string;
  duplicateError: string;
  duplicateRoomError: string;
  backendError: string;
  unauthorizedError: string;
  invalidPhotoError: string;
  photoTooLargeError: string;
  invalidRoomError: string;
  roomNameAvailable: string;
  roomNameUnavailable: string;
  staffManagement: string;
  existingStaff: string;
  addStaff: string;
  noStaff: string;
  finishNewStaffFirst: string;
  name: string;
  surname: string;
  birthDate: string;
  phone: string;
  salary: string;
  email: string;
  password: string;
  showPassword: string;
  hidePassword: string;
  receptionist: string;
  staffSaveError: string;
  staffDuplicateError: string;
  staffInvalidInputError: string;
};

type BranchManagementViewProps = {
  branches: Branch[];
  labels: BranchManagementLabels;
  rooms: Room[];
  staff: StaffMember[];
  stats: BranchStats[];
};

const initialCreateState: CreateBranchState = {};
const initialDeleteState: DeleteBranchState = {};

export function BranchManagementView({
  branches,
  labels,
  rooms,
  staff,
  stats,
}: BranchManagementViewProps) {
  const router = useRouter();
  const [selectedBranch, setSelectedBranch] = useState<Branch | null>(null);
  const [branchOverrides, setBranchOverrides] = useState<
    Record<string, Branch>
  >({});
  const [deletedBranchIDs, setDeletedBranchIDs] = useState<Set<string>>(
    () => new Set(),
  );
  const [createOpen, setCreateOpen] = useState(false);
  const [createBranchName, setCreateBranchName] = useState("");
  const createFileInputRef = useRef<HTMLInputElement | null>(null);
  const createPreviewURLRef = useRef<string | null>(null);
  const [createPhotoPreview, setCreatePhotoPreview] = useState("");
  const [createState, createFormAction] = useActionState(
    createBranchAction,
    initialCreateState,
  );
  const [createIdempotencyKey, setCreateIdempotencyKey] = useState(() =>
    crypto.randomUUID(),
  );
  const roomsByBranch = useMemo(() => groupRoomsByBranch(rooms), [rooms]);
  const staffByBranch = useMemo(() => groupStaffByBranch(staff), [staff]);
  const visibleBranches = useMemo(
    () =>
      branches
        .filter((branch) => !deletedBranchIDs.has(branch.id))
        .map((branch) => branchOverrides[branch.id] ?? branch),
    [branchOverrides, branches, deletedBranchIDs],
  );
  const createBranchNameStatus = getNameAvailability(
    createBranchName,
    visibleBranches,
  );
  const activeBranches = visibleBranches.length;
  const archivedBranches = 0;
  const totalRooms = rooms.length;
  const totalCapacity = rooms.reduce((sum, room) => sum + room.capacity, 0);
  const totalStudents = stats.reduce(
    (sum, item) => sum + item.active_students,
    0,
  );
  const capacityPercent =
    totalCapacity > 0 ? Math.round((totalStudents / totalCapacity) * 100) : 0;

  useEffect(() => {
    if (!createState.branch) {
      return;
    }

    window.queueMicrotask(() => {
      setCreateOpen(false);
      setCreateBranchName("");
      setCreateIdempotencyKey(crypto.randomUUID());
      clearCreatePhotoPreview();
      router.refresh();
    });
  }, [createState.branch, router]);

  useEffect(() => {
    if (
      createState.error !== "invalid_photo" &&
      createState.error !== "photo_too_large"
    ) {
      return;
    }

    window.queueMicrotask(() => {
      clearCreatePhotoPreview();
    });
  }, [createState.error]);

  useEffect(() => {
    return () => {
      if (createPreviewURLRef.current) {
        URL.revokeObjectURL(createPreviewURLRef.current);
      }
    };
  }, []);

  function clearCreatePhotoPreview() {
    if (createPreviewURLRef.current) {
      URL.revokeObjectURL(createPreviewURLRef.current);
      createPreviewURLRef.current = null;
    }
    if (createFileInputRef.current) {
      createFileInputRef.current.value = "";
    }
    setCreatePhotoPreview("");
  }

  function handleCreateOpenChange(open: boolean) {
    setCreateOpen(open);
    if (!open) {
      clearCreatePhotoPreview();
    }
  }

  function handleCreatePhotoChange(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) {
      clearCreatePhotoPreview();
      return;
    }
    if (createPreviewURLRef.current) {
      URL.revokeObjectURL(createPreviewURLRef.current);
    }
    const nextURL = URL.createObjectURL(file);
    createPreviewURLRef.current = nextURL;
    setCreatePhotoPreview(nextURL);
  }

  if (selectedBranch) {
    return (
      <BranchDetailPage
        branch={branchOverrides[selectedBranch.id] ?? selectedBranch}
        labels={labels}
        onCancel={() => setSelectedBranch(null)}
        onSaved={(branch) => {
          setBranchOverrides((current) => ({
            ...current,
            [branch.id]: branch,
          }));
          setSelectedBranch(null);
        }}
        rooms={roomsByBranch.get(selectedBranch.id) ?? []}
        branches={visibleBranches}
      />
    );
  }

  return (
    <div className="space-y-6">
      <section className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-3xl font-black tracking-tight">{labels.title}</h1>
          <p className="mt-1 text-sm text-kw-c-59667a dark:text-kw-c-a7b0bf">
            {labels.description}
          </p>
        </div>
        <Sheet open={createOpen} onOpenChange={handleCreateOpenChange}>
          <SheetTrigger asChild>
            <Button className="h-[38px] gap-2 rounded-lg bg-kw-c-ef2334 px-5 font-bold text-white shadow-kw-action transition hover:bg-kw-c-d91f30">
              <Plus className="size-4" />
              {labels.addNewBranch}
            </Button>
          </SheetTrigger>
          <SheetContent className="z-[1100] w-full overflow-y-auto p-0 sm:max-w-xl dark:border-kw-c-293445">
            <SheetHeader className="border-b border-kw-c-dce3ee p-6 dark:border-kw-c-293445">
              <SheetTitle className="text-2xl font-black">
                {labels.addNewBranch}
              </SheetTitle>
              <SheetDescription>{labels.addBranchDescription}</SheetDescription>
            </SheetHeader>
            <form action={createFormAction} className="space-y-5 p-6">
              <input
                name="idempotency_key"
                type="hidden"
                value={createIdempotencyKey}
              />
              <div className="space-y-3">
                <Label>{labels.branchPhoto}</Label>
                <div className="relative mx-auto size-36">
                  <label className="group relative flex size-36 cursor-pointer items-center justify-center overflow-hidden rounded-full border border-dashed border-kw-c-b9c5d6 bg-kw-c-f7f8fb text-kw-c-0a284b shadow-inner transition hover:border-kw-c-ef2334 hover:bg-kw-c-f1f4f8 dark:border-kw-c-3a4658 dark:bg-kw-c-202b3a dark:text-kw-c-f3f6fa dark:hover:border-kw-c-ff3b4f dark:hover:bg-kw-c-263448">
                    {createPhotoPreview ? (
                      <ObjectCoverImage src={createPhotoPreview} />
                    ) : (
                      <ImagePlus className="size-8 transition-transform group-hover:scale-110" />
                    )}
                    <input
                      accept="image/jpeg,image/png,image/webp,.jpg,.jpeg,.png,.webp"
                      className="sr-only"
                      name="photo"
                      onChange={handleCreatePhotoChange}
                      ref={createFileInputRef}
                      type="file"
                    />
                  </label>
                  {createPhotoPreview ? (
                    <button
                      aria-label={labels.removePhoto}
                      className="absolute right-0 top-0 grid size-8 cursor-pointer place-items-center rounded-full bg-kw-c-ef2334 text-white shadow-lg ring-4 ring-white transition hover:bg-kw-c-d91f30 dark:bg-kw-c-ff3b4f dark:ring-kw-c-1b2635 dark:hover:bg-kw-c-ff5a69"
                      type="button"
                      onClick={clearCreatePhotoPreview}
                    >
                      <X className="size-4" />
                    </button>
                  ) : null}
                </div>
                <CreateError
                  state={createState}
                  labels={labels}
                  field="photo"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="branch-name">{labels.branchName}</Label>
                <div className="relative">
                  <Input
                    className="pr-10"
                    id="branch-name"
                    name="name"
                    onChange={(event) =>
                      setCreateBranchName(event.target.value)
                    }
                    placeholder={labels.branchNamePlaceholder}
                    required
                    value={createBranchName}
                  />
                  <AvailabilityIcon
                    availableLabel={labels.branchNameAvailable}
                    status={createBranchNameStatus}
                    unavailableLabel={labels.branchNameUnavailable}
                  />
                </div>
                <CreateError state={createState} labels={labels} field="name" />
              </div>
              <div className="space-y-2">
                <Label htmlFor="branch-address">{labels.address}</Label>
                <Textarea
                  id="branch-address"
                  name="address"
                  placeholder={labels.addressPlaceholder}
                  required
                />
                <CreateError
                  state={createState}
                  labels={labels}
                  field="address"
                />
              </div>
              <div className="grid gap-4">
                <div className="space-y-2">
                  <Label htmlFor="create-opening-time">{labels.openTime}</Label>
                  <Input
                    defaultValue="09:00"
                    id="create-opening-time"
                    name="opening_time"
                    type="time"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="create-closing-time">{labels.closeTime}</Label>
                  <Input
                    defaultValue="21:00"
                    id="create-closing-time"
                    name="closing_time"
                    type="time"
                  />
                </div>
              </div>
              <CreateError state={createState} labels={labels} field="form" />
              <CreateBranchSubmit
                disabled={createBranchNameStatus === "taken"}
                label={labels.saveBranch}
                pending={labels.saving}
              />
            </form>
          </SheetContent>
        </Sheet>
      </section>

      <section className="grid gap-4 md:grid-cols-3">
        <MiniStat
          icon={Building2}
          label={labels.totalBranches}
          value={`${activeBranches} ${labels.active} / ${archivedBranches} ${labels.archived}`}
        />
        <MiniStat
          icon={DoorOpen}
          label={labels.totalClassrooms}
          value={String(totalRooms)}
        />
        <Card className="rounded-lg border-kw-c-dce3ee dark:border-kw-c-3a4658 dark:bg-kw-c-1b2635">
          <CardContent className="space-y-3 p-5">
            <div className="flex items-center justify-between">
              <div className="text-sm font-bold text-kw-c-59667a dark:text-kw-c-a7b0bf">
                {labels.capacityStatus}
              </div>
              <Users className="size-5 text-kw-c-ef2334 dark:text-kw-c-ff3b4f" />
            </div>
            <div className="text-2xl font-black tabular-nums">
              {capacityPercent}%
            </div>
            <Progress value={capacityPercent} />
          </CardContent>
        </Card>
      </section>

      <Card className="overflow-hidden rounded-xl border-white/55 bg-white/65 py-0 shadow-kw-panel-strong backdrop-blur-xl dark:border-white/10 dark:bg-kw-c-1b2635/62 kw-dark-shadow-panel-strong">
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow className="h-14 border-white/40 bg-kw-c-eef3f8 hover:bg-kw-c-eef3f8 dark:border-white/10 dark:bg-kw-c-202b3a dark:hover:bg-kw-c-202b3a">
                <TableHead className="px-5 py-4">
                  {labels.branchProfile}
                </TableHead>
                <TableHead>{labels.address}</TableHead>
                <TableHead>{labels.rooms}</TableHead>
                <TableHead>{labels.staff}</TableHead>
                <TableHead>{labels.status}</TableHead>
                <TableHead className="pr-5 text-right">
                  {labels.actions}
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {visibleBranches.map((branch) => {
                const branchRooms = roomsByBranch.get(branch.id) ?? [];
                const branchStaff = staffByBranch.get(branch.id) ?? [];

                return (
                  <TableRow
                    className="cursor-pointer border-white/35 hover:bg-white/35 dark:border-white/10 dark:hover:bg-kw-c-202b3a/50"
                    key={branch.id}
                    onClick={() => setSelectedBranch(branch)}
                  >
                    <TableCell className="px-5 py-5">
                      <div className="flex items-center gap-3">
                        <BranchProfileAvatar branch={branch} />
                        <div className="min-w-0">
                          <div className="truncate font-bold">
                            {branch.name}
                          </div>
                          <div className="mt-1 text-xs text-kw-c-687386 dark:text-kw-c-6f7a8a">
                            {labels.branchID}: {branch.id.slice(0, 8)}
                          </div>
                        </div>
                      </div>
                    </TableCell>
                    <TableCell className="max-w-[320px] py-5">
                      <div className="flex items-center gap-2 text-kw-c-59667a dark:text-kw-c-a7b0bf">
                        <MapPin className="size-4 shrink-0 text-kw-c-ef2334 dark:text-kw-c-ff3b4f" />
                        <span className="truncate">
                          {branch.address || labels.addressPlaceholder}
                        </span>
                      </div>
                    </TableCell>
                    <TableCell className="py-5 font-semibold">
                      {branchRooms.length} {labels.rooms}
                    </TableCell>
                    <TableCell className="py-5">
                      <StaffCell
                        label={labels.unassigned}
                        receptionistLabel={labels.receptionist}
                        staff={branchStaff}
                      />
                    </TableCell>
                    <TableCell className="py-5">
                      <Badge className="bg-emerald-600 text-white hover:bg-emerald-600">
                        {labels.active}
                      </Badge>
                    </TableCell>
                    <TableCell className="py-5 pr-5 text-right">
                      <BranchActions
                        branch={branch}
                        labels={labels}
                        onDeleted={(branchID) => {
                          setDeletedBranchIDs((current) => {
                            const next = new Set(current);
                            next.add(branchID);
                            return next;
                          });
                          setBranchOverrides((current) => {
                            const next = { ...current };
                            delete next[branchID];
                            return next;
                          });
                          router.refresh();
                        }}
                        onSelect={setSelectedBranch}
                      />
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  );
}

function BranchActions({
  branch,
  labels,
  onDeleted,
  onSelect,
}: {
  branch: Branch;
  labels: BranchManagementLabels;
  onDeleted: (branchID: string) => void;
  onSelect: (branch: Branch) => void;
}) {
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deleteStep, setDeleteStep] = useState<"name" | "warning">("name");
  const [confirmName, setConfirmName] = useState("");
  const [deleteState, deleteFormAction] = useActionState(
    deleteBranchAction,
    initialDeleteState,
  );
  const [deleteIdempotencyKey, setDeleteIdempotencyKey] = useState(() =>
    crypto.randomUUID(),
  );
  const deleteNameMatches = confirmName.trim() === branch.name;

  useEffect(() => {
    if (!deleteState.branch) {
      return;
    }

    window.queueMicrotask(() => {
      onDeleted(deleteState.branch?.id ?? branch.id);
      setDeleteIdempotencyKey(crypto.randomUUID());
      setDeleteOpen(false);
      setConfirmName("");
      setDeleteStep("name");
    });
  }, [branch.id, deleteState.branch, onDeleted]);

  function selectBranch(event: Event) {
    event.stopPropagation();
    onSelect(branch);
  }

  function openDeleteDialog(event: Event) {
    event.stopPropagation();
    setConfirmName("");
    setDeleteStep("name");
    setDeleteOpen(true);
  }

  function stopRowSelection(event: { stopPropagation: () => void }) {
    event.stopPropagation();
  }

  function handleDeleteOpenChange(open: boolean) {
    setDeleteOpen(open);
    if (!open) {
      setConfirmName("");
      setDeleteStep("name");
    }
  }

  function closeDeleteFlow() {
    handleDeleteOpenChange(false);
  }

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            className="cursor-pointer"
            size="icon"
            type="button"
            variant="ghost"
            onClick={(event) => event.stopPropagation()}
            onPointerDown={(event) => event.stopPropagation()}
          >
            <MoreHorizontal className="size-5" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent
          align="end"
          className="z-[1100] min-w-48"
          onClick={stopRowSelection}
          onPointerDown={stopRowSelection}
        >
          <DropdownMenuItem
            className="cursor-pointer gap-2"
            onSelect={selectBranch}
          >
            <Pencil className="size-4" />
            {labels.edit}
          </DropdownMenuItem>
          <DropdownMenuItem
            className="cursor-pointer gap-2"
            onSelect={selectBranch}
          >
            <DoorOpen className="size-4" />
            {labels.manageRooms}
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            className="cursor-pointer gap-2 text-kw-c-ef2334 hover:text-kw-c-ef2334 focus:bg-kw-c-fff1f2 focus:text-kw-c-ef2334 data-[highlighted]:text-kw-c-ef2334 [&_svg]:text-kw-c-ef2334 [&_svg]:stroke-kw-c-ef2334 dark:text-kw-c-ff3b4f dark:focus:bg-kw-c-3a1e2a dark:focus:text-kw-c-ff3b4f dark:data-[highlighted]:text-kw-c-ff3b4f dark:[&_svg]:text-kw-c-ff3b4f dark:[&_svg]:stroke-kw-c-ff3b4f"
            onSelect={(event) => event.stopPropagation()}
          >
            <Archive className="size-4" />
            {labels.archive}
          </DropdownMenuItem>
          <DropdownMenuItem
            className="cursor-pointer gap-2 text-kw-c-8f1020 hover:text-kw-c-8f1020 focus:bg-kw-c-fff1f2 focus:text-kw-c-8f1020 data-[highlighted]:text-kw-c-8f1020 [&_svg]:text-kw-c-8f1020 [&_svg]:stroke-kw-c-8f1020 dark:text-kw-c-ff6b7a dark:focus:bg-kw-c-3a1e2a dark:focus:text-kw-c-ff6b7a dark:data-[highlighted]:text-kw-c-ff6b7a dark:[&_svg]:text-kw-c-ff6b7a dark:[&_svg]:stroke-kw-c-ff6b7a"
            onSelect={openDeleteDialog}
          >
            <Trash2 className="size-4" />
            {labels.delete}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      <Dialog open={deleteOpen} onOpenChange={handleDeleteOpenChange}>
        <DialogContent
          className="z-[1200] border-kw-c-dce3ee dark:border-kw-c-3a4658 dark:bg-kw-c-1b2635 dark:text-kw-c-f3f6fa"
          onClick={(event) => event.stopPropagation()}
        >
          {deleteStep === "name" ? (
            <>
              <DialogHeader>
                <DialogTitle className="text-xl font-black">
                  {labels.deleteBranchTitle}
                </DialogTitle>
                <div className="rounded-lg border border-kw-c-f87171 bg-kw-c-fff1f2 px-4 py-3 text-sm font-black text-kw-c-991b1b dark:bg-kw-c-3b1218 dark:text-kw-c-ffe4e6">
                  {labels.deleteBranchIrreversibleWarning}
                </div>
                <DialogDescription>
                  {labels.deleteBranchDescription}
                </DialogDescription>
              </DialogHeader>
              <div className="space-y-2">
                <Label htmlFor={`delete-branch-${branch.id}`}>
                  {labels.deleteBranchNameLabel}
                </Label>
                <Input
                  id={`delete-branch-${branch.id}`}
                  placeholder={labels.deleteBranchNamePlaceholder.replace(
                    "{name}",
                    branch.name,
                  )}
                  value={confirmName}
                  onChange={(event) => setConfirmName(event.target.value)}
                />
                {confirmName && !deleteNameMatches ? (
                  <p className="text-sm font-semibold text-kw-c-ef2334 dark:text-kw-c-ff6b7a">
                    {labels.deleteBranchNameMismatch}
                  </p>
                ) : null}
              </div>
              <DialogFooter className="bg-transparent px-0 pb-0">
                <Button
                  type="button"
                  variant="outline"
                  onClick={closeDeleteFlow}
                >
                  {labels.cancel}
                </Button>
                <Button
                  className="bg-kw-c-8f1020 text-white hover:bg-kw-c-74101d"
                  disabled={!deleteNameMatches}
                  type="button"
                  onClick={() => setDeleteStep("warning")}
                >
                  {labels.delete}
                </Button>
              </DialogFooter>
            </>
          ) : (
            <>
              <DialogHeader>
                <DialogTitle className="text-xl font-black text-kw-c-8f1020 dark:text-kw-c-ff6b7a">
                  {labels.deleteBranchWarningTitle}
                </DialogTitle>
                <DialogDescription className="text-kw-c-59667a dark:text-kw-c-cbd5e1">
                  {labels.deleteBranchWarningDescription}
                </DialogDescription>
              </DialogHeader>
              <form action={deleteFormAction}>
                <input name="branch_id" type="hidden" value={branch.id} />
                <input
                  name="idempotency_key"
                  type="hidden"
                  value={deleteIdempotencyKey}
                />
                <DeleteBranchError state={deleteState} labels={labels} />
                <DialogFooter className="mt-4 bg-transparent px-0 pb-0">
                  <Button
                    type="button"
                    variant="outline"
                    onClick={closeDeleteFlow}
                  >
                    {labels.cancel}
                  </Button>
                  <DeleteBranchSubmit
                    label={labels.delete}
                    pending={labels.deleting}
                  />
                </DialogFooter>
              </form>
            </>
          )}
        </DialogContent>
      </Dialog>
    </>
  );
}

function CreateBranchSubmit({
  disabled,
  label,
  pending,
}: {
  disabled?: boolean;
  label: string;
  pending: string;
}) {
  const status = useFormStatus();
  const submitLock = useSubmitLock(status.pending);

  const blocked = status.pending || Boolean(disabled) || submitLock.locked;

  return (
    <Button
      className="w-full rounded-lg bg-kw-c-ef2334 font-bold text-white hover:bg-kw-c-d91f30"
      disabled={blocked}
      type="submit"
      onClick={submitLock.onClick}
    >
      {status.pending ? pending : label}
    </Button>
  );
}

function DeleteBranchSubmit({
  label,
  pending,
}: {
  label: string;
  pending: string;
}) {
  const status = useFormStatus();
  const submitLock = useSubmitLock(status.pending);

  const blocked = status.pending || submitLock.locked;

  return (
    <Button
      className="bg-kw-c-8f1020 text-white hover:bg-kw-c-74101d"
      disabled={blocked}
      type="submit"
      onClick={submitLock.onClick}
    >
      {status.pending ? pending : label}
    </Button>
  );
}

function CreateError({
  field,
  labels,
  state,
}: {
  field: "address" | "form" | "name" | "photo";
  labels: BranchManagementLabels;
  state: CreateBranchState;
}) {
  const message =
    field === "name" && state.fieldErrors?.name?.length
      ? labels.branchNameRequired
      : field === "address" && state.fieldErrors?.address?.length
        ? labels.addressRequired
        : field === "photo" && state.error === "invalid_photo"
          ? labels.invalidPhotoError
          : field === "photo" && state.error === "photo_too_large"
            ? labels.photoTooLargeError
            : field === "form" && state.error === "duplicate"
              ? labels.duplicateError
              : field === "form" && state.error === "unauthorized"
                ? labels.unauthorizedError
                : field === "form" && state.error === "backend"
                  ? labels.backendError
                  : null;

  if (!message) {
    return null;
  }

  return <p className="text-sm font-semibold text-kw-c-ef2334">{message}</p>;
}

function DeleteBranchError({
  labels,
  state,
}: {
  labels: BranchManagementLabels;
  state: DeleteBranchState;
}) {
  const message =
    state.error === "not_found"
      ? labels.deleteBranchNotFoundError
      : state.error === "unauthorized"
        ? labels.deleteBranchUnauthorizedError
        : state.error === "backend"
          ? labels.deleteBranchBackendError
          : null;

  if (!message) {
    return null;
  }

  return (
    <div className="mt-4 rounded-lg border border-kw-c-f87171 bg-kw-c-fff1f2 px-4 py-3 text-sm font-semibold text-kw-c-991b1b dark:bg-kw-c-3b1218 dark:text-kw-c-ffe4e6">
      {message}
    </div>
  );
}

function groupRoomsByBranch(rooms: Room[]) {
  const map = new Map<string, Room[]>();
  for (const room of rooms) {
    const current = map.get(room.branch_id) ?? [];
    current.push(room);
    map.set(room.branch_id, current);
  }

  return map;
}
function groupStaffByBranch(staff: StaffMember[]) {
  const map = new Map<string, StaffMember[]>();
  for (const member of staff) {
    const current = map.get(member.branch_id) ?? [];
    current.push(member);
    map.set(member.branch_id, current);
  }

  return map;
}
