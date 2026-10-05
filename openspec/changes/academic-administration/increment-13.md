# Incremento 13: notas individuales por miembro (T2c-3)

Especificación previa al código. T2c / MDL-012, tercera parte. Depende de los incrementos 11 (grupos) y 12 (entrega grupal compartida). No acredita el 98 % ni equivalencia Moodle.

## Reglas y aceptación

- GI1: una nota deja de ser una por entrega y pasa a ser una por **estudiante**: `submission_grades` incorpora `student_id` obligatorio con `UNIQUE(submission_id,student_id)` y relleno desde el autor de cada entrega. En una tarea individual nada cambia: la única nota posible es la de su autor.
- GI2: en una entrega grupal el docente califica a cada miembro por separado, con la misma escala, rúbrica publicada, revisión optimista e idempotencia que hoy. Calificar a un miembro no altera la nota de otro. Solo puede calificar a estudiantes vinculados (`teacher_students`) e inscritos en el curso; un miembro no vinculado se muestra como tal y su nota se rechaza con 404 en lugar de conceder acceso a un estudiante ajeno.
- GI3: la publicación es por estudiante, como hoy: una nota en borrador no se revela al alumno y no cuenta en el libro. Publicar la devolución de un miembro no publica las de los demás. El alumno ve únicamente su propia nota, su retroalimentación y su evaluación por rúbrica; nunca la de sus compañeros.
- GI4: el libro muestra la nota individual publicada de cada miembro cuando existe. Si un miembro no tiene nota individual, su celda queda pendiente y **no** hereda la nota de otro miembro; un grupo sin entrega enviada no participa. Los contadores, el peso total y el promedio ponderado siguen contando la actividad una sola vez por estudiante y no convierten ausencias en cero.
- GI5: reabrir el grupo conserva las notas anteriores de cada miembro en el historial y el nuevo intento empieza sin nota, igual que hoy con las entregas individuales. Quitar a un miembro conserva su nota publicada en el historial y deja de contar en el libro, según la regla vigente de matrícula y vínculo.
- GI6: la interfaz del corrector lista a los miembros del grupo con su estado, permite calificar y publicar a cada uno por separado, conserva lo escrito ante un error y advierte de quién queda sin calificar. El alumno ve su nota con la misma presentación que una entrega individual.

## Datos y contratos

Migración 012: `student_id` en `submission_grades` con FK a `users(id)`, `UNIQUE(submission_id,student_id)` y relleno desde `submissions.user_id`; ajuste de `gradebook_entries` para unir la nota del miembro y no la de la entrega. Rutas: `PUT /teacher/submissions/{submissionId}/grades/{studentId}` y `POST` de publicación por estudiante, conservando las actuales para entregas individuales. OpenAPI antes del código.

## Validación y límites

Integración PostgreSQL: notas independientes por miembro; publicación selectiva; privacidad entre compañeros; conflicto de revisión por miembro; promedio del libro con notas mixtas (un miembro publicado y otro pendiente); reapertura que conserva el historial de notas; miembro no vinculado rechazado con 404; tarea individual sin cambios. Navegador: el docente califica a dos miembros con notas distintas, publica solo una y el alumno sin publicar no la ve. Regresión completa de libro, tareas y entrega grupal.

Fuera de alcance: nota grupal única compartida, moderación, anonimato y varios evaluadores (T3a).
