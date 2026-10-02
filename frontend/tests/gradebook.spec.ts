import { test, expect, type APIRequestContext } from "@playwright/test";
import type { Gradebook } from "../src/lib/api";

test("GB5 libro, cero publicado, entrega exacta, tablet y recuperación", async ({ page, browser }) => {
  test.skip(process.env.E2E_ACADEMIC !== "1", "Requiere Go/PostgreSQL sintético");
  test.setTimeout(120_000);
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
  const list = await (await page.request.get(`/api/v1/teacher/courses/${course.id}/activities`)).json();
  const title = `Nota cero ${Date.now()}`;
  let activity = await mutate(page.request, "POST", `/teacher/courses/${course.id}/activities`, { moduleId: list.modules[0].id, title, description: "Prueba del libro", type: "assignment", body: "Explica tu respuesta" });
  activity = await mutate(page.request, "POST", `/teacher/activities/${activity.id}/publish`, { version: 1 });
  const context = await browser.newContext({ baseURL: origin });
  try {
    await mutate(context.request, "POST", "/auth/login", { username: "luna", password: process.env.E2E_STUDENT_PASSWORD });
    const submission = await mutate(context.request, "PUT", `/lessons/${activity.lessonId}/submission`, { version: 0, lessonVersion: 1, body: "Mi entrega para revisar" });
    await mutate(context.request, "POST", `/lessons/${activity.lessonId}/submission/submit`, { version: submission.version });
    await mutate(page.request, "PUT", `/teacher/submissions/${submission.id}/grade`, { version: 0, score: 0, feedback: "Vamos a revisarlo juntos" });
    await mutate(page.request, "POST", `/teacher/submissions/${submission.id}/grade/publish`, { version: 1 });
    let activityPage = 1;
    let book: Gradebook;
    for (;;) {
      book = await (await page.request.get(`/api/v1/teacher/courses/${course.id}/gradebook?activityPage=${activityPage}`)).json();
      if (book.activities.some((a) => a.activityId === activity.id)) break;
      expect(activityPage * book.activityPageSize).toBeLessThan(book.totalActivities);
      activityPage++;
    }
    await page.goto(`/teacher/courses/${course.id}`);
    await page.getByRole("link", { name: "Ver calificaciones" }).click();
    await expect(page.getByRole("heading", { name: "Libro de calificaciones" })).toBeVisible();
    for (let i = 1; i < activityPage; i++) await page.getByRole("link", { name: "Más tareas" }).click();
    const column = book.activities.findIndex((a) => a.activityId === activity.id);
    const row = page.getByRole("row").filter({ has: page.getByRole("rowheader", { name: "Luna", exact: true }) });
    const cell = row.getByRole("cell").nth(column);
    await expect(cell).toContainText("Nota publicada");
    await expect(cell).toContainText("0 / 100");
    const region = page.getByRole("region", { name: "Tabla de calificaciones desplazable" });
    await region.focus();
    await expect(region).toBeFocused();
    await page.keyboard.press("ArrowRight");
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBeTruthy();
    await page.screenshot({ path: "../docs/screenshots/gradebook-tablet.png", fullPage: true, animations: "disabled" });
    await cell.getByRole("link", { name: /Revisar entrega de Luna/ }).click();
    await expect(page).toHaveURL(new RegExp(`submissionId=${submission.id}#submission-${submission.id}$`));
    await expect(page.locator(`#submission-${submission.id}`)).toContainText("Mi entrega para revisar");
    await expect(page.locator('academic-form[data-kind="grade"]')).toHaveCount(1);
    await page.goto(`/teacher/courses/${course.id}/gradebook?page=0`);
    await expect(page.getByRole("alert")).toBeVisible();
    await expect(page.getByRole("table")).toHaveCount(0);
    await page.getByRole("link", { name: "Volver a intentar" }).click();
    await expect(page.getByRole("table")).toBeVisible();
  } finally { await context.close(); }
});
