# Plan de validación

Plan completo; el subconjunto del incremento 1 ya tiene pruebas ejecutadas, registradas en `docs/academic-validation.md`. Las pruebas de funciones posteriores siguen pendientes. IDs A1..A9 corresponden a escenarios del spec. Cada prueba debe registrar comando, fecha, entorno, resultado y artefactos. El paso T0 añadirá casos por capacidad Moodle; estos nueve escenarios no bastan para certificar 98 %.

| ID | Prueba requerida | Evidencia mínima |
|---|---|---|
| A1 | Autoría y publicación con docente propio/ajeno; borrador privado | Handler Go + PostgreSQL + E2E docente/estudiante |
| A2 | Guardar, enviar, recargar; reintento idéntico, revisión conflictiva y dos envíos concurrentes | Integración PostgreSQL con recuento/versión + E2E |
| A3 | Borrador de nota invisible, publicación visible, dos correctores editando | Transacciones + 409 de versión obsoleta + historial intacto |
| A4 | 80/100 × 60 % + 30/50 × 40 % = 72; faltante/exento/cero y nota oculta | Casos unitarios de agregación y API de estudiante |
| A5 | Rúbrica nueva no altera nota previa | Integración con dos versiones y recarga |
| A6 | Score manipulado, respuestas privadas, reloj/intententos y reintento | Handler, servicio y navegador; claves ausentes de HTML/JSON |
| A7 | Otro curso/estudiante, archivos, CSV, sesión revocada y auditoría | Matriz positiva/negativa por endpoint, sin cambios ante rechazo |
| A8 | Teclado, tablet 768×1024, movimiento reducido, caída y recuperación | Playwright con capturas y aserciones de estado/foco |
| A9 | Denominador incompleto y pruebas omitidas no producen certificado | Informe por capacidad; comprobación exacta de fórmula |

Ejecutar pruebas PostgreSQL en base separada terminada en _test, nunca sobre datos del usuario. Sembrar perfiles sintéticos de dos cursos y dos docentes; no asumir que el vínculo teacher_students basta para autoría de curso. Añadir regresiones de HUD/logout y recompensas idempotentes antes de integrar notas con progreso.

Comandos existentes a ejecutar cuando haya código: npm --prefix frontend run check; npm --prefix frontend test; npm --prefix frontend run build; npm --prefix frontend run test:e2e con entorno preparado; desde backend, go test ./... y go vet ./...; integración con TEST_DATABASE_URL separada. No atribuir a estos comandos cobertura que todavía no contienen.

Validación del incremento 1: integración Go/PostgreSQL, Astro/TypeScript, build SSR, stores y navegador. OpenSpec 1.14.0 validó el cambio en modo estricto. No se ejecutó una comparación funcional en Moodle ni una certificación de paridad. A4–A6 y A9 completos siguen pendientes; A7 no acredita todavía archivos ni CSV. Véase el informe de resultados para el alcance exacto.
