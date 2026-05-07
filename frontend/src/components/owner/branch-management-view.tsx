"use client";

import { useRouter } from "next/navigation";
import {
  useActionState,
  type ChangeEvent,
  type ReactNode,
  useEffect,
  useMemo,
  useRef,
  useState,
  useTransition,
} from "react";
import { useFormStatus } from "react-dom";
import {
  Archive,
  ArrowLeft,
  Building2,
  Calendar,
  CheckCircle2,
  Clock,
  CircleDollarSign,
  DoorOpen,
  Eye,
  EyeOff,
  ImagePlus,
  KeyRound,
  Mail,
  MapPin,
  MoreHorizontal,
  Pencil,
  Phone,
  Plus,
  Save,
  Trash2,
  Upload,
  UserRound,
  Users,
  X,
  XCircle,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import {
  Avatar,
  AvatarFallback,
  AvatarGroup,
  AvatarImage,
} from "@/components/ui/avatar";
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
import type { Branch, Room } from "@/lib/api/types";
import {
  createBranchAction,
  createStaffAction,
  deleteBranchAction,
  deleteStaffAction,
  saveBranchManagementAction,
  updateStaffAction,
  type CreateBranchState,
  type DeleteBranchState,
  type SaveBranchManagementState,
  type StaffManagementState,
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

type RoomDraft = {
  id: string;
  name: string;
  capacity: number;
};

const initialCreateState: CreateBranchState = {};
const initialSaveState: SaveBranchManagementState = {};
const initialDeleteState: DeleteBranchState = {};
const initialStaffState: StaffManagementState = {};

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
          <p className="mt-1 text-sm text-[#59667a] dark:text-[#a7b0bf]">
            {labels.description}
          </p>
        </div>
        <Sheet open={createOpen} onOpenChange={handleCreateOpenChange}>
          <SheetTrigger asChild>
            <Button className="h-[38px] gap-2 rounded-lg bg-[#ef2334] px-5 font-bold text-white shadow-[0_12px_26px_rgba(239,35,52,0.28)] transition hover:bg-[#d91f30]">
              <Plus className="size-4" />
              {labels.addNewBranch}
            </Button>
          </SheetTrigger>
          <SheetContent className="z-[1100] w-full overflow-y-auto p-0 sm:max-w-xl dark:border-[#293445]">
            <SheetHeader className="border-b border-[#dce3ee] p-6 dark:border-[#293445]">
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
                  <label className="group flex size-36 cursor-pointer items-center justify-center overflow-hidden rounded-full border border-dashed border-[#b9c5d6] bg-[#f7f8fb] text-[#0a284b] shadow-inner transition hover:border-[#ef2334] hover:bg-[#f1f4f8] dark:border-[#3a4658] dark:bg-[#202b3a] dark:text-[#f3f6fa] dark:hover:border-[#ff3b4f] dark:hover:bg-[#263448]">
                    {createPhotoPreview ? (
                      // eslint-disable-next-line @next/next/no-img-element
                      <img
                        alt=""
                        className="size-full object-cover"
                        src={createPhotoPreview}
                      />
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
                      className="absolute right-0 top-0 grid size-8 cursor-pointer place-items-center rounded-full bg-[#ef2334] text-white shadow-lg ring-4 ring-white transition hover:bg-[#d91f30] dark:bg-[#ff3b4f] dark:ring-[#1b2635] dark:hover:bg-[#ff5a69]"
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
        <Card className="rounded-lg border-[#dce3ee] dark:border-[#3a4658] dark:bg-[#1b2635]">
          <CardContent className="space-y-3 p-5">
            <div className="flex items-center justify-between">
              <div className="text-sm font-bold text-[#59667a] dark:text-[#a7b0bf]">
                {labels.capacityStatus}
              </div>
              <Users className="size-5 text-[#ef2334] dark:text-[#ff3b4f]" />
            </div>
            <div className="text-2xl font-black tabular-nums">
              {capacityPercent}%
            </div>
            <Progress value={capacityPercent} />
          </CardContent>
        </Card>
      </section>

      <Card className="overflow-hidden rounded-xl border-white/55 bg-white/65 py-0 shadow-[0_24px_70px_rgba(10,40,75,0.14)] backdrop-blur-xl dark:border-white/10 dark:bg-[#1b2635]/62 dark:shadow-[0_24px_80px_rgba(0,0,0,0.34)]">
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow className="h-14 border-white/40 bg-[#eef3f8] hover:bg-[#eef3f8] dark:border-white/10 dark:bg-[#202b3a] dark:hover:bg-[#202b3a]">
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
                    className="cursor-pointer border-white/35 hover:bg-white/35 dark:border-white/10 dark:hover:bg-[#202b3a]/50"
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
                          <div className="mt-1 text-xs text-[#687386] dark:text-[#6f7a8a]">
                            {labels.branchID}: {branch.id.slice(0, 8)}
                          </div>
                        </div>
                      </div>
                    </TableCell>
                    <TableCell className="max-w-[320px] py-5">
                      <div className="flex items-center gap-2 text-[#59667a] dark:text-[#a7b0bf]">
                        <MapPin className="size-4 shrink-0 text-[#ef2334] dark:text-[#ff3b4f]" />
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

function BranchDetailPage({
  branch,
  branches,
  labels,
  onCancel,
  onSaved,
  rooms,
}: {
  branch: Branch;
  branches: Branch[];
  labels: BranchManagementLabels;
  onCancel: () => void;
  onSaved: (branch: Branch) => void;
  rooms: Room[];
}) {
  const router = useRouter();
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const previewURLRef = useRef<string | null>(null);
  const [saveState, saveFormAction] = useActionState(
    saveBranchManagementAction,
    initialSaveState,
  );
  const saveIdempotencyKey = useMemo(() => crypto.randomUUID(), []);
  const [branchNameValue, setBranchNameValue] = useState(branch.name);
  const [photoPreview, setPhotoPreview] = useState("");
  const [photoRemoved, setPhotoRemoved] = useState(false);
  const [addRoomOpen, setAddRoomOpen] = useState(false);
  const [roomName, setRoomName] = useState("");
  const [roomCapacity, setRoomCapacity] = useState("1");
  const [editingRoomID, setEditingRoomID] = useState<string | null>(null);
  const [pendingRooms, setPendingRooms] = useState<RoomDraft[]>([]);
  const [updatedRooms, setUpdatedRooms] = useState<RoomDraft[]>([]);
  const [removedRoomIDs, setRemovedRoomIDs] = useState<string[]>([]);
  const visibleRooms = rooms.filter(
    (room) => !removedRoomIDs.includes(room.id),
  ).map(
    (room) => {
      const updated = updatedRooms.find((item) => item.id === room.id);
      return updated
        ? { ...room, capacity: updated.capacity, name: updated.name }
        : room;
    },
  );
  const [roomDeleteTarget, setRoomDeleteTarget] = useState<Room | null>(null);
  const selectedPhoto = photoRemoved
    ? ""
    : photoPreview || branch.photo_url || "";
  const branchNameStatus = getNameAvailability(
    branchNameValue,
    branches,
    branch.id,
  );
  const roomNameStatus = getRoomNameAvailability(
    roomName,
    visibleRooms,
    pendingRooms,
    editingRoomID,
  );

  useEffect(() => {
    const savedBranch = saveState.branch;
    if (!savedBranch) {
      return;
    }

    window.queueMicrotask(() => {
      onSaved(savedBranch);
      router.refresh();
    });
  }, [onSaved, router, saveState.branch]);

  useEffect(() => {
    if (
      saveState.error !== "invalid_photo" &&
      saveState.error !== "photo_too_large"
    ) {
      return;
    }
    window.queueMicrotask(() => {
      if (previewURLRef.current) {
        URL.revokeObjectURL(previewURLRef.current);
        previewURLRef.current = null;
      }
      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }
      setPhotoPreview("");
    });
  }, [saveState.error]);

  useEffect(() => {
    return () => {
      if (previewURLRef.current) {
        URL.revokeObjectURL(previewURLRef.current);
      }
    };
  }, []);

  function handlePhotoChange(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) {
      return;
    }
    if (previewURLRef.current) {
      URL.revokeObjectURL(previewURLRef.current);
    }
    const nextURL = URL.createObjectURL(file);
    previewURLRef.current = nextURL;
    setPhotoPreview(nextURL);
    setPhotoRemoved(false);
  }

  function removePhoto() {
    if (previewURLRef.current) {
      URL.revokeObjectURL(previewURLRef.current);
      previewURLRef.current = null;
    }
    if (fileInputRef.current) {
      fileInputRef.current.value = "";
    }
    setPhotoPreview("");
    setPhotoRemoved(true);
  }

  function addRoom() {
    const name = roomName.trim();
    if (!name) {
      return;
    }
    if (roomNameStatus === "taken") {
      return;
    }
    const capacity = Math.max(1, Number.parseInt(roomCapacity, 10) || 1);
    if (
      editingRoomID &&
      pendingRooms.some((room) => room.id === editingRoomID)
    ) {
      setPendingRooms((current) =>
        current.map((room) =>
          room.id === editingRoomID ? { ...room, capacity, name } : room,
        ),
      );
    } else if (
      editingRoomID &&
      rooms.some((room) => room.id === editingRoomID)
    ) {
      setUpdatedRooms((current) => {
        const nextRoom = { id: editingRoomID, capacity, name };
        return current.some((room) => room.id === editingRoomID)
          ? current.map((room) => (room.id === editingRoomID ? nextRoom : room))
          : [...current, nextRoom];
      });
    } else {
      setPendingRooms((current) => [
        ...current,
        {
          id: crypto.randomUUID(),
          name,
          capacity,
        },
      ]);
    }
    setRoomName("");
    setRoomCapacity("1");
    setEditingRoomID(null);
    setAddRoomOpen(false);
  }

  return (
    <form action={saveFormAction} className="space-y-6">
      <input name="branch_id" type="hidden" value={branch.id} />
      <input
        name="idempotency_key"
        type="hidden"
        value={saveIdempotencyKey}
      />
      <input
        name="photo_file_id"
        type="hidden"
        value={branch.photo_file_id ?? ""}
      />
      <input
        name="remove_photo"
        type="hidden"
        value={photoRemoved ? "1" : "0"}
      />
      <input
        name="new_rooms"
        type="hidden"
        value={JSON.stringify(
          pendingRooms.map((room) => ({
            capacity: room.capacity,
            name: room.name,
          })),
        )}
      />
      <input
        name="removed_room_ids"
        type="hidden"
        value={JSON.stringify(removedRoomIDs)}
      />
      <input
        name="updated_rooms"
        type="hidden"
        value={JSON.stringify(updatedRooms)}
      />

      <section className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-3xl font-black tracking-tight">
            {labels.detailsTitle}
          </h1>
          <p className="mt-1 text-sm text-[#59667a] dark:text-[#a7b0bf]">
            {labels.detailsDescription}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            className="gap-2"
            onClick={onCancel}
            type="button"
            variant="outline"
          >
            <ArrowLeft className="size-4" />
            {labels.cancel}
          </Button>
          <SaveBranchSubmit
            disabled={branchNameStatus === "taken"}
            label={labels.saveChanges}
            pending={labels.savingChanges}
          />
        </div>
      </section>

      <SaveError labels={labels} state={saveState} />

      <Card className="rounded-xl border-[#dce3ee] bg-white/80 shadow-[0_24px_70px_rgba(10,40,75,0.12)] backdrop-blur-xl dark:border-[#3a4658] dark:bg-[#1b2635]">
        <CardContent className="space-y-8 p-6">
          <section className="flex flex-col items-center gap-4">
            <div className="relative size-36">
              <label className="group flex size-36 cursor-pointer items-center justify-center overflow-hidden rounded-full border border-dashed border-[#b9c5d6] bg-[#f7f8fb] text-[#0a284b] shadow-inner transition hover:border-[#ef2334] hover:bg-[#f1f4f8] dark:border-[#3a4658] dark:bg-[#202b3a] dark:text-[#f3f6fa] dark:hover:border-[#ff3b4f] dark:hover:bg-[#263448]">
                {selectedPhoto ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    alt={branch.name}
                    className="size-full object-cover"
                    src={selectedPhoto}
                  />
                ) : (
                  <ImagePlus className="size-8 transition-transform group-hover:scale-110" />
                )}
                <input
                  accept="image/jpeg,image/png,image/webp,.jpg,.jpeg,.png,.webp"
                  className="sr-only"
                  name="photo"
                  onChange={handlePhotoChange}
                  ref={fileInputRef}
                  type="file"
                />
              </label>
              {selectedPhoto ? (
                <button
                  aria-label={labels.removePhoto}
                  className="absolute right-0 top-0 grid size-8 cursor-pointer place-items-center rounded-full bg-[#ef2334] text-white shadow-lg ring-4 ring-white transition hover:bg-[#d91f30] dark:bg-[#ff3b4f] dark:ring-[#1b2635] dark:hover:bg-[#ff5a69]"
                  type="button"
                  onClick={removePhoto}
                >
                  <X className="size-4" />
                </button>
              ) : null}
            </div>
          </section>

          <section className="grid gap-5 md:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="edit-branch-name">
                {labels.currentBranchName}
              </Label>
              <div className="relative">
                <Input
                  className="pr-10"
                  id="edit-branch-name"
                  name="name"
                  onChange={(event) => setBranchNameValue(event.target.value)}
                  required
                  value={branchNameValue}
                />
                <AvailabilityIcon
                  availableLabel={labels.branchNameAvailable}
                  status={branchNameStatus}
                  unavailableLabel={labels.branchNameUnavailable}
                />
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="edit-branch-address">
                {labels.currentAddress}
              </Label>
              <Textarea
                className="min-h-9"
                defaultValue={branch.address ?? ""}
                id="edit-branch-address"
                name="address"
                required
              />
            </div>
          </section>

          <section className="rounded-lg border border-[#dce3ee] p-4 dark:border-[#3a4658]">
            <div className="mb-4 flex items-center gap-2 font-black">
              <Clock className="size-5 text-[#ef2334] dark:text-[#ff3b4f]" />
              {labels.operationalHours}
            </div>
            <div className="grid gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="opening-time">{labels.openTime}</Label>
                <Input
                  defaultValue={branch.opening_time || "09:00"}
                  id="opening-time"
                  name="opening_time"
                  type="time"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="closing-time">{labels.closeTime}</Label>
                <Input
                  defaultValue={branch.closing_time || "21:00"}
                  id="closing-time"
                  name="closing_time"
                  type="time"
                />
              </div>
            </div>
          </section>
        </CardContent>
      </Card>

      <Card className="rounded-xl border-[#dce3ee] bg-white/80 shadow-[0_24px_70px_rgba(10,40,75,0.12)] backdrop-blur-xl dark:border-[#3a4658] dark:bg-[#1b2635]">
        <CardContent className="space-y-5 p-6">
          <div className="flex items-center gap-2 font-black">
            <DoorOpen className="size-5 text-[#ef2334] dark:text-[#ff3b4f]" />
            {labels.roomManagement}
          </div>

          <RoomList
            editLabel={labels.edit}
            emptyLabel={labels.noRooms}
            label={labels.existingRooms}
            onEdit={(room) => {
              setEditingRoomID(room.id);
              setRoomName(room.name);
              setRoomCapacity(String(room.capacity));
              setAddRoomOpen(true);
            }}
            onRemove={setRoomDeleteTarget}
            removeLabel={labels.removeRoom}
            rooms={visibleRooms}
          />
          <div className="flex flex-col items-center gap-4">
            {addRoomOpen ? (
              <div
                className="w-full max-w-2xl rounded-xl border border-[#dce3ee] bg-[#f8fafc] p-4 shadow-sm dark:border-[#3a4658] dark:bg-[#202b3a]"
                data-testid="room-editor"
              >
                <div className="grid gap-3 md:grid-cols-[1fr_140px]">
                  <div className="relative">
                    <Input
                      autoFocus
                      aria-invalid={roomNameStatus === "taken"}
                      className="pr-10"
                      onChange={(event) => setRoomName(event.target.value)}
                      onKeyDown={(event) => {
                        if (event.key === "Enter") {
                          event.preventDefault();
                          addRoom();
                        }
                      }}
                      placeholder={labels.roomPlaceholder}
                      value={roomName}
                    />
                    <AvailabilityIcon
                      availableLabel={labels.roomNameAvailable}
                      status={roomNameStatus}
                      unavailableLabel={labels.roomNameUnavailable}
                    />
                  </div>
                  <Input
                    aria-label={labels.roomCapacity}
                    min={1}
                    onChange={(event) => setRoomCapacity(event.target.value)}
                    placeholder={labels.roomCapacity}
                    type="number"
                    value={roomCapacity}
                  />
                </div>
                <div className="mt-3 flex justify-center gap-2">
                  <Button
                    className="min-w-28 bg-emerald-600 font-bold text-white hover:bg-emerald-700"
                    disabled={roomNameStatus !== "available"}
                    onClick={addRoom}
                    type="button"
                  >
                    {labels.saveChanges}
                  </Button>
                  <Button
                    className="min-w-28"
                    onClick={() => {
                      setRoomName("");
                      setRoomCapacity("1");
                      setEditingRoomID(null);
                      setAddRoomOpen(false);
                    }}
                    type="button"
                    variant="outline"
                  >
                    {labels.cancel}
                  </Button>
                </div>
              </div>
            ) : null}
            {!addRoomOpen ? (
              <Button
                className="gap-2 rounded-full px-5"
                onClick={() => {
                  setEditingRoomID(null);
                  setRoomName("");
                  setRoomCapacity("1");
                  setAddRoomOpen(true);
                }}
                type="button"
                variant="outline"
              >
                <Plus className="size-4" />
                {labels.addRoom}
              </Button>
            ) : null}
          </div>
          <DraftRoomList
            editLabel={labels.edit}
            label={labels.pendingRooms}
            onEdit={(room) => {
              setEditingRoomID(room.id);
              setRoomName(room.name);
              setRoomCapacity(String(room.capacity));
              setAddRoomOpen(true);
            }}
            onRemove={(id) =>
              setPendingRooms((current) =>
                current.filter((room) => room.id !== id),
              )
            }
            removeLabel={labels.removeRoom}
            rooms={pendingRooms}
          />
        </CardContent>
      </Card>
      <Dialog
        open={Boolean(roomDeleteTarget)}
        onOpenChange={(open) => {
          if (!open) {
            setRoomDeleteTarget(null);
          }
        }}
      >
        <DialogContent className="z-[1200] border-[#dce3ee] dark:border-[#3a4658] dark:bg-[#1b2635] dark:text-[#f3f6fa]">
          <DialogHeader>
            <DialogTitle className="text-xl font-black">
              {labels.deleteRoomTitle}
            </DialogTitle>
            <DialogDescription className="text-[#59667a] dark:text-[#cbd5e1]">
              {labels.deleteRoomWarningDescription}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="bg-transparent px-0 pb-0">
            <Button
              onClick={() => setRoomDeleteTarget(null)}
              type="button"
              variant="outline"
            >
              {labels.cancel}
            </Button>
            <Button
              className="bg-[#8f1020] text-white hover:bg-[#74101d]"
              onClick={() => {
                if (roomDeleteTarget) {
                  setRemovedRoomIDs((current) =>
                    current.includes(roomDeleteTarget.id)
                      ? current
                      : [...current, roomDeleteTarget.id],
                  );
                  setUpdatedRooms((current) =>
                    current.filter((room) => room.id !== roomDeleteTarget.id),
                  );
                }
                setRoomDeleteTarget(null);
              }}
              type="button"
            >
              {labels.delete}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </form>
  );
}

type StaffFormMode = "create" | "edit" | null;

type StaffFormValues = {
  birthDate: string;
  email: string;
  firstName: string;
  lastName: string;
  password: string;
  phone: string;
  photoFileID: string;
  photoRemoved: boolean;
  photoURL: string;
  salary: string;
  staffID: string;
};

const emptyStaffForm: StaffFormValues = {
  birthDate: "",
  email: "",
  firstName: "",
  lastName: "",
  password: "",
  phone: "",
  photoFileID: "",
  photoRemoved: false,
  photoURL: "",
  salary: "",
  staffID: "",
};

// Staff UI moved to Receptionist Management; this legacy component will be removed in a cleanup pass.
// eslint-disable-next-line @typescript-eslint/no-unused-vars
function StaffManagementSection({
  branchID,
  initialStaff,
  labels,
}: {
  branchID: string;
  initialStaff: StaffMember[];
  labels: BranchManagementLabels;
}) {
  const router = useRouter();
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const previewURLRef = useRef<string | null>(null);
  const [isPending, startTransition] = useTransition();
  const [mode, setMode] = useState<StaffFormMode>(null);
  const [formValues, setFormValues] = useState<StaffFormValues>(emptyStaffForm);
  const [staffMembers, setStaffMembers] = useState(initialStaff);
  const [staffState, setStaffState] =
    useState<StaffManagementState>(initialStaffState);
  const [warning, setWarning] = useState("");
  const [passwordVisible, setPasswordVisible] = useState(false);
  const selectedPhoto =
    formValues.photoRemoved || !formValues.photoURL ? "" : formValues.photoURL;

  useEffect(() => {
    window.queueMicrotask(() => {
      setStaffMembers(initialStaff);
    });
  }, [initialStaff]);

  useEffect(() => {
    return () => {
      if (previewURLRef.current) {
        URL.revokeObjectURL(previewURLRef.current);
      }
    };
  }, []);

  function resetStaffForm() {
    if (previewURLRef.current) {
      URL.revokeObjectURL(previewURLRef.current);
      previewURLRef.current = null;
    }
    if (fileInputRef.current) {
      fileInputRef.current.value = "";
    }
    setFormValues(emptyStaffForm);
    setMode(null);
    setWarning("");
    setPasswordVisible(false);
  }

  function openCreate() {
    setStaffState(initialStaffState);
    setWarning("");
    setFormValues(emptyStaffForm);
    setPasswordVisible(false);
    setMode("create");
  }

  function openEdit(member: StaffMember) {
    if (mode === "create") {
      setWarning(labels.finishNewStaffFirst);
      return;
    }
    setStaffState(initialStaffState);
    setWarning("");
    setPasswordVisible(false);
    setFormValues({
      birthDate: member.birth_date ?? "",
      email: member.email,
      firstName: member.first_name,
      lastName: member.last_name,
      password: "",
      phone: member.phone ?? "",
      photoFileID: member.profile_photo_file_id ?? "",
      photoRemoved: false,
      photoURL: member.profile_photo_url ?? "",
      salary: String(member.salary_amount_azn || ""),
      staffID: member.id,
    });
    setMode("edit");
  }

  function handleStaffPhotoChange(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) {
      return;
    }
    if (previewURLRef.current) {
      URL.revokeObjectURL(previewURLRef.current);
    }
    const nextURL = URL.createObjectURL(file);
    previewURLRef.current = nextURL;
    setFormValues((current) => ({
      ...current,
      photoRemoved: false,
      photoURL: nextURL,
    }));
  }

  function removeStaffPhoto() {
    if (previewURLRef.current) {
      URL.revokeObjectURL(previewURLRef.current);
      previewURLRef.current = null;
    }
    if (fileInputRef.current) {
      fileInputRef.current.value = "";
    }
    setFormValues((current) => ({
      ...current,
      photoRemoved: true,
      photoURL: "",
    }));
  }

  function saveStaff() {
    const formData = new FormData();
    formData.set("branch_id", branchID);
    formData.set("staff_id", formValues.staffID);
    formData.set("first_name", formValues.firstName);
    formData.set("last_name", formValues.lastName);
    formData.set("birth_date", formValues.birthDate);
    formData.set("phone", formValues.phone);
    formData.set("salary", formValues.salary);
    formData.set("email", formValues.email);
    formData.set("password", formValues.password);
    formData.set("profile_photo_file_id", formValues.photoFileID);
    formData.set("remove_photo", formValues.photoRemoved ? "1" : "0");
    const photo = fileInputRef.current?.files?.[0];
    if (photo) {
      formData.set("photo", photo);
    }

    startTransition(async () => {
      const result =
        mode === "edit"
          ? await updateStaffAction(initialStaffState, formData)
          : await createStaffAction(initialStaffState, formData);
      setStaffState(result);
      if (!result.staff) {
        return;
      }
      const savedStaff = {
        ...result.staff,
        profile_photo_url:
          result.staff.profile_photo_url ??
          (!formValues.photoRemoved ? formValues.photoURL : undefined),
      };
      setStaffMembers((current) => {
        const exists = current.some((member) => member.id === savedStaff.id);
        return exists
          ? current.map((member) =>
              member.id === savedStaff.id ? savedStaff : member,
            )
          : [...current, savedStaff];
      });
      resetStaffForm();
      router.refresh();
    });
  }

  function deleteStaffMember(staffID: string) {
    const formData = new FormData();
    formData.set("staff_id", staffID);
    startTransition(async () => {
      const result = await deleteStaffAction(initialStaffState, formData);
      setStaffState(result);
      if (result.deletedStaffID) {
        setStaffMembers((current) =>
          current.filter((member) => member.id !== result.deletedStaffID),
        );
        router.refresh();
      }
    });
  }

  return (
    <Card className="rounded-xl border-[#dce3ee] bg-white/80 shadow-[0_24px_70px_rgba(10,40,75,0.12)] backdrop-blur-xl dark:border-[#3a4658] dark:bg-[#1b2635]">
      <CardContent className="space-y-5 p-6">
        <div className="flex items-center gap-2 font-black">
          <UserRound className="size-5 text-[#ef2334] dark:text-[#ff3b4f]" />
          {labels.staffManagement}
        </div>
        <StaffList
          labels={labels}
          onDelete={deleteStaffMember}
          onEdit={openEdit}
          staff={staffMembers}
        />
        {warning ? (
          <div className="rounded-lg border border-[#f87171] bg-[#fff1f2] px-4 py-3 text-sm font-semibold text-[#991b1b] dark:bg-[#3b1218] dark:text-[#ffe4e6]">
            {warning}
          </div>
        ) : null}
        <StaffError labels={labels} state={staffState} />
        {mode ? (
          <div className="rounded-xl border border-[#dce3ee] bg-[#f8fafc] p-5 dark:border-[#3a4658] dark:bg-[#202b3a]">
            <div className="flex flex-col items-center gap-3">
              <div className="grid size-28 place-items-center overflow-hidden rounded-full border border-dashed border-[#b9c5d6] bg-[#eef2f7] dark:border-[#414d60] dark:bg-[#2a3444]">
                {selectedPhoto ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    alt=""
                    className="size-full object-cover"
                    src={selectedPhoto}
                  />
                ) : (
                  <UserRound className="size-9 text-[#0a284b] dark:text-[#f3f6fa]" />
                )}
              </div>
              <div className="flex flex-wrap justify-center gap-2">
                <Button
                  className="gap-2"
                  onClick={() => fileInputRef.current?.click()}
                  type="button"
                  variant="outline"
                >
                  <Upload className="size-4" />
                  {labels.editPhoto}
                </Button>
                <Button
                  className="gap-2 text-[#ef2334] hover:text-[#ef2334] dark:text-[#ff3b4f] dark:hover:text-[#ff3b4f]"
                  onClick={removeStaffPhoto}
                  type="button"
                  variant="outline"
                >
                  <Trash2 className="size-4" />
                  {labels.removePhoto}
                </Button>
              </div>
              <input
                accept="image/jpeg,image/png,image/webp,.jpg,.jpeg,.png,.webp"
                className="sr-only"
                onChange={handleStaffPhotoChange}
                ref={fileInputRef}
                type="file"
              />
            </div>
            <div className="mt-5 grid gap-4 md:grid-cols-2">
              <StaffInput
                icon={UserRound}
                label={labels.name}
                onChange={(value) =>
                  setFormValues((current) => ({ ...current, firstName: value }))
                }
                value={formValues.firstName}
              />
              <StaffInput
                icon={UserRound}
                label={labels.surname}
                onChange={(value) =>
                  setFormValues((current) => ({ ...current, lastName: value }))
                }
                value={formValues.lastName}
              />
              <StaffInput
                icon={Calendar}
                label={labels.birthDate}
                onChange={(value) =>
                  setFormValues((current) => ({
                    ...current,
                    birthDate: formatBirthDateInput(value),
                  }))
                }
                placeholder="DD/MM/YYYY"
                value={formValues.birthDate}
              />
              <StaffInput
                icon={Phone}
                label={labels.phone}
                onChange={(value) =>
                  setFormValues((current) => ({ ...current, phone: value }))
                }
                value={formValues.phone}
              />
              <StaffInput
                icon={CircleDollarSign}
                label={labels.salary}
                onChange={(value) =>
                  setFormValues((current) => ({
                    ...current,
                    salary: value.replace(/\D/g, ""),
                  }))
                }
                prefix="₼"
                value={formValues.salary}
              />
              <StaffInput
                icon={Mail}
                label={labels.email}
                onChange={(value) =>
                  setFormValues((current) => ({ ...current, email: value }))
                }
                value={formValues.email}
              />
              <StaffInput
                icon={KeyRound}
                label={labels.password}
                onChange={(value) =>
                  setFormValues((current) => ({ ...current, password: value }))
                }
                trailing={
                  <button
                    aria-label={
                      passwordVisible ? labels.hidePassword : labels.showPassword
                    }
                    className="absolute right-3 top-1/2 -translate-y-1/2 cursor-pointer text-[#687386] transition hover:text-[#0a284b] dark:text-[#a7b0bf] dark:hover:text-white"
                    onClick={() => setPasswordVisible((current) => !current)}
                    type="button"
                  >
                    {passwordVisible ? (
                      <EyeOff className="size-4" />
                    ) : (
                      <Eye className="size-4" />
                    )}
                  </button>
                }
                type={passwordVisible ? "text" : "password"}
                value={formValues.password}
              />
            </div>
            <div className="mt-5 flex justify-center gap-2">
              <Button
                className="min-w-28 bg-emerald-600 font-bold text-white hover:bg-emerald-700"
                disabled={isPending}
                onClick={saveStaff}
                type="button"
              >
                {isPending ? labels.savingChanges : labels.saveChanges}
              </Button>
              <Button
                className="min-w-28"
                onClick={resetStaffForm}
                type="button"
                variant="outline"
              >
                {labels.cancel}
              </Button>
            </div>
          </div>
        ) : null}
        <div className="flex justify-center">
          <Button
            className="gap-2 rounded-full px-5"
            disabled={Boolean(mode) || isPending}
            onClick={openCreate}
            type="button"
            variant="outline"
          >
            <Plus className="size-4" />
            {labels.addStaff}
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}

function StaffList({
  labels,
  onDelete,
  onEdit,
  staff,
}: {
  labels: BranchManagementLabels;
  onDelete: (staffID: string) => void;
  onEdit: (member: StaffMember) => void;
  staff: StaffMember[];
}) {
  return (
    <section className="space-y-3">
      <div className="text-sm font-black text-[#0a284b] dark:text-[#f3f6fa]">
        {labels.existingStaff}
      </div>
      {staff.length === 0 ? (
        <div className="rounded-lg border border-dashed border-[#cbd5e1] px-4 py-5 text-center text-sm font-semibold text-[#687386] dark:border-[#3a4658] dark:text-[#a7b0bf]">
          {labels.noStaff}
        </div>
      ) : (
        <div className="space-y-2">
          {staff.map((member) => (
            <div
              className="flex flex-wrap items-center gap-3 rounded-xl border border-[#dce3ee] bg-white px-4 py-3 shadow-sm dark:border-[#3a4658] dark:bg-[#202b3a]"
              key={member.id}
            >
              <Avatar className="size-12 border border-white/60 shadow-sm dark:border-[#414d60]">
                {member.profile_photo_url ? (
                  <AvatarImage
                    alt={`${member.first_name} ${member.last_name}`}
                    src={member.profile_photo_url}
                  />
                ) : null}
                <AvatarFallback className="bg-[#e8edf5] text-sm font-black text-[#0a284b] dark:bg-[#2a3444] dark:text-[#f3f6fa]">
                  {staffInitials(member)}
                </AvatarFallback>
              </Avatar>
              <div className="min-w-0 flex-1">
                <div className="truncate text-sm font-black text-[#0a284b] dark:text-[#f3f6fa]">
                  {member.first_name} {member.last_name}
                </div>
                <div className="text-xs font-semibold text-[#687386] dark:text-[#a7b0bf]">
                  {labels.receptionist}
                </div>
              </div>
              <div className="rounded-full border border-[#dce3ee] px-3 py-1 text-sm font-black text-[#0a284b] dark:border-[#3a4658] dark:text-[#f3f6fa]">
                ₼ {member.salary_amount_azn}
              </div>
              <div className="ml-auto flex items-center gap-2">
                <Button
                  aria-label={labels.edit}
                  className="size-9 rounded-full p-0"
                  onClick={() => onEdit(member)}
                  type="button"
                  variant="outline"
                >
                  <Pencil className="size-4" />
                </Button>
                <Button
                  aria-label={labels.delete}
                  className="size-9 rounded-full p-0 text-[#ef2334] hover:text-[#ef2334] dark:text-[#ff3b4f] dark:hover:text-[#ff3b4f]"
                  onClick={() => onDelete(member.id)}
                  type="button"
                  variant="outline"
                >
                  <Trash2 className="size-4" />
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}
    </section>
  );
}

function StaffInput({
  icon: Icon,
  label,
  onChange,
  placeholder,
  prefix,
  trailing,
  type = "text",
  value,
}: {
  icon: LucideIcon;
  label: string;
  onChange: (value: string) => void;
  placeholder?: string;
  prefix?: string;
  trailing?: ReactNode;
  type?: string;
  value: string;
}) {
  const salaryInput = Boolean(prefix);

  return (
    <div className="space-y-2">
      <Label>{label}</Label>
      <div className="relative">
        <Icon className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-[#687386] dark:text-[#a7b0bf]" />
        {prefix ? (
          <span className="pointer-events-none absolute left-9 top-1/2 -translate-y-1/2 text-sm font-black text-[#687386] dark:text-[#a7b0bf]">
            {prefix}
          </span>
        ) : null}
        <Input
          className={`${prefix ? "pl-20" : "pl-10"} ${trailing ? "pr-10" : ""}`}
          inputMode={salaryInput ? "numeric" : undefined}
          onChange={(event) => onChange(event.target.value)}
          placeholder={placeholder}
          type={type}
          value={value}
        />
        {trailing}
      </div>
    </div>
  );
}

function StaffError({
  labels,
  state,
}: {
  labels: BranchManagementLabels;
  state: StaffManagementState;
}) {
  const message =
    state.error === "duplicate"
      ? labels.staffDuplicateError
      : state.error === "invalid_input" ||
          state.error === "invalid_photo" ||
          state.error === "photo_too_large"
        ? labels.staffInvalidInputError
        : state.error === "unauthorized"
          ? labels.unauthorizedError
          : state.error
            ? labels.staffSaveError
            : "";

  if (!message) {
    return null;
  }

  return (
    <div className="rounded-lg border border-[#f87171] bg-[#fff1f2] px-4 py-3 text-sm font-semibold text-[#991b1b] dark:bg-[#3b1218] dark:text-[#ffe4e6]">
      {message}
    </div>
  );
}

function MiniStat({
  icon: Icon,
  label,
  value,
}: {
  icon: LucideIcon;
  label: string;
  value: string;
}) {
  return (
    <Card className="rounded-lg border-[#dce3ee] dark:border-[#3a4658] dark:bg-[#1b2635]">
      <CardContent className="p-5">
        <div className="flex items-center justify-between">
          <div className="text-sm font-bold text-[#59667a] dark:text-[#a7b0bf]">
            {label}
          </div>
          <Icon className="size-5 text-[#ef2334] dark:text-[#ff3b4f]" />
        </div>
        <div className="mt-4 text-2xl font-black tabular-nums">{value}</div>
      </CardContent>
    </Card>
  );
}

type AvailabilityStatus = "available" | "empty" | "taken";

function AvailabilityIcon({
  availableLabel,
  status,
  unavailableLabel,
}: {
  availableLabel: string;
  status: AvailabilityStatus;
  unavailableLabel: string;
}) {
  if (status === "empty") {
    return null;
  }

  const isAvailable = status === "available";
  const label = isAvailable ? availableLabel : unavailableLabel;
  const Icon = isAvailable ? CheckCircle2 : XCircle;

  return (
    <span
      aria-label={label}
      className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2"
      title={label}
    >
      <Icon
        className={
          isAvailable
            ? "size-5 text-emerald-500"
            : "size-5 text-[#ef2334] dark:text-[#ff5a66]"
        }
      />
    </span>
  );
}

function getNameAvailability(
  name: string,
  branches: Branch[],
  currentBranchID?: string,
): AvailabilityStatus {
  const normalized = normalizeComparableName(name);
  if (!normalized) {
    return "empty";
  }

  return branches.some(
    (branch) =>
      branch.id !== currentBranchID &&
      normalizeComparableName(branch.name) === normalized,
  )
    ? "taken"
    : "available";
}

function getRoomNameAvailability(
  name: string,
  rooms: Room[],
  pendingRooms: RoomDraft[],
  editingRoomID: string | null,
): AvailabilityStatus {
  const normalized = normalizeComparableName(name);
  if (!normalized) {
    return "empty";
  }

  const existsInSavedRooms = rooms.some(
    (room) =>
      room.id !== editingRoomID && normalizeComparableName(room.name) === normalized,
  );
  const existsInPendingRooms = pendingRooms.some(
    (room) =>
      room.id !== editingRoomID &&
      normalizeComparableName(room.name) === normalized,
  );

  return existsInSavedRooms || existsInPendingRooms ? "taken" : "available";
}

function normalizeComparableName(value: string) {
  return value.trim().replace(/\s+/g, " ").toLocaleLowerCase("en-US");
}

function staffInitials(member: StaffMember) {
  return `${member.first_name[0] ?? ""}${member.last_name[0] ?? ""}`
    .trim()
    .toUpperCase();
}

function formatBirthDateInput(value: string) {
  const digits = value.replace(/\D/g, "").slice(0, 8);
  if (digits.length <= 2) {
    return digits;
  }
  if (digits.length <= 4) {
    return `${digits.slice(0, 2)}/${digits.slice(2)}`;
  }

  return `${digits.slice(0, 2)}/${digits.slice(2, 4)}/${digits.slice(4)}`;
}

function StaffCell({
  label,
  receptionistLabel,
  staff,
}: {
  label: string;
  receptionistLabel: string;
  staff: StaffMember[];
}) {
  if (staff.length > 0) {
    return (
      <div className="flex items-center gap-3">
        <AvatarGroup>
          {staff.slice(0, 3).map((member) => (
            <Avatar
              className="border border-white/60 dark:border-[#414d60]"
              key={member.id}
              size="sm"
            >
              {member.profile_photo_url ? (
                <AvatarImage
                  alt={`${member.first_name} ${member.last_name}`}
                  src={member.profile_photo_url}
                />
              ) : null}
              <AvatarFallback className="bg-[#eef2f7] text-[10px] font-black text-[#0a284b] dark:bg-[#2a3444] dark:text-[#f3f6fa]">
                {staffInitials(member)}
              </AvatarFallback>
            </Avatar>
          ))}
        </AvatarGroup>
        <span className="text-sm font-semibold text-[#59667a] dark:text-[#a7b0bf]">
          {staff.length} {receptionistLabel}
        </span>
      </div>
    );
  }

  return (
    <div className="flex items-center gap-3">
      <AvatarGroup>
        <Avatar size="sm">
          <AvatarFallback className="bg-[#eef2f7] text-[10px] font-black text-[#0a284b] dark:bg-[#2a3444] dark:text-[#f3f6fa]">
            --
          </AvatarFallback>
        </Avatar>
      </AvatarGroup>
      <span className="text-sm text-[#687386] dark:text-[#6f7a8a]">
        {label}
      </span>
    </div>
  );
}

function BranchProfileAvatar({ branch }: { branch: Branch }) {
  return (
    <Avatar className="size-11 border border-white/50 shadow-sm dark:border-[#414d60]">
      {branch.photo_url ? (
        <AvatarImage alt={branch.name} src={branch.photo_url} />
      ) : null}
      <AvatarFallback className="bg-[#e8edf5] text-sm font-black text-[#0a284b] dark:bg-[#2a3444] dark:text-[#f3f6fa]">
        {branchInitials(branch.name)}
      </AvatarFallback>
    </Avatar>
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
  const deleteNameMatches = confirmName.trim() === branch.name;

  useEffect(() => {
    if (!deleteState.branch) {
      return;
    }

    window.queueMicrotask(() => {
      onDeleted(deleteState.branch?.id ?? branch.id);
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
            className="cursor-pointer gap-2 text-[#ef2334] hover:text-[#ef2334] focus:bg-[#fff1f2] focus:text-[#ef2334] data-[highlighted]:text-[#ef2334] [&_svg]:text-[#ef2334] [&_svg]:stroke-[#ef2334] dark:text-[#ff3b4f] dark:focus:bg-[#3a1e2a] dark:focus:text-[#ff3b4f] dark:data-[highlighted]:text-[#ff3b4f] dark:[&_svg]:text-[#ff3b4f] dark:[&_svg]:stroke-[#ff3b4f]"
            onSelect={(event) => event.stopPropagation()}
          >
            <Archive className="size-4" />
            {labels.archive}
          </DropdownMenuItem>
          <DropdownMenuItem
            className="cursor-pointer gap-2 text-[#8f1020] hover:text-[#8f1020] focus:bg-[#fff1f2] focus:text-[#8f1020] data-[highlighted]:text-[#8f1020] [&_svg]:text-[#8f1020] [&_svg]:stroke-[#8f1020] dark:text-[#ff6b7a] dark:focus:bg-[#3a1e2a] dark:focus:text-[#ff6b7a] dark:data-[highlighted]:text-[#ff6b7a] dark:[&_svg]:text-[#ff6b7a] dark:[&_svg]:stroke-[#ff6b7a]"
            onSelect={openDeleteDialog}
          >
            <Trash2 className="size-4" />
            {labels.delete}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      <Dialog open={deleteOpen} onOpenChange={handleDeleteOpenChange}>
        <DialogContent
          className="z-[1200] border-[#dce3ee] dark:border-[#3a4658] dark:bg-[#1b2635] dark:text-[#f3f6fa]"
          onClick={(event) => event.stopPropagation()}
        >
          {deleteStep === "name" ? (
            <>
              <DialogHeader>
                <DialogTitle className="text-xl font-black">
                  {labels.deleteBranchTitle}
                </DialogTitle>
                <div className="rounded-lg border border-[#f87171] bg-[#fff1f2] px-4 py-3 text-sm font-black text-[#991b1b] dark:bg-[#3b1218] dark:text-[#ffe4e6]">
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
                  <p className="text-sm font-semibold text-[#ef2334] dark:text-[#ff6b7a]">
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
                  className="bg-[#8f1020] text-white hover:bg-[#74101d]"
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
                <DialogTitle className="text-xl font-black text-[#8f1020] dark:text-[#ff6b7a]">
                  {labels.deleteBranchWarningTitle}
                </DialogTitle>
                <DialogDescription className="text-[#59667a] dark:text-[#cbd5e1]">
                  {labels.deleteBranchWarningDescription}
                </DialogDescription>
              </DialogHeader>
              <form action={deleteFormAction}>
                <input name="branch_id" type="hidden" value={branch.id} />
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

function RoomList({
  editLabel,
  emptyLabel,
  label,
  onEdit,
  onRemove,
  removeLabel,
  rooms,
}: {
  editLabel: string;
  emptyLabel: string;
  label: string;
  onEdit: (room: Room) => void;
  onRemove: (room: Room) => void;
  removeLabel: string;
  rooms: Room[];
}) {
  return (
    <div className="space-y-3">
      <div className="text-sm font-black text-[#59667a] dark:text-[#a7b0bf]">
        {label}
      </div>
      {rooms.length === 0 ? (
        <div className="rounded-lg border border-dashed border-[#dce3ee] px-3 py-6 text-center text-sm text-[#687386] dark:border-[#3a4658] dark:text-[#6f7a8a]">
          {emptyLabel}
        </div>
      ) : (
        <div className="grid gap-2">
          {rooms.map((room) => (
            <RoomRow
              capacity={room.capacity}
              editLabel={editLabel}
              key={room.id}
              name={room.name}
              onEdit={() => onEdit(room)}
              onRemove={() => onRemove(room)}
              removeLabel={removeLabel}
            />
          ))}
        </div>
      )}
    </div>
  );
}

function DraftRoomList({
  editLabel,
  label,
  onEdit,
  onRemove,
  removeLabel,
  rooms,
}: {
  editLabel: string;
  label: string;
  onEdit: (room: RoomDraft) => void;
  onRemove: (id: string) => void;
  removeLabel: string;
  rooms: RoomDraft[];
}) {
  if (rooms.length === 0) {
    return null;
  }

  return (
    <div className="space-y-3">
      <div className="text-sm font-black text-[#59667a] dark:text-[#a7b0bf]">
        {label}
      </div>
      <div className="grid gap-2">
        {rooms.map((room) => (
          <RoomRow
            capacity={room.capacity}
            editLabel={editLabel}
            key={room.id}
            name={room.name}
            onEdit={() => onEdit(room)}
            onRemove={() => onRemove(room.id)}
            removeLabel={removeLabel}
          />
        ))}
      </div>
    </div>
  );
}

function RoomRow({
  capacity,
  editLabel,
  name,
  onEdit,
  onRemove,
  removeLabel,
}: {
  capacity: number;
  editLabel?: string;
  name: string;
  onEdit?: () => void;
  onRemove: () => void;
  removeLabel: string;
}) {
  return (
    <div
      className="flex items-center justify-between gap-3 rounded-lg border border-[#dce3ee] bg-[#f8fafc] px-4 py-3 dark:border-[#3a4658] dark:bg-[#202b3a]"
      data-testid="room-row"
    >
      <div>
        <div className="font-bold">{name}</div>
        <div className="text-xs font-semibold text-[#687386] dark:text-[#6f7a8a]">
          {capacity}
        </div>
      </div>
      <div className="flex items-center gap-1">
        {onEdit ? (
          <Button
            aria-label={editLabel}
            className="text-[#0a284b] hover:text-[#0a284b] dark:text-[#f3f6fa] dark:hover:text-white"
            onClick={onEdit}
            size="icon"
            type="button"
            variant="ghost"
          >
            <Pencil className="size-4" />
          </Button>
        ) : null}
        <Button
          aria-label={removeLabel}
          className="text-[#ef2334] hover:text-[#ef2334] dark:text-[#ff3b4f] dark:hover:text-[#ff3b4f]"
          onClick={onRemove}
          size="icon"
          type="button"
          variant="ghost"
        >
          <Trash2 className="size-4" />
        </Button>
      </div>
    </div>
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
      className="w-full rounded-lg bg-[#ef2334] font-bold text-white hover:bg-[#d91f30]"
      disabled={blocked}
      type="submit"
      onClick={submitLock.onClick}
    >
      {status.pending ? pending : label}
    </Button>
  );
}

function SaveBranchSubmit({
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
      className="gap-2 rounded-lg bg-emerald-600 px-5 font-bold text-white hover:bg-emerald-700"
      disabled={blocked}
      type="submit"
      onClick={submitLock.onClick}
    >
      <Save className="size-4" />
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
      className="bg-[#8f1020] text-white hover:bg-[#74101d]"
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

  return <p className="text-sm font-semibold text-[#ef2334]">{message}</p>;
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
    <div className="mt-4 rounded-lg border border-[#f87171] bg-[#fff1f2] px-4 py-3 text-sm font-semibold text-[#991b1b] dark:bg-[#3b1218] dark:text-[#ffe4e6]">
      {message}
    </div>
  );
}

function SaveError({
  labels,
  state,
}: {
  labels: BranchManagementLabels;
  state: SaveBranchManagementState;
}) {
  const message = state.fieldErrors?.name?.length
    ? labels.branchNameRequired
    : state.fieldErrors?.address?.length
      ? labels.addressRequired
      : state.error === "invalid_photo"
        ? labels.invalidPhotoError
        : state.error === "photo_too_large"
          ? labels.photoTooLargeError
          : state.error === "invalid_room"
            ? labels.invalidRoomError
            : state.error === "invalid_input"
              ? labels.backendError
              : state.error === "duplicate"
                ? labels.duplicateError
                : state.error === "duplicate_room"
                  ? labels.duplicateRoomError
                  : state.error === "unauthorized"
                    ? labels.unauthorizedError
                    : state.error === "backend"
                      ? labels.backendError
                      : null;

  if (!message) {
    return null;
  }

  return (
    <div className="rounded-lg border border-[#f87171] bg-[#fff1f2] px-4 py-3 text-sm font-semibold text-[#991b1b] dark:bg-[#3b1218] dark:text-[#ffe4e6]">
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

function branchInitials(name: string) {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) {
    return "--";
  }

  return parts
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase())
    .join("");
}
