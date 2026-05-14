import type {
  Branch,
  Course,
  CreateBranchInput,
  CreateStudentAccountInput,
  CreateStudentInput,
  CreateSalaryModelInput,
  CreateTeacherInput,
  CreateTeacherResult,
  CreateRoomInput,
  CreateStaffInput,
  DashboardRecord,
  DownloadURLResult,
  EmailAvailability,
  FileObject,
  LoginResult,
  Role,
  Room,
  SalaryModel,
  Session,
  StaffMember,
  Student,
  StudentAssignmentHubFilters,
  StudentAssignmentHubPage,
  Teacher,
  TeacherFinanceFilters,
  TeacherFinanceRecord,
  UpdateBranchInput,
  UpdateRoomInput,
  UpdateStaffInput,
  UpdateStudentInput,
  UpdateTeacherInput,
} from "./types";

type ApiFetchOptions = Omit<RequestInit, "body"> & {
  body?: unknown;
  token?: string;
  idempotencyKey?: string;
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
  if (options.idempotencyKey && !headers.has("Idempotency-Key")) {
    headers.set("Idempotency-Key", options.idempotencyKey);
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

export function logoutSession(token: string) {
  return apiFetch<{ ok: boolean }>("/v1/auth/logout", {
    method: "POST",
    token,
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

export function createBranch(
  input: CreateBranchInput,
  token: string,
  idempotencyKey?: string,
) {
  return apiFetch<Branch>("/v1/branches", {
    method: "POST",
    token,
    idempotencyKey,
    body: input,
  });
}

export function updateBranch(
  branchId: string,
  input: UpdateBranchInput,
  token: string,
  idempotencyKey?: string,
) {
  return apiFetch<Branch>(`/v1/branches/${branchId}`, {
    method: "PATCH",
    token,
    idempotencyKey,
    body: input,
  });
}

export function deleteBranch(
  branchId: string,
  token: string,
  idempotencyKey?: string,
) {
  return apiFetch<Branch>(`/v1/branches/${branchId}`, {
    method: "DELETE",
    token,
    idempotencyKey,
  });
}

export function listRooms(token: string, branchId?: string) {
  const query = branchId ? `?branch_id=${encodeURIComponent(branchId)}` : "";
  return apiFetch<Room[]>(`/v1/rooms${query}`, { token });
}

export function createRoom(
  input: CreateRoomInput,
  token: string,
  idempotencyKey?: string,
) {
  return apiFetch<Room>("/v1/rooms", {
    method: "POST",
    token,
    idempotencyKey,
    body: input,
  });
}

export function deleteRoom(
  roomId: string,
  token: string,
  idempotencyKey?: string,
) {
  return apiFetch<Room>(`/v1/rooms/${roomId}`, {
    method: "DELETE",
    token,
    idempotencyKey,
  });
}

export function updateRoom(
  roomId: string,
  input: UpdateRoomInput,
  token: string,
  idempotencyKey?: string,
) {
  return apiFetch<Room>(`/v1/rooms/${roomId}`, {
    method: "PATCH",
    token,
    idempotencyKey,
    body: input,
  });
}

export function listBranchStaff(branchId: string, token: string) {
  return apiFetch<StaffMember[]>(`/v1/branches/${branchId}/staff`, { token });
}

export function createBranchStaff(
  branchId: string,
  input: CreateStaffInput,
  token: string,
  idempotencyKey?: string,
) {
  return apiFetch<StaffMember>(`/v1/branches/${branchId}/staff`, {
    method: "POST",
    token,
    idempotencyKey,
    body: input,
  });
}

export function updateStaff(
  staffId: string,
  input: UpdateStaffInput,
  token: string,
  idempotencyKey?: string,
) {
  return apiFetch<StaffMember>(`/v1/staff/${staffId}`, {
    method: "PATCH",
    token,
    idempotencyKey,
    body: input,
  });
}

export function deleteStaff(
  staffId: string,
  token: string,
  idempotencyKey?: string,
) {
  return apiFetch<StaffMember>(`/v1/staff/${staffId}`, {
    method: "DELETE",
    token,
    idempotencyKey,
  });
}

export function listTeacherFinanceRecords(
  filters: TeacherFinanceFilters,
  token: string,
) {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(filters)) {
    if (value) {
      params.set(key, value);
    }
  }
  const query = params.toString();
  return apiFetch<TeacherFinanceRecord[]>(
    `/v1/teacher-finance${query ? `?${query}` : ""}`,
    { token },
  );
}

export function listStudentAssignmentHub(
  filters: StudentAssignmentHubFilters,
  token: string,
) {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(filters)) {
    if (value !== undefined && value !== "") {
      params.set(key, String(value));
    }
  }
  const query = params.toString();
  return apiFetch<StudentAssignmentHubPage>(
    `/v1/student-assignment-hub${query ? `?${query}` : ""}`,
    { token },
  );
}

export function listCourses(token: string, branchId?: string) {
  const query = branchId ? `?branch_id=${encodeURIComponent(branchId)}` : "";
  return apiFetch<Course[]>(`/v1/courses${query}`, { token });
}

export function createStudent(
  input: CreateStudentInput,
  token: string,
  idempotencyKey?: string,
) {
  return apiFetch<Student>("/v1/students", {
    method: "POST",
    token,
    idempotencyKey,
    body: input,
  });
}

export function updateStudent(
  studentId: string,
  input: UpdateStudentInput,
  token: string,
  idempotencyKey?: string,
) {
  return apiFetch<Student>(`/v1/students/${studentId}`, {
    method: "PATCH",
    token,
    idempotencyKey,
    body: input,
  });
}

export function createStudentAccount(
  studentId: string,
  input: CreateStudentAccountInput,
  token: string,
  idempotencyKey?: string,
) {
  return apiFetch<{ student: Student; user: unknown }>(
    `/v1/students/${studentId}/account`,
    {
      method: "POST",
      token,
      idempotencyKey,
      body: input,
    },
  );
}

export function deleteTeacher(
  teacherId: string,
  token: string,
  idempotencyKey?: string,
) {
  return apiFetch<Teacher>(`/v1/teachers/${teacherId}`, {
    method: "DELETE",
    token,
    idempotencyKey,
  });
}

export function getTeacher(teacherId: string, token: string) {
  return apiFetch<Teacher>(`/v1/teachers/${teacherId}`, { token });
}

export function createTeacher(
  input: CreateTeacherInput,
  token: string,
  idempotencyKey?: string,
) {
  return apiFetch<CreateTeacherResult>("/v1/teachers", {
    method: "POST",
    token,
    idempotencyKey,
    body: input,
  });
}

export function updateTeacher(
  teacherId: string,
  input: UpdateTeacherInput,
  token: string,
  idempotencyKey?: string,
) {
  return apiFetch<Teacher>(`/v1/teachers/${teacherId}`, {
    method: "PATCH",
    token,
    idempotencyKey,
    body: input,
  });
}

export function listSalaryModels(token: string, branchId?: string) {
  const query = branchId ? `?branch_id=${encodeURIComponent(branchId)}` : "";
  return apiFetch<SalaryModel[]>(`/v1/salary-models${query}`, { token });
}

export function createSalaryModel(
  input: CreateSalaryModelInput,
  token: string,
  idempotencyKey?: string,
) {
  return apiFetch<SalaryModel>("/v1/salary-models", {
    method: "POST",
    token,
    idempotencyKey,
    body: input,
  });
}

export function checkEmailAvailability(email: string, token: string) {
  const params = new URLSearchParams({ email });
  return apiFetch<EmailAvailability>(
    `/v1/users/email-availability?${params.toString()}`,
    { token },
  );
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

export function uploadTeacherPhoto(
  branchId: string,
  teacherId: string,
  photo: File,
  token: string,
) {
  const formData = new FormData();
  formData.set("branch_id", branchId);
  formData.set("owner_type", "teacher");
  formData.set("owner_id", teacherId);
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

export function uploadStudentPhoto(
  branchId: string,
  studentId: string,
  photo: File,
  token: string,
) {
  const formData = new FormData();
  formData.set("branch_id", branchId);
  formData.set("owner_type", "student");
  formData.set("owner_id", studentId);
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

export function deleteFile(
  fileId: string,
  token: string,
  idempotencyKey?: string,
) {
  return apiFetch<FileObject>(`/v1/files/${fileId}`, {
    method: "DELETE",
    token,
    idempotencyKey,
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
