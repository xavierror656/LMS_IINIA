import { test, expect } from "@playwright/test";
test("LMS013 docente edita plugins y estudiante observa cambios", async ({
  page,
  browser,
}) => {
  test.skip(process.env.E2E_DEMO !== "1", "Edición exclusiva de demo");
  await page.goto("/login");
  await page.getByLabel("Tu usuario").fill("profe");
  await page.getByLabel("Tu contraseña").fill("aulaquest-demo");
  await page.getByRole("button", { name: "Entrar a mi aventura" }).click();
  await page.getByRole("link", { name: "Editar plugins" }).click();
  const original = await page.evaluate(
    async () => (await (await fetch("/api/v1/plugins")).json()).items,
  );
  try {
    const form = page.locator('[data-plugin="code"]');
    await form
      .getByLabel("Nombre del plugin")
      .fill("Laboratorio de descubrimientos");
    await form
      .getByLabel("Instrucciones para estudiantes")
      .fill("Imagina una idea y escríbela.");
    await form.getByLabel("Lenguaje inicial").selectOption("python");
    await form.getByRole("button", { name: "Guardar cambios" }).click();
    await expect(form.getByRole("status")).toContainText("Cambios guardados");
    await page.reload();
    await expect(page.locator("#code-title")).toHaveValue(
      "Laboratorio de descubrimientos",
    );
    const context = await browser.newContext();
    try {
      const student = await context.newPage();
      await student.goto(new URL("/login", page.url()).href);
      await student.getByLabel("Tu usuario").fill("luna");
      await student.getByLabel("Tu contraseña").fill("aulaquest-demo");
      await student
        .getByRole("button", { name: "Entrar a mi aventura" })
        .click();
      await expect(student).toHaveURL(/courses/);
      await student.goto(new URL("/lessons/2", page.url()).href);
      await expect(
        student.getByRole("heading", {
          name: "Laboratorio de descubrimientos",
        }),
      ).toBeVisible();
      await expect(student.locator("select")).toHaveValue("python");
      await student.goto(new URL("/teacher/plugins", page.url()).href);
      await expect(student).toHaveURL(/courses/);
    } finally {
      await context.close();
    }
  } finally {
    await page.evaluate(async (plugins) => {
      for (const { id, ...body } of plugins) {
        const response = await fetch(`/api/v1/teacher/plugins/${id}`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        });
        if (!response.ok)
          throw new Error("No se pudo restaurar la configuración");
      }
    }, original);
  }
});
