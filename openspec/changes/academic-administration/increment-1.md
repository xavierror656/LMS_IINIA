# Incremento ejecutable 1: autoría, entrega textual y devolución

Especificación previa al código. Subconjunto incremental de T1/T2/T3; no sustituye el alcance global ni certifica el 98 %.

## Contrato y estados
- GET /api/v1/teacher/courses y GET /api/v1/teacher/courses/{courseId}/activities: cursos asignados, módulos y borradores paginados (page, 20 por página).
- POST /api/v1/teacher/courses/{courseId}/activities: moduleId, title (1–160), description (0–1000), type reading|assignment, body (1–12000 caracteres). Devuelve 201 y borrador versión 1. Texto plano escapado, sin HTML activo. La asignación docente por curso es explícita; no se infiere de teacher_students.
- GET /api/v1/teacher/activities/{id}: borrador propio del curso.
- PUT /api/v1/teacher/activities/{id}: title, description, body, version entero obligatorio. Tipo y módulo inmutables. Devuelve versión canónica incrementada; conflicto 409 conserva edición previa.
- POST /api/v1/teacher/activities/{id}/publish: version. Publicación transaccional toma snapshot inmutable, crea/actualiza una lección y devuelve borrador con publishedVersion y lessonId. Repetir la misma publicación no duplica lecciones. Una edición posterior no cambia la lección visible hasta volver a publicar.
- GET /api/v1/lessons/{lessonId}/submission: entrega propia o null.
- PUT /api/v1/lessons/{lessonId}/submission: body (1–12000), version (0 para primera entrega), lessonVersion publicado visto por estudiante. Crea o reemplaza borrador; rechaza cambios de actividad publicados entre lectura y guardado. Devuelve entrega canónica; guardar la primera respuesta marca la lección en progreso sin XP.
- POST /api/v1/lessons/{lessonId}/submission/submit: version. Fija contenido y snapshot publicado; repetición idéntica devuelve la misma entrega. Envío ya definitivo no es editable.
- GET /api/v1/teacher/activities/{id}/submissions: entregas enviadas, paginadas; nunca borradores privados del estudiante.
- PUT /api/v1/teacher/submissions/{id}/grade: score entero 0–100, feedback (0–4000), version (0 sin nota). Guarda nota oculta y revisión de auditoría. Editar nota publicada vuelve a borrador hasta publicación explícita.
- POST /api/v1/teacher/submissions/{id}/grade/publish: version. Publica la revisión guardada, con auditoría, sin XP.

Todas las rutas requieren sesión, Origin para escrituras, rol y autorización por curso/inscripción/objeto. Para consultar o calificar una entrega se requieren ambos vínculos: course_staff al curso y teacher_students al estudiante. La asignación de curso no amplía implícitamente los estudiantes autorizados. 400 para entrada inválida/campos desconocidos; 401 sesión; 403 rol/Origin; 404 objeto no accesible; 409 versión obsoleta o transición inválida; 503 DB. REST se incorpora a OpenAPI junto con implementación. WS no cambia.

## Modelo y restricciones
course_staff con PK(course_id,user_id). authored_activities conserva borrador, revisión y relación con lección publicada. activity_publications con UNIQUE(activity_id,version), contenido inmutable. Lecciones previas conservan acceso; lecciones nuevas solo se crean al publicar. Envíos con UNIQUE(lesson_id,user_id); snapshot de instrucciones guardado en FK de publicación; notas y sus revisiones en transacción. Locks de fila y claves únicas resuelven concurrencia. Seed asigna profe a los cursos sintéticos, sin inferir permisos en migraciones de datos reales.

## Escenarios de aceptación ejecutables
I1: docente asignado crea lectura; estudiante no la ve hasta publicación; aparece tras publicar; recarga conserva texto.
I2: docente ajeno/estudiante no puede editar ni consultar borradores; módulo ajeno rechazado; HTML se muestra como texto.
I3: dos guardados con igual versión dan un éxito y un 409; dos publicaciones idénticas producen una sola lección. Edición posterior no cambia publicación anterior.
I4: estudiante guarda y envía tarea; borrador no aparece en bandeja docente; estudiante ajeno no lo consulta; envío repetido no duplica; no acepta identidad ni nota del cliente.
I5: docente califica; alumno no ve borrador; publicar muestra nota/comentario; historial persiste; dos escrituras obsoletas no se pisan. Cambios no duplican XP.
I6: UI tablet/teclado/reduced motion; fallo API conserva texto y muestra error, cierre de sesión impide acceder; borrador de formulario avisa antes de abandonar.

Fuera de este incremento: adjuntos, plazos, reentregas, rúbricas, cuestionarios, agregación y gestión de asignaciones desde interfaz. Permanecen en tareas futuras. Nota manual tiene escala fija 0–100. Roles iniciales student/teacher; admin Go y gestión de usuarios siguen pendientes. La demo Node conserva su comportamiento y no anuncia las nuevas funciones como disponibles: se verifican contra Go/PostgreSQL.
