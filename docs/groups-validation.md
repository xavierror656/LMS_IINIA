# Grupos de curso: incremento 11 (T2c-1)

Especificación previa `openspec/changes/academic-administration/increment-11.md`, CG1–CG7 / T2c / MDL-011. Primera parte de T2c: la entrega grupal compartida queda en `increment-12.md` y las notas individuales por miembro en `increment-13.md`. No acredita el 98 % ni equivalencia Moodle.

## Implementación y decisiones

Un docente con `course_staff` vigente crea grupos dentro de su curso y asigna estudiantes vinculados e inscritos, con la misma base que el roster de prórrogas. El nombre es único por curso y renombrar exige la versión exacta. Un estudiante pertenece como máximo a un grupo por curso: la regla vive en `UNIQUE(course_id,user_id)` con el curso desnormalizado en la pertenencia, así que no depende de que el llamador la respete. Añadir dos veces al mismo grupo es idempotente; quitar a quien no está devuelve 404. La lista de grupos viaja junto al roster en una sola respuesta, para que la página no necesite otra consulta.

Eliminar un grupo borra sus pertenencias y **no toca entregas, notas ni progreso**: en este incremento las entregas siguen siendo individuales, así que ninguna operación de grupo puede destruir trabajo académico. Las claves foráneas compuestas impiden que una pertenencia apunte a un grupo de otro curso.

Este incremento no cambia ninguna tabla existente: la migración 010 solo añade `course_groups` y `group_members`, de modo que tareas, libro, adjuntos, reentregas y cuestionarios conservan su comportamiento y sus pruebas.

## Hallazgo: los mensajes específicos nunca llegaban al cliente

Al comprobar el nombre duplicado apareció un defecto preexistente: el manejador de errores de Fiber reemplazaba el mensaje de cualquier `*fiber.Error` por su propia tabla de textos genéricos. Por eso los mensajes cuidados que ya existían —tarea cerrada, cuestionario cerrado o tiempo agotado— **nunca se mostraban**: el usuario recibía siempre «Hay cambios más recientes o esta actividad ya no admite esa acción». La corrección conserva el mensaje propio cuando el manejador lo aporta y usa la tabla amable solo para los errores del marco, cuyo texto es el del código HTTP. Una prueba de integración fija ahora el mensaje exacto del nombre duplicado, y la regresión de cuestionarios y fechas siguió aprobando.

## Archivos y contratos

- `010_course_groups.sql`: `course_groups` (id, curso, nombre, versión, autor, fecha, `UNIQUE(course_id,name)` y `UNIQUE(course_id,id)`) y `group_members` (grupo, curso denormalizado con FK compuesta, estudiante, fecha, `PRIMARY KEY(group_id,user_id)`, `UNIQUE(course_id,user_id)`).
- `models/group.go`, `services/groups.go`, `handlers/groups.go` y seis rutas nuevas bajo `/teacher/courses/{courseId}/groups`. `services.ErrGroupNameTaken` y `services.ErrGroupMemberElsewhere` se traducen a 409 con mensaje propio.
- `handlers/app.go`: tabla `friendlyError` y conservación del mensaje específico.
- OpenAPI 1.9.0 con siete esquemas y tres plantillas de ruta nuevas (53 rutas, 84 esquemas); tipos regenerados.
- `frontend/src/pages/teacher/courses/[courseId]/groups.astro` con el elemento `group-admin`, y enlace desde la página del curso. Sin islas React ni dependencias nuevas.

## Evidencia ejecutada

- `go vet ./...`: sin diagnósticos. `gofmt` sobre copia normalizada: sin hallazgos.
- `go test ./... -count=1` con `TEST_DATABASE_URL` contra PostgreSQL 18.6 real dentro de WSL: todas las pruebas aprobadas, `handlers` 26.4 s. La prueba nueva aplica migraciones y seed dos veces, y cubre nombre duplicado con su mensaje exacto, nombres inválidos, docente ajeno 404 y alumno 403, renombrado con revisión correcta y obsoleta, estudiante no vinculado 404, segunda pertenencia 409, adición idempotente, retirada inexistente 404, roster con el grupo de cada estudiante y sin estudiantes ajenos, borrado con sus pertenencias, y la comprobación de que entregas, notas y progreso no cambian: la misma entrega calificada con 80 sigue en 80.
- Navegador `groups.spec.ts`: aprobada, 5.7 s. Crea dos grupos, comprueba el mensaje de nombre repetido, añade a Luna, verifica por API que un segundo grupo del mismo curso responde 409, renombra, retira del grupo (el roster vuelve a mostrar grupo 0), elimina ambos grupos por su tarjeta y confirma que ninguno permanece. Captura `screenshots/groups-tablet.png`.
- Regresión en la misma ejecución: `quizzes.spec.ts` y `schedule.spec.ts` aprobadas (36.1 s), sin cambios por la corrección del manejador de errores.
- `astro check`: 59 archivos, 0 errores, 0 warnings, 0 hints. `vitest`: 4 aprobadas. `astro build` aprobado con el aviso existente de chunks superiores a 500 kB.
- Incidencia de entorno: una ejecución de navegador falló porque el servidor de desarrollo de Astro estaba reoptimizando dependencias en ese momento y el formulario de acceso no se hidrató; se reinició el servidor, se confirmó la página lista y se repitió. No se modificó ninguna prueba para ocultarlo.

## Reproducción

```bash
# backend
go run ./cmd/migrate && go run ./cmd/api
# frontend
npm --prefix frontend run dev
# pruebas
E2E_ACADEMIC=1 E2E_STUDENT_PASSWORD='change-this-student-password' E2E_TEACHER_PASSWORD='change-this-teacher-password' \
  npm --prefix frontend run test:e2e -- groups.spec.ts
```

En el curso, **Grupos del curso** permite crear, renombrar, eliminar y asignar estudiantes. El docente solo ve su roster vinculado e inscrito.

## Límites pendientes

La entrega grupal compartida (incremento 12) y las notas individuales por miembro (incremento 13) no están implementadas: hoy un grupo no cambia cómo se entrega ni cómo se califica. No hay agrupamientos, autoselección de grupo por el alumno, grupos en cuestionarios ni restricciones de acceso por grupo. La lista de grupos pagina 20 por página sin total exacto. Docker, carga, accesibilidad completa y equivalencia Moodle siguen sin ejecutarse.
