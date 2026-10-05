import { test, expect, type Page, type APIRequestContext } from "@playwright/test";

type Finding = { rule: string; page: string; detail: string };

async function auditPage(page: Page, path: string, findings: Finding[]) {
  const issues = await page.evaluate(() => {
    const out: { rule: string; detail: string }[] = [];
    const label = (el: Element) =>
      ((el.getAttribute("aria-label") || el.textContent || "").replace(/\s+/g, " ").trim()).slice(0, 50);
    const headings = [...document.querySelectorAll("h1,h2,h3,h4,h5,h6")];
    const h1 = headings.filter((h) => h.tagName === "H1");
    if (h1.length !== 1) out.push({ rule: "encabezados", detail: `${h1.length} elementos h1` });
    let previous = 0;
    for (const heading of headings) {
      const level = Number(heading.tagName[1]);
      if (previous && level > previous + 1) {
        out.push({ rule: "encabezados", detail: `salto h${previous}→h${level}: ${label(heading)}` });
      }
      previous = level;
    }
    for (const field of document.querySelectorAll("input,select,textarea")) {
      const input = field as HTMLInputElement;
      if (input.type === "hidden") continue;
      const associated = input.id && document.querySelector(`label[for="${CSS.escape(input.id)}"]`);
      const wrapped = input.closest("label");
      const aria = input.getAttribute("aria-label") || input.getAttribute("aria-labelledby");
      if (!associated && !wrapped && !aria) out.push({ rule: "etiquetas", detail: input.outerHTML.slice(0, 100) });
    }
    for (const control of document.querySelectorAll("button,a[href]")) {
      if (!control.getAttribute("aria-label") && !(control.textContent || "").trim()) {
        out.push({ rule: "nombres", detail: control.outerHTML.slice(0, 100) });
      }
    }
    for (const target of document.querySelectorAll("button,a[href],input,select,textarea,summary")) {
      const rect = target.getBoundingClientRect();
      if (rect.width === 0 || rect.height === 0) continue;
      let width = rect.width;
      let height = rect.height;
      // A form control inside its label is clicked through the label: the union
      // of both rectangles is the real touch target.
      const associated =
        target.closest("label") ??
        (target.id ? document.querySelector(`label[for="${CSS.escape(target.id)}"]`) : null);
      if (associated) {
        const labelRect = associated.getBoundingClientRect();
        const left = Math.min(rect.left, labelRect.left);
        const top = Math.min(rect.top, labelRect.top);
        const right = Math.max(rect.right, labelRect.right);
        const bottom = Math.max(rect.bottom, labelRect.bottom);
        width = right - left;
        height = bottom - top;
      }
      if (height < 44 || width < 44) {
        out.push({ rule: "objetivos", detail: `${target.tagName} ${label(target)} · ${Math.round(width)}×${Math.round(height)}` });
      }
    }
    const running = [...document.querySelectorAll("*")].filter((el) => {
      const style = getComputedStyle(el);
      return style.animationName !== "none" && style.animationIterationCount === "infinite" && parseFloat(style.animationDuration) > 0;
    });
    for (const el of running) out.push({ rule: "movimiento", detail: `animación infinita en ${el.tagName}.${el.className}` });
    return out;
  });
  for (const issue of issues) findings.push({ ...issue, page: path });
}

test("AC1 etiquetas, encabezados, foco, objetivos, contraste y movimiento", async ({ page, browser }) => {
  test.skip(process.env.E2E_ACADEMIC !== "1", "Requiere Go/PostgreSQL sintético");
  test.setTimeout(180_000);
  const origin = process.env.E2E_BASE_URL ?? "http://localhost:4321";
  const findings: Finding[] = [];
  async function mutate(request: APIRequestContext, method: string, path: string, data: unknown) {
    const res = await request.fetch(`/api/v1${path}`, { method, data, headers: { Origin: origin } });
    expect(res.ok(), await res.text()).toBeTruthy();
    return res.json();
  }
  await page.goto("/login");
  findings.length = 0;
  await page.evaluate(() => {
    if (!matchMedia("(prefers-reduced-motion: reduce)").matches) throw new Error("reduced motion not honored");
  });
  await auditPage(page, "/login", findings);
  await page.getByLabel("Tu usuario").fill("profe");
  await page.getByLabel("Tu contraseña").fill(process.env.E2E_TEACHER_PASSWORD!);
  await page.getByRole("button", { name: "Entrar a mi aventura" }).click();
  await expect(page).toHaveURL(/\/teacher$/);
  const courses = await (await page.request.get("/api/v1/teacher/courses")).json();
  const courseId = courses.items[0].id;
  const activities = await (await page.request.get(`/api/v1/teacher/courses/${courseId}/activities`)).json();
  let activity = activities.items.find((item: { type: string }) => item.type === "assignment");
  if (!activity) {
    activity = await mutate(page.request, "POST", `/teacher/courses/${courseId}/activities`, {
      moduleId: activities.modules[0].id,
      title: `Accesibilidad ${Date.now()}`,
      description: "",
      type: "assignment",
      body: "Revisión de accesibilidad",
    });
    activity = await mutate(page.request, "POST", `/teacher/activities/${activity.id}/publish`, { version: 1 });
  }
  for (const path of [
    "/teacher",
    `/teacher/courses/${courseId}`,
    `/teacher/courses/${courseId}/gradebook`,
    `/teacher/activities/${activity.id}/evaluation`,
    `/teacher/courses/${courseId}/questions`,
  ]) {
    await page.goto(path);
    await auditPage(page, path, findings);
  }
  // Focus must stay visible along the first keyboard stops of a dense page.
  await page.goto(`/teacher/courses/${courseId}/gradebook`);
  const focusStops: string[] = [];
  for (let i = 0; i < 8; i++) {
    await page.keyboard.press("Tab");
    focusStops.push(
      await page.evaluate(() => {
        const el = document.activeElement as HTMLElement | null;
        const style = el ? getComputedStyle(el) : null;
        const visible = !!style && ((style.outlineStyle !== "none" && parseFloat(style.outlineWidth) > 0) || style.boxShadow !== "none");
        return `${el?.tagName ?? "?"}:${visible ? "foco" : "sin-foco"}`;
      }),
    );
  }
  const withoutFocus = focusStops.filter((stop) => stop.endsWith("sin-foco"));
  expect(withoutFocus, `Foco no visible en ${withoutFocus.join(", ")}`).toEqual([]);

  const student = await browser.newContext({ viewport: { width: 820, height: 1180 }, reducedMotion: "reduce" });
  const studentPage = await student.newPage();
  try {
    await studentPage.goto("/login");
    await studentPage.getByLabel("Tu usuario").fill("luna");
    await studentPage.getByLabel("Tu contraseña").fill(process.env.E2E_STUDENT_PASSWORD!);
    await studentPage.getByRole("button", { name: "Entrar a mi aventura" }).click();
    await expect(studentPage).toHaveURL(/\/courses$/);
    const map = await (await studentPage.request.get(`/api/v1/courses/${courseId}`)).json();
    const reading = map.lessons.find((lesson: { type: string }) => lesson.type === "reading") ?? map.lessons[0];
    for (const path of ["/courses", `/courses/${courseId}`, `/lessons/${reading.id}`]) {
      await studentPage.goto(path);
      await auditPage(studentPage, path, findings);
    }
  } finally {
    await student.close();
  }
  expect(
    findings.filter((finding) => finding.rule === "encabezados" || finding.rule === "etiquetas" || finding.rule === "nombres"),
    JSON.stringify(findings, null, 2),
  ).toEqual([]);
  // Touch targets smaller than 44 px are recorded as findings; this suite fails
  // so the list stays explicit instead of silently ignored.
  expect(findings, JSON.stringify(findings, null, 2)).toEqual([]);
});
