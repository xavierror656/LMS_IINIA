import { test, expect, type Page } from "@playwright/test";

test.describe("Administración académica · Go/PostgreSQL", () => {
  test.skip(
    process.env.E2E_ACADEMIC !== "1",
    "Requiere backend Go y base sintética separada",
  );
  test.setTimeout(120_000);

  async function login(page: Page, username: string) {
    await page.goto("/login");
    await page.getByLabel("Tu usuario").fill(username);
    await page
      .getByLabel("Tu contraseña")
      .fill(
        username === "profe"
          ? process.env.E2E_TEACHER_PASSWORD!
          : process.env.E2E_STUDENT_PASSWORD!,
      );
    await page.getByRole("button", { name: "Entrar a mi aventura" }).click();
    await expect(page).toHaveURL(
      username === "profe" ? /\/teacher$/ : /\/courses$/,
    );
  }

  test("I1–I6 crear, publicar, entregar, calificar y recuperar errores", async ({
    page,
    browser,
  }) => {
    await login(page, "profe");
    await page
      .getByRole("link", { name: "Crear actividades y calificar" })
      .click();
    await page
      .getByRole("link", { name: "Gestionar actividades" })
      .first()
      .click();
    await expect(page).toHaveURL(/\/teacher\/courses\/\d+\/?$/);
    const courseId = new URL(page.url()).pathname.match(/courses\/(\d+)/)![1];
    const title = `Mi descubrimiento ${Date.now()}`;
    const creator = page.locator('academic-form[data-kind="create"]');
    await creator.getByLabel("Tipo de actividad").selectOption("assignment");
    await creator.getByLabel("Título", { exact: true }).fill(title);
    await creator
      .getByLabel("Descripción breve")
      .fill("Explica con tus propias palabras.");
    await creator
      .getByLabel("Contenido o instrucciones")
      .fill("Cuenta qué aprendiste. <script>window.badAuthoring=true</script>");
    await creator.getByRole("button", { name: "Guardar borrador" }).click();
    await expect(page).toHaveURL(/\/teacher\/activities\/\d+$/);
    const activityURL = page.url();
    const activityId = activityURL.split("/").at(-1)!;
    const editor = page.locator('academic-form[data-kind="activity"]');
    await expect(editor.getByLabel("Título", { exact: true })).toHaveValue(
      title,
    );

    const studentContext = await browser.newContext({
      viewport: { width: 768, height: 1024 },
      reducedMotion: "reduce",
    });
    const student = await studentContext.newPage();
    const secondContext = await browser.newContext();
    const secondStudent = await secondContext.newPage();
    try {
      await login(student, "luna");
      await student.goto(`/courses/${courseId}`);
      await expect(student.getByRole("heading", { name: title })).toHaveCount(
        0,
      );
      const denied = await student.request.get(
        `/api/v1/teacher/activities/${activityId}`,
      );
      expect(denied.status()).toBe(403);

      await editor.getByRole("button", { name: "Publicar actividad" }).click();
      await expect(
        page.getByText("Última versión publicada: 1.", { exact: false }),
      ).toBeVisible();
      await student.reload();
      await student
        .getByRole("link")
        .filter({ has: student.getByRole("heading", { name: title }) })
        .click();
      await expect(student.locator(".reading-body").first()).toContainText(
        "<script>",
      );
      expect(await student.evaluate(() => "badAuthoring" in window)).toBe(
        false,
      );
      const lessonURL = student.url();
      const answer = student.getByLabel("Escribe tu trabajo");
      await answer.fill("Aprendí a ordenar mis ideas.");

      // Failure must preserve text and must not imply a successful save.
      await student.route("**/api/v1/lessons/*/submission", (route) =>
        route.abort(),
      );
      await student.getByRole("button", { name: "Guardar borrador" }).click();
      await expect(
        student.locator("academic-form [role=status]"),
      ).toContainText("Tu texto permanece");
      await expect(answer).toHaveValue("Aprendí a ordenar mis ideas.");
      await expect(
        student.getByRole("button", { name: "Enviar al docente" }),
      ).toBeDisabled();
      await student.unroute("**/api/v1/lessons/*/submission");

      // Native keyboard submission and persisted reload.
      const save = student.getByRole("button", { name: "Guardar borrador" });
      await save.focus();
      await student.keyboard.press("Enter");
      await expect(
        student.locator("academic-form [role=status]"),
      ).toContainText("Borrador guardado");
      await student.reload();
      await expect(answer).toHaveValue("Aprendí a ordenar mis ideas.");
      await page.reload();
      await expect(
        page.getByText("No hay entregas enviadas en esta página."),
      ).toBeVisible();

      await login(secondStudent, "sol");
      await secondStudent.goto(lessonURL);
      await expect(secondStudent.getByLabel("Escribe tu trabajo")).toHaveValue(
        "",
      );
      await secondStudent
        .getByLabel("Escribe tu trabajo")
        .fill("Respuesta de Sol para su docente vinculado.");
      await secondStudent
        .getByRole("button", { name: "Guardar borrador" })
        .click();
      await expect(
        secondStudent.locator("academic-form [role=status]"),
      ).toContainText("Borrador guardado");
      await secondStudent
        .getByRole("button", { name: "Enviar al docente" })
        .click();
      await expect(
        secondStudent.getByRole("heading", { name: "Tu trabajo está enviado" }),
      ).toBeVisible();
      const solLessonId = new URL(lessonURL).pathname.split("/").at(-1)!;
      const solResponse = await secondStudent.request.get(
        `/api/v1/lessons/${solLessonId}/submission`,
      );
      const solSubmission = (await solResponse.json()) as {
        submission: { id: number };
      };
      const forbiddenGrade = await page.request.put(
        `/api/v1/teacher/submissions/${solSubmission.submission.id}/grade`,
        {
          headers: { Origin: new URL(page.url()).origin },
          data: { score: 100, feedback: "", version: 0 },
        },
      );
      expect(forbiddenGrade.status()).toBe(404);

      await student.getByRole("button", { name: "Enviar al docente" }).click();
      await expect(
        student.getByRole("heading", { name: "Tu trabajo está enviado" }),
      ).toBeVisible();
      await expect(student.getByLabel("Escribe tu trabajo")).toHaveCount(0);
      await page.reload();
      await expect(
        page.getByRole("heading", { name: "Entrega de Luna" }),
      ).toBeVisible();
      await expect(
        page.getByRole("heading", { name: "Entrega de Sol" }),
      ).toHaveCount(0);
      await expect(
        page.getByText("Respuesta de Sol para su docente vinculado."),
      ).toHaveCount(0);
      const grading = page
        .locator('academic-form[data-kind="grade"]')
        .filter({ has: page.getByLabel("Calificación de Luna (0 a 100)") });
      await grading.getByLabel("Calificación de Luna (0 a 100)").fill("87");
      await grading
        .getByLabel("Comentario para Luna")
        .fill("Explicaste muy bien tus ideas.");
      await grading
        .getByRole("button", { name: "Guardar calificación" })
        .click();
      await expect(grading.getByRole("status")).toContainText(
        "Borrador guardado",
      );
      await student.reload();
      await expect(
        student.getByText("Explicaste muy bien tus ideas."),
      ).toHaveCount(0);
      await grading
        .getByRole("button", { name: "Publicar devolución" })
        .click();
      await expect(
        page.getByText("Devolución publicada", { exact: true }),
      ).toBeVisible();
      await student.reload();
      await expect(student.getByText("Calificación: 87 / 100")).toBeVisible();
      await expect(
        student.getByText("Explicaste muy bien tus ideas."),
      ).toBeVisible();
      await student.screenshot({
        path: "../docs/screenshots/academic-student-tablet.png",
        fullPage: true,
      });
      await page.screenshot({
        path: "../docs/screenshots/academic-grading-tablet.png",
        fullPage: true,
      });
      const dimensions = await grading
        .getByRole("button", { name: "Guardar calificación" })
        .boundingBox();
      expect(dimensions!.height).toBeGreaterThanOrEqual(44);
      expect(
        await student.evaluate(
          () => document.documentElement.scrollWidth <= window.innerWidth,
        ),
      ).toBe(true);

      // Protect edits from client-router navigation as well as hard reloads.
      await editor
        .getByLabel("Título", { exact: true })
        .fill(title + " sin guardar");
      page.once("dialog", (dialog) => dialog.dismiss());
      await page
        .getByRole("link", { name: "← Mis cursos", exact: true })
        .click();
      await expect(page).toHaveURL(activityURL);
      await editor.getByRole("button", { name: "Guardar borrador" }).click();
      await expect(editor.getByRole("status")).toContainText(
        "Borrador guardado",
      );
      await page.getByRole("button", { name: "Salir", exact: true }).click();
      await expect(page).toHaveURL(/\/login$/);
      await page.goto(activityURL);
      await expect(page).toHaveURL(/\/login$/);
    } finally {
      await studentContext.close();
      await secondContext.close();
    }
  });
});
