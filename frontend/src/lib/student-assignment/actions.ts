"use server";

import { redirect } from "next/navigation";
import {
  ApiError,
  checkEmailAvailability,
  createStudent,
  createStudentAccount,
  getFileDownloadURL,
  listStudentAssignmentHub,
  updateStudent,
  uploadStudentPhoto,
} from "@/lib/api/client";
import type {
  StudentAssignmentHubFilters,
  StudentAssignmentHubPage,
  StudentAssignmentHubRecord,
  StudentParentInput,
} from "@/lib/api/types";
import { getAuthToken } from "@/lib/auth/session";

export type StudentAssignmentHubActionResult =
  | { data: StudentAssignmentHubPage }
  | { error: "unauthorized" | "backend" };

export async function fetchStudentAssignmentHubAction(
  filters: StudentAssignmentHubFilters,
): Promise<StudentAssignmentHubActionResult> {
  const token = await getAuthToken();
  if (!token) {
    return { error: "unauthorized" };
  }

  try {
    const data = await listStudentAssignmentHub(filters, token);
    return { data: await attachStudentHubPhotos(data, token) };
  } catch {
    return { error: "backend" };
  }
}

export type CreateStudentState = {
  error?:
    | "invalid"
    | "duplicate_fin"
    | "duplicate_email"
    | "invalid_photo"
    | "photo_too_large"
    | "unauthorized"
    | "backend";
};

export async function createStudentAction(
  _previous: CreateStudentState,
  formData: FormData,
): Promise<CreateStudentState> {
  const token = await getAuthToken();
  if (!token) {
    return { error: "unauthorized" };
  }

  const locale = stringField(formData, "locale") || "en";
  const idempotencyKey = stringField(formData, "idempotency_key") || undefined;
  const firstName = stringField(formData, "first_name");
  const lastName = stringField(formData, "last_name");
  const branchID = stringField(formData, "branch_id");
  const fin = stringField(formData, "fin").toUpperCase();
  const birthDate = stringField(formData, "birth_date");
  const email = stringField(formData, "email");
  const password = stringField(formData, "password");
  const courseIDs = formData.getAll("course_ids").map(String).filter(Boolean);
  const startDate = stringField(formData, "start_date");
  const parents = parseParents(formData);
  const courseAssignments = courseIDs.map((courseID) => ({
    course_id: courseID,
    teacher_id: stringField(formData, `teacher_${courseID}`),
    monthly_amount_cents: manatToCents(stringField(formData, `amount_${courseID}`)),
    start_date: startDate,
  }));

  if (
    !firstName ||
    !lastName ||
    !branchID ||
    !/^[A-Z0-9]{7}$/.test(fin) ||
    !validShortDate(birthDate) ||
    !email ||
    password.length < 8 ||
    courseAssignments.length === 0 ||
    !validShortDate(startDate) ||
    courseAssignments.some((item) => item.monthly_amount_cents <= 0)
  ) {
    return { error: "invalid" };
  }

  const photo = formData.get("profile_photo");
  if (photo instanceof File && photo.size > 0) {
    if (photo.size > 10 * 1024 * 1024) {
      return { error: "photo_too_large" };
    }
    if (
      photo.type &&
      !["image/jpeg", "image/png", "image/webp"].includes(photo.type)
    ) {
      return { error: "invalid_photo" };
    }
  }

  let stage: "student" | "photo" | "account" = "student";
  try {
    const student = await createStudent(
      {
        address: stringField(formData, "address"),
        birth_date: birthDate,
        branch_id: branchID,
        courses: courseAssignments,
        fin,
        first_name: firstName,
        gender: stringField(formData, "gender"),
        last_name: lastName,
        parents,
        phone: stringField(formData, "phone"),
        status: "active",
      },
      token,
      idempotencyKey,
    );

    stage = "photo";
    if (photo instanceof File && photo.size > 0) {
      const uploaded = await uploadStudentPhoto(branchID, student.id, photo, token);
      await updateStudent(
        student.id,
        {
          address: student.address,
          birth_date: student.birth_date,
          first_name: student.first_name,
          gender: student.gender,
          last_name: student.last_name,
          phone: student.phone,
          profile_photo_file_id: uploaded.id,
          status: student.status,
        },
        token,
      );
    }

    stage = "account";
    await createStudentAccount(
      student.id,
      {
        email,
        first_name: firstName,
        last_name: lastName,
        password,
      },
      token,
      idempotencyKey ? `${idempotencyKey}:account` : undefined,
    );
  } catch (error) {
    if (error instanceof ApiError) {
      if (error.status === 409) {
        return { error: stage === "account" ? "duplicate_email" : "duplicate_fin" };
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

  redirect(`/${locale}/dashboard/owner?view=student-assignment`);
}

export async function checkStudentEmailAvailabilityAction(email: string) {
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

function parseParents(formData: FormData): StudentParentInput[] {
  const indexes = formData.getAll("parent_indexes").map(String);
  return indexes
    .map((index) => ({
      name: stringField(formData, `parent_name_${index}`),
      phones: formData
        .getAll(`parent_phone_${index}`)
        .map((value) => String(value).trim())
        .filter(Boolean),
      relation: stringField(formData, `parent_relation_${index}`) as StudentParentInput["relation"],
    }))
    .filter((parent) => parent.name || parent.phones.length > 0);
}

function manatToCents(value: string) {
  const normalized = value.replace(",", ".").replace(/[^\d.]/g, "");
  const amount = Number.parseFloat(normalized);
  if (!Number.isFinite(amount)) {
    return 0;
  }

  return Math.round(amount * 100);
}

function validShortDate(value: string) {
  return /^\d{2}\/\d{2}\/\d{4}$/.test(value);
}

async function attachStudentHubPhotos(
  page: StudentAssignmentHubPage,
  token: string,
): Promise<StudentAssignmentHubPage> {
  const items = await Promise.all(
    page.items.map((student) => attachStudentHubPhoto(student, token)),
  );
  return { ...page, items };
}

async function attachStudentHubPhoto(
  student: StudentAssignmentHubRecord,
  token: string,
): Promise<StudentAssignmentHubRecord> {
  if (!student.profile_photo_file_id) {
    return student;
  }
  try {
    const download = await getFileDownloadURL(student.profile_photo_file_id, token);
    return { ...student, profile_photo_url: download.url };
  } catch {
    return student;
  }
}
