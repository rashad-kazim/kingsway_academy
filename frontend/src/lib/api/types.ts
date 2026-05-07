export type Role = "owner" | "receptionist" | "teacher" | "student";

export type User = {
  id: string;
  branch_id?: string;
  role: Role;
  email: string;
  first_name: string;
  last_name: string;
  is_active: boolean;
  last_login_at?: string;
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

export type UpdateRoomInput = {
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
  gender?: string;
  phone?: string;
  address?: string;
  hired_at?: string;
  salary_amount_azn: number;
  profile_photo_file_id?: string;
  profile_photo_url?: string;
  is_active: boolean;
  last_login_at?: string;
  created_at: string;
  updated_at: string;
};

export type TeacherStatus = "pending_owner_approval" | "active" | "terminated";
export type StudentStatus = "active" | "left" | "graduated";

export type SalaryModelType = "fixed" | "percent" | "hybrid";

export type Teacher = {
  id: string;
  branch_id: string;
  user_id: string;
  status: TeacherStatus;
  birth_date?: string;
  gender?: string;
  phone?: string;
  address?: string;
  profile_photo_file_id?: string;
  created_at: string;
  updated_at: string;
};

export type TeacherFinanceRecord = {
  id: string;
  branch_id: string;
  branch_name: string;
  user_id: string;
  email: string;
  first_name: string;
  last_name: string;
  profile_photo_file_id?: string;
  profile_photo_url?: string;
  subject?: string;
  status: TeacherStatus;
  salary_type?: SalaryModelType;
  assigned_students: number;
  calculated_salary_amount_cents: number;
  created_at: string;
  updated_at: string;
};

export type TeacherFinanceFilters = {
  branch_id?: string;
  subject?: string;
  status?: TeacherStatus;
  salary_model?: SalaryModelType;
};

export type StudentAssignmentHubRecord = {
  id: string;
  branch_id: string;
  branch_name: string;
  user_id?: string;
  fin: string;
  first_name: string;
  last_name: string;
  profile_photo_file_id?: string;
  profile_photo_url?: string;
  status: StudentStatus;
  active_teacher_id?: string;
  active_teacher_first_name?: string;
  active_teacher_last_name?: string;
  registered_at: string;
  created_at: string;
  updated_at: string;
};

export type Course = {
  id: string;
  branch_id: string;
  name: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
};

export type StudentAssignmentHubPage = {
  items: StudentAssignmentHubRecord[];
  total: number;
  limit: number;
  offset: number;
};

export type StudentAssignmentHubFilters = {
  branch_id?: string;
  status?: StudentStatus;
  teacher_id?: string;
  q?: string;
  limit?: number;
  offset?: number;
};

export type CreateTeacherInput = {
  branch_id: string;
  email: string;
  password: string;
  first_name: string;
  last_name: string;
  birth_date?: string;
  gender?: string;
  phone: string;
  address?: string;
  profile_photo_file_id?: string;
  subjects: string[];
  salary_model: SalaryModelType;
  fixed_monthly_amount_cents?: number;
  student_percent_basis_points?: number;
};

export type StudentParentInput = {
  relation: "father" | "mother" | "sister" | "brother" | "other";
  name: string;
  phones: string[];
};

export type StudentCourseAssignInput = {
  course_id: string;
  teacher_id?: string;
  monthly_amount_cents: number;
  start_date: string;
};

export type Student = {
  id: string;
  branch_id: string;
  user_id?: string;
  fin: string;
  first_name: string;
  last_name: string;
  birth_date?: string;
  gender?: string;
  phone?: string;
  address?: string;
  profile_photo_file_id?: string;
  status: StudentStatus;
  left_reason?: string;
  created_at: string;
  updated_at: string;
};

export type CreateStudentInput = {
  branch_id: string;
  fin: string;
  first_name: string;
  last_name: string;
  birth_date?: string;
  gender?: string;
  phone?: string;
  address?: string;
  profile_photo_file_id?: string;
  status?: StudentStatus;
  parents?: StudentParentInput[];
  courses?: StudentCourseAssignInput[];
};

export type CreateStudentAccountInput = {
  email: string;
  password: string;
  first_name?: string;
  last_name?: string;
};

export type UpdateStudentInput = Partial<
  Omit<CreateStudentInput, "branch_id" | "courses" | "parents">
>;

export type UpdateTeacherInput = {
  branch_id?: string;
  email?: string;
  password?: string;
  first_name?: string;
  last_name?: string;
  birth_date?: string;
  gender?: string;
  phone?: string;
  address?: string;
  profile_photo_file_id?: string;
  subjects?: string[];
};

export type SalaryModel = {
  id: string;
  branch_id: string;
  teacher_id: string;
  model_type: SalaryModelType;
  fixed_monthly_amount_cents?: number;
  student_percent_basis_points?: number;
  active_from: string;
  active_to?: string;
  approved_by_owner_user_id: string;
  created_at: string;
};

export type CreateSalaryModelInput = {
  branch_id: string;
  teacher_id: string;
  model_type: SalaryModelType;
  fixed_monthly_amount_cents?: number;
  student_percent_basis_points?: number;
  active_from: string;
};

export type CreateTeacherResult = {
  teacher: Teacher;
  user: User;
  salary_model?: SalaryModel | null;
};

export type EmailAvailability = {
  email: string;
  available: boolean;
};

export type CreateStaffInput = {
  first_name: string;
  last_name: string;
  birth_date?: string;
  gender?: string;
  phone?: string;
  address?: string;
  hired_at?: string;
  salary_amount_azn: number;
  email: string;
  password: string;
  profile_photo_file_id?: string;
};

export type UpdateStaffInput = Omit<CreateStaffInput, "password"> & {
  branch_id?: string;
  is_active?: boolean;
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
