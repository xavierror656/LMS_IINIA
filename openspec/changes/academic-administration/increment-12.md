# Incremento 12: entrega grupal compartida (T2c-2)

Especificación previa al código. T2c / MDL-011..012, segunda parte. Depende del incremento 11 (grupos y pertenencia, ya verificados). Las notas individuales por miembro quedan en `increment-13.md`. No acredita el 98 % ni equivalencia Moodle.

## Reglas y aceptación

- GS1: una tarea admite `groupSubmission` booleano (falso por defecto). Se guarda como borrador con la versión de actividad y entra en vigor **al publicar**, igual que el calendario, la rúbrica o los intentos; publicar copia el valor a `lessons` y al snapshot de la publicación, de modo que una entrega existente conserva el modo con el que fue creada. Lecturas y cuestionarios rechazan el campo con 400. Se sirve con una ruta propia `PUT /teacher/activities/{activityId}/group-mode` para no mezclarlo con el contenido ni con la evaluación.
- GS2: en una tarea grupal publicada la identidad de la entrega es el **grupo** y la matrícula sigue siendo el requisito de acceso. Cualquier miembro puede guardar, adjuntar, enviar y consultar el borrador compartido; dos miembros que escriben a la vez se serializan por el mismo bloqueo y solo uno gana la revisión, sin duplicar la entrega. Un alumno inscrito sin grupo recibe 409 con mensaje propio y no puede crear una entrega individual en esa tarea. Un alumno de un grupo de otro curso, o ajeno al grupo de la tarea, recibe 404.
- GS3: la entrega grupal conserva intento, reapertura e historial contando por grupo: `UNIQUE(lesson_id,group_id,attempt)` y un único borrador activo por grupo. Reabrir crea el siguiente intento del grupo con la misma pertenencia y publicación vigente, exige motivo y revisión, y autoriza al docente vinculado a algún miembro del grupo o al autor de la entrega. Quitar un miembro después de enviar no borra la entrega ni la nota: el retirado pierde el acceso y el resto lo conserva.
- GS4: los adjuntos pertenecen a la entrega y son compartidos. Se registra quién subió cada archivo: la cuota de 50 MiB se descuenta del cargador y se le devuelve al borrar, no al autor de la entrega. Cualquier miembro puede descargar y quitar archivos de la entrega de su grupo. Los límites existentes no cambian: 5 archivos, 2 MiB por archivo, 128 KiB de JSON y 2 MiB + 64 KiB solo en multipart.
- GS5: el libro asigna a **cada miembro** la entrega enviada del grupo y, mientras no existan notas individuales, su nota publicada compartida: nadie aparece como «sin entregar» por no haber sido quien guardó. Los contadores y el promedio ponderado siguen sumando la actividad una sola vez por estudiante, y un grupo sin entrega enviada no participa.
- GS6: ninguna operación grupal concede XP, estrellas ni gemas; la finalización de lectura mantiene su recompensa única e idempotente. El docente ve el grupo en la entrega y en el libro; el alumno ve su grupo y que el trabajo es compartido; un alumno sin grupo ve un aviso claro y ninguna caja de texto, no un error.

## Datos y contratos

Migración 011: `group_submission` en `authored_activities`, `lessons` y `activity_publications`; `group_id` en `submissions` con FK a `course_groups`; índices únicos parciales por grupo; `uploaded_by` en `submission_attachments` con relleno desde el autor de cada entrega; y sustitución de la vista `gradebook_entries` para que cada miembro reciba la entrega del grupo y las individuales mantengan su regla. Sin borrado de datos: las entregas existentes quedan individuales.

## Validación y límites

Integración PostgreSQL: sin grupo 409 sin crear entrega; dos miembros guardando a la vez sin duplicar; reapertura por grupo con revisión y motivo; cuota descontada al cargador y devuelta al borrar; miembro retirado que pierde acceso sin borrar la entrega; libro que asigna la entrega del grupo a todos los miembros sin duplicar la actividad; tarea individual sin cambios de comportamiento. Navegador: docente activa la entrega grupal y publica; dos alumnos ven la misma entrega y uno envía; el docente la califica y el libro muestra a ambos. Regresión de tareas, libro, adjuntos y reentregas.

Fuera de alcance: notas individuales por miembro (incremento 13), agrupamientos, autoselección, grupos en cuestionarios y restricciones por grupo.
