import http from "node:http";
import { defaultPlugins, validatePlugin } from "./plugins.mjs";
import { randomBytes, randomUUID } from "node:crypto";
import {
  existsSync,
  readFileSync,
  mkdirSync,
  writeFileSync,
  renameSync,
} from "node:fs";
import { dirname } from "node:path";
import { WebSocketServer, WebSocket } from "ws";
import {
  users,
  courses,
  modules,
  lessons,
  demoPassword,
  publicUser,
  initialState,
} from "./data.mjs";

const problem = (status, message) =>
  Object.assign(new Error(message), { status });
export function createDemoServer({
  stateFile,
  origin = "http://localhost:4321",
}) {
  if (process.env.NODE_ENV === "production")
    throw new Error("La API demo solo admite desarrollo local.");
  const allowedOrigins = new Set([origin]);
  const alternate = new URL(origin);
  if (["localhost", "127.0.0.1"].includes(alternate.hostname)) {
    alternate.hostname =
      alternate.hostname === "localhost" ? "127.0.0.1" : "localhost";
    allowedOrigins.add(alternate.origin);
  }
  let state = initialState();
  if (existsSync(stateFile)) {
    try {
      state = JSON.parse(readFileSync(stateFile, "utf8"));
      if (state.version !== 1 || !Array.isArray(state.attempts)) throw Error();
      for (const id of [1, 2]) {
        if (
          !Array.isArray(state.completed?.[id]) ||
          !Array.isArray(state.started?.[id])
        )
          throw Error();
        if (
          state.completed[id].some(
            (value) =>
              !lessons.some((l) => l.id === value && l.type === "reading"),
          )
        )
          throw Error();
        if (new Set(state.completed[id]).size !== state.completed[id].length)
          throw Error();
      }
    } catch {
      throw new Error(
        `Datos demo inválidos en ${stateFile}. Conserva una copia y renombra el archivo para comenzar otra demostración.`,
      );
    }
  }
  if (state.plugins === undefined) state.plugins = defaultPlugins();
  for (const id of ["code", "h5p"]) {
    const { id: storedId, ...settings } = state.plugins[id] ?? {};
    if (storedId !== id) throw new Error("Configuración de plugins inválida.");
    state.plugins[id] = validatePlugin(id, settings);
  }
  function save(next) {
    mkdirSync(dirname(stateFile), { recursive: true });
    const temp = `${stateFile}.tmp`;
    writeFileSync(temp, JSON.stringify(next, null, 2) + "\n", { mode: 0o600 });
    renameSync(temp, stateFile);
    state = next;
  }
  const sessions = new Map();
  const connections = new Map();
  const tokenOf = (req) =>
    /(?:^|;\s*)aq_session=([a-f0-9]{64})(?:;|$)/.exec(
      req.headers.cookie ?? "",
    )?.[1];
  function userOf(req) {
    const session = sessions.get(tokenOf(req));
    if (!session || session.expires <= Date.now())
      throw problem(401, "Vuelve a entrar con una cuenta demo.");
    return users.find((u) => u.id === session.userId);
  }
  function progress(user) {
    const completed = state.completed[user.id].length;
    return {
      userId: user.id,
      alias: user.alias,
      xp: completed * 25,
      level: 1 + Math.floor(completed / 4),
      stars: completed,
      gems: completed * 2,
      lives: 5,
      completed,
    };
  }
  function lessonFor(user, id) {
    const lesson = lessons.find((l) => l.id === id);
    if (!lesson) throw problem(404, "Lección no disponible.");
    return {
      ...lesson,
      config:
        lesson.type === "code" &&
        state.plugins.code.defaultLanguage !== "lesson"
          ? {
              ...lesson.config,
              language: state.plugins.code.defaultLanguage,
              starter:
                state.plugins.code.defaultLanguage === "python"
                  ? "# Escribe tu primera idea\nprint('¡Hola, aventura!')"
                  : "// Escribe tu primera idea\nconsole.log('¡Hola, aventura!');",
            }
          : lesson.config,
      status: state.completed[user.id].includes(id)
        ? "completed"
        : state.started[user.id].includes(id)
          ? "in_progress"
          : "available",
    };
  }
  function readBody(req) {
    return new Promise((resolve, reject) => {
      if (req.headers["content-type"]?.split(";")[0] !== "application/json") {
        req.resume();
        reject(problem(400, "Envía datos JSON."));
        return;
      }
      let length = 0;
      const chunks = [];
      req.on("data", (chunk) => {
        length += chunk.length;
        if (length > 24 * 1024) {
          reject(problem(413, "Solicitud demasiado grande."));
        } else chunks.push(chunk);
      });
      req.on("end", () => {
        try {
          resolve(JSON.parse(Buffer.concat(chunks).toString()));
        } catch {
          reject(problem(400, "Datos inválidos."));
        }
      });
      req.on("error", reject);
    });
  }
  function fields(body, allowed) {
    if (
      !body ||
      typeof body !== "object" ||
      Array.isArray(body) ||
      Object.keys(body).some((k) => !allowed.includes(k))
    )
      throw problem(400, "Datos no permitidos.");
  }
  function revoke(token) {
    sessions.delete(token);
    for (const { ws, sessionToken } of connections.values())
      if (sessionToken === token) ws.close(1000, "Sesión cerrada");
  }
  const server = http.createServer(async (req, res) => {
    const requestId = randomUUID();
    res.setHeader("Cache-Control", "no-store");
    res.setHeader("X-Request-ID", requestId);
    res.setHeader("X-Content-Type-Options", "nosniff");
    const json = (status, data) => {
      res.writeHead(status, {
        "Content-Type": "application/json; charset=utf-8",
      });
      res.end(JSON.stringify(data));
    };
    try {
      const url = new URL(req.url, origin);
      const path = url.pathname;
      const method = req.method;
      if (method !== "GET" && !allowedOrigins.has(req.headers.origin))
        throw problem(403, "Origen no permitido.");
      if (method === "GET" && ["/healthz", "/readyz"].includes(path))
        return json(200, { status: "demo" });
      if (method === "POST" && path === "/api/v1/auth/login") {
        const body = await readBody(req);
        fields(body, ["username", "password"]);
        const user = users.find((u) => u.username === body.username);
        if (!user || body.password !== demoPassword)
          throw problem(
            401,
            "Usa luna, sol, profe o admin con la contraseña aulaquest-demo.",
          );
        revoke(tokenOf(req));
        const token = randomBytes(32).toString("hex");
        sessions.set(token, {
          userId: user.id,
          expires: Date.now() + 12 * 3600_000,
        });
        res.setHeader(
          "Set-Cookie",
          `aq_session=${token}; Path=/; HttpOnly; SameSite=Lax; Max-Age=43200`,
        );
        return json(200, publicUser(user));
      }
      const user = userOf(req);
      if (method === "GET" && path === "/api/v1/auth/session")
        return json(200, publicUser(user));
      if (method === "POST" && path === "/api/v1/auth/logout") {
        revoke(tokenOf(req));
        res.setHeader(
          "Set-Cookie",
          "aq_session=; Path=/; HttpOnly; SameSite=Lax; Max-Age=0",
        );
        return json(200, { ok: true });
      }
      if (method === "GET" && path === "/api/v1/plugins")
        return json(200, { items: Object.values(state.plugins) });
      const pluginMatch = /^\/api\/v1\/(teacher|admin)\/plugins\/([^/]+)$/.exec(
        path,
      );
      if (method === "POST" && pluginMatch) {
        if (user.role !== pluginMatch[1])
          throw problem(403, "No tienes acceso a esta configuración.");
        const id = pluginMatch[2];
        if (!["code", "h5p"].includes(id))
          throw problem(404, "Plugin no disponible.");
        const body = await readBody(req);
        let plugin;
        try {
          plugin = validatePlugin(id, body);
        } catch (error) {
          throw problem(400, error.message);
        }
        const next = structuredClone(state);
        next.plugins[id] = plugin;
        save(next);
        return json(200, plugin);
      }
      if (path.startsWith("/api/v1/admin/")) {
        if (user.role !== "admin")
          throw problem(403, "Este espacio es del administrador.");
        if (method === "GET" && path === "/api/v1/admin/overview")
          return json(200, {
            users: users.map((u) => ({
              ...publicUser(u),
              username: u.username,
            })),
            courses: courses.map((c) => ({
              ...c,
              lessonCount: lessons.filter((l) =>
                modules.some((m) => m.id === l.moduleId && m.courseId === c.id),
              ).length,
            })),
            plugins: Object.keys(state.plugins).length,
          });
        throw problem(404, "Ruta administrativa no disponible.");
      }
      const page = Number(url.searchParams.get("page") ?? 1);
      if (!Number.isInteger(page) || page < 1 || page > 10000)
        throw problem(400, "Página inválida.");
      if (path.startsWith("/api/v1/teacher/")) {
        if (user.role !== "teacher")
          throw problem(403, "Este espacio es del docente.");
        if (method === "GET" && path === "/api/v1/teacher/students")
          return json(200, {
            items: page === 1 ? [publicUser(users[0])] : [],
            page,
            pageSize: 20,
          });
        if (method === "GET" && path === "/api/v1/teacher/students/1/progress")
          return json(200, {
            progress: progress(users[0]),
            courses: courses.map((c) => {
              const ids = lessons
                .filter((l) =>
                  modules.some(
                    (m) => m.id === l.moduleId && m.courseId === c.id,
                  ),
                )
                .map((l) => l.id);
              return {
                id: c.id,
                title: c.title,
                total: ids.length,
                completed: ids.filter((id) => state.completed[1].includes(id))
                  .length,
              };
            }),
          });
        throw problem(404, "Estudiante no vinculado.");
      }
      if (user.role !== "student")
        throw problem(403, "Esta aventura es para estudiantes.");
      if (method === "GET" && path === "/api/v1/me/progress")
        return json(200, progress(user));
      if (method === "GET" && path === "/api/v1/courses")
        return json(200, {
          items: courses.slice((page - 1) * 20, page * 20),
          page,
          pageSize: 20,
        });
      const courseMatch = /^\/api\/v1\/courses\/(\d+)$/.exec(path);
      if (method === "GET" && courseMatch) {
        const course = courses.find((c) => c.id === Number(courseMatch[1]));
        if (!course) throw problem(404, "Curso no disponible.");
        const courseModules = modules.filter((m) => m.courseId === course.id);
        return json(200, {
          course,
          modules: courseModules,
          lessons: lessons
            .filter((l) => courseModules.some((m) => m.id === l.moduleId))
            .map((l) => lessonFor(user, l.id)),
        });
      }
      const match =
        /^\/api\/v1\/lessons\/(\d+)(?:\/(complete|attempts))?$/.exec(path);
      if (match) {
        const lesson = lessonFor(user, Number(match[1]));
        if (method === "GET" && !match[2]) return json(200, lesson);
        if (method === "POST" && match[2] === "complete") {
          fields(await readBody(req), []);
          if (lesson.type !== "reading")
            throw problem(
              409,
              "Solo las lecturas admiten finalización declarada.",
            );
          if (!state.completed[user.id].includes(lesson.id)) {
            const next = structuredClone(state);
            next.completed[user.id].push(lesson.id);
            save(next);
          }
          return json(200, progress(user));
        }
        if (method === "POST" && match[2] === "attempts") {
          const body = await readBody(req);
          fields(body, ["verb", "score"]);
          if (lesson.type !== "h5p")
            throw problem(409, "Esta lección no es H5P.");
          if (
            !["completed", "answered", "passed", "failed"].includes(
              body.verb,
            ) ||
            (body.score !== undefined &&
              (typeof body.score !== "number" ||
                body.score < 0 ||
                body.score > 1))
          )
            throw problem(400, "Intento inválido.");
          const next = structuredClone(state);
          next.attempts.push({
            userId: user.id,
            lessonId: lesson.id,
            ...body,
            trust: "client_reported",
          });
          next.attempts = next.attempts.slice(-100);
          if (!next.started[user.id].includes(lesson.id))
            next.started[user.id].push(lesson.id);
          save(next);
          return json(200, {
            trust: "client_reported",
            progress: progress(user),
          });
        }
      }
      throw problem(404, "Ruta no disponible en demo.");
    } catch (error) {
      json(error.status ?? 500, {
        error: {
          code: String(error.status ?? 500),
          message: error.status
            ? error.message
            : "No se pudo guardar o leer la demo local.",
          requestId,
        },
      });
    }
  });
  const sockets = new WebSocketServer({
    noServer: true,
    maxPayload: 24 * 1024,
  });
  server.on("upgrade", (req, socket, head) => {
    try {
      if (req.url !== "/ws/code" || !allowedOrigins.has(req.headers.origin))
        throw problem(403, "Origin");
      const user = userOf(req);
      if (user.role !== "student" || connections.has(user.id))
        throw problem(403, "Role or connection limit");
      sockets.handleUpgrade(req, socket, head, (ws) =>
        sockets.emit("connection", ws, req, user),
      );
    } catch (e) {
      socket.end(
        `HTTP/1.1 ${e.status ?? 403} Forbidden\r\nConnection: close\r\n\r\n`,
      );
    }
  });
  sockets.on("connection", (ws, req, user) => {
    connections.set(user.id, { ws, sessionToken: tokenOf(req) });
    let run;
    let alive = true;
    const send = (type, requestId, payload, runId, seq = 0) => {
      if (ws.readyState === WebSocket.OPEN)
        ws.send(JSON.stringify({ v: 1, type, requestId, runId, seq, payload }));
    };
    const failed = (id, code, message) =>
      send("run.failed", id, { code, message });
    const finish = (status) => {
      if (!run) return;
      for (const timer of run.timers) clearTimeout(timer);
      send("run.finished", run.requestId, { status }, run.id, ++run.seq);
      run = undefined;
    };
    const heartbeat = setInterval(() => {
      try {
        userOf(req);
        if (!alive) return ws.terminate();
        alive = false;
        ws.ping();
      } catch {
        ws.close(1000, "Sesión expirada");
      }
    }, 20_000);
    ws.on("pong", () => {
      alive = true;
    });
    ws.on("message", (data, binary) => {
      let event;
      try {
        userOf(req);
        if (binary) throw Error();
        event = JSON.parse(data.toString());
        fields(event, ["v", "type", "requestId", "runId", "payload"]);
        if (
          event.v !== 1 ||
          typeof event.requestId !== "string" ||
          !event.requestId.length ||
          event.requestId.length > 64
        )
          throw Error();
        if (event.type === "run.cancel") {
          fields(event.payload, []);
          if (!run || event.runId !== run.id)
            return failed(
              event.requestId,
              "invalid_run",
              "Ejecución no disponible.",
            );
          finish("cancelled");
          return;
        }
        if (event.type !== "run.start") throw Error();
        if (run)
          return failed(
            event.requestId,
            "busy",
            "Ya hay una simulación activa.",
          );
        const p = event.payload;
        fields(p, ["lessonId", "language", "code"]);
        if (
          !["javascript", "python"].includes(p.language) ||
          typeof p.code !== "string" ||
          Buffer.byteLength(p.code) > 16384 ||
          event.runId ||
          lessonFor(user, p.lessonId).type !== "code"
        )
          throw Error();
        const next = structuredClone(state);
        if (!next.started[user.id].includes(p.lessonId))
          next.started[user.id].push(p.lessonId);
        save(next);
        run = {
          id: randomUUID(),
          requestId: event.requestId,
          seq: 1,
          timers: [],
        };
        send(
          "run.accepted",
          run.requestId,
          { simulated: true },
          run.id,
          run.seq,
        );
        run.timers.push(
          setTimeout(() => {
            if (run)
              send(
                "run.stdout",
                run.requestId,
                { text: "Ejecución simulada: tu código no se ejecuta." },
                run.id,
                ++run.seq,
              );
          }, 350),
        );
        run.timers.push(
          setTimeout(() => {
            if (run)
              send(
                "run.stdout",
                run.requestId,
                {
                  text: "El laboratorio demo está conectado por WebSocket. ¡Sigue explorando!",
                },
                run.id,
                ++run.seq,
              );
          }, 700),
        );
        run.timers.push(setTimeout(() => finish("completed"), 1200));
      } catch {
        failed(
          event?.requestId ?? "",
          "invalid_message",
          "Solicitud inválida o sesión cerrada.",
        );
      }
    });
    ws.on("error", () => {});
    ws.on("close", () => {
      clearInterval(heartbeat);
      if (run) for (const timer of run.timers) clearTimeout(timer);
      connections.delete(user.id);
    });
  });
  return {
    server,
    async listen(port = 0) {
      await new Promise((resolve, reject) => {
        server.once("error", reject);
        server.listen(port, "127.0.0.1", () => {
          server.off("error", reject);
          resolve();
        });
      });
      return server.address().port;
    },
    async close() {
      for (const { ws } of connections.values()) ws.terminate();
      await new Promise((resolve) => sockets.close(resolve));
      server.closeIdleConnections();
      await new Promise((resolve) => server.close(resolve));
    },
  };
}
