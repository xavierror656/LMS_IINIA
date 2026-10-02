# Animaciones y edición de plugins — LMS-012 / LMS-013

Implementación incremental sobre la demo local existente. Especificaciones en `openspec/changes/playful-motion` y `openspec/changes/teacher-plugin-settings`. Contrato adicional en `contracts/demo-plugins.md`; no modifica ni amplía la API Go.

- Entradas CSS finitas de tarjetas, mapa e ilustración; interacciones táctiles y de foco.
- HUD con barra animada y celebración breve de aumentos canónicos. Movimiento reducido conserva texto y suprime partículas.
- Formularios Astro docentes para registro cerrado code/h5p. Validación de campos, rol y Origin en servidor Node. Persistencia JSON atómica compatible con archivos de demo anteriores.
- Nombres e instrucciones aparecen en lecciones; lenguaje opcional reemplaza ejemplo inicial. H5P continúa pendiente de paquete válido.

Pruebas: `frontend/scripts/demo/server.test.mjs` cubre persistencia tras reinicio, permisos y entradas inválidas. `frontend/tests/plugins.spec.ts` cubre edición, recarga y vista de estudiante, restaurando ajustes. `frontend/tests/motion.spec.ts` utiliza respuestas API controladas para comprobar aumentos, ausencia de celebración al recargar y preferencias de movimiento. Resultados finales se registran al concluir validación.

## Resultados ejecutados

- OpenSpec validación estricta: ambos cambios válidos.
- Astro check: 27 archivos, 0 errores, 0 warnings, 0 hints.
- Build SSR: correcto; continúa el aviso de bundle superior a 500 kB del editor.
- API demo: 8/8 pruebas correctas.
- Playwright: 8 escenarios aprobados entre la ejecución inicial y la repetición de los 4 que comenzaron antes de que Astro estuviera listo (ERR_CONNECTION_REFUSED). Las pruebas nuevas de animaciones normal/reducida y edición docente aprobaron.
- No se ejecutaron pruebas Go/PostgreSQL en este incremento; no se modificaron. La edición de plugins de producción y H5P con paquete real siguen pendientes.


## Cierre administrativo — 2026-10-02

LMS-014: páginas `/admin` y `/admin/plugins`, sesión admin demo, navegación por rol y editor Astro compartido en `frontend/src/components/astro/PluginSettings.astro`. Contrato local `contracts/demo-admin.md`. Cuenta admin / aulaquest-demo, inicio con `npm run demo` desde la raíz (Node >=22.12).

Validación ejecutada con Node 22.23.3, Chromium 153.0.8010.12 y viewport Playwright 820×1180:

- `npm --prefix frontend run check`: 30 archivos, 0 errores, 0 warnings, 0 hints.
- `npm --prefix frontend run build`: correcto; persiste aviso de chunk mayor a 500 kB.
- `node --test frontend/scripts/demo/server.test.mjs`: 9/9 aprobadas. Incluye autorización administrativa, rechazo de Origin ajeno, validación, persistencia y cierre de sesión.
- `E2E_DEMO=1 E2E_STUDENT_PASSWORD=aulaquest-demo E2E_TEACHER_PASSWORD=aulaquest-demo npm --prefix frontend run test:e2e`: 10/10 aprobadas en 46.5 s, sin omitidas. Incluye admin, plugins, estudiante/docente, error de API y movimiento normal/reducido. Los ajustes usados por pruebas se restauran.

Se repitió Astro check con permisos ampliados porque la ruta del workspace difiere en mayúsculas de su resolución en Windows; el primer intento falló EROFS. Chromium y bibliotecas se recuperaron en /tmp tras reiniciarse el entorno. No se modificaron dependencias del proyecto.

Pendiente fuera de este incremento: CRUD de usuarios/cursos, rol admin y plugins en Go/PostgreSQL, contenido H5P real y runner aislado. No se repitieron pruebas Go porque su código no cambió. La demo sigue siendo local y no apta para producción.
