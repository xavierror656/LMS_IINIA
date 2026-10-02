import { test } from "node:test";
import assert from "node:assert/strict";
import { mkdtempSync, rmSync, readFileSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { once } from "node:events";
import WebSocket from "ws";
import { createDemoServer } from "./server.mjs";

async function fixture(t) {
  const directory = mkdtempSync(join(tmpdir(), "aulaquest-demo-test-"));
  const stateFile = join(directory, "state.json");
  let api = createDemoServer({ stateFile });
  let port = await api.listen();
  t.after(async () => {
    await api.close();
    rmSync(directory, { recursive: true });
  });
  const request = (
    path,
    {
      cookie = "",
      body,
      origin = "http://localhost:4321",
      method = body === undefined ? "GET" : "POST",
    } = {},
  ) =>
    fetch(`http://127.0.0.1:${port}/api/v1${path}`, {
      method,
      headers: {
        Cookie: cookie,
        Origin: origin,
        "Content-Type": "application/json",
      },
      ...(body !== undefined ? { body: JSON.stringify(body) } : {}),
    });
  const login = async (username = "luna") => {
    const r = await request("/auth/login", {
      body: { username, password: "aulaquest-demo" },
    });
    assert.equal(r.status, 200);
    assert.match(r.headers.get("set-cookie"), /HttpOnly/);
    return r.headers.get("set-cookie").split(";")[0];
  };
  return {
    request,
    login,
    stateFile,
    get port() {
      return port;
    },
    restart: async () => {
      await api.close();
      api = createDemoServer({ stateFile });
      port = await api.listen();
    },
  };
}
test("LMS-011 catálogo, sesiones, roles, Origin y alumno no vinculado", async (t) => {
  const f = await fixture(t);
  assert.equal((await f.request("/courses")).status, 401);
  const cookie = await f.login();
  const teacher = await f.login("profe");
  const courses = await (await f.request("/courses", { cookie })).json();
  assert.equal(courses.items.length, 2);
  const course = await (await f.request("/courses/1", { cookie })).json();
  assert.equal(course.modules.length, 2);
  assert.equal(course.lessons.length, 5);
  assert.equal((await f.request("/teacher/students", { cookie })).status, 403);
  assert.equal(
    (await f.request("/teacher/students/2/progress", { cookie: teacher }))
      .status,
    404,
  );
  assert.equal(
    (await f.request("/teacher/students/1/progress", { cookie: teacher }))
      .status,
    200,
  );
  assert.equal(
    (
      await f.request("/lessons/1/complete", {
        cookie,
        body: {},
        origin: "https://evil.example",
      })
    ).status,
    403,
  );
  assert.equal(
    (await f.request("/auth/logout", { cookie, method: "POST" })).status,
    200,
  );
  assert.equal((await f.request("/auth/session", { cookie })).status, 401);
});
test("LMS-011 recompensa concurrente única y persistencia tras reinicio", async (t) => {
  const f = await fixture(t);
  let cookie = await f.login();
  const responses = await Promise.all(
    Array.from({ length: 8 }, () =>
      f.request("/lessons/1/complete", { cookie, body: {} }),
    ),
  );
  assert.ok(responses.every((r) => r.status === 200));
  const p = await (await f.request("/me/progress", { cookie })).json();
  assert.equal(p.xp, 25);
  assert.equal(p.stars, 1);
  assert.equal(p.gems, 2);
  assert.deepEqual(
    JSON.parse(readFileSync(f.stateFile, "utf8")).completed[1],
    [1],
  );
  await f.restart();
  assert.equal((await f.request("/auth/session", { cookie })).status, 401);
  cookie = await f.login();
  assert.equal(
    (await (await f.request("/me/progress", { cookie })).json()).xp,
    25,
  );
});
test("LMS-011 H5P reportado no concede premios y payloads ajenos se rechazan", async (t) => {
  const f = await fixture(t);
  const cookie = await f.login();
  assert.equal(
    (
      await f.request("/lessons/1/complete", {
        cookie,
        body: { xp_to_add: 999 },
      })
    ).status,
    400,
  );
  assert.equal(
    (await f.request("/lessons/2/complete", { cookie, body: {} })).status,
    409,
  );
  const attempt = await (
    await f.request("/lessons/3/attempts", {
      cookie,
      body: { verb: "passed", score: 1 },
    })
  ).json();
  assert.equal(attempt.trust, "client_reported");
  assert.equal(attempt.progress.xp, 0);
});
test("LMS-011 WebSocket real, simulación y cancelación liberan ejecución", async (t) => {
  const f = await fixture(t);
  const cookie = await f.login();
  const ws = new WebSocket(`ws://127.0.0.1:${f.port}/ws/code`, {
    headers: { Origin: "http://localhost:4321", Cookie: cookie },
  });
  await once(ws, "open");
  t.after(() => ws.terminate());
  const events = [];
  ws.on("message", (data) => events.push(JSON.parse(data)));
  const wait = async (type) => {
    for (let i = 0; i < 100; i++) {
      const index = events.findIndex((e) => e.type === type);
      if (index !== -1) return events.splice(index, 1)[0];
      await new Promise((r) => setTimeout(r, 20));
    }
    throw Error(`Missing ${type}`);
  };
  const start = (requestId) =>
    ws.send(
      JSON.stringify({
        v: 1,
        type: "run.start",
        requestId,
        payload: {
          lessonId: 2,
          language: "javascript",
          code: "throw new Error()",
        },
      }),
    );
  start("first");
  const accepted = await wait("run.accepted");
  assert.equal(accepted.payload.simulated, true);
  const stdout = await wait("run.stdout");
  assert.match(stdout.payload.text, /no se ejecuta/);
  ws.send(
    JSON.stringify({
      v: 1,
      type: "run.cancel",
      requestId: "cancel",
      runId: accepted.runId,
      payload: {},
    }),
  );
  assert.equal((await wait("run.finished")).payload.status, "cancelled");
  start("second");
  assert.equal((await wait("run.accepted")).requestId, "second");
  ws.close();
  await once(ws, "close");
});
test("LMS-011 archivo corrupto se conserva", () => {
  const directory = mkdtempSync(join(tmpdir(), "aulaquest-demo-invalid-"));
  const stateFile = join(directory, "state.json");
  writeFileSync(stateFile, "{bad");
  try {
    assert.throws(
      () => createDemoServer({ stateFile }),
      /Datos demo inválidos/,
    );
    assert.equal(readFileSync(stateFile, "utf8"), "{bad");
  } finally {
    rmSync(directory, { recursive: true });
  }
});

test("LMS-011 no se permite iniciar demo en producción", () => {
  const previous = process.env.NODE_ENV;
  process.env.NODE_ENV = "production";
  try {
    assert.throws(
      () => createDemoServer({ stateFile: "/unused" }),
      /desarrollo local/,
    );
  } finally {
    if (previous === undefined) delete process.env.NODE_ENV;
    else process.env.NODE_ENV = previous;
  }
});

test("LMS-011 admite ambos orígenes loopback exactos", async (t) => {
  const f = await fixture(t);
  const response = await f.request("/auth/login", {
    origin: "http://127.0.0.1:4321",
    body: { username: "luna", password: "aulaquest-demo" },
  });
  assert.equal(response.status, 200);
  const denied = await f.request("/auth/login", {
    origin: "http://localhost:9999",
    body: { username: "luna", password: "aulaquest-demo" },
  });
  assert.equal(denied.status, 403);
});

test("LMS-013 plugins: permisos, validación, persistencia y aplicación", async (t) => {
  const f = await fixture(t);
  const student = await f.login();
  const teacher = await f.login("profe");
  const body = {
    title: "Mi laboratorio",
    instructions: "Prueba una idea.",
    defaultLanguage: "python",
  };
  assert.equal((await f.request("/plugins")).status, 401);
  assert.equal(
    (await f.request("/teacher/plugins/code", { cookie: student, body }))
      .status,
    403,
  );
  assert.equal(
    (
      await f.request("/teacher/plugins/code", {
        cookie: teacher,
        body,
        origin: "https://evil.example",
      })
    ).status,
    403,
  );
  for (const invalid of [
    { ...body, title: " " },
    { ...body, title: "a".repeat(61) },
    { ...body, defaultLanguage: "shell" },
    { ...body, enabled: true },
  ])
    assert.equal(
      (
        await f.request("/teacher/plugins/code", {
          cookie: teacher,
          body: invalid,
        })
      ).status,
      400,
    );
  assert.equal(
    (await f.request("/teacher/plugins/unknown", { cookie: teacher, body }))
      .status,
    404,
  );
  assert.equal(
    (await f.request("/teacher/plugins/code", { cookie: teacher, body }))
      .status,
    200,
  );
  assert.equal(
    (
      await f.request("/teacher/plugins/h5p", {
        cookie: teacher,
        body: { title: "Mi reto", instructions: "Explora." },
      })
    ).status,
    200,
  );
  await f.restart();
  const cookie = await f.login();
  const result = await (await f.request("/plugins", { cookie })).json();
  assert.equal(result.items[0].title, "Mi laboratorio");
  const lesson = await (await f.request("/lessons/2", { cookie })).json();
  assert.equal(lesson.config.language, "python");
  assert.match(lesson.config.starter, /print\(/);
  assert.equal(
    (await (await f.request("/me/progress", { cookie })).json()).xp,
    0,
  );
});

test("LMS-014 admin: resumen y permisos separados de estudiante/docente", async (t) => {
  const f = await fixture(t);
  assert.equal((await f.request("/admin/overview")).status, 401);
  const body = {
    title: "Reto del administrador",
    instructions: "Explora el reto.",
  };
  for (const username of ["luna", "profe"]) {
    const cookie = await f.login(username);
    assert.equal((await f.request("/admin/overview", { cookie })).status, 403);
    assert.equal(
      (await f.request("/admin/plugins/h5p", { cookie, body })).status,
      403,
    );
  }
  const cookie = await f.login("admin");
  const r = await f.request("/admin/overview", { cookie });
  assert.equal(r.status, 200);
  assert.equal(r.headers.get("cache-control"), "no-store");
  const overview = await r.json();
  assert.equal(overview.users.length, 4);
  assert.equal(overview.courses[0].lessonCount, 5);
  assert.equal(
    overview.users.some((u) => "password" in u),
    false,
  );
  assert.equal((await f.request("/teacher/students", { cookie })).status, 403);
  assert.equal((await f.request("/me/progress", { cookie })).status, 403);
  assert.equal(
    (await f.request("/admin/plugins/h5p", { cookie, body })).status,
    200,
  );
  assert.equal(
    (
      await f.request("/admin/plugins/h5p", {
        cookie,
        body: { ...body, url: "https://evil.example" },
      })
    ).status,
    400,
  );
  assert.equal(
    (
      await f.request("/admin/plugins/h5p", {
        cookie,
        body,
        origin: "https://evil.example",
      })
    ).status,
    403,
  );
  await f.request("/auth/logout", { cookie, body: {} });
  assert.equal((await f.request("/admin/overview", { cookie })).status, 401);
});
