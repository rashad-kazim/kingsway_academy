import type { Role } from "@/lib/api/types";

export const roles: Role[] = ["owner", "receptionist", "teacher", "student"];

export function isRole(value: string): value is Role {
  return roles.includes(value as Role);
}

export function dashboardPathForRole(role: Role) {
  return `/dashboard/${role}`;
}

export function roleLabel(role: Role) {
  switch (role) {
    case "owner":
      return "Owner";
    case "receptionist":
      return "Receptionist";
    case "teacher":
      return "Teacher";
    case "student":
      return "Student";
  }
}
