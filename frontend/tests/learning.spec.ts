import { test, expect, type Page } from "@playwright/test";
const studentPassword = process.env.E2E_STUDENT_PASSWORD;
const teacherPassword = process.env.E2E_TEACHER_PASSWORD;
async function login(page: Page, username: string, password: string) {
  await page.goto("/login");
  if (process.env.E2E_DEMO === "1") {
    await expect(page.getByLabel("Modo de demostración")).toBeVisible();
    await expect(page.getByText("Contraseña para todos:")).toContainText("aulaquest-demo");
  }
  await page.getByLabel("Tu usuario").fill(username);
  await page.getByLabel("Tu contraseña").fill(password);
  await page.getByRole("button", { name: "Entrar a mi aventura" }).click();
  await expect(page).toHaveURL(
    username === "profe" ? /\/teacher$/ : /\/courses$/,
  );
}
test("LMS001/003/004/008/009 student flow, persistent HUD, refresh and logout", async ({
  page,
}) => {
  test.skip(!studentPassword, "E2E_STUDENT_PASSWORD required");
  await login(page, "luna", studentPassword!);
  await expect(
    page.getByRole("heading", { name: "Mis aventuras", exact: true }),
  ).toBeVisible();
  await expect(page.getByLabel("Tu progreso")).toContainText("Luna");
  await page
    .locator("astro-island")
    .filter({ has: page.getByLabel("Tu progreso") })
    .evaluate((el) => el.setAttribute("data-persistence-check", "kept"));
  await page.getByRole("link", { name: "Ver mi camino" }).first().click();
  await expect(page.getByLabel("Tu progreso")).toContainText("Luna");
  await expect(page.locator('[data-persistence-check="kept"]')).toHaveCount(1);
  await page.getByRole("link").filter({ hasText: "Lectura ·" }).first().click();
  const complete = page.locator("#complete-reading");
  await expect(complete).toBeVisible();
  if (await complete.isEnabled()) {
    await complete.click();
    await expect(page.locator("#completion-status")).toContainText(
      "Tu progreso está guardado",
    );
  }
  await expect(page.getByLabel("Tu progreso")).toContainText("25 XP");
  await page.reload();
  await expect(page.getByLabel("Tu progreso")).toContainText("25 XP");
  await expect(
    page.getByRole("button", { name: "Lectura completada" }),
  ).toBeDisabled();
  await page.getByRole("button", { name: "Salir", exact: true }).click();
  await expect(page).toHaveURL(/login/);
  await expect(page.getByLabel("Tu progreso")).toHaveCount(0);
  await page.goto("/courses");
  await expect(page).toHaveURL(/login/);
});
test("LMS006 code websocket and LMS007 missing H5P", async ({ page }) => {
  test.skip(!studentPassword, "E2E_STUDENT_PASSWORD required");
  await login(page, "luna", studentPassword!);
  await page.getByRole("link", { name: "Ver mi camino" }).first().click();
  await page.getByRole("link").filter({ hasText: "Laboratorio ·" }).first().click();
  await expect(
    page.getByText("Ejecución simulada", { exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Ejecutar", exact: true }),
  ).toBeEnabled();
  await page.getByRole("button", { name: "Ejecutar", exact: true }).click();
  await expect(page.getByRole("log")).toContainText("tu código no se ejecuta");
  await expect(page.getByRole("log")).toContainText("Simulación finalizada");
  await page.getByRole("button", { name: "Ejecutar", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "Detener", exact: true }),
  ).toBeEnabled();
  await page.getByRole("button", { name: "Detener", exact: true }).click();
  await expect(page.getByRole("log")).toContainText("Simulación detenida");
  await page
    .getByRole("link", { name: "Mis aventuras", exact: true })
    .first()
    .click();
  await page.getByRole("link", { name: "Ver mi camino" }).first().click();
  await page.getByRole("link").filter({ hasText: "Actividad ·" }).click();
  await expect(page.getByRole("status")).toContainText(
    "Actividad aún no configurada",
  );
});
test("LMS005 teacher sees linked student progress", async ({ page }) => {
  test.skip(!teacherPassword, "E2E_TEACHER_PASSWORD required");
  await login(page, "profe", teacherPassword!);
  await expect(
    page.getByRole("cell", { name: "Luna", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("cell", { name: "Sol", exact: true }),
  ).toHaveCount(0);
  await page.getByRole("link", { name: "Ver progreso de Luna" }).click();
  await expect(
    page.getByRole("heading", { name: "El camino de Luna" }),
  ).toBeVisible();
  await expect(page.getByRole("table")).toContainText(
    "Exploradores del código",
  );
});
test("LMS009 keyboard and reduced motion on tablet", async ({ page }) => {
  await page.goto("/login");
  await page.keyboard.press("Tab");
  await expect(
    page.getByRole("link", { name: "Saltar al contenido" }),
  ).toBeFocused();
  await page.getByLabel("Tu usuario").focus();
  await page.keyboard.type("luna");
  await page.keyboard.press("Tab");
  await expect(page.getByLabel("Tu contraseña")).toBeFocused();
  expect(
    await page.evaluate(
      () => matchMedia("(prefers-reduced-motion: reduce)").matches,
    ),
  ).toBe(true);
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
  const height = await page
    .getByRole("button", { name: "Entrar a mi aventura" })
    .evaluate((el) => el.getBoundingClientRect().height);
  expect(height).toBeGreaterThanOrEqual(44);
});
test("LMS008 API outage shows retry and recovers", async ({ page }) => {
  test.skip(!studentPassword, "E2E_STUDENT_PASSWORD required");
  await login(page, "luna", studentPassword!);
  await page.route("**/api/v1/me/progress", (route) => route.abort());
  await page.reload();
  await expect(page.getByLabel("Tu progreso")).toContainText(
    "No podemos conectar",
  );
  await expect(page.getByLabel("Tu progreso")).not.toContainText("25 XP");
  await page.unroute("**/api/v1/me/progress");
  await page.getByRole("button", { name: "Reintentar", exact: true }).click();
  await expect(page.getByLabel("Tu progreso")).toContainText("Luna");
});
