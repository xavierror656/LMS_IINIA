# Rúbricas y ponderaciones: evidencia del incremento 3

Especificación previa: `openspec/changes/academic-administration/increment-3.md`, RB1–RB5; LMS-018/019/021/022. Implementación Go/PostgreSQL y Astro SSR, con custom elements para edición. Sin nuevas dependencias ni servicios.

## Comportamiento implementado

Cada tarea admite peso relativo entero 1..1000 y rúbrica opcional de hasta diez criterios, con dos a seis niveles por criterio. Los puntos son enteros crecientes desde cero; la suma de los máximos define el total de la rúbrica. El servidor transforma los niveles elegidos a nota 0..100 y redondea la mitad hacia arriba. El alumno no envía ni modifica esa nota.

Guardar evaluación incrementa la versión privada de la actividad. Publicar aplica peso y criterios. La entrega conserva los criterios de su publicación; publicar otra rúbrica no reinterpreta una respuesta existente. El docente corrige con el snapshot y la devolución publicada muestra niveles alcanzados. La edición posterior de una nota vuelve a ocultarla, conforme al flujo existente. Auditoría conserva las selecciones tanto en borrador como en publicación.

El libro muestra promedio ponderado de notas publicadas y peso evaluado/total; el contrato mantiene también el promedio aritmético. Una nota ausente no es cero y una nota oculta no participa. Los agregados recorren todas las tareas, independientemente de las columnas visibles. No se presenta como nota final.

## Archivos principales

- `backend/migrations/003_rubrics_weights.sql`: migración aditiva; datos anteriores conservan peso 1 y rúbrica null.
- `backend/internal/models/rubric.go`: validación y cálculo; `services/evaluation.go`, modificaciones de autoría/corrección y consulta de libro.
- `contracts/openapi.json`: versión 1.3.0; tipos regenerados en frontend.
- `frontend/src/pages/teacher/activities/[activityId]/evaluation.astro`, componentes `RubricFields.astro` y `RubricView.astro`; formularios y libro existentes integrados.
- `backend/internal/models/rubric_test.go`, casos RB1–RB5 de `academic_test.go` y `frontend/tests/rubric.spec.ts`.

## Validación ejecutada

- Go test con `TEST_DATABASE_URL` en `academic_test`: aprobado. Integración handlers 3.788 s; migraciones/seed ejecutados dos veces en esquema nuevo. Verifica permisos, revisiones obsoletas, selección manipulada, rechazo de nota manual, conflicto concurrente, borrador oculto, publicación idempotente, auditoría y rúbrica histórica.
- Cálculos probados: rúbrica 6/8 → 75, 1/8 → 13; ponderación (90+0+30+75×3)/6 → 57.50 y, al publicar peso 9, (90+0+30+75×9)/12 → 66.25. Peso sin publicar no altera el libro; este caso usa segunda página de columnas.
- `go vet ./...`: sin diagnósticos; API compilada. Migración aplicada a `academic_preview` sin borrar datos.
- Astro check: 39 archivos, cero errores, advertencias o hints. Build SSR aprobado; aviso previo de chunks superiores a 500 kB permanece.
- Vitest con `--pool=threads`: cuatro pruebas aprobadas, 45.38 s.
- Playwright de rúbricas: una prueba integral aprobada, 34.3 s total. Crea configuración desde UI, conserva campos tras un 400 real, recarga, publica, entrega, califica por criterios, comprueba nota oculta y devolución de 63/100 para 5/8 puntos. Capturas tablet inspeccionadas: `screenshots/rubric-editor-tablet.png` y `screenshots/rubric-feedback-tablet.png`.
- Regresión final: siete pruebas anteriores aprobadas en 15.5 s tras corregir el supuesto de XP fijo y renovar la ventana del limitador. Ocho escenarios E2E aprobados en dos ejecuciones separadas (rúbricas y regresión).
- OpenSpec strict validó el cambio. Sin auditoría WCAG completa ni pruebas de carga.

La primera regresión de siete pruebas dio cinco aprobadas y dos fallos: la prueba de aprendizaje suponía 25 XP absolutos, aunque la cuenta sintética tenía 50 por uso previo; se corrigió para verificar progreso previo + 25 solo cuando la lectura se completa por primera vez. La otra prueba recibió HTTP 429 tras ejecutar suites seguidas contra la misma IP (límite global 180/min). Se conserva la protección y se repite después de renovar la ventana; no se borraron avances ni se aumentó el límite. Queda pendiente revisar límites por usuario/proxy para uso simultáneo en aula.

## Reproducir y probar

Con las variables de entorno del README y PostgreSQL configurados:

```bash
# Terminal 1, desde backend
go run ./cmd/migrate
go run ./cmd/api
# Terminal 2, desde la raíz
npm --prefix frontend run dev
# Desde frontend, en entorno sintético separado
E2E_ACADEMIC=1 E2E_STUDENT_PASSWORD='change-this-student-password' E2E_TEACHER_PASSWORD='change-this-teacher-password' npm run test:e2e -- rubric.spec.ts
```

Dejar renovar la ventana de un minuto del limitador antes de concatenar suites completas que comparten IP. No compilar mientras se ejecutan pruebas contra el servidor dev.

Demo temporal de esta sesión: `http://localhost:4323/login`, `profe` / `academic-teacher-pass`; estudiante `luna` / `academic-student-pass`. Ruta docente: Crear actividades y calificar → Gestionar actividades → Editar una tarea → Configurar rúbrica y peso. Guardar y volver a la actividad para publicar. No está disponible en la API Node de `npm run demo`.

## Pendientes del objetivo completo

Categorías, otras agregaciones, guías de evaluación, plantillas compartidas, archivos privados, fechas/prórrogas, reentregas, cuestionarios y dominios PL1–PL7 aún tienen trabajo pendiente. La comparación con Moodle no fue ejecutada; el inventario no acredita 98 %. MockRunner permanece simulado y H5P sin paquete real verificado. No se ejecutaron Docker ni pruebas de producción en este incremento.
