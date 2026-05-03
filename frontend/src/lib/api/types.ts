export type Role = "owner" | "receptionist" | "teacher" | "student";

export type User = {
  id: string;
  branch_id?: string;
  role: Role;
  email: string;
  first_name: string;
  last_name: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
};

export type Principal = {
  user_id: string;
  branch_id?: string;
  role: Role;
};

export type Branch = {
  id: string;
  name: string;
  slug: string;
  address?: string;
  opening_time?: string;
  closing_time?: string;
  photo_url?: string;
  photo_file_id?: string;
  created_at: string;
  updated_at: string;
};

export type CreateBranchInput = {
  name: string;
  slug: string;
  address?: string;
  opening_time?: string;
  closing_time?: string;
};

export type UpdateBranchInput = CreateBranchInput;

export type Room = {
  id: string;
  branch_id: string;
  name: string;
  capacity: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
};

export type CreateRoomInput = {
  branch_id: string;
  name: string;
  capacity: number;
};

export type StaffMember = {
  id: string;
  branch_id: string;
  user_id: string;
  role: Role;
  email: string;
  first_name: string;
  last_name: string;
  birth_date?: string;
  phone?: string;
  salary_amount_azn: number;
  profile_photo_file_id?: string;
  profile_photo_url?: string;
  created_at: string;
  updated_at: string;
};

export type CreateStaffInput = {
  first_name: string;
  last_name: string;
  birth_date?: string;
  phone?: string;
  salary_amount_azn: number;
  email: string;
  password: string;
  profile_photo_file_id?: string;
};

export type UpdateStaffInput = Omit<CreateStaffInput, "password"> & {
  password?: string;
};

export type FileObject = {
  id: string;
  branch_id: string;
  uploader_user_id: string;
  owner_type: string;
  owner_id: string;
  category: "standard" | "special";
  purpose: string;
  original_filename: string;
  mime_type: string;
  original_size_bytes: number;
  stored_size_bytes: number;
  original_sha256: string;
  storage_bucket: string;
  storage_key: string;
  retention_until?: string;
  deleted_at?: string;
  created_at: string;
};

export type DownloadURLResult = {
  url: string;
  expires_at: string;
  file: FileObject;
};

export type Session = {
  user: User;
  principal: Principal;
  branch?: Branch;
  capabilities: string[];
  dashboard_path: string;
};

export type LoginResult = {
  token: string;
  user: User;
};

export type AcademicSummary = {
  branch_id?: string;
  active_students: number;
  active_teachers: number;
  active_classes: number;
  upcoming_schedule: number;
  pending_payments: number;
  writing_files_retained: number;
};

export type DashboardRecord = {
  role: Role;
  branch_id?: string;
  summary?: AcademicSummary;
  branches?: Branch[];
  teacher?: Record<string, unknown>;
  student?: Record<string, unknown>;
  classes?: Record<string, unknown>[];
  schedule?: Record<string, unknown>[];
  assignments?: Record<string, unknown>[];
  payments?: Record<string, unknown>[];
  exams?: Record<string, unknown>[];
  results?: Record<string, unknown>[];
};
