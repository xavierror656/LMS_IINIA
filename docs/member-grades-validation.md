# Notas individuales por miembro: incremento 13 (T2c-3)

Especificación previa `openspec/changes/academic-administration/increment-13.md`, GI1–GI6 / MDL-012. Tercera y última parte de T2c; depende de los incrementos 11 (grupos) y 12 (entrega grupal compartida), ya verificados. No acredita el 98 % ni equivalencia Moodle.

## Implementación y decisiones

Una nota deja de ser una por entrega y pasa a ser una por **estudiante**. La clave primaria de `submission_grades` era la entrega; ahora es `(submission_id, student_id)`, con el miembro rellenado desde el autor de cada entrega existente, así que nada cambia para una tarea individual: su única nota posible sigue siendo la de su autor. El índice único del historial de auditoría también pasa a ser por miembro, para que dos compañeros de equipo puedan tener cada uno su propia secuencia de revisiones.

Calificar es por estudiante y **autoriza por estudiante**: el docente debe ser del curso y estar vinculado e inscrito con el miembro concreto, no con «alguien del grupo». Un miembro que no pertenece a la entrega responde 404, y en una entrega individual pedir a otra persona también es 404. La revisión optimista, el cálculo por rúbrica publicada y la idempotencia funcionan igual que antes, pero por miembro: guardar la nota de uno no toca la del otro y **publicar la devolución de un miembro no publica las demás**.

La privacidad se sostiene en la consulta, no en la interfaz: el alumno recibe **su** nota publicada y nunca el listado de miembros; ese listado solo lo recibe el lado docente, con la nota de cada uno. El libro entrega a cada miembro la entrega del grupo y **su propia** nota: quien no tiene nota queda pendiente y **no hereda** la de un compañero, de modo que la actividad sigue contando una sola vez por estudiante y una ausencia no se convierte en cero.

Las rutas individuales se conservan para las entregas individuales y **rechazan** una entrega de equipo con 409 y un mensaje que explica que hay que calificar miembro a miembro: es preferible a calificar a todo el equipo por error.

## Archivos y contratos

- `012_member_grades.sql`: `student_id` en `submission_grades` (obligatorio, con relleno) y clave primaria `(submission_id,student_id)`; `student_id` en `grade_revisions` y su unicidad por miembro; y `gradebook_entries` sustituida para unir la nota del miembro y no la de la entrega.
- `models.MemberGrade` y `Submission.Members` (solo lado docente); `Repository.Submission` recibe el espectador cuya nota adjuntar y devuelve los miembros con sus notas al docente.
- `services.AcademicService.Grade`/`GradeWithRubric` reciben el estudiante; `ErrGradePerMember` se traduce a 409.
- Rutas nuevas `PUT /teacher/submissions/{submissionId}/grades/{studentId}` y `POST .../grades/{studentId}/publish`; OpenAPI con `MemberGrade` y `Submission.members` (56 rutas, 86 esquemas) y tipos regenerados.
- Interfaz: la bandeja del docente muestra un bloque de calificación **por miembro** con su estado, y la tarjeta etiqueta la entrega como de equipo.

## Evidencia ejecutada

- `gofmt` sobre copia normalizada y `go vet ./...`: sin hallazgos.
- `go test ./... -count=1` con `TEST_DATABASE_URL` contra PostgreSQL 18.6 real dentro de WSL: **todo aprobado**, `handlers` 28.6 s. La prueba de entrega grupal se amplió para cubrir GI2–GI5: la vía individual rechaza el equipo con 409; calificar a un miembro funciona y publicar solo publica al suyo; **el otro miembro no ve la nota ajena ni hereda puntuación** en el libro aunque sí recibe la entrega; la bandeja docente trae a los dos miembros con su estado; las notas quedan independientes (75 y 60) y el historial de auditoría registra cuatro revisiones, dos por miembro. La regresión de calificación individual, libro, rúbricas, intentos y archivos sigue aprobada.
- Navegador: la prueba `group-submission.spec.ts` se amplió con la calificación por miembro: el docente abre la entrega del equipo, califica a Luna, guarda el borrador, ve que la vía individual responde 409, publica su devolución, y **la alumna ve su propia nota (70 / 100) y su comentario**. Aprobada en ventana limpia (9.7 s). Regresión `gradebook.spec.ts` y `attempts.spec.ts` aprobadas.
- `astro check`: 59 archivos, 0 errores. `vitest`: 4 aprobadas. `astro build`: correcto.

## Incidencias de entorno y regresiones propias

- **Etiqueta duplicada**: al añadir el estado por miembro dentro del bloque de calificación, una entrega individual mostraba dos veces «Devolución publicada» y `attempts.spec.ts` lo detectó con su aserción estricta. Ahora esa etiqueta por miembro solo aparece en entregas de equipo.
- **Duplicado de método**: el proyecto ya tenía `GradeWithRubric` en `services/evaluation.go`; mi primera versión la duplicó y el compilador lo señaló. Se conservó una sola, extendida con el estudiante.
- **Restricción de auditoría**: `grade_revisions` tenía unicidad por entrega, así que el segundo miembro calificado fallaba con error de base y salía 503. Se cambió a unicidad por miembro.
- El guardado de un borrador de calificación **no recarga la página** (el formulario confirma con «Borrador guardado · versión N»); solo publicar recarga. La prueba de navegador lo refleja en lugar de esperar una recarga que no ocurre.

## Reproducción

```bash
go run ./cmd/migrate && go run ./cmd/api
npm --prefix frontend run dev
E2E_ACADEMIC=1 E2E_STUDENT_PASSWORD='change-this-student-password' E2E_TEACHER_PASSWORD='change-this-teacher-password' \
  npm --prefix frontend run test:e2e -- group-submission.spec.ts
```

En la bandeja de la actividad, cada miembro del equipo tiene su propio bloque de calificación y su propia publicación.

## Límites pendientes

No hay nota grupal única compartida, moderación, anonimato ni varios evaluadores (T3a). La publicación es por miembro pero no hay programación de publicación ni notificación al alumno. Quitar a un miembro conserva su nota publicada en el historial y deja de contar en el libro según la regla vigente de matrícula y vínculo. Docker, carga, accesibilidad completa y equivalencia Moodle siguen sin ejecutarse.
