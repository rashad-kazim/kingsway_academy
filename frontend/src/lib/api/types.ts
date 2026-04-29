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
  created_at: string;
  updated_at: string;
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
