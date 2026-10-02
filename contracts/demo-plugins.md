# Extensiones de demostración — LMS-013
Solo Node demo; no pertenece al contrato Go de producción. Todas las respuestas no-store.
GET /api/v1/plugins: sesión student/teacher; devuelve {items: Plugin[]}.
POST /api/v1/teacher/plugins/:id: sesión teacher y Origin local autorizado; reemplaza ajustes de id code o h5p y devuelve Plugin.
Plugin: {id, title, instructions, defaultLanguage?}. title texto recortado 1–60 caracteres; instructions 1–300. Solo code admite defaultLanguage: lesson|javascript|python; lesson conserva el lenguaje original de cada lección. Campos adicionales rechazados.
Errores: 400 entrada inválida, 401 sesión ausente, 403 rol/origen, 404 id desconocido, 500 fallo de persistencia. Sin cambios en error. Escritura JSON atómica; último guardado gana. No admite URLs, paquetes, scripts, recompensas ni cambio de estado de H5P.
Pruebas: server.test.mjs LMS-013 y tests/plugins.spec.ts.
