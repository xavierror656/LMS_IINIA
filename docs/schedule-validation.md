# Fechas y prórrogas: incremento 4

Especificación previa: `openspec/changes/academic-administration/increment-4.md`, DT1–DT5 / LMS-016/021/022 / MDL-003..005. El objetivo general del 98 % permanece activo y sin acreditar.

## Implementación

Calendario opcional de apertura, vencimiento y cierre por tarea; borrador/revisión de actividad y publicación explícita. Horas RFC3339 en API, timestamptz en PostgreSQL y zona UTC indicada en formularios y vistas. El frontend no decide autorización con su reloj.

Vencimiento permite enviar tarde hasta cierre; cierre y apertura bloquean guardar/enviar, manteniendo lectura de instrucciones y borrador guardado. El docente concede/revoca prórrogas a estudiantes vinculados e inscritos en un curso asignado. Se exige motivo, revisión optimista y se conserva auditoría. Una entrega definitiva registra vencimiento efectivo y tardanza; cambios posteriores y reintentos no reescriben esos datos ni otorgan recompensas.

Se adquiere bloqueo compartido de lección antes de matrícula y entrega. Los escritores de calendario/prórroga bloquean la lección exclusivamente. El envío consulta `clock_timestamp()` después de esperar los bloqueos; no utiliza la hora del inicio de transacción para aceptar un plazo vencido.

Archivos: `004_assignment_schedule.sql`, `models/schedule.go`, `services/schedule.go`, `handlers/schedule.go`; OpenAPI 1.4.0 y tipos generados. Páginas docentes `schedule.astro`/`extensions.astro`, componente `ScheduleView.astro` y formulario de tarea integrados sin React adicional ni nuevas dependencias.

## Evidencia ejecutada

- `go test -count=1 ./...` con PostgreSQL separado: aprobado. Integración handlers 3.776 s. Migraciones/seed repetidos; fronteras exactas en pruebas unitarias (apertura inclusive, vencimiento puntual, cierre exclusivo), fechas inválidas, horario privado no filtrado, envío tardío, lectura tras cierre, bloqueo de escritura, permiso de prórroga, alumno no vinculado rechazado, conflicto concurrente de revisiones, revocación y auditoría.
- Prueba de carrera: una transacción modifica cierre y retiene bloqueo de lección; un envío concurrente espera y, tras commit, devuelve 409. También comprueba que no cambia clasificación al repetir envío ya definitivo después del cierre.
- `go vet ./...`: sin diagnósticos; API compilada. Migración 004 aplicada a academic_preview sin borrar datos existentes.
- Astro check: 42 archivos, cero errores, warnings/hints. Build SSR aprobado, aviso existente de chunks mayores de 500 kB permanece.
- E2E `schedule.spec.ts`: aprobada, 19.8 s total. Crea calendario desde formulario, publica, verifica instrucciones accesibles y 409 al intentar eludir cierre, otorga prórroga desde UI, permite guardar y enviar tarde, revoca y confirma historial inmutable. Capturas tablet inspeccionadas: `screenshots/extensions-tablet.png`, `screenshots/schedule-student-tablet.png`.
- Vitest `--pool=threads`: cuatro pruebas aprobadas, 27.46 s.
- Regresión final `academic.spec.ts gradebook.spec.ts learning.spec.ts`: siete pruebas aprobadas, 32.9 s. Con DT5, ocho escenarios de navegador aprobados en dos ejecuciones separadas.
- OpenSpec strict validó el cambio. Sin auditoría WCAG completa ni pruebas de carga.

## Acceso y reproducción

Demo temporal: `http://localhost:4323/login`; docente `profe` / `academic-teacher-pass`, alumno `luna` / `academic-student-pass`. En una tarea: **Configurar fechas**, guardar y volver para **Publicar actividad**. Tras publicar aparece **Gestionar prórrogas**; sus cambios se aplican al guardar. Fechas explícitas UTC, no hora local implícita.

Con configuración del README:

```bash
# backend, terminal 1
go run ./cmd/migrate
go run ./cmd/api
# raíz, terminal 2
npm --prefix frontend run dev
# frontend, entorno sintético con servidores activos
E2E_ACADEMIC=1 E2E_STUDENT_PASSWORD='change-this-student-password' E2E_TEACHER_PASSWORD='change-this-teacher-password' npm run test:e2e -- schedule.spec.ts
```

Los escenarios E2E agregan datos sintéticos y no eliminan avances. Separar suites intensivas por la ventana del limitador de solicitudes existente; no compilar en paralelo con el servidor dev durante E2E. Este flujo es Go/PostgreSQL y no está en `npm run demo`.

## Límites pendientes

Sin selector IANA/localización de edición ni excepción de apertura individual; UTC explícito evita conversiones ambiguas por horario estacional. Las entregas previas a migración tienen fecha efectiva null (sin evidencia de vencimiento), no se recalifican retrospectivamente. Falta administración de historial de prórrogas desde UI; auditoría persistida en PostgreSQL. Reentregas, adjuntos, fechas de cuestionarios y demás dominios mantienen tareas pendientes. MockRunner simulado y H5P sin contenido real verificado. Docker/producción y equivalencia con Moodle no ejecutados.

Incidencia de regresión: una ejecución arrancó antes de que Astro escuchara tras reinicio; los siete escenarios fallaron por ERR_CONNECTION_REFUSED. Se confirmó el mensaje ready del mismo proceso vivo antes de repetir. No se modificaron pruebas para ocultar el fallo ni se reinició un proceso que seguía arrancando.
