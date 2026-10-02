export const defaultPlugins = () => ({
  code: {
    id: "code",
    title: "Laboratorio de código",
    instructions:
      "Explora tus ideas con JavaScript o Python. La ejecución es simulada.",
    defaultLanguage: "lesson",
  },
  h5p: {
    id: "h5p",
    title: "Actividad interactiva",
    instructions: "Explora el reto que prepara tu docente.",
  },
});
export function validatePlugin(id, body) {
  const allowed =
    id === "code"
      ? ["title", "instructions", "defaultLanguage"]
      : ["title", "instructions"];
  if (
    !["code", "h5p"].includes(id) ||
    !body ||
    typeof body !== "object" ||
    Array.isArray(body) ||
    Object.keys(body).some((k) => !allowed.includes(k))
  )
    throw new Error("Datos no permitidos.");
  for (const [key, max] of [
    ["title", 60],
    ["instructions", 300],
  ]) {
    if (
      typeof body[key] !== "string" ||
      !body[key].trim() ||
      body[key].trim().length > max
    )
      throw new Error(
        `Revisa el nombre (1–60) y las instrucciones (1–300 caracteres).`,
      );
  }
  if (
    id === "code" &&
    !["lesson", "javascript", "python"].includes(body.defaultLanguage)
  )
    throw new Error("Selecciona un lenguaje válido.");
  return {
    id,
    title: body.title.trim(),
    instructions: body.instructions.trim(),
    ...(id === "code" ? { defaultLanguage: body.defaultLanguage } : {}),
  };
}
