import type {
  Branch,
  CreateBranchInput,
  CreateRoomInput,
  CreateStaffInput,
  DashboardRecord,
  DownloadURLResult,
  FileObject,
  LoginResult,
  Role,
  Room,
  Session,
  StaffMember,
  UpdateBranchInput,
  UpdateStaffInput,
} from "./types";

type ApiFetchOptions = Omit<RequestInit, "body"> & {
  body?: unknown;
  token?: string;
};

export class ApiError extends Error {
  status: number;
  payload: unknown;

  constructor(status: number, payload: unknown) {
    super(extractErrorMessage(payload) ?? `Backend request failed: ${status}`);
    this.name = "ApiError";
    this.status = status;
    this.payload = payload;
  }
}

export async function apiFetch<T>(
  path: string,
  options: ApiFetchOptions = {},
): Promise<T> {
  const headers = new Headers(options.headers);
  if (options.token) {
    headers.set("Authorization", `Bearer ${options.token}`);
  }
  if (options.body !== undefined && !(options.body instanceof FormData)) {
    headers.set("Content-Type", "application/json");
  }
  if (!headers.has("X-Request-ID")) {
    headers.set("X-Request-ID", crypto.randomUUID());
  }

  const response = await fetch(backendURL(path), {
    ...options,
    headers,
    cache: "no-store",
    body:
      options.body === undefined
        ? undefined
        : options.body instanceof FormData
          ? options.body
          : JSON.stringify(options.body),
  });

  const payload = await readPayload(response);
  if (!response.ok) {
    throw new ApiError(response.status, payload);
  }

  return payload as T;
}

export function login(email: string, password: string) {
  return apiFetch<LoginResult>("/v1/auth/login", {
    method: "POST",
    body: { email, password },
  });
}

export function getSession(token: string) {
  return apiFetch<Session>("/v1/session", { token });
}

export function getDashboard(role: Role, token: string, branchId?: string) {
  const query = branchId ? `?branch_id=${encodeURIComponent(branchId)}` : "";
  return apiFetch<DashboardRecord>(`/v1/dashboard/${role}${query}`, { token });
}

export function listBranches(token: string) {
  return apiFetch<Branch[]>("/v1/branches", { token });
}

export function createBranch(input: CreateBranchInput, token: string) {
  return apiFetch<Branch>("/v1/branches", {
    method: "POST",
    token,
    body: input,
  });
}

export function updateBranch(
  branchId: string,
  input: UpdateBranchInput,
  token: string,
) {
  return apiFetch<Branch>(`/v1/branches/${branchId}`, {
    method: "PATCH",
    token,
    body: input,
  });
}

export function deleteBranch(branchId: string, token: string) {
  return apiFetch<Branch>(`/v1/branches/${branchId}`, {
    method: "DELETE",
    token,
  });
}

export function listRooms(token: string, branchId?: string) {
  const query = branchId ? `?branch_id=${encodeURIComponent(branchId)}` : "";
  return apiFetch<Room[]>(`/v1/rooms${query}`, { token });
}

export function createRoom(input: CreateRoomInput, token: string) {
  return apiFetch<Room>("/v1/rooms", {
    method: "POST",
    token,
    body: input,
  });
}

export function deleteRoom(roomId: string, token: string) {
  return apiFetch<Room>(`/v1/rooms/${roomId}`, {
    method: "DELETE",
    token,
  });
}

export function listBranchStaff(branchId: string, token: string) {
  return apiFetch<StaffMember[]>(`/v1/branches/${branchId}/staff`, { token });
}

export function createBranchStaff(
  branchId: string,
  input: CreateStaffInput,
  token: string,
) {
  return apiFetch<StaffMember>(`/v1/branches/${branchId}/staff`, {
    method: "POST",
    token,
    body: input,
  });
}

export function updateStaff(
  staffId: string,
  input: UpdateStaffInput,
  token: string,
) {
  return apiFetch<StaffMember>(`/v1/staff/${staffId}`, {
    method: "PATCH",
    token,
    body: input,
  });
}

export function deleteStaff(staffId: string, token: string) {
  return apiFetch<StaffMember>(`/v1/staff/${staffId}`, {
    method: "DELETE",
    token,
  });
}

export function uploadBranchPhoto(
  branchId: string,
  photo: File,
  token: string,
) {
  const formData = new FormData();
  formData.set("branch_id", branchId);
  formData.set("owner_type", "branch");
  formData.set("owner_id", branchId);
  formData.set("category", "standard");
  formData.set("purpose", "profile_photo");
  formData.set("file", photo);

  return apiFetch<FileObject>("/v1/files/upload", {
    method: "POST",
    token,
    body: formData,
  });
}

export function uploadStaffPhoto(
  branchId: string,
  staffId: string,
  photo: File,
  token: string,
) {
  const formData = new FormData();
  formData.set("branch_id", branchId);
  formData.set("owner_type", "staff");
  formData.set("owner_id", staffId);
  formData.set("category", "standard");
  formData.set("purpose", "profile_photo");
  formData.set("file", photo);

  return apiFetch<FileObject>("/v1/files/upload", {
    method: "POST",
    token,
    body: formData,
  });
}

export function listBranchProfileFiles(branchId: string, token: string) {
  const params = new URLSearchParams({
    branch_id: branchId,
    owner_type: "branch",
    owner_id: branchId,
    purpose: "profile_photo",
    limit: "1",
  });

  return apiFetch<FileObject[]>(`/v1/files?${params.toString()}`, { token });
}

export function getFileDownloadURL(fileId: string, token: string) {
  return apiFetch<DownloadURLResult>(`/v1/files/${fileId}/download-url`, {
    token,
  });
}

export function deleteFile(fileId: string, token: string) {
  return apiFetch<FileObject>(`/v1/files/${fileId}`, {
    method: "DELETE",
    token,
  });
}

function backendURL(path: string) {
  const base = process.env.KINGSWAY_API_BASE_URL ?? "http://127.0.0.1:8080";
  return new URL(path, base).toString();
}

async function readPayload(response: Response) {
  const contentType = response.headers.get("content-type") ?? "";
  if (contentType.includes("application/json")) {
    return response.json();
  }

  const text = await response.text();
  return text.length > 0 ? text : null;
}

function extractErrorMessage(payload: unknown) {
  if (
    payload &&
    typeof payload === "object" &&
    "error" in payload &&
    typeof payload.error === "string"
  ) {
    return payload.error;
  }

  return null;
}
