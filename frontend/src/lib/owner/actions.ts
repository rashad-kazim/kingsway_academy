"use server";

import { z } from "zod";
import {
  ApiError,
  createRoom,
  createBranch,
  createBranchStaff,
  deleteBranch,
  deleteFile,
  deleteRoom,
  deleteStaff,
  getFileDownloadURL,
  updateStaff,
  updateBranch,
  uploadBranchPhoto,
  uploadStaffPhoto,
} from "@/lib/api/client";
import type { Branch, Room, StaffMember } from "@/lib/api/types";
import { getAuthToken } from "@/lib/auth/session";

export type CreateBranchState = {
  branch?: Branch;
  error?:
    | "invalid_input"
    | "duplicate"
    | "unauthorized"
    | "backend"
    | "invalid_photo"
    | "photo_too_large";
  fieldErrors?: {
    name?: string[];
    address?: string[];
  };
};

export type UploadBranchPhotoState = {
  photoUrl?: string;
  error?: "invalid_photo" | "photo_too_large" | "unauthorized" | "backend";
};

export type SaveBranchManagementState = {
  branch?: Branch;
  rooms?: Room[];
  error?:
    | "invalid_input"
    | "duplicate"
    | "duplicate_room"
    | "unauthorized"
    | "backend"
    | "invalid_photo"
    | "photo_too_large"
    | "invalid_room";
  fieldErrors?: {
    name?: string[];
    address?: string[];
  };
};

export type DeleteBranchState = {
  branch?: Branch;
  error?: "not_found" | "unauthorized" | "backend";
};

export type StaffManagementState = {
  staff?: StaffMember;
  deletedStaffID?: string;
  error?:
    | "invalid_input"
    | "duplicate"
    | "unauthorized"
    | "backend"
    | "invalid_photo"
    | "photo_too_large";
};

const MAX_BRANCH_PHOTO_BYTES = 10 * 1024 * 1024;
const ALLOWED_BRANCH_PHOTO_TYPES = new Set([
  "image/jpeg",
  "image/jpg",
  "image/pjpeg",
  "image/png",
  "image/x-png",
  "image/webp",
]);

const branchFormSchema = z.object({
  name: z.string().trim().min(2).max(120),
  address: z.string().trim().min(3).max(1000),
});

const branchCreateFormSchema = branchFormSchema.extend({
  openingTime: z.string().trim().regex(/^$|^\d{2}:\d{2}$/),
  closingTime: z.string().trim().regex(/^$|^\d{2}:\d{2}$/),
});

const branchManagementFormSchema = branchFormSchema.extend({
  branchID: z.string().trim().min(1),
  openingTime: z
    .string()
    .trim()
    .regex(/^$|^\d{2}:\d{2}$/),
  closingTime: z
    .string()
    .trim()
    .regex(/^$|^\d{2}:\d{2}$/),
  photoFileID: z.string().trim(),
  removePhoto: z.boolean(),
});

const roomDraftSchema = z.array(
  z.object({
    name: z.string().trim().min(1).max(80),
    capacity: z.coerce.number().int().min(1).max(500),
  }),
);

const removedRoomIDsSchema = z.array(z.string().trim().min(1));

const staffBaseSchema = z.object({
  branchID: z.string().trim().min(1),
  staffID: z.string().trim(),
  firstName: z.string().trim().min(2).max(80),
  lastName: z.string().trim().min(2).max(80),
  birthDate: z.string().trim().regex(/^$|^\d{2}\/\d{2}\/\d{4}$/),
  phone: z.string().trim().max(40),
  salary: z.string().trim().regex(/^\d*$/),
  email: z.string().trim().email().max(160),
  password: z.string(),
  profilePhotoFileID: z.string().trim(),
  removePhoto: z.boolean(),
});

export async function createBranchAction(
  _prevState: CreateBranchState,
  formData: FormData,
): Promise<CreateBranchState> {
  const parsed = branchCreateFormSchema.safeParse({
    name: formData.get("name"),
    address: formData.get("address"),
    openingTime: formData.get("opening_time") ?? "",
    closingTime: formData.get("closing_time") ?? "",
  });

  if (!parsed.success) {
    const flattened = parsed.error.flatten();
    return {
      error: "invalid_input",
      fieldErrors: {
        name: flattened.fieldErrors.name,
        address: flattened.fieldErrors.address,
      },
    };
  }

  const token = await getAuthToken();
  if (!token) {
    return { error: "unauthorized" };
  }

  const photo = formData.get("photo");
  const photoValidation = validateBranchPhoto(photo);
  if (photoValidation.error) {
    return { error: photoValidation.error };
  }

  try {
    let branch = await createBranch(
      {
        name: parsed.data.name,
        slug: slugFromName(parsed.data.name),
        address: parsed.data.address,
        opening_time: parsed.data.openingTime,
        closing_time: parsed.data.closingTime,
      },
      token,
    );
    if (photoValidation.file) {
      try {
        const file = await uploadBranchPhoto(
          branch.id,
          photoValidation.file,
          token,
        );
        const download = await getFileDownloadURL(file.id, token);
        branch = { ...branch, photo_url: download.url };
      } catch {
        // The branch is already saved. Keep the user moving and let refresh
        // show the branch even if object storage is temporarily unavailable.
      }
    }
    return { branch };
  } catch (error) {
    if (error instanceof ApiError && error.status === 409) {
      return { error: "duplicate" };
    }
    if (
      error instanceof ApiError &&
      (error.status === 401 || error.status === 403)
    ) {
      return { error: "unauthorized" };
    }
    return { error: "backend" };
  }
}

export async function uploadBranchPhotoAction(
  branchID: string,
  formData: FormData,
): Promise<UploadBranchPhotoState> {
  const token = await getAuthToken();
  if (!token) {
    return { error: "unauthorized" };
  }

  const photoValidation = validateBranchPhoto(formData.get("photo"));
  if (photoValidation.error) {
    return { error: photoValidation.error };
  }
  if (!photoValidation.file) {
    return { error: "invalid_photo" };
  }

  try {
    const file = await uploadBranchPhoto(branchID, photoValidation.file, token);
    const download = await getFileDownloadURL(file.id, token);
    return { photoUrl: download.url };
  } catch {
    return { error: "backend" };
  }
}

export async function saveBranchManagementAction(
  _prevState: SaveBranchManagementState,
  formData: FormData,
): Promise<SaveBranchManagementState> {
  const parsed = branchManagementFormSchema.safeParse({
    branchID: formData.get("branch_id"),
    name: formData.get("name"),
    address: formData.get("address"),
    openingTime: formData.get("opening_time") ?? "",
    closingTime: formData.get("closing_time") ?? "",
    photoFileID: formData.get("photo_file_id") ?? "",
    removePhoto: formData.get("remove_photo") === "1",
  });

  if (!parsed.success) {
    const flattened = parsed.error.flatten();
    return {
      error: "invalid_input",
      fieldErrors: {
        name: flattened.fieldErrors.name,
        address: flattened.fieldErrors.address,
      },
    };
  }

  const newRooms = parseJSON(formData.get("new_rooms"), roomDraftSchema);
  const removedRoomIDs = parseJSON(
    formData.get("removed_room_ids"),
    removedRoomIDsSchema,
  );
  if (!newRooms.ok || !removedRoomIDs.ok) {
    return { error: "invalid_room" };
  }

  const token = await getAuthToken();
  if (!token) {
    return { error: "unauthorized" };
  }

  const photoValidation = validateBranchPhoto(formData.get("photo"));
  if (photoValidation.error) {
    return { error: photoValidation.error };
  }

  try {
    let branch: Branch;
    try {
      branch = await updateBranch(
        parsed.data.branchID,
        {
          name: parsed.data.name,
          slug: slugFromName(parsed.data.name),
          address: parsed.data.address,
          opening_time: parsed.data.openingTime,
          closing_time: parsed.data.closingTime,
        },
        token,
      );
    } catch (error) {
      if (error instanceof ApiError && error.status === 409) {
        return { error: "duplicate" };
      }
      throw error;
    }

    if (parsed.data.removePhoto && parsed.data.photoFileID) {
      try {
        await deleteFile(parsed.data.photoFileID, token);
      } catch (error) {
        if (!(error instanceof ApiError && error.status === 404)) {
          throw error;
        }
      }
      branch = { ...branch, photo_file_id: undefined, photo_url: undefined };
    }

    if (photoValidation.file) {
      let file;
      try {
        file = await uploadBranchPhoto(
          parsed.data.branchID,
          photoValidation.file,
          token,
        );
      } catch (error) {
        if (error instanceof ApiError && error.status === 400) {
          return { error: "invalid_photo" };
        }
        throw error;
      }

      let photoURL: string | undefined;
      try {
        const download = await getFileDownloadURL(file.id, token);
        photoURL = download.url;
      } catch {
        photoURL = undefined;
      }
      branch = {
        ...branch,
        photo_file_id: file.id,
        photo_url: photoURL,
      };
    }

    const rooms: Room[] = [];
    for (const id of removedRoomIDs.data) {
      try {
        await deleteRoom(id, token);
      } catch (error) {
        if (!(error instanceof ApiError && error.status === 404)) {
          throw error;
        }
      }
    }
    for (const room of newRooms.data) {
      try {
        rooms.push(
          await createRoom(
            {
              branch_id: parsed.data.branchID,
              name: room.name,
              capacity: room.capacity,
            },
            token,
          ),
        );
      } catch (error) {
        if (error instanceof ApiError && error.status === 409) {
          return { error: "duplicate_room" };
        }
        throw error;
      }
    }

    return { branch, rooms };
  } catch (error) {
    if (error instanceof ApiError && error.status === 409) {
      return { error: "duplicate" };
    }
    if (
      error instanceof ApiError &&
      (error.status === 401 || error.status === 403)
    ) {
      return { error: "unauthorized" };
    }
    return { error: "backend" };
  }
}

export async function deleteBranchAction(
  _prevState: DeleteBranchState,
  formData: FormData,
): Promise<DeleteBranchState> {
  const branchID = String(formData.get("branch_id") ?? "").trim();
  if (!branchID) {
    return { error: "not_found" };
  }

  const token = await getAuthToken();
  if (!token) {
    return { error: "unauthorized" };
  }

  try {
    const branch = await deleteBranch(branchID, token);
    return { branch };
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) {
      return { error: "not_found" };
    }
    if (error instanceof ApiError && error.status === 401) {
      return { error: "unauthorized" };
    }
    return { error: "backend" };
  }
}

export async function createStaffAction(
  _prevState: StaffManagementState,
  formData: FormData,
): Promise<StaffManagementState> {
  const parsed = staffBaseSchema.safeParse(staffFormValues(formData));
  if (!parsed.success || parsed.data.password.length < 8) {
    return { error: "invalid_input" };
  }

  const token = await getAuthToken();
  if (!token) {
    return { error: "unauthorized" };
  }

  const photoValidation = validateBranchPhoto(formData.get("photo"));
  if (photoValidation.error) {
    return { error: photoValidation.error };
  }

  try {
    let staff = await createBranchStaff(
      parsed.data.branchID,
      {
        birth_date: parsed.data.birthDate,
        email: parsed.data.email,
        first_name: parsed.data.firstName,
        last_name: parsed.data.lastName,
        password: parsed.data.password,
        phone: parsed.data.phone,
        salary_amount_azn: salaryAmount(parsed.data.salary),
      },
      token,
    );

    if (photoValidation.file) {
      const file = await uploadStaffPhoto(
        parsed.data.branchID,
        staff.id,
        photoValidation.file,
        token,
      );
      const download = await getFileDownloadURL(file.id, token);
      staff = await updateStaff(
        staff.id,
        {
          birth_date: parsed.data.birthDate,
          email: parsed.data.email,
          first_name: parsed.data.firstName,
          last_name: parsed.data.lastName,
          phone: parsed.data.phone,
          profile_photo_file_id: file.id,
          salary_amount_azn: salaryAmount(parsed.data.salary),
        },
        token,
      );
      staff = {
        ...staff,
        profile_photo_file_id: file.id,
        profile_photo_url: download.url,
      };
    }

    return { staff };
  } catch (error) {
    if (error instanceof ApiError && error.status === 409) {
      return { error: "duplicate" };
    }
    if (error instanceof ApiError && (error.status === 401 || error.status === 403)) {
      return { error: "unauthorized" };
    }
    return { error: "backend" };
  }
}

export async function updateStaffAction(
  _prevState: StaffManagementState,
  formData: FormData,
): Promise<StaffManagementState> {
  const parsed = staffBaseSchema.safeParse(staffFormValues(formData));
  if (!parsed.success || !parsed.data.staffID) {
    return { error: "invalid_input" };
  }
  if (parsed.data.password && parsed.data.password.length < 8) {
    return { error: "invalid_input" };
  }

  const token = await getAuthToken();
  if (!token) {
    return { error: "unauthorized" };
  }

  const photoValidation = validateBranchPhoto(formData.get("photo"));
  if (photoValidation.error) {
    return { error: photoValidation.error };
  }

  try {
    let profilePhotoFileID = parsed.data.profilePhotoFileID;
    let profilePhotoURL: string | undefined;

    if (parsed.data.removePhoto && profilePhotoFileID) {
      try {
        await deleteFile(profilePhotoFileID, token);
      } catch (error) {
        if (!(error instanceof ApiError && error.status === 404)) {
          throw error;
        }
      }
      profilePhotoFileID = "";
    }

    if (photoValidation.file) {
      const file = await uploadStaffPhoto(
        parsed.data.branchID,
        parsed.data.staffID,
        photoValidation.file,
        token,
      );
      const download = await getFileDownloadURL(file.id, token);
      profilePhotoFileID = file.id;
      profilePhotoURL = download.url;
    }

    const staff = await updateStaff(
      parsed.data.staffID,
      {
        birth_date: parsed.data.birthDate,
        email: parsed.data.email,
        first_name: parsed.data.firstName,
        last_name: parsed.data.lastName,
        password: parsed.data.password || undefined,
        phone: parsed.data.phone,
        profile_photo_file_id: profilePhotoFileID,
        salary_amount_azn: salaryAmount(parsed.data.salary),
      },
      token,
    );

    return {
      staff: {
        ...staff,
        profile_photo_file_id: profilePhotoFileID || undefined,
        profile_photo_url: profilePhotoURL,
      },
    };
  } catch (error) {
    if (error instanceof ApiError && error.status === 409) {
      return { error: "duplicate" };
    }
    if (error instanceof ApiError && (error.status === 401 || error.status === 403)) {
      return { error: "unauthorized" };
    }
    return { error: "backend" };
  }
}

export async function deleteStaffAction(
  _prevState: StaffManagementState,
  formData: FormData,
): Promise<StaffManagementState> {
  const staffID = String(formData.get("staff_id") ?? "").trim();
  if (!staffID) {
    return { error: "invalid_input" };
  }

  const token = await getAuthToken();
  if (!token) {
    return { error: "unauthorized" };
  }

  try {
    const staff = await deleteStaff(staffID, token);
    return { deletedStaffID: staff.id };
  } catch (error) {
    if (error instanceof ApiError && (error.status === 401 || error.status === 403)) {
      return { error: "unauthorized" };
    }
    return { error: "backend" };
  }
}

function staffFormValues(formData: FormData) {
  return {
    birthDate: formData.get("birth_date") ?? "",
    branchID: formData.get("branch_id"),
    email: formData.get("email"),
    firstName: formData.get("first_name"),
    lastName: formData.get("last_name"),
    password: String(formData.get("password") ?? ""),
    phone: formData.get("phone") ?? "",
    profilePhotoFileID: formData.get("profile_photo_file_id") ?? "",
    removePhoto: formData.get("remove_photo") === "1",
    salary: formData.get("salary") ?? "",
    staffID: formData.get("staff_id") ?? "",
  };
}

function salaryAmount(value: string) {
  return Number.parseInt(value || "0", 10) || 0;
}

function validateBranchPhoto(value: FormDataEntryValue | null): {
  file?: File;
  error?: "invalid_photo" | "photo_too_large";
} {
  if (!(value instanceof File) || value.size === 0) {
    return {};
  }
  if (!isAllowedBranchPhoto(value)) {
    return { error: "invalid_photo" };
  }
  if (value.size > MAX_BRANCH_PHOTO_BYTES) {
    return { error: "photo_too_large" };
  }

  return { file: value };
}

function isAllowedBranchPhoto(file: File) {
  const type = file.type.trim().toLowerCase().split(";")[0];
  if (ALLOWED_BRANCH_PHOTO_TYPES.has(type)) {
    return true;
  }
  if (type !== "" && type !== "application/octet-stream") {
    return false;
  }

  return /\.(jpe?g|png|webp)$/i.test(file.name);
}

function parseJSON<T>(
  value: FormDataEntryValue | null,
  schema: z.ZodType<T>,
): { data: T; ok: true } | { ok: false } {
  if (typeof value !== "string") {
    return { ok: false };
  }

  try {
    const parsed = schema.safeParse(JSON.parse(value));
    if (!parsed.success) {
      return { ok: false };
    }

    return { data: parsed.data, ok: true };
  } catch {
    return { ok: false };
  }
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
