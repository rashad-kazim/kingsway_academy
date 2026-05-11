"use server";

import { redirect } from "next/navigation";
import {
  ApiError,
  checkEmailAvailability,
  createSalaryModel,
  createTeacher,
  deleteTeacher,
  updateTeacher,
  uploadTeacherPhoto,
} from "@/lib/api/client";
import type { SalaryModelType, Teacher } from "@/lib/api/types";
import { getAuthToken } from "@/lib/auth/session";

export type DeleteTeacherState = {
  teacher?: Teacher;
  error?: "assigned_students" | "not_found" | "unauthorized" | "backend";
};

export type CreateTeacherState = {
  error?:
    | "invalid"
    | "duplicate_email"
    | "invalid_photo"
    | "photo_too_large"
    | "unauthorized"
    | "backend";
};

export type UpdateTeacherState = CreateTeacherState;

export async function deleteTeacherAction(
  _previous: DeleteTeacherState,
  formData: FormData,
): Promise<DeleteTeacherState> {
  const teacherID = String(formData.get("teacher_id") ?? "").trim();
  if (!teacherID) {
    return { error: "backend" };
  }

  const token = await getAuthToken();
  if (!token) {
    return { error: "unauthorized" };
  }

  try {
    const teacher = await deleteTeacher(teacherID, token);
    return { teacher };
  } catch (error) {
    if (error instanceof ApiError) {
      if (error.status === 409) {
        return { error: "assigned_students" };
      }
      if (error.status === 404) {
        return { error: "not_found" };
      }
      if (error.status === 401 || error.status === 403) {
        return { error: "unauthorized" };
      }
    }

    return { error: "backend" };
  }
}

export async function createTeacherAction(
  _previous: CreateTeacherState,
  formData: FormData,
): Promise<CreateTeacherState> {
  const token = await getAuthToken();
  if (!token) {
    return { error: "unauthorized" };
  }

  const idempotencyKey = stringField(formData, "idempotency_key") || undefined;
  const locale = stringField(formData, "locale") || "en";
  const salaryModel = stringField(formData, "salary_model") as SalaryModelType;
  const salaryAmount = manatToCents(stringField(formData, "salary_amount"));
  const percentage = percentToBasisPoints(stringField(formData, "percentage"));
  const subjects = formData
    .getAll("subjects")
    .map((value) => String(value).trim())
    .filter(Boolean);

  if (
    !stringField(formData, "branch_id") ||
    !stringField(formData, "first_name") ||
    !stringField(formData, "last_name") ||
    !stringField(formData, "birth_date") ||
    !stringField(formData, "phone") ||
    !stringField(formData, "email") ||
    !stringField(formData, "password") ||
    subjects.length === 0 ||
    !salaryModel ||
    ((salaryModel === "fixed" || salaryModel === "hybrid") &&
      salaryAmount <= 0) ||
    ((salaryModel === "percent" || salaryModel === "hybrid") &&
      percentage < 0)
  ) {
    return { error: "invalid" };
  }

  const photo = formData.get("profile_photo");
  if (photo instanceof File && photo.size > 0) {
    if (photo.size > 15 * 1024 * 1024) {
      return { error: "photo_too_large" };
    }
    if (
      photo.type &&
      !["image/jpeg", "image/png", "image/webp"].includes(photo.type)
    ) {
      return { error: "invalid_photo" };
    }
  }

  try {
    const result = await createTeacher(
      {
        branch_id: stringField(formData, "branch_id"),
        first_name: stringField(formData, "first_name"),
        last_name: stringField(formData, "last_name"),
        birth_date: stringField(formData, "birth_date"),
        gender: stringField(formData, "gender"),
        phone: stringField(formData, "phone"),
        address: stringField(formData, "address"),
        email: stringField(formData, "email"),
        password: stringField(formData, "password"),
        subjects,
        salary_model: salaryModel,
        fixed_monthly_amount_cents:
          salaryModel === "fixed" || salaryModel === "hybrid"
            ? salaryAmount
            : 0,
        student_percent_basis_points:
          salaryModel === "percent" || salaryModel === "hybrid"
            ? percentage
            : 0,
      },
      token,
      idempotencyKey,
    );

    if (photo instanceof File && photo.size > 0) {
      const uploaded = await uploadTeacherPhoto(
        result.teacher.branch_id,
        result.teacher.id,
        photo,
        token,
      );
      await updateTeacher(
        result.teacher.id,
        {
          birth_date: result.teacher.birth_date,
          gender: result.teacher.gender,
          phone: result.teacher.phone,
          address: result.teacher.address,
          profile_photo_file_id: uploaded.id,
          subjects,
        },
        token,
      );
    }
  } catch (error) {
    if (error instanceof ApiError) {
      if (error.status === 409) {
        return { error: "duplicate_email" };
      }
      if (error.status === 401 || error.status === 403) {
        return { error: "unauthorized" };
      }
      if (error.status === 400) {
        return { error: "invalid" };
      }
    }

    return { error: "backend" };
  }

  redirect(`/${locale}/dashboard/owner?view=teacher-finance`);
}

export async function updateTeacherAction(
  _previous: UpdateTeacherState,
  formData: FormData,
): Promise<UpdateTeacherState> {
  const token = await getAuthToken();
  if (!token) {
    return { error: "unauthorized" };
  }

  const idempotencyKey = stringField(formData, "idempotency_key") || undefined;
  const locale = stringField(formData, "locale") || "en";
  const teacherID = stringField(formData, "teacher_id");
  const existingPhotoFileID = stringField(formData, "profile_photo_file_id");
  const removePhoto = stringField(formData, "remove_photo") === "1";
  const salaryModel = stringField(formData, "salary_model") as SalaryModelType;
  const salaryAmount = manatToCents(stringField(formData, "salary_amount"));
  const percentage = percentToBasisPoints(stringField(formData, "percentage"));
  const subjects = formData
    .getAll("subjects")
    .map((value) => String(value).trim())
    .filter(Boolean);

  if (
    !teacherID ||
    !stringField(formData, "branch_id") ||
    !stringField(formData, "first_name") ||
    !stringField(formData, "last_name") ||
    !stringField(formData, "birth_date") ||
    !stringField(formData, "phone") ||
    !stringField(formData, "email") ||
    subjects.length === 0 ||
    !salaryModel ||
    ((salaryModel === "fixed" || salaryModel === "hybrid") &&
      salaryAmount <= 0) ||
    ((salaryModel === "percent" || salaryModel === "hybrid") &&
      percentage < 0)
  ) {
    return { error: "invalid" };
  }

  const password = stringField(formData, "password");
  if (password && password.length < 8) {
    return { error: "invalid" };
  }

  const photo = formData.get("profile_photo");
  if (photo instanceof File && photo.size > 0) {
    if (photo.size > 15 * 1024 * 1024) {
      return { error: "photo_too_large" };
    }
    if (
      photo.type &&
      !["image/jpeg", "image/png", "image/webp"].includes(photo.type)
    ) {
      return { error: "invalid_photo" };
    }
  }

  try {
    let profilePhotoFileID = removePhoto ? "" : existingPhotoFileID;
    if (photo instanceof File && photo.size > 0) {
      const uploaded = await uploadTeacherPhoto(
        stringField(formData, "branch_id"),
        teacherID,
        photo,
        token,
      );
      profilePhotoFileID = uploaded.id;
    }

    await updateTeacher(
      teacherID,
      {
        address: stringField(formData, "address"),
        birth_date: stringField(formData, "birth_date"),
        branch_id: stringField(formData, "branch_id"),
        email: stringField(formData, "email"),
        first_name: stringField(formData, "first_name"),
        gender: stringField(formData, "gender"),
        last_name: stringField(formData, "last_name"),
        password,
        phone: stringField(formData, "phone"),
        profile_photo_file_id: profilePhotoFileID,
        subjects,
      },
      token,
    );

    await createSalaryModel(
      {
        active_from: new Date().toISOString(),
        branch_id: stringField(formData, "branch_id"),
        fixed_monthly_amount_cents:
          salaryModel === "fixed" || salaryModel === "hybrid"
            ? salaryAmount
            : 0,
        model_type: salaryModel,
        student_percent_basis_points:
          salaryModel === "percent" || salaryModel === "hybrid"
            ? percentage
            : 0,
        teacher_id: teacherID,
      },
      token,
      idempotencyKey ? `${idempotencyKey}:salary` : undefined,
    );
  } catch (error) {
    if (error instanceof ApiError) {
      if (error.status === 409) {
        return { error: "duplicate_email" };
      }
      if (error.status === 401 || error.status === 403) {
        return { error: "unauthorized" };
      }
      if (error.status === 400) {
        return { error: "invalid" };
      }
    }

    return { error: "backend" };
  }

  redirect(`/${locale}/dashboard/owner?view=teacher-finance`);
}

export async function checkTeacherEmailAvailabilityAction(email: string) {
  const token = await getAuthToken();
  if (!token) {
    return { available: false, error: "unauthorized" as const };
  }
  try {
    return await checkEmailAvailability(email, token);
  } catch {
    return { available: false, error: "backend" as const };
  }
}

function stringField(formData: FormData, key: string) {
  return String(formData.get(key) ?? "").trim();
}

function manatToCents(value: string) {
  const normalized = value.replace(",", ".").replace(/[^\d.]/g, "");
  const amount = Number.parseFloat(normalized);
  if (!Number.isFinite(amount)) {
    return 0;
  }

  return Math.round(amount * 100);
}

function percentToBasisPoints(value: string) {
  const normalized = value.replace(",", ".").replace(/[^\d.]/g, "");
  const amount = Number.parseFloat(normalized);
  if (!Number.isFinite(amount)) {
    return -1;
  }

  return Math.round(amount * 100);
}
