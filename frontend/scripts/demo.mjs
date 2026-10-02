import { spawn } from "node:child_process";
import net from "node:net";
import { fileURLToPath } from "node:url";
import { createDemoServer } from "./demo/server.mjs";

const [major, minor] = process.versions.node.split(".").map(Number);
if (major < 22 || (major === 22 && minor < 12)) {
  console.error(
    "AulaQuest necesita Node 22.12 o superior. Instala Node 22 LTS y vuelve a ejecutar npm run demo.",
  );
  process.exit(1);
}
if (process.env.NODE_ENV === "production") {
  console.error("La demo solo se inicia en desarrollo local.");
  process.exit(1);
}
const root = new URL("../", import.meta.url);
let demo,
  child,
  stopping = false;
async function stop(code = 0) {
  if (stopping) return;
  stopping = true;
  if (child && child.exitCode === null) child.kill("SIGTERM");
  await demo?.close();
  process.exitCode = code;
}
try {
  // Fail rather than silently selecting another origin (cookies and CSRF depend on it).
  const check = net.createServer();
  await new Promise((resolve, reject) => {
    check.once("error", reject);
    check.listen(4321, "127.0.0.1", resolve);
  });
  await new Promise((resolve) => check.close(resolve));
  await import("./copy-h5p.mjs");
  demo = createDemoServer({
    stateFile: fileURLToPath(new URL(".demo/state.json", root)),
  });
  const port = await demo.listen();
  child = spawn(
    process.execPath,
    [
      fileURLToPath(new URL("node_modules/astro/bin/astro.mjs", root)),
      "dev",
      "--host",
      "127.0.0.1",
      "--port",
      "4321",
      "--ignore-lock",
    ],
    {
      cwd: fileURLToPath(root),
      stdio: "inherit",
      env: {
        ...process.env,
        NODE_ENV: "development",
        ASTRO_TELEMETRY_DISABLED: "1",
        API_INTERNAL_URL: `http://127.0.0.1:${port}`,
        PUBLIC_DEMO_MODE: "true",
      },
    },
  );
  console.log(
    "\nAulaQuest · DEMO LOCAL SIN DOCKER\nAbre http://localhost:4321 cuando Astro indique ready.\nEstudiante: luna | Docente: profe | Admin: admin | Contraseña: aulaquest-demo\nProgreso ficticio: frontend/.demo/state.json\nCtrl+C detiene la demo.\n",
  );
  child.on("error", async (error) => {
    console.error(error.message);
    await stop(1);
  });
  child.on("exit", (code) => {
    void stop(code ?? 0);
  });
  process.on("SIGINT", () => void stop());
  process.on("SIGTERM", () => void stop());
} catch (error) {
  console.error(
    error.code === "EADDRINUSE"
      ? "El puerto 4321 está ocupado. Cierra la otra instancia de AulaQuest y vuelve a intentar."
      : error.message,
  );
  await stop(1);
}
