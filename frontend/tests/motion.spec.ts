import { test, expect } from "@playwright/test";
const password = process.env.E2E_STUDENT_PASSWORD;
for (const reducedMotion of ["no-preference", "reduce"] as const) {
  test.describe(`LMS012 motion=${reducedMotion}`, () => {
    test.use({ reducedMotion });
    test("entrada finita y celebración solo tras progreso confirmado", async ({
      page,
    }) => {
      test.skip(!password, "E2E_STUDENT_PASSWORD required");
      let progress = {
        userId: 1,
        alias: "Luna",
        xp: 75,
        level: 1,
        stars: 3,
        gems: 6,
        lives: 5,
        completed: 3,
      };
      // Simulates consecutive canonical API responses; never changes demo/production data.
      await page.route("**/api/v1/me/progress", (route) =>
        route.fulfill({ json: progress }),
      );
      await page.goto("/login");
      await page.getByLabel("Tu usuario").fill("luna");
      await page.getByLabel("Tu contraseña").fill(password!);
      await page.getByRole("button", { name: "Entrar a mi aventura" }).click();
      await expect(page.getByLabel("Tu progreso")).toContainText("75 XP");
      await expect(page.locator(".reward-toast")).toHaveCount(0);
      const animation = await page
        .locator(".course-card")
        .first()
        .evaluate((el) => ({
          name: getComputedStyle(el).animationName,
          iterations: getComputedStyle(el).animationIterationCount,
        }));
      expect(animation.name).toBe(
        reducedMotion === "reduce" ? "none" : "quest-arrive",
      );
      if (reducedMotion !== "reduce") expect(animation.iterations).toBe("1");
      progress = {
        ...progress,
        xp: 100,
        level: 2,
        stars: 4,
        gems: 8,
        completed: 4,
      };
      await page.getByRole("link", { name: "Ver mi camino" }).first().click();
      await expect(page.locator(".reward-toast")).toContainText(
        "¡Llegaste al nivel 2!",
      );
      await expect(page.locator(".reward-toast")).toContainText("+25 XP");
      await expect(page.locator(".reward-sparks")).toHaveCount(
        reducedMotion === "reduce" ? 0 : 1,
      );
      await expect(page.getByLabel("Tu progreso")).toContainText("100 XP");
      await expect(page.locator(".reward-toast")).toHaveCount(0, {
        timeout: 5000,
      });
      await page.reload();
      await expect(page.getByLabel("Tu progreso")).toContainText("100 XP");
      await expect(page.locator(".reward-toast")).toHaveCount(0);
      expect(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth,
        ),
      ).toBe(true);
    });
  });
}
