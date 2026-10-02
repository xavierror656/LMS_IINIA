import { cp, mkdir } from "node:fs/promises";
const source = new URL("../node_modules/h5p-standalone/dist/", import.meta.url);
const dest = new URL("../public/h5p/player/", import.meta.url);
await mkdir(dest, { recursive: true });
for (const file of ["frame.bundle.js", "styles", "fonts"]) {
  try {
    await cp(new URL(file, source), new URL(file, dest), { recursive: true });
  } catch (e) {
    if (e.code !== "ENOENT") throw e;
  }
}
await cp(
  new URL("../node_modules/h5p-standalone/LICENSE", import.meta.url),
  new URL("LICENSE", dest),
).catch(() => {});
