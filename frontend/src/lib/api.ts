import type { components } from "./generated/api";
export type User = components["schemas"]["User"];
/** Admin is a local demo extension, not a production Go role. */
export type SessionUser = Omit<User, "role"> & { role: User["role"] | "admin" };
export const roleHome = (role?: SessionUser["role"]) =>
  role === "admin" ? "/admin" : role === "teacher" ? "/teacher" : "/courses";
export type Progress = components["schemas"]["Progress"];
export type Course = components["schemas"]["Course"];
export type Lesson = components["schemas"]["Lesson"];
export type CourseMap = components["schemas"]["CourseMap"];
export type StudentProgress = components["schemas"]["StudentProgress"];
export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
  }
}
export async function api<T>(
  path: string,
  options: RequestInit = {},
  cookie?: string,
): Promise<T> {
  const server = typeof window === "undefined";
  const base = server
    ? process.env.API_INTERNAL_URL ||
      import.meta.env.API_INTERNAL_URL ||
      "http://127.0.0.1:8080"
    : "";
  const headers = new Headers(options.headers);
  if (options.body && !(options.body instanceof FormData)) headers.set("Content-Type", "application/json");
  if (server && cookie) {
    const token = cookie
      .split(";")
      .map((s) => s.trim())
      .find((s) => /^aq_session=[a-f0-9]{64}$/.test(s));
    if (token) headers.set("Cookie", token);
  }
  let response: Response;
  try {
    response = await fetch(base + "/api/v1" + path, {
      ...options,
      headers,
      credentials: "same-origin",
      cache: "no-store",
      signal: options.signal ?? AbortSignal.timeout(8000),
    });
  } catch {
    throw new ApiError(
      503,
      "No podemos conectar. Tu progreso guardado sigue en el servidor. Vuelve a intentarlo.",
    );
  }
  if (!response.ok) {
    const body = (await response.json().catch(() => null)) as {
      error?: { message?: string };
    } | null;
    throw new ApiError(
      response.status,
      body?.error?.message || "No se pudo completar la solicitud.",
    );
  }
  return response.json() as Promise<T>;
}

export type Activity = components["schemas"]["Activity"];
export type ActivityList = components["schemas"]["ActivityList"];
export type Submission = components["schemas"]["Submission"];
export type Grade = components["schemas"]["Grade"];

export type Gradebook = components["schemas"]["Gradebook"];
export type GradeCategory = components["schemas"]["GradeCategory"];
export type GradeCategoryList = components["schemas"]["GradeCategoryList"];
export type StudentGrades = components["schemas"]["StudentGrades"];

export type Rubric = components["schemas"]["Rubric"];
export type RubricAssessment = components["schemas"]["RubricAssessment"];

export type Schedule = components["schemas"]["Schedule"];
export type Availability = components["schemas"]["Availability"];
export type ExtensionList = components["schemas"]["ExtensionList"];

export type Attachment = components["schemas"]["Attachment"];
