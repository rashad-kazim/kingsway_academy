import { promises as fs } from "node:fs";
import {
  expect,
  type APIRequestContext,
  type Locator,
  type Page,
  type TestInfo,
  test,
} from "@playwright/test";

const appBaseURL = process.env.KINGSWAY_E2E_BASE_URL ?? "http://127.0.0.1:3000";
const appOrigin = new URL(appBaseURL).origin;
const backendBaseURL =
  process.env.KINGSWAY_E2E_BACKEND_URL ?? "http://127.0.0.1:8080";
const ownerEmail = process.env.KINGSWAY_E2E_OWNER_EMAIL ?? "owner@kingsway.local";
const ownerPassword = process.env.KINGSWAY_E2E_OWNER_PASSWORD ?? "Kingsway123!";

type ApiBranch = {
  id: string;
  name: string;
};

type ApiCourse = {
  id: string;
  branch_id: string;
  name: string;
  is_active: boolean;
};

type ApiTeacherFinanceRecord = {
  id: string;
  branch_id: string;
  first_name: string;
  last_name: string;
};

test.describe.configure({ mode: "serial" });

test.describe("Owner regression smoke flows", () => {
  test("login/logout, add/edit/delete flows, filters, uploads, and idempotency stay functional", async ({
    page,
    request,
  }, testInfo) => {
    const runtimeErrors: string[] = [];
    page.on("console", (message) => {
      if (message.type() === "error") {
        runtimeErrors.push(message.text());
      }
    });
    page.on("pageerror", (error) => runtimeErrors.push(error.message));

    const runID = Date.now().toString(36);
    const photoPath = await createTestPNG(testInfo);
    const token = await backendLogin(request);

    const branchName = `E2E Branch ${runID}`;
    const branchAddress = `E2E Address ${runID}`;
    const editedBranchAddress = `E2E Updated Address ${runID}`;
    const roomName = `E2E Room ${runID}`;
    const editedRoomName = `E2E Room Edited ${runID}`;
    const teacherFirstName = `E2E${runID}`;
    const teacherLastName = "Teacher";
    const teacherFullName = `${teacherFirstName} ${teacherLastName}`;
    const teacherEmail = `e2e.teacher.${runID}@kingsway.local`;
    const deleteTeacherFirstName = `Del${runID}`;
    const deleteTeacherLastName = "Teacher";
    const deleteTeacherFullName = `${deleteTeacherFirstName} ${deleteTeacherLastName}`;
    const deleteTeacherEmail = `e2e.teacher.delete.${runID}@kingsway.local`;
    const receptionistFirstName = `E2E${runID}`;
    const receptionistLastName = "Reception";
    const receptionistFullName = `${receptionistFirstName} ${receptionistLastName}`;
    const receptionistEmail = `e2e.reception.${runID}@kingsway.local`;
    const studentFirstName = `E2E${runID}`;
    const studentLastName = "Student";
    const studentFullName = `${studentFirstName} ${studentLastName}`;
    const studentEmail = `e2e.student.${runID}@kingsway.local`;
    const studentFIN = makeFIN(runID, 0);
    let branchID = "";

    await loginAsOwner(page);
    await logoutFromSidebar(page);
    await loginAsOwner(page);

    try {
      await createBranch(page, branchName, branchAddress, photoPath, true);
      const branch = await findBranchByName(request, token, branchName);
      branchID = branch.id;

      await editBranchPhotoRoomsAndAddress(page, {
        branchName,
        editedBranchAddress,
        editedRoomName,
        photoPath,
        roomName,
      });

      await createReceptionist(page, {
        branchName,
        email: receptionistEmail,
        firstName: receptionistFirstName,
        lastName: receptionistLastName,
        photoPath,
      });
      await editReceptionist(page, receptionistFullName);
      await deleteReceptionist(page, receptionistFullName);

      await createTeacher(page, {
        branchName,
        email: teacherEmail,
        firstName: teacherFirstName,
        lastName: teacherLastName,
        photoPath,
      });
      await editTeacher(page, teacherFullName, photoPath);
      await assertTeacherPhotoVisible(page, teacherFullName);

      await createTeacher(page, {
        branchName,
        email: deleteTeacherEmail,
        firstName: deleteTeacherFirstName,
        lastName: deleteTeacherLastName,
        photoPath,
      });
      await deleteTeacherFullFlow(page, deleteTeacherFullName);

      const teacherRecord = await findTeacherByName(request, token, teacherFullName);
      const course = await ensureCourse(request, token, branchID, "IELTS");
      await seedStudentsForPagination(request, token, {
        branchID,
        courseID: course.id,
        runID,
        teacherID: teacherRecord.id,
      });

      await createStudent(page, {
        branchName,
        email: studentEmail,
        fin: studentFIN,
        firstName: studentFirstName,
        lastName: studentLastName,
        photoPath,
        teacherFullName,
      });
      await exerciseStudentHubFiltersAndPagination(page, {
        branchName,
        studentFullName,
        teacherFullName,
      });

      await exerciseTeacherFilters(page, branchName);
      await exerciseAPIIdempotencyAndConcurrency(request, token, runID);

      await deleteBranchWithPopupEdgeCases(page, branchName);
      branchID = "";

      expect(runtimeErrors).toEqual([]);
    } finally {
      if (branchID) {
        await apiDelete(request, token, `/v1/branches/${branchID}`).catch(() => undefined);
      }
    }
  });
});

async function loginAsOwner(page: Page) {
  await page.goto("/en/login");
  await expect(page.getByRole("heading", { name: "Login" })).toBeVisible();
  await page.getByLabel("Email").fill(ownerEmail);
  await page.getByRole("textbox", { name: "Password" }).fill(ownerPassword);
  await page.getByRole("button", { name: "Login" }).click();
  await expect(page).toHaveURL(/\/en\/dashboard\/owner/);
  await expect(page.getByText("Command Center")).toBeVisible();
}

async function logoutFromSidebar(page: Page) {
  await page.getByLabel("Profile menu").last().click();
  await page.getByRole("menuitem", { name: "Sign out" }).click();
  await expect(page).toHaveURL(/\/en\/login/);
}

async function createBranch(
  page: Page,
  branchName: string,
  address: string,
  photoPath: string,
  hammerSubmit: boolean,
) {
  await page.goto("/en/dashboard/owner?view=branches");
  await expect(page.getByRole("heading", { name: "Branch Management" })).toBeVisible();
  await page.getByRole("button", { name: "Add New Branch" }).click();

  const dialog = page.getByRole("dialog");
  await expect(dialog.getByRole("heading", { name: "Add New Branch" })).toBeVisible();
  await dialog.locator('input[name="photo"]').setInputFiles(photoPath);
  await dialog.locator('input[name="name"]').fill(branchName);
  await dialog.locator('textarea[name="address"]').fill(address);

  if (hammerSubmit) {
    await hammerSubmitAndExpectSinglePost(
      page,
      dialog.getByRole("button", { name: "Save Branch" }),
    );
  } else {
    await submitAndExpectPost(page, dialog.getByRole("button", { name: "Save Branch" }));
  }
  const row = page.getByRole("row").filter({ hasText: branchName });
  await expect(row).toBeVisible();
  await expect(row.locator("img")).toHaveCount(1);
}

async function editBranchPhotoRoomsAndAddress(
  page: Page,
  input: {
    branchName: string;
    editedBranchAddress: string;
    editedRoomName: string;
    photoPath: string;
    roomName: string;
  },
) {
  await openBranchDetail(page, input.branchName);
  await page.locator('input[name="photo"]').setInputFiles(input.photoPath);
  await page.locator('textarea[name="address"]').fill(input.editedBranchAddress);
  await addRoomDraft(page, input.roomName, "12");
  await submitAndExpectPost(page, page.getByRole("button", { name: "Save" }));
  await expect(page.getByRole("row").filter({ hasText: input.branchName })).toContainText("1 Rooms");

  await openBranchDetail(page, input.branchName);
  const roomRow = page.getByTestId("room-row").filter({ hasText: input.roomName });
  await roomRow.getByRole("button", { name: "Edit" }).click();
  const roomEditor = page.getByTestId("room-editor");
  await roomEditor.getByPlaceholder("Room name, for example Room 101").fill(input.editedRoomName);
  await roomEditor.getByRole("button", { name: "Save" }).click();
  await submitAndExpectPost(page, page.getByRole("button", { name: "Save" }).first());
  await expect(page.getByRole("row").filter({ hasText: input.branchName })).toContainText("1 Rooms");

  await openBranchDetail(page, input.branchName);
  const editedRoomRow = page.getByTestId("room-row").filter({ hasText: input.editedRoomName });
  await editedRoomRow.getByRole("button", { name: "Remove room" }).click();
  const deleteDialog = page.getByRole("dialog");
  await expect(deleteDialog.getByText("Schedule, Events, and Tasks")).toBeVisible();
  await deleteDialog.getByRole("button", { name: "Cancel" }).click();
  await expect(editedRoomRow).toBeVisible();
  await editedRoomRow.getByRole("button", { name: "Remove room" }).click();
  await page.getByRole("dialog").getByRole("button", { name: "Delete" }).click();
  await submitAndExpectPost(page, page.getByRole("button", { name: "Save" }));
  await expect(page.getByRole("row").filter({ hasText: input.branchName })).toContainText("0 Rooms");
}

async function openBranchDetail(page: Page, branchName: string) {
  await page.goto("/en/dashboard/owner?view=branches");
  const row = page.getByRole("row").filter({ hasText: branchName });
  await expect(row).toBeVisible();
  await row.click();
  await expect(page.getByRole("heading", { name: "Branch Detail" })).toBeVisible();
}

async function addRoomDraft(page: Page, roomName: string, capacity: string) {
  await page.getByRole("button", { name: "Add Room" }).click();
  const roomEditor = page.getByTestId("room-editor");
  await roomEditor.getByPlaceholder("Room name, for example Room 101").fill(roomName);
  await roomEditor.getByLabel("Capacity").fill(capacity);
  await roomEditor.getByRole("button", { name: "Save" }).click();
  await expect(page.getByTestId("room-row").filter({ hasText: roomName })).toBeVisible();
}

async function createTeacher(
  page: Page,
  input: {
    branchName: string;
    email: string;
    firstName: string;
    lastName: string;
    photoPath: string;
  },
) {
  await page.goto("/en/dashboard/owner?view=teacher-add");
  await expect(page.getByRole("heading", { name: "Add Teacher" })).toBeVisible();
  await page.locator('input[name="profile_photo"]').setInputFiles(input.photoPath);
  await page.locator('input[name="first_name"]').fill(input.firstName);
  await page.locator('input[name="last_name"]').fill(input.lastName);
  await page.locator('input[name="birth_date"]').fill("15/07/1998");
  await page.locator('input[name="phone"]').fill("+994501112233");
  await page.locator('textarea[name="address"]').fill("E2E teacher address");
  await page.locator('select[name="branch_id"]').selectOption({ label: input.branchName });
  await selectCheckboxOption(page, "Select subjects", "IELTS");
  await page.locator('input[name="salary_amount"]').fill("100");
  await page.locator('input[name="email"]').fill(input.email);
  await expect(page.getByText("Email is available.")).toBeVisible();
  await page.locator('input[name="password"]').fill("Kingsway123!");

  await submitAndExpectPost(page, page.getByRole("button", { name: "Save" }));
  await expect(page).toHaveURL(/view=teacher-finance/);
  await expect(page.getByText(`${input.firstName} ${input.lastName}`)).toBeVisible();
}

async function editTeacher(page: Page, teacherFullName: string, photoPath: string) {
  await page.goto("/en/dashboard/owner?view=teacher-finance");
  const row = page.getByRole("row").filter({ hasText: teacherFullName });
  await expect(row).toBeVisible();
  await row.getByRole("link", { name: "Edit" }).click();
  await expect(page.getByRole("heading", { name: "Edit" })).toBeVisible();
  await page.locator('input[name="profile_photo"]').setInputFiles(photoPath);
  await page.locator('input[name="salary_amount"]').fill("101");

  await submitAndExpectPost(page, page.getByRole("button", { name: "Save" }));
  await expect(page).toHaveURL(/view=teacher-finance/);
  await expect(page.getByText(teacherFullName)).toBeVisible();
}

async function assertTeacherPhotoVisible(page: Page, teacherFullName: string) {
  await page.goto("/en/dashboard/owner?view=teacher-finance");
  const row = page.getByRole("row").filter({ hasText: teacherFullName });
  await expect(row).toBeVisible();
  await expect(row.locator("img")).toHaveCount(1);
}

async function deleteTeacherFullFlow(page: Page, teacherFullName: string) {
  await page.goto("/en/dashboard/owner?view=teacher-finance");
  const row = page.getByRole("row").filter({ hasText: teacherFullName });
  await expect(row).toBeVisible();
  await row.getByRole("button", { name: "Delete" }).click();

  const dialog = page.getByRole("dialog");
  await dialog.getByLabel("Teacher full name").fill(`${teacherFullName}x`);
  await expect(dialog.getByRole("button", { name: "Delete" })).toBeDisabled();
  await dialog.getByLabel("Teacher full name").fill(teacherFullName);
  await dialog.getByRole("button", { name: "Delete" }).click();
  await expect(dialog.getByText("This deletion is permanent")).toBeVisible();
  await dialog.getByRole("button", { name: "Cancel" }).click();
  await expect(row).toBeVisible();

  await row.getByRole("button", { name: "Delete" }).click();
  await page.getByRole("dialog").getByLabel("Teacher full name").fill(teacherFullName);
  await page.getByRole("dialog").getByRole("button", { name: "Delete" }).click();
  await submitAndExpectPost(
    page,
    page.getByRole("dialog").getByRole("button", { name: "Continue" }),
  );
  await expect(page.getByRole("row").filter({ hasText: teacherFullName })).toHaveCount(0);
}

async function createReceptionist(
  page: Page,
  input: {
    branchName: string;
    email: string;
    firstName: string;
    lastName: string;
    photoPath: string;
  },
) {
  await page.goto("/en/dashboard/owner?view=receptionists");
  await expect(page.getByRole("heading", { name: "Receptionist Management" })).toBeVisible();
  await page.getByRole("button", { name: "Add Staff" }).click();
  const form = page.getByTestId("staff-editor-create");
  await form.locator('input[name="photo"]').setInputFiles(input.photoPath);
  await form.getByTestId("staff-first-name").fill(input.firstName);
  await form.getByTestId("staff-last-name").fill(input.lastName);
  await form.getByTestId("staff-branch-id").selectOption({ label: input.branchName });
  await form.getByTestId("staff-birth-date").fill("15/07/1998");
  await form.getByTestId("staff-gender").selectOption("female");
  await form.getByTestId("staff-phone").fill("+994701112233");
  await form.getByTestId("staff-address").fill("E2E receptionist address");
  await form.getByTestId("staff-salary").fill("650");
  await form.getByTestId("staff-hired-at").fill("01/05/2026");
  await form.getByTestId("staff-email").fill(input.email);
  await form.getByTestId("staff-password").fill("Kingsway123!");
  await expect(form.getByTestId("staff-save")).toBeEnabled();
  await submitAndExpectPost(page, form.getByTestId("staff-save"));
  await expect(page.getByText(`${input.firstName} ${input.lastName}`)).toBeVisible();
}

async function editReceptionist(page: Page, receptionistFullName: string) {
  await page.goto("/en/dashboard/owner?view=receptionists");
  const row = page.getByText(receptionistFullName).locator("xpath=ancestor::*[@data-testid='receptionist-row'][1]");
  await row.getByRole("button", { name: "Edit" }).click();
  const form = page.getByTestId("staff-editor-edit");
  await form.getByTestId("staff-salary").fill("777");
  await form.getByTestId("staff-status").selectOption("inactive");
  await submitAndExpectPost(page, form.getByTestId("staff-save"));
  await expect(page.getByText(receptionistFullName)).toBeVisible();
  await expect(page.getByText("Inactive")).toBeVisible();
  await expect(page.getByText("777")).toBeVisible();
}

async function deleteReceptionist(page: Page, receptionistFullName: string) {
  await page.goto("/en/dashboard/owner?view=receptionists");
  const row = page.getByText(receptionistFullName).locator("xpath=ancestor::*[@data-testid='receptionist-row'][1]");
  await row.getByRole("button", { name: "Delete" }).click();
  const dialog = page.getByRole("dialog");
  await dialog.getByLabel("Receptionist full name").fill(`${receptionistFullName}x`);
  await expect(dialog.getByRole("button", { name: "Delete" })).toBeDisabled();
  await dialog.getByLabel("Receptionist full name").fill(receptionistFullName);
  await dialog.getByRole("button", { name: "Delete" }).click();
  await expect(dialog.getByText("This deletion is permanent")).toBeVisible();
  await dialog.getByRole("button", { name: "Cancel" }).click();
  await expect(page.getByText(receptionistFullName)).toBeVisible();

  await row.getByRole("button", { name: "Delete" }).click();
  await page.getByRole("dialog").getByLabel("Receptionist full name").fill(receptionistFullName);
  await page.getByRole("dialog").getByRole("button", { name: "Delete" }).click();
  await submitAndExpectPost(
    page,
    page.getByRole("dialog").getByRole("button", { name: "Continue" }),
  );
  await expect(page.getByText(receptionistFullName)).toHaveCount(0);
}

async function createStudent(
  page: Page,
  input: {
    branchName: string;
    email: string;
    fin: string;
    firstName: string;
    lastName: string;
    photoPath: string;
    teacherFullName: string;
  },
) {
  await page.goto("/en/dashboard/owner?view=student-add");
  await expect(page.getByRole("heading", { name: "Add Student" })).toBeVisible();
  await page.locator('input[name="profile_photo"]').setInputFiles(input.photoPath);
  await page.locator('input[name="first_name"]').fill(input.firstName);
  await page.locator('input[name="last_name"]').fill(input.lastName);
  await page.locator('input[name="fin"]').fill(input.fin);
  await page.locator('input[name="birth_date"]').fill("15/07/2012");
  await page.locator('select[name="gender"]').selectOption("male");
  await page.locator('textarea[name="address"]').fill("E2E student address");
  await page.locator('input[name="phone"]').fill("+994551112233");
  await page.locator('select[name="branch_id"]').selectOption({ label: input.branchName });
  await selectCheckboxOption(page, "Select courses", "IELTS");

  const teacherSelect = page.locator('select[name^="teacher_"]').first();
  if (await teacherSelect.count()) {
    await teacherSelect.selectOption({ label: input.teacherFullName });
  }
  await page.locator('input[name^="amount_"]').first().fill("150");
  await page.locator('input[name="start_date"]').fill("01/09/2026");
  await page.locator('input[name="email"]').fill(input.email);
  await expect(page.getByText("This email is available.")).toBeVisible();
  await page.locator('input[name="password"]').fill("Kingsway123!");

  await submitAndExpectPost(page, page.getByRole("button", { name: "Save" }));
  await expect(page).toHaveURL(/view=student-assignment/);
}

async function exerciseStudentHubFiltersAndPagination(
  page: Page,
  input: {
    branchName: string;
    studentFullName: string;
    teacherFullName: string;
  },
) {
  await page.goto("/en/dashboard/owner?view=student-assignment");
  await expect(page.getByRole("heading", { name: "Student & Assignment Hub" })).toBeVisible();
  await page.getByLabel("Branch").selectOption({ label: input.branchName });
  await expect(page.getByText(`Branch: ${input.branchName}`)).toBeVisible();
  await page.getByLabel("Status").selectOption({ label: "Active" });
  await expect(page.getByText("Status: Active")).toBeVisible();
  await page.getByLabel("Teacher").selectOption({ label: input.teacherFullName });
  await expect(page.getByText(`Teacher: ${input.teacherFullName}`)).toBeVisible();

  await page.getByLabel("Rows").selectOption("20");
  await expect(page.getByText(/Showing 20 of/)).toBeVisible();
  const pageTwo = page.getByRole("button", { name: "2" });
  if (await pageTwo.isVisible()) {
    const responsePromise = page.waitForResponse((response) =>
      response.url().includes("/v1/student-assignment-hub") &&
      response.request().method() === "GET",
    );
    await pageTwo.click();
    await responsePromise;
  }

  await page.getByLabel("Search").fill(input.studentFullName.slice(0, 4));
  await expect(page.getByText(input.studentFullName)).toBeVisible();
  await expect(page.getByText(/Search:/)).toBeVisible();
}

async function exerciseTeacherFilters(page: Page, branchName: string) {
  await page.goto("/en/dashboard/owner?view=teacher-finance");
  await page.getByLabel("Branch").selectOption({ label: branchName });
  await expect(page.getByText(`Branch: ${branchName}`)).toBeVisible();
  await page.getByRole("button", { name: "Reset Filter" }).click();
  await expect(page.getByText(`Branch: ${branchName}`)).toHaveCount(0);
}

async function deleteBranchWithPopupEdgeCases(page: Page, branchName: string) {
  await page.goto("/en/dashboard/owner?view=branches");
  const row = page.getByRole("row").filter({ hasText: branchName });
  await expect(row).toBeVisible();
  await row.locator("button").last().click();
  await page.getByRole("menuitem", { name: "Delete" }).click();
  let dialog = page.getByRole("dialog");
  await expect(dialog.getByText("permanently and irreversibly deleted")).toBeVisible();
  await dialog.getByLabel("Branch name").fill(`${branchName}x`);
  await expect(dialog.getByRole("button", { name: "Delete" })).toBeDisabled();
  await expect(dialog.getByText("Branch name does not match.")).toBeVisible();
  await dialog.getByRole("button", { name: "Cancel" }).click();
  await expect(row).toBeVisible();

  await row.locator("button").last().click();
  await page.getByRole("menuitem", { name: "Delete" }).click();
  dialog = page.getByRole("dialog");
  await dialog.getByLabel("Branch name").fill(branchName);
  await dialog.getByRole("button", { name: "Delete" }).click();
  await expect(dialog.getByText("Delete this branch?")).toBeVisible();
  await dialog.getByRole("button", { name: "Cancel" }).click();
  await expect(row).toBeVisible();

  await row.locator("button").last().click();
  await page.getByRole("menuitem", { name: "Delete" }).click();
  dialog = page.getByRole("dialog");
  await dialog.getByLabel("Branch name").fill(branchName);
  await dialog.getByRole("button", { name: "Delete" }).click();
  await submitAndExpectPost(page, dialog.getByRole("button", { name: "Delete" }));
  await expect(page.getByRole("row").filter({ hasText: branchName })).toHaveCount(0);
}

async function selectCheckboxOption(page: Page, triggerText: string, optionText: string) {
  await page.getByRole("button", { name: new RegExp(triggerText, "i") }).click();
  await page.getByRole("menuitemcheckbox", { name: optionText }).click();
  await page.keyboard.press("Escape");
}

async function submitAndExpectPost(page: Page, submitButton: Locator) {
  const responsePromise = page.waitForResponse(
    (response) =>
      response.request().method() === "POST" &&
      response.url().startsWith(appOrigin) &&
      !response.url().includes("/api/dev-asset"),
    { timeout: 30_000 },
  );
  await submitButton.click();
  const response = await responsePromise;
  expect(response.status()).toBeLessThan(500);
  return response;
}

async function hammerSubmitAndExpectSinglePost(page: Page, submitButton: Locator) {
  let postCount = 0;
  const onRequest = (request: { method: () => string; url: () => string }) => {
    if (
      request.method() === "POST" &&
      request.url().startsWith(appOrigin) &&
      !request.url().includes("/api/dev-asset")
    ) {
      postCount += 1;
    }
  };
  page.on("request", onRequest);
  try {
    const responsePromise = page.waitForResponse(
      (response) =>
        response.request().method() === "POST" &&
        response.url().startsWith(appOrigin) &&
        !response.url().includes("/api/dev-asset"),
      { timeout: 30_000 },
    );
    await submitButton.evaluate((element) => {
      for (let index = 0; index < 50; index += 1) {
        (element as HTMLButtonElement).click();
      }
    });
    const response = await responsePromise;
    expect(response.status()).toBeLessThan(500);
    await page.waitForTimeout(750);
    expect(postCount).toBe(1);
  } finally {
    page.off("request", onRequest);
  }
}

async function backendLogin(request: APIRequestContext) {
  const response = await request.post(`${backendBaseURL}/v1/auth/login`, {
    data: { email: ownerEmail, password: ownerPassword },
  });
  expect(response.status()).toBe(200);
  const body = (await response.json()) as { token: string };
  return body.token;
}

async function apiGet<T>(request: APIRequestContext, token: string, path: string) {
  const response = await request.get(`${backendBaseURL}${path}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(response.status()).toBeLessThan(400);
  return (await response.json()) as T;
}

async function apiPost<T>(
  request: APIRequestContext,
  token: string,
  path: string,
  data: unknown,
  idempotencyKey?: string,
) {
  const response = await request.post(`${backendBaseURL}${path}`, {
    data,
    headers: {
      Authorization: `Bearer ${token}`,
      ...(idempotencyKey ? { "Idempotency-Key": idempotencyKey } : {}),
    },
  });
  expect(response.status()).toBeLessThan(500);
  return { body: (await response.json()) as T, response };
}

async function apiDelete(request: APIRequestContext, token: string, path: string) {
  const response = await request.delete(`${backendBaseURL}${path}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(response.status()).toBeLessThan(500);
  return response;
}

async function findBranchByName(
  request: APIRequestContext,
  token: string,
  name: string,
) {
  const branches = await apiGet<ApiBranch[]>(request, token, "/v1/branches");
  const branch = branches.find((item) => item.name === name);
  expect(branch, `Branch ${name} should exist`).toBeTruthy();
  return branch as ApiBranch;
}

async function findTeacherByName(
  request: APIRequestContext,
  token: string,
  fullName: string,
) {
  const teachers = await apiGet<ApiTeacherFinanceRecord[]>(
    request,
    token,
    "/v1/teacher-finance",
  );
  const teacher = teachers.find(
    (item) => `${item.first_name} ${item.last_name}`.trim() === fullName,
  );
  expect(teacher, `Teacher ${fullName} should exist`).toBeTruthy();
  return teacher as ApiTeacherFinanceRecord;
}

async function ensureCourse(
  request: APIRequestContext,
  token: string,
  branchID: string,
  name: string,
) {
  const courses = await apiGet<ApiCourse[]>(
    request,
    token,
    `/v1/courses?branch_id=${encodeURIComponent(branchID)}`,
  );
  const existing = courses.find((course) => course.name === name && course.is_active);
  if (existing) {
    return existing;
  }
  const created = await apiPost<{ course: ApiCourse }>(
    request,
    token,
    "/v1/courses",
    { branch_id: branchID, categories: [], name },
    `e2e-course-${branchID}-${name}`,
  );
  return created.body.course;
}

async function seedStudentsForPagination(
  request: APIRequestContext,
  token: string,
  input: {
    branchID: string;
    courseID: string;
    runID: string;
    teacherID: string;
  },
) {
  for (let index = 1; index <= 24; index += 1) {
    const firstName = `Seed${index}${input.runID}`;
    await apiPost(
      request,
      token,
      "/v1/students",
      {
        branch_id: input.branchID,
        birth_date: "01/01/2012",
        courses: [
          {
            course_id: input.courseID,
            monthly_amount_cents: 15000,
            start_date: "01/09/2026",
            teacher_id: input.teacherID,
          },
        ],
        fin: makeFIN(input.runID, index),
        first_name: firstName,
        last_name: "Student",
        phone: `+99455000${String(index).padStart(4, "0")}`,
        status: "active",
      },
      `e2e-student-${input.runID}-${index}`,
    );
  }
}

async function exerciseAPIIdempotencyAndConcurrency(
  request: APIRequestContext,
  token: string,
  runID: string,
) {
  const replayName = `E2E Replay ${runID}`;
  const replayPayload = {
    address: "E2E replay address",
    name: replayName,
    slug: `e2e-replay-${runID}`,
  };
  const replayKey = `e2e-replay-${runID}`;
  const first = await apiPost<ApiBranch>(
    request,
    token,
    "/v1/branches",
    replayPayload,
    replayKey,
  );
  const second = await apiPost<ApiBranch>(
    request,
    token,
    "/v1/branches",
    replayPayload,
    replayKey,
  );
  expect(second.body.id).toBe(first.body.id);

  const conflict = await request.post(`${backendBaseURL}/v1/branches`, {
    data: { ...replayPayload, name: `${replayName} conflict` },
    headers: {
      Authorization: `Bearer ${token}`,
      "Idempotency-Key": replayKey,
    },
  });
  expect(conflict.status()).toBe(409);
  await apiDelete(request, token, `/v1/branches/${first.body.id}`);

  const concurrentName = `E2E Concurrent ${runID}`;
  const concurrentPayload = {
    address: "E2E concurrent address",
    name: concurrentName,
    slug: `e2e-concurrent-${runID}`,
  };
  const concurrentKey = `e2e-concurrent-${runID}`;
  const responses = await Promise.all(
    Array.from({ length: 50 }, () =>
      request.post(`${backendBaseURL}/v1/branches`, {
        data: concurrentPayload,
        headers: {
          Authorization: `Bearer ${token}`,
          "Idempotency-Key": concurrentKey,
        },
      }),
    ),
  );
  expect(responses.every((response) => [201, 409].includes(response.status()))).toBe(true);
  const branches = await apiGet<ApiBranch[]>(request, token, "/v1/branches");
  const created = branches.filter((branch) => branch.name === concurrentName);
  expect(created).toHaveLength(1);
  await apiDelete(request, token, `/v1/branches/${created[0].id}`);
}

async function createTestPNG(testInfo: TestInfo) {
  const pngPath = testInfo.outputPath("kingsway-e2e-profile.png");
  const base64 =
    "iVBORw0KGgoAAAANSUhEUgAAAAIAAAACCAIAAAD91JpzAAAAGElEQVR4nGP8z8Dwn4GBgYGJgQEYAAAN+wICeY4Z9AAAAABJRU5ErkJggg==";
  await fs.writeFile(pngPath, Buffer.from(base64, "base64"));
  return pngPath;
}

function makeFIN(seed: string, index: number) {
  const numeric = parseInt(seed.replace(/[^a-z0-9]/gi, "").slice(-6), 36) || 0;
  const value = (numeric + index).toString(36).toUpperCase().padStart(6, "0");
  return `T${value}`.slice(0, 7);
}
