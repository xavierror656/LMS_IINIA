import { test, expect } from "@playwright/test";
test("LMS014 admin consulta plataforma, edita plugins y cierra sesión", async ({
  page,
}) => {
  test.skip(process.env.E2E_DEMO !== "1", "Solo demo");
  await page.goto("/admin");
  await expect(page).toHaveURL(/login/);
  await page.getByLabel("Tu usuario").fill("admin");
  await page.getByLabel("Tu contraseña").fill("aulaquest-demo");
  await page.getByRole("button", { name: "Entrar a mi aventura" }).click();
  await expect(page).toHaveURL(/\/admin$/);
  await expect(
    page.getByRole("heading", { name: "Usuarios", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("cell", { name: "Administrador", exact: true }),
  ).toBeVisible();
  await expect(page.getByLabel("Tu progreso")).toHaveCount(0);
  await page.getByRole("link", { name: "Editar plugins" }).click();
  const form = page.locator('[data-plugin="h5p"]');
  await expect(form.getByLabel("Nombre del plugin")).toBeVisible();
  const original = await form.getByLabel("Nombre del plugin").inputValue();
  try {
    await form.getByLabel("Nombre del plugin").fill("Reto del administrador");
    await form.getByRole("button", { name: "Guardar cambios" }).click();
    await expect(form.getByRole("status")).toContainText("Cambios guardados");
    await page.reload();
    await expect(form.getByLabel("Nombre del plugin")).toHaveValue(
      "Reto del administrador",
    );
    await expect(form).toContainText("Actividad aún no configurada");
    const height = await form
      .getByLabel("Nombre del plugin")
      .evaluate((el) => el.getBoundingClientRect().height);
    expect(height).toBeGreaterThanOrEqual(44);
  } finally {
    await form.getByLabel("Nombre del plugin").fill(original);
    await form.getByRole("button", { name: "Guardar cambios" }).click();
    await expect(form.getByRole("status")).toContainText("Cambios guardados");
  }
  await page.getByRole("button", { name: "Salir", exact: true }).click();
  await expect(page).toHaveURL(/login/);
  await page.goto("/admin");
  await expect(page).toHaveURL(/login/);
});
test("LMS014 docente no entra al panel admin", async ({ page }) => {
  test.skip(process.env.E2E_DEMO !== "1", "Solo demo");
  await page.goto("/login");
  await page.getByLabel("Tu usuario").fill("profe");
  await page.getByLabel("Tu contraseña").fill("aulaquest-demo");
  await page.getByRole("button", { name: "Entrar a mi aventura" }).click();
  await expect(page).toHaveURL(/\/teacher$/);
  await page.goto("/admin");
  await expect(page).toHaveURL(/\/teacher$/);
});
