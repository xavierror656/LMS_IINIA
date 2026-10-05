# Entrega grupal compartida: incremento 12 (T2c-2)

Especificación previa `openspec/changes/academic-administration/increment-12.md`, GS1–GS6 / MDL-011..012. Segunda parte de T2c; depende del incremento 11 (grupos, ya verificados). Las notas individuales por miembro siguen en `increment-13.md`. No acredita el 98 % ni equivalencia Moodle.

## Implementación y decisiones

Una tarea admite `groupSubmission`, se guarda como borrador con revisión y entra en vigor **al publicar**, igual que el calendario, la rúbrica o el límite de intentos: publicar copia el valor a la lección y al snapshot de la publicación, de modo que una entrega existente conserva el modo con el que nació. Una lectura lo rechaza con 400. Se sirve con una ruta propia para no mezclarlo con el contenido ni con la evaluación.

La identidad de la entrega la decide un único resolutor: el grupo en una tarea grupal, el estudiante en una individual. **Los doce puntos que leen o escriben entregas pasan por él** —guardar, enviar, adjuntar, borrar adjunto, historial, entrega propia, bandeja del docente, descarga, calificación, reapertura, progreso y publicación—, de modo que no hay dos caminos con reglas distintas. Guardar marca el progreso de **todos** los miembros. Un alumno inscrito sin grupo recibe 409 con un mensaje propio y **no se crea ninguna entrega**: no hay estado a medias que limpiar después.

Los intentos y las reaperturas cuentan por grupo: el índice único por `(lesson_id,group_id,attempt)` y el borrador único por grupo sostienen la regla en la base, no en el código. Reabrir exige motivo y revisión, autoriza al docente vinculado a **cualquier** miembro y crea el siguiente intento del grupo. Quitar a un miembro después de enviar no borra la entrega ni la nota: solo esa persona pierde el acceso.

Los adjuntos son del equipo y su cuota se descuenta a **quien sube el archivo**, no al autor de la entrega, y se le devuelve al borrar. Cualquier miembro puede descargarlos y quitarlos. El libro asigna a cada miembro la entrega enviada del grupo con su nota publicada, así que nadie aparece como «sin entregar» por no haber sido quien guardó; la actividad sigue contando una sola vez por estudiante y un grupo sin entrega no participa. Ninguna operación grupal concede XP, estrellas ni gemas.

## Hallazgos

**Orden de comprobaciones en el borrado de adjuntos.** Al reescribir la autorización quedó la comprobación de revisión antes que la de propiedad, así que intentar borrar el archivo de otra persona devolvía 409 en lugar de 404: filtraba el estado de una entrega ajena. Lo cazó la suite existente de adjuntos; ahora la propiedad se decide primero.

**Un grupo con entregas no se puede eliminar.** La clave foránea de `submissions.group_id` hizo que borrar un grupo con entregas fallara como error de base y saliera un 503. El incremento 11 prometía que eliminar un grupo nunca toca el trabajo académico, y con entregas grupales eso solo se cumple si el borrado se **rechaza**: ahora responde 409 explicando que hay que vaciar el grupo en lugar de eliminarlo. La prueba de integración y la de navegador fijan esa regla.

## Archivos y contratos

- `011_group_submissions.sql`: `group_submission` en actividad, lección y publicación; `submissions.group_id` con `UNIQUE(lesson_id,group_id,attempt)` y borrador único por grupo; `submission_attachments.uploaded_by` con relleno desde el autor de cada entrega ya existente; y `gradebook_entries` sustituida para que cada miembro reciba la entrega del grupo. Ninguna entrega existente cambia: las antiguas quedan individuales con `group_id` nulo.
- `services/scope.go` con el resolutor, la progresión por grupo, la entrega propia y el historial; `services/attachments.go` y `submission_attempts.go` reescritos sobre el alcance; consultas de docente, descarga y calificación con vínculo por cualquier miembro.
- `PUT /teacher/activities/{activityId}/group-mode`; OpenAPI con los campos nuevos en `Activity`, `Lesson`, `Submission` y `Attachment` y el esquema `GroupModeInput` (54 rutas, 85 esquemas); tipos regenerados.
- Interfaz: conmutador **Trabajo en equipo** en la actividad con aviso de que entra en vigor al publicar; en la tarea del alumno, el nombre del equipo, el aviso claro cuando no tiene grupo y la etiqueta de la entrega compartida; la bandeja del docente etiqueta la entrega con su grupo.

## Evidencia ejecutada

- `gofmt` sobre copia normalizada y `go vet ./...`: sin hallazgos.
- `go test ./... -count=1` con `TEST_DATABASE_URL` contra PostgreSQL 18.6 real dentro de WSL: **todo aprobado**, `handlers` 27.5 s. La prueba nueva `TestGroupSubmissionIntegration` cubre GS1–GS6: el modo no se activa en una lectura, el alumno sin grupo recibe 409 sin crear entrega, la entrega es del grupo y la ve el segundo miembro, dos escrituras simultáneas de dos miembros producen exactamente un éxito y un conflicto con **una sola** entrega, enviar desde el otro miembro funciona, el progreso queda para los dos, la nota publicada llega a **ambos** en `gradebook_entries`, la reapertura es por grupo e idempotente y conserva el grupo, quitar a un miembro no altera las entregas, y un grupo con entregas no se elimina (409).
- Regresión interna: la suite completa de incrementos anteriores sigue aprobada, incluidas entregas individuales, libro, rúbricas, fechas, adjuntos y reanudaciones.
- Navegador `group-submission.spec.ts`: **aprobada** (8.0 s). El docente activa el modo y sobrevive a la recarga; el alumno sin grupo ve el aviso y la API responde 409 con su mensaje; tras asignarle grupo, la entrega se crea con `groupName`, se envía, el alumno ve el equipo en la tarea y el docente lo ve en su bandeja. Captura `screenshots/group-mode-tablet.png`.
- Regresión de navegador completa: **17 pruebas aprobadas y 3 omitidas**, ejecutada en tandas. Observación honesta: el límite de inicio de sesión de TSEC1 (10 por minuto y por dirección) hace que una tanda larga agote el cupo y las últimas suites fallen al entrar, no por un defecto del código; se comprobó con doce intentos seguidos (el 11.º responde 429) y se repitieron en ventana limpia, aprobando las cinco. **No se relajó ningún límite para pasar las pruebas**; la solución es ejecutar por tandas.

## Reproducción

```bash
go run ./cmd/migrate && go run ./cmd/api          # backend
npm --prefix frontend run dev                      # frontend
# entrega grupal y regresión (en tandas por el límite de acceso)
E2E_ACADEMIC=1 E2E_STUDENT_PASSWORD='change-this-student-password' E2E_TEACHER_PASSWORD='change-this-teacher-password' \
  npm --prefix frontend run test:e2e -- group-submission.spec.ts
```

En la actividad: **Trabajo en equipo** → marcar → guardar → publicar. Los grupos se crean en **Grupos del curso**, en la página del curso.

## Límites pendientes

La nota sigue siendo **una por entrega**: todavía no hay notas individuales por miembro (incremento 13), así que todos los miembros comparten la nota publicada. No hay agrupamientos, autoselección de grupo por el alumno, grupos en cuestionarios ni restricciones de acceso por grupo. Un grupo que ya tiene entregas no se puede eliminar: hay que vaciarlo. La bandeja de entregas del docente pagina 20 y una entrega concreta se alcanza con `?submissionId=`. Docker, carga, accesibilidad completa y equivalencia Moodle siguen sin ejecutarse.
