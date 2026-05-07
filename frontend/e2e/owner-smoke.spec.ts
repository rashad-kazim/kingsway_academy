import { expect, type Locator, type Page, test } from "@playwright/test";

const appBaseURL = process.env.KINGSWAY_E2E_BASE_URL ?? "http://127.0.0.1:3000";
const appOrigin = new URL(appBaseURL).origin;
const ownerEmail = process.env.KINGSWAY_E2E_OWNER_EMAIL ?? "owner@kingsway.local";
const ownerPassword = process.env.KINGSWAY_E2E_OWNER_PASSWORD ?? "Kingsway123!";

test.describe.configure({ mode: "serial" });

test.describe("Owner smoke flows", () => {
  test("login, branch save, teacher save, and student create stay functional", async ({
    page,
  }) => {
    const runtimeErrors: string[] = [];
    page.on("console", (message) => {
      if (message.type() === "error") {
        runtimeErrors.push(message.text());
      }
    });
    page.on("pageerror", (error) => runtimeErrors.push(error.message));

    const runID = Date.now().toString(36);
    const branchName = `E2E Branch ${runID}`;
    const editedBranchAddress = `E2E Updated Address ${runID}`;
    const teacherFirstName = `E2E${runID}`;
    const teacherLastName = "Teacher";
    const teacherFullName = `${teacherFirstName} ${teacherLastName}`;
    const teacherEmail = `e2e.teacher.${runID}@kingsway.local`;
    const studentFirstName = `E2E${runID}`;
    const studentLastName = "Student";
    const studentFullName = `${studentFirstName} ${studentLastName}`;
    const studentEmail = `e2e.student.${runID}@kingsway.local`;
    const studentFIN = runID.replace(/[^a-z0-9]/gi, "").toUpperCase().padEnd(7, "A").slice(0, 7);

    await loginAsOwner(page);

    try {
      await createBranch(page, branchName, `E2E Address ${runID}`);
      await editBranch(page, branchName, editedBranchAddress);
      await createTeacher(page, {
        branchName,
        email: teacherEmail,
        firstName: teacherFirstName,
        lastName: teacherLastName,
      });
      await editTeacher(page, teacherFullName);
      await createStudent(page, {
        branchName,
        email: studentEmail,
        fin: studentFIN,
        firstName: studentFirstName,
        lastName: studentLastName,
        teacherFullName,
      });

      await page.goto("/en/dashboard/owner?view=student-assignment");
      await expect(page.getByRole("heading", { name: "Student & Assignment Hub" })).toBeVisible();
      await expect(page.getByText(studentFullName)).toBeVisible();
      expect(runtimeErrors).toEqual([]);
    } finally {
      await cleanupBranch(page, branchName).catch(() => undefined);
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

async function createBranch(page: Page, branchName: string, address: string) {
  await page.goto("/en/dashboard/owner?view=branches");
  await expect(page.getByRole("heading", { name: "Branch Management" })).toBeVisible();
  await page.getByRole("button", { name: "Add New Branch" }).click();

  const dialog = page.getByRole("dialog");
  await expect(dialog.getByRole("heading", { name: "Add New Branch" })).toBeVisible();
  await dialog.locator('input[name="name"]').fill(branchName);
  await dialog.locator('textarea[name="address"]').fill(address);

  await submitAndExpectPost(page, dialog.getByRole("button", { name: "Save Branch" }));
  await expect(page.getByRole("row").filter({ hasText: branchName })).toBeVisible();
}

async function editBranch(page: Page, branchName: string, address: string) {
  await page.goto("/en/dashboard/owner?view=branches");
  const row = page.getByRole("row").filter({ hasText: branchName });
  await expect(row).toBeVisible();
  await row.click();
  await expect(page.getByRole("heading", { name: "Branch Detail" })).toBeVisible();
  await page.locator('textarea[name="address"]').fill(address);

  await submitAndExpectPost(page, page.getByRole("button", { name: "Save" }));
  await expect(page.getByRole("heading", { name: "Branch Management" })).toBeVisible();
  await expect(page.getByRole("row").filter({ hasText: branchName })).toContainText(address);
}

async function createTeacher(
  page: Page,
  input: {
    branchName: string;
    email: string;
    firstName: string;
    lastName: string;
  },
) {
  await page.goto("/en/dashboard/owner?view=teacher-add");
  await expect(page.getByRole("heading", { name: "Add Teacher" })).toBeVisible();
  await page.locator('input[name="first_name"]').fill(input.firstName);
  await page.locator('input[name="last_name"]').fill(input.lastName);
  await page.locator('input[name="birth_date"]').fill("15/07/1998");
  await page.locator('input[name="phone"]').fill("+994501112233");
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

async function editTeacher(page: Page, teacherFullName: string) {
  await page.goto("/en/dashboard/owner?view=teacher-finance");
  const row = page.getByRole("row").filter({ hasText: teacherFullName });
  await expect(row).toBeVisible();
  await row.getByRole("link", { name: "Edit" }).click();
  await expect(page.getByRole("heading", { name: "Edit" })).toBeVisible();
  await page.locator('input[name="salary_amount"]').fill("101");

  await submitAndExpectPost(page, page.getByRole("button", { name: "Save" }));
  await expect(page).toHaveURL(/view=teacher-finance/);
  await expect(page.getByText(teacherFullName)).toBeVisible();
}

async function createStudent(
  page: Page,
  input: {
    branchName: string;
    email: string;
    fin: string;
    firstName: string;
    lastName: string;
    teacherFullName: string;
  },
) {
  await page.goto("/en/dashboard/owner?view=student-add");
  await expect(page.getByRole("heading", { name: "Add Student" })).toBeVisible();
  await page.locator('input[name="first_name"]').fill(input.firstName);
  await page.locator('input[name="last_name"]').fill(input.lastName);
  await page.locator('input[name="fin"]').fill(input.fin);
  await page.locator('input[name="birth_date"]').fill("15/07/2012");
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

async function cleanupBranch(page: Page, branchName: string) {
  await page.goto("/en/dashboard/owner?view=branches");
  const row = page.getByRole("row").filter({ hasText: branchName });
  if ((await row.count()) === 0) {
    return;
  }

  await row.first().locator("button").last().click();
  await page.getByRole("menuitem", { name: "Delete" }).click();
  const dialog = page.getByRole("dialog");
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
    { timeout: 20_000 },
  );
  await submitButton.click();
  const response = await responsePromise;
  expect(response.status()).toBeLessThan(500);
  return response;
}
