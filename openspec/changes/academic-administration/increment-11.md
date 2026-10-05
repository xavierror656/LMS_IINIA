# Incremento 11: grupos de curso (T2c-1)

Especificación previa al código. T2c / MDL-011..012, primera parte. El turno anterior verificó límites por usuario y proxy confiable (TSEC1): progreso. Este incremento añade **grupos de curso y su pertenencia**; la entrega grupal compartida queda especificada en `increment-12.md` y las notas individuales por miembro en `increment-13.md`. No acredita el 98 % ni equivalencia Moodle.

## Alcance

Un docente asignado a un curso crea grupos y asigna estudiantes vinculados e inscritos. Nada más cambia: las entregas siguen siendo individuales en este incremento, de modo que el comportamiento existente no se altera. Fuera de alcance: entrega grupal (incremento 12), notas individuales (incremento 13), agrupamientos, autoselección de grupo por el alumno, grupos en cuestionarios y restricciones de acceso por grupo.

## Reglas y aceptación

- CG1: un grupo tiene nombre de 1 a 100 caracteres, curso inmutable, autor y versión. El nombre es único dentro del curso: repetirlo devuelve 409 y no crea un grupo parcial ni duplica la fila existente.
- CG2: solo un docente con `course_staff` vigente en el curso lista, crea, renombra y elimina grupos del curso. Un docente ajeno recibe 404 —no revela que el curso existe— y un alumno 403. Renombrar exige la versión exacta; una revisión obsoleta devuelve 409 y conserva el nombre anterior.
- CG3: un estudiante pertenece **como máximo a un grupo por curso**. Añadirlo a un segundo grupo del mismo curso devuelve 409 sin crear la pertenencia. Añadirlo exige que esté inscrito en el curso y vinculado a ese docente (`teacher_students`), la misma base que el roster de prórrogas; si no cumple, 404. Añadir dos veces al mismo grupo es idempotente y no duplica la fila. Un alumno no puede añadirse ni quitarse a sí mismo.
- CG4: quitar un miembro que no pertenece a ese grupo devuelve 404. Eliminar un grupo borra sus pertenencias y nunca toca entregas, notas ni progreso: como este incremento no vincula entregas a grupos, no puede destruir trabajo. Repetir la eliminación devuelve 404.
- CG5: la lista de grupos del curso pagina 20 por página y devuelve, además, el roster de estudiantes vinculados e inscritos con el grupo al que pertenece cada uno (0 si no tiene), para que la interfaz no necesite otra consulta. Solo aparecen estudiantes `student` vinculados a ese docente e inscritos en ese curso.
- CG6: ninguna operación de grupo concede XP, estrellas ni gemas ni modifica el progreso de una lección; publicar actividades, entregar y calificar siguen exactamente igual porque las entregas no cambian en este incremento.
- CG7: la interfaz permite al docente ver los grupos de un curso, crear uno, renombrarlo, eliminarlo y añadir o quitar estudiantes; los formularios conservan lo escrito ante un error, advierten de cambios sin guardar, funcionan con teclado y en tablet, y no muestran acciones a quien no tiene el curso asignado.

## Datos y contratos

Migración 010: `course_groups` (id, course_id, nombre, versión, autor, fecha, `UNIQUE(course_id,name)` y `UNIQUE(course_id,id)`) y `group_members` (grupo, curso denormalizado con FK compuesta, estudiante, fecha, `PRIMARY KEY(group_id,user_id)` y `UNIQUE(course_id,user_id)`). Sin cambios en tablas existentes: las entregas permanecen individuales.

Rutas nuevas: `GET /teacher/courses/{courseId}/groups?page`, `POST /teacher/courses/{courseId}/groups` (`{name}`), `PUT /teacher/courses/{courseId}/groups/{groupId}` (`{name,version}`), `DELETE /teacher/courses/{courseId}/groups/{groupId}`, `PUT /teacher/courses/{courseId}/groups/{groupId}/members/{studentId}` y `DELETE` de la misma ruta. OpenAPI antes del código; se conservan sesiones, `Origin`, límites por usuario y `no-store`.

## Validación y límites

Unitarias: validación del nombre (vacío, solo espacios, más de 100 caracteres) y rangos de versión. Integración PostgreSQL con esquema separado y migraciones/seed repetidos: crear y listar; nombre duplicado 409; renombrar con revisión correcta y obsoleta; docente ajeno 404 y alumno 403; añadir estudiante no vinculado o no inscrito 404; segunda pertenencia en el mismo curso 409; añadir dos veces idempotente; quitar miembro inexistente 404; eliminar grupo y comprobar que entregas, notas y progreso no cambian; roster con el grupo de cada estudiante y sin estudiantes ajenos. Navegador: docente crea un grupo, añade a Luna, renombra y quita; comprueba que Sol no aparece. Regresión de tareas, libro, adjuntos y reentregas. Ejecutar `go vet`, `go test -count=1`, `astro check`, `astro build` y `vitest`.

No se acreditan: entrega grupal compartida ni notas individuales (incrementos 12 y 13), agrupamientos, autoselección, grupos en cuestionarios ni paridad Moodle.
