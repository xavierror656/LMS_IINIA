# Categorías de calificación y agregación: evidencia del incremento 17

Especificación previa: `openspec/changes/academic-administration/increment-17.md`, GC1–GC8; T4/T4h, LMS-018 / A4. Implementación Go/PostgreSQL y Astro SSR con un custom element para administrar categorías. Sin dependencias nuevas del producto.

## Comportamiento implementado

El docente del curso administra categorías planas con nombre único (sin distinguir mayúsculas ni espacios sobrantes), peso entero 1..1000, posición y versión optimista; el máximo es cien por curso. Crear agrega al final; reordenar exige la lista completa del curso y rechaza listas incompletas, repetidas o ajenas; eliminar una categoría con actividades responde 409 y no reasigna nada en silencio. Docente ajeno 404, alumno 403, categoría de otro curso no se puede asignar (400) ni operar (404).

Publicar (o crear) sin categoría usa la primera del curso por posición y crea «General» con peso 1 si no existe. Publicar fija la categoría en `activity_publications` y `lessons`, igual que el peso y la rúbrica; un `CHECK` impide que una lección publicada de tarea o cuestionario quede sin categoría. La migración asigna «General» a todo lo existente, de modo que una sola categoría con peso 1 reproduce el promedio anterior.

El total de categoría es la media ponderada de las actividades **publicadas** con nota publicada: `sum(score × grade_weight) / sum(grade_weight)`, redondeo mitad arriba a centésimas y `null` sin ítems. El total del curso es `sum(total_categoría × peso_categoría) / sum(peso_categoría)` solo sobre categorías con total, también `null` sin datos. La política de faltantes del curso es `exclude` (predeterminada) o `zero`: con `zero` toda actividad publicada participa con 0 cuando no tiene nota publicada; una nota en borrador se trata como ausente, nunca como cero revelable. El alumno consulta su propia vista con categorías, total e ítems, siempre solo con notas publicadas.

El libro muestra el total del curso, el total por categoría y conserva el promedio publicado como dato de cobertura; el CSV añade una columna `Total <categoría>` y `Total del curso` con los mismos números y neutraliza fórmulas también en el nombre de la categoría. Las pantallas vacías, de error y de política explican el cálculo y aclaran que no es un boletín oficial.

## Archivos principales

- `backend/migrations/015_grade_categories.sql`: categorías, política de faltantes, FKs en borradores/publicaciones/lecciones, backfill «General» y `CHECK` de lección calificable.
- `backend/internal/models/grade_category.go` y `gradebook.go`; `services/grade_categories.go`, `student_grades.go`, cambios en `gradebook.go`, `evaluation.go`, `quiz.go` y `academic.go` (publicación y `courseId` de lectura); `repositories/gradebook.go`; handlers `grade_categories.go`, `gradebook.go` y rutas.
- `contracts/openapi.json` 1.10.0 con los esquemas y cinco rutas nuevas; tipos regenerados en `frontend/src/lib/generated/api.d.ts`.
- `frontend/src/components/astro/GradeCategories.astro`, `RubricFields.astro`, `QuizComposer.astro`; páginas del libro, evaluación, cuestionario y curso del alumno.

## Validación ejecutada

- Unitarias: `CategoryAverage` (excluir/cero, nulo y redondeo medio arriba), `WeightedHundredths` (80.00×60 + 60.00×40 = 72.00, una categoría cancela su peso) y validación de nombre, peso y política. Aprobadas.
- Integración PostgreSQL 18.6 real con `TEST_DATABASE_URL` en base `_test`: `go test ./... -count=1` completo aprobado; `internal/handlers` 19.4 s y `migrations` 0.08 s. La prueba `TestGradeCategoriesIntegration` aplica migraciones y seed dos veces en un esquema nuevo y verifica: creación automática de General, duplicado 409, edición obsoleta 409, alta/orden/borrado con 409 si hay actividades, reordenamiento incompleto/repetido/ajeno 400, categoría ajena 400 y 404, publicación con categoría fija, totales con `exclude` (80.00 y curso 80.00), política `zero` con borrador oculto contado como cero (curso 26.67) y publicado después (curso 90.00), CSV con columnas y cifras idénticas, vista del alumno solo con publicadas, 403/404 de rol y alumno no inscrito.
- `TestGradeCategoryBackfill` (paquete `migrations`): aplica 001–014 a mano, inserta una tarea heredada con peso 3 y nota 80 publicada, ejecuta 015 y comprueba «General», la propagación a lección, borrador y publicación, la política por defecto y que el peso no cambia. Aprobada.
- `go vet ./...`: sin diagnósticos.
- Astro check: 61 archivos, 0 errores, 0 advertencias. Build SSR completado.
- Vitest: 4 pruebas aprobadas (51 s).
- Playwright contra Go/PostgreSQL/Astro reales con datos sintéticos: nueva suite `grade-categories.spec.ts` aprobada (crea categoría y cambia la política desde la interfaz, asigna la categoría al publicar, comprueba totales en el libro y en el CSV y la vista «Mis notas» del alumno tras recargar). Regresión `rubric.spec.ts` y `gradebook.spec.ts` aprobada tras ajustar el selector del encabezado del libro. Capturas: `screenshots/grade-categories-tablet.png` y `screenshots/student-grades-tablet.png`.

Dos defectos se detectaron y corrigieron durante la verificación, no después: el reordenamiento aceptaba una lista que no cubría todas las categorías (el servicio ahora compara contra el total del curso) y la vista del alumno exponía la nota en borrador de un ítem porque la vista `gradebook_entries` publica el score aunque el estado sea `draft` (la consulta ahora solo proyecta el score con estado `published`). Ambos casos quedaron cubiertos por la prueba de integración.

Entorno: WSL2 Debian 13, Go 1.27.1 oficial, Node 22.23.3, PostgreSQL 18.6 mediante binarios zonky y Chromium headless con bibliotecas extraídas sin privilegios de administrador. Docker, Nginx y pruebas con lectores de pantalla no se ejecutaron.

## Reproducir y probar

```bash
# Terminal 1, desde backend (DATABASE_URL y TEST_DATABASE_URL configurados)
go run ./cmd/migrate
go run ./cmd/seed
go run ./cmd/api
# Terminal 2, desde la raíz
npm --prefix frontend run dev
# Desde frontend, contra la base sintética
E2E_ACADEMIC=1 E2E_STUDENT_PASSWORD='...' E2E_TEACHER_PASSWORD='...' npm run test:e2e -- grade-categories.spec.ts
```

La validación estricta de OpenSpec del cambio se ejecutó con la CLI 1.14.0 (`openspec validate academic-administration --strict --no-interactive`) y aprobó.

## Pendientes del objetivo completo

Categorías anidadas, otros métodos de agregación (suma, mediana, natural, más baja), exclusión de la nota más mínima, exenciones individuales (`exento`), escalas cualitativas, importación/exportación en otros formatos y cálculos personalizados siguen fuera de este incremento. AC1 (accesibilidad) continúa sin auditoría completa, la comparación con Moodle no se ejecutó y no se declara cobertura del 98 % ni paridad. MockRunner sigue simulado y H5P sin paquete real verificado.
