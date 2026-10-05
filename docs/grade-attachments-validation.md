# Adjuntos de devolución: incremento 15 (T4g-2)

Especificación previa `openspec/changes/academic-administration/increment-15.md`, FD1–FD5. Cierra la cuarta parte de T4g, tras el incremento 14 (formatos, instrucciones y retención). No acredita el 98 % ni equivalencia Moodle.

## Implementación y decisiones

Los archivos de devolución cuelgan de la **nota**, no de la entrega: la clave foránea compuesta a `submission_grades(submission_id, student_id)` hace imposible que existan sin calificación. Adjuntar sin nota responde 409 con un mensaje que explica que hay que guardarla primero, y la prueba comprueba además que **no queda ningún archivo huérfano**. En una entrega de equipo cada miembro tiene su propia devolución y sus propios archivos.

La autorización es la misma que la de calificar, y ahora **literalmente la misma**: la resolución del miembro («¿sobre quién puede actuar este docente en esta entrega?») se extrajo a un único sitio que usan tanto calificar y publicar como listar, subir, borrar y descargar. Antes esa regla estaba escrita dentro de la calificación; duplicarla para los archivos habría garantizado que las dos versiones divergieran con el tiempo.

La privacidad se sostiene en la consulta: el alumno descarga por identificador de archivo, y la consulta exige que la nota esté **publicada** y que el archivo sea **suyo**. Un compañero de equipo recibe 404 y una devolución en borrador no se filtra por ninguna ruta. Igual que el resto de descargas, el archivo se entrega opaco, con sandbox y `nosniff`, nunca en línea.

Los archivos se cuentan en la cuota de quien los sube —el docente— y se le devuelven al borrar. La retención del incremento 14 **no** toca devoluciones: su regla solo alcanza borradores abandonados, y una devolución publicada pertenece a una entrega enviada.

## Archivos y contratos

- `014_grade_attachments.sql`: tabla `grade_attachments` con clave foránea compuesta a la nota, ranura 1..5, unicidad por miembro y ranura e índice por cargador.
- `services/grade_attachments.go` con `gradableMember` (regla única), listar, subir y borrar; `services/academic.go` pasa a usar esa misma resolución. `models.Grade.Files` y su carga en la entrega y en la bandeja del docente.
- `handlers/grade_attachments.go` y siete rutas: tres del docente para una entrega individual (`/grade-attachments`), tres para un miembro concreto (`/grades/{studentId}/attachments`) y la descarga del alumno (`/lessons/{lessonId}/feedback/{attachmentId}`). OpenAPI con `Grade.files` (65 rutas, 87 esquemas) y tipos regenerados.
- Interfaz: cada bloque de calificación incluye subida, listado y borrado de sus archivos de devolución; la tarea del alumno los lista bajo su nota publicada.

## Evidencia ejecutada

- `gofmt` sobre copia normalizada y `go vet ./...`: sin hallazgos.
- `go test ./... -count=1` con `TEST_DATABASE_URL` contra PostgreSQL 18.6 real dentro de WSL: **todo aprobado**, `handlers` 34.3 s. La prueba de entrega grupal cubre FD1–FD5: sin calificación, 409 y **cero archivos almacenados**; la subida queda a nombre del docente; un docente ajeno recibe 404 y un alumno 403; **el compañero de equipo recibe 404 y la dueña 200** en la descarga. La regresión de libro, rúbricas, intentos, adjuntos de entrega, instrucciones y grupos sigue aprobada.
- Navegador: `group-submission.spec.ts` ampliada y **aprobada** (6.3 s): el docente adjunta la devolución con archivo desde la bandeja, la alumna la descarga y el texto coincide.
- `astro check`: 60 archivos, 0 errores. `vitest`: 4 aprobadas. `astro build`: correcto.

## Incidencia propia

El primer intento de navegador falló al no aparecer el archivo tras subirlo, aunque la API **sí** lo había guardado (201): la bandeja del docente cargaba las notas de cada miembro pero no sus archivos. La prueba de integración no lo detectaba porque consulta la ruta de listado, no la de la bandeja. Corregido cargando los archivos por miembro en la bandeja; es un recordatorio de que una prueba verde no cubre caminos que no ejercita.

## Reproducción

```bash
go run ./cmd/migrate && go run ./cmd/api
npm --prefix frontend run dev
E2E_ACADEMIC=1 E2E_STUDENT_PASSWORD='change-this-student-password' E2E_TEACHER_PASSWORD='change-this-teacher-password' \
  npm --prefix frontend run test:e2e -- group-submission.spec.ts
```

En la bandeja de una actividad, cada bloque de calificación tiene su subida de archivos de devolución. El alumno los ve bajo **Tu devolución** cuando la nota está publicada.

## Límites

No hay análisis antivirus ni previsualización. Los cuestionarios no admiten archivos de devolución: la devolución de un intento es solo texto. No hay cuotas por curso ni borrado automático de devoluciones antiguas. Nginx y Docker siguen sin poder ejecutarse en este entorno, así que la validación del proxy para las rutas nuevas queda pendiente de un entorno que los tenga.
