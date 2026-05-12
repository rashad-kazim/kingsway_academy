import { useRouter } from "next/navigation";
import {
  useActionState,
  type ChangeEvent,
  useEffect,
  useRef,
  useState,
} from "react";
import { useFormStatus } from "react-dom";
import {
  ArrowLeft,
  CheckCircle2,
  Clock,
  DoorOpen,
  ImagePlus,
  Pencil,
  Plus,
  Save,
  Trash2,
  X,
  XCircle,
} from "lucide-react";
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
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { ObjectCoverImage } from "@/components/owner/shared/object-cover-image";
import type { Branch, Room } from "@/lib/api/types";
import {
  saveBranchManagementAction,
  type SaveBranchManagementState,
} from "@/lib/owner/actions";
import { useSubmitLock } from "@/lib/forms/use-submit-lock";
import type { BranchManagementLabels } from "./branch-management-view";

type RoomDraft = {
  id: string;
  name: string;
  capacity: number;
};

const initialSaveState: SaveBranchManagementState = {};

export function BranchDetailPage({
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
  const [saveIdempotencyKey, setSaveIdempotencyKey] = useState(() =>
    crypto.randomUUID(),
  );
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
      setSaveIdempotencyKey(crypto.randomUUID());
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
          <p className="mt-1 text-sm text-kw-c-59667a dark:text-kw-c-a7b0bf">
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

      <Card className="rounded-xl border-kw-c-dce3ee bg-white/80 shadow-kw-panel backdrop-blur-xl dark:border-kw-c-3a4658 dark:bg-kw-c-1b2635">
        <CardContent className="space-y-8 p-6">
          <section className="flex flex-col items-center gap-4">
            <div className="relative size-36">
              <label className="group relative flex size-36 cursor-pointer items-center justify-center overflow-hidden rounded-full border border-dashed border-kw-c-b9c5d6 bg-kw-c-f7f8fb text-kw-c-0a284b shadow-inner transition hover:border-kw-c-ef2334 hover:bg-kw-c-f1f4f8 dark:border-kw-c-3a4658 dark:bg-kw-c-202b3a dark:text-kw-c-f3f6fa dark:hover:border-kw-c-ff3b4f dark:hover:bg-kw-c-263448">
                {selectedPhoto ? (
                  <ObjectCoverImage alt={branch.name} src={selectedPhoto} />
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
                  className="absolute right-0 top-0 grid size-8 cursor-pointer place-items-center rounded-full bg-kw-c-ef2334 text-white shadow-lg ring-4 ring-white transition hover:bg-kw-c-d91f30 dark:bg-kw-c-ff3b4f dark:ring-kw-c-1b2635 dark:hover:bg-kw-c-ff5a69"
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

          <section className="rounded-lg border border-kw-c-dce3ee p-4 dark:border-kw-c-3a4658">
            <div className="mb-4 flex items-center gap-2 font-black">
              <Clock className="size-5 text-kw-c-ef2334 dark:text-kw-c-ff3b4f" />
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

      <Card className="rounded-xl border-kw-c-dce3ee bg-white/80 shadow-kw-panel backdrop-blur-xl dark:border-kw-c-3a4658 dark:bg-kw-c-1b2635">
        <CardContent className="space-y-5 p-6">
          <div className="flex items-center gap-2 font-black">
            <DoorOpen className="size-5 text-kw-c-ef2334 dark:text-kw-c-ff3b4f" />
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
                className="w-full max-w-2xl rounded-xl border border-kw-c-dce3ee bg-kw-c-f8fafc p-4 shadow-sm dark:border-kw-c-3a4658 dark:bg-kw-c-202b3a"
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
        <DialogContent className="z-[1200] border-kw-c-dce3ee dark:border-kw-c-3a4658 dark:bg-kw-c-1b2635 dark:text-kw-c-f3f6fa">
          <DialogHeader>
            <DialogTitle className="text-xl font-black">
              {labels.deleteRoomTitle}
            </DialogTitle>
            <DialogDescription className="text-kw-c-59667a dark:text-kw-c-cbd5e1">
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
              className="bg-kw-c-8f1020 text-white hover:bg-kw-c-74101d"
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


type AvailabilityStatus = "available" | "empty" | "taken";

export function AvailabilityIcon({
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
            : "size-5 text-kw-c-ef2334 dark:text-kw-c-ff5a66"
        }
      />
    </span>
  );
}

export function getNameAvailability(
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
      <div className="text-sm font-black text-kw-c-59667a dark:text-kw-c-a7b0bf">
        {label}
      </div>
      {rooms.length === 0 ? (
        <div className="rounded-lg border border-dashed border-kw-c-dce3ee px-3 py-6 text-center text-sm text-kw-c-687386 dark:border-kw-c-3a4658 dark:text-kw-c-6f7a8a">
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
      <div className="text-sm font-black text-kw-c-59667a dark:text-kw-c-a7b0bf">
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
      className="flex items-center justify-between gap-3 rounded-lg border border-kw-c-dce3ee bg-kw-c-f8fafc px-4 py-3 dark:border-kw-c-3a4658 dark:bg-kw-c-202b3a"
      data-testid="room-row"
    >
      <div>
        <div className="font-bold">{name}</div>
        <div className="text-xs font-semibold text-kw-c-687386 dark:text-kw-c-6f7a8a">
          {capacity}
        </div>
      </div>
      <div className="flex items-center gap-1">
        {onEdit ? (
          <Button
            aria-label={editLabel}
            className="text-kw-c-0a284b hover:text-kw-c-0a284b dark:text-kw-c-f3f6fa dark:hover:text-white"
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
          className="text-kw-c-ef2334 hover:text-kw-c-ef2334 dark:text-kw-c-ff3b4f dark:hover:text-kw-c-ff3b4f"
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
    <div className="rounded-lg border border-kw-c-f87171 bg-kw-c-fff1f2 px-4 py-3 text-sm font-semibold text-kw-c-991b1b dark:bg-kw-c-3b1218 dark:text-kw-c-ffe4e6">
      {message}
    </div>
  );
}

