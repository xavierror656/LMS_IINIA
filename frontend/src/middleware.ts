import { defineMiddleware } from "astro:middleware";
import { api, ApiError, type SessionUser, roleHome } from "./lib/api";
export const onRequest = defineMiddleware(async (context, next) => {
  const privatePage = /^\/(courses|lessons|teacher|admin)(\/|$)/.test(
    context.url.pathname,
  );
  if (privatePage) {
    try {
      context.locals.user = await api<SessionUser>(
        "/auth/session",
        {},
        context.request.headers.get("cookie") ?? "",
      );
    } catch (error) {
      if (error instanceof ApiError && error.status === 401)
        return context.redirect("/login");
      context.locals.apiError =
        error instanceof Error ? error.message : "Servicio no disponible";
    }
    const user = context.locals.user;
    if (user) {
      const expectedRole = context.url.pathname.startsWith("/admin")
        ? "admin"
        : context.url.pathname.startsWith("/teacher")
          ? "teacher"
          : "student";
      if (user.role !== expectedRole)
        return context.redirect(roleHome(user.role));
    }
  }
  const response = await next();
  if (privatePage || context.url.pathname === "/login")
    response.headers.set("Cache-Control", "private, no-store");
  response.headers.set("X-Content-Type-Options", "nosniff");
  response.headers.set("Referrer-Policy", "same-origin");
  return response;
});
