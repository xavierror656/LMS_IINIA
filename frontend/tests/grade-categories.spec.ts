import { test, expect, type APIRequestContext } from "@playwright/test";

test("GC1–GC7 categorías, totales y notas del alumno", async ({ page, browser }) => {
  test.skip(process.env.E2E_ACADEMIC !== "1", "Requiere Go/PostgreSQL sintético");
  test.setTimeout(180_000);
  const origin = process.env.E2E_BASE_URL ?? "http://localhost:4321";
  async function mutate(request: APIRequestContext, method: string, path: string, data: unknown) {
    const res = await request.fetch(`/api/v1${path}`, { method, data, headers: { Origin: origin } });
    expect(res.ok(), await res.text()).toBeTruthy();
    return res.json();
  }
  await page.goto("/login");
  await page.getByLabel("Tu usuario").fill("profe");
  await page.getByLabel("Tu contraseña").fill(process.env.E2E_TEACHER_PASSWORD!);
  await page.getByRole("button", { name: "Entrar a mi aventura" }).click();
  await expect(page).toHaveURL(/\/teacher$/);
  const courses = await (await page.request.get("/api/v1/teacher/courses")).json();
  const course = courses.items[0];
  const courseId = course.id;
  const categoryName = `Parciales E2E ${Date.now()}`;

  // GC1: the teacher creates a weighted category and changes the missing policy.
  await page.goto(`/teacher/courses/${courseId}/gradebook`);
  const manager = page.locator("grade-categories");
  await manager.locator("form[data-create]").getByLabel("Nueva categoría").fill(categoryName);
  await manager.locator("form[data-create]").getByLabel("Peso").fill("2");
  await manager.locator("form[data-create]").getByRole("button", { name: "Crear categoría" }).click();
  await expect(manager.locator(`form[data-save] input[name="name"][value="${categoryName}"]`)).toHaveCount(1);
  await manager.locator("form[data-policy]").getByLabel("Política de actividades sin nota").selectOption("zero");
  await manager.locator("form[data-policy]").getByRole("button", { name: "Guardar política" }).click();
  await expect(manager.locator("form[data-policy]").getByLabel("Política de actividades sin nota")).toHaveValue("zero");
  await page.screenshot({ path: "../docs/screenshots/grade-categories-tablet.png", animations: "disabled" });

  // GC2: a new assignment is authored, assigned to the category and published.
  const title = `Actividad categoría ${Date.now()}`;
  await page.goto(`/teacher/courses/${courseId}`);
  const creator = page.locator('academic-form[data-kind="create"]');
  await creator.getByLabel("Tipo de actividad").selectOption("assignment");
  await creator.getByLabel("Título", { exact: true }).fill(title);
  await creator.getByLabel("Contenido o instrucciones").fill("Resuelve y explica.");
  await creator.getByRole("button", { name: "Guardar borrador" }).click();
  await expect(page).toHaveURL(/\/teacher\/activities\/\d+$/);
  const activityId = page.url().split("/").at(-1)!;
  await page.getByRole("link", { name: "Configurar rúbrica y peso" }).click();
  await page.getByLabel("Categoría de calificación").selectOption({ label: `${categoryName} · peso 2` });
  await page.getByRole("button", { name: "Guardar borrador" }).click();
  await page.getByRole("link", { name: "Volver a la actividad" }).click();
  await page.getByRole("button", { name: "Publicar actividad" }).click();
  await expect(page.getByText("Última versión publicada: 2.", { exact: false })).toBeVisible();
  const activity = await (await page.request.get(`/api/v1/teacher/activities/${activityId}`)).json();
  expect(activity.categoryId).toBeGreaterThan(0);

  // GC3/GC4: Luna delivers, the teacher publishes 80 and the totals move.
  const context = await browser.newContext({ viewport: { width: 768, height: 1024 }, reducedMotion: "reduce" });
  const student = await context.newPage();
  try {
    await mutate(context.request, "POST", "/auth/login", { username: "luna", password: process.env.E2E_STUDENT_PASSWORD });
    const submission = await mutate(context.request, "PUT", `/lessons/${activity.lessonId}/submission`, { version: 0, lessonVersion: 2, body: "Mi respuesta de la categoría" });
    await mutate(context.request, "POST", `/lessons/${activity.lessonId}/submission/submit`, { version: submission.version });
    await mutate(page.request, "PUT", `/teacher/submissions/${submission.id}/grade`, { version: 0, score: 80, feedback: "Bien planteado" });
    await mutate(page.request, "POST", `/teacher/submissions/${submission.id}/grade/publish`, { version: 1 });

    await page.goto(`/teacher/courses/${courseId}/gradebook`);
    await expect(page.getByRole("columnheader", { name: "Total del curso" })).toBeVisible();
    const row = page.getByRole("row").filter({ has: page.getByRole("rowheader", { name: "Luna", exact: true }) });
    await expect(row.getByText(`${categoryName}: 80.00`, { exact: true })).toBeVisible();
    // The database may hold graded work from earlier suites: compare the screen
    // against the canonical book instead of a hardcoded course total.
    const bookNow = await (await page.request.get(`/api/v1/teacher/courses/${courseId}/gradebook`)).json();
    const luna = bookNow.rows.find((candidate: { alias: string }) => candidate.alias === "Luna");
    const expectedTotal = (luna.summary.courseTotalHundredths / 100).toFixed(2);
    await expect(row.getByText(`${expectedTotal} / 100`, { exact: true })).toBeVisible();

    // GC6: the CSV repeats the same category and course totals.
    const csv = await (await page.request.get(`/api/v1/teacher/courses/${courseId}/gradebook.csv`)).text();
    expect(csv).toContain(`Total ${categoryName}`);
    expect(csv).toContain("Total del curso");
    expect(csv).toContain("80.00");

    // GC7: the student reads only published grades and keeps them after reload.
    const own = await (await context.request.get(`/api/v1/me/courses/${courseId}/grades`)).json();
    const expectedMe = (own.courseTotalHundredths / 100).toFixed(2);
    await student.goto(`/courses/${courseId}`);
    await expect(student.getByRole("heading", { name: "Mis notas" })).toBeVisible();
    await expect(student.getByText(`Total del curso: ${expectedMe} / 100`, { exact: true })).toBeVisible();
    await expect(student.getByText(`${categoryName}`).first()).toBeVisible();
    await expect(student.getByText("80 / 100").first()).toBeVisible();
    await student.reload();
    await expect(student.getByText(`Total del curso: ${expectedMe} / 100`, { exact: true })).toBeVisible();
    await student.screenshot({ path: "../docs/screenshots/student-grades-tablet.png", animations: "disabled" });
  } finally {
    await context.close();
  }
});
