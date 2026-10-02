# Incremento 2: libro de calificaciones por curso

Especificación previa al código. Desarrolla LMS-018/021/022, T4a; el resto de T4 (categorías, ponderaciones, rúbricas y archivos) permanece pendiente.

## Alcance y reglas
El docente ve una matriz de estudiantes por tarea publicada y puede abrir la entrega exacta para corregirla. Solo estudiantes student inscritos en el curso y vinculados mediante teacher_students; además requiere course_staff. Se excluyen lecturas, código, H5P y tareas aún sin publicar. Encabezados usan el título publicado, no cambios privados del editor.

Estados por celda: not_submitted (Sin entregar), submitted (Por calificar), graded (Nota sin publicar), published (Nota publicada). Un borrador de respuesta se presenta idéntico a ninguna respuesta: sin ID, contenido ni fecha. La nota 0 es una nota real; ausencia de nota es null. Notas en borrador son visibles solo al docente autorizado y no cuentan en el promedio.

Resumen de cada estudiante sobre TODAS las tareas publicadas del curso, aunque haya varias páginas de columnas: totalActivities, notSubmitted, pendingReview, pendingPublication, published, averageHundredths. Media aritmética de notas publicadas, peso igual, escala 0–100; promedio se transmite como centésimas enteras (ejemplo 72.50 → 7250). Redondeo mitad hacia arriba; null si no hay notas publicadas. No tratar pendientes como cero. No es calificación final ni media ponderada configurada. Todos los contadores de estado suman totalActivities.

Paginación independiente: page de estudiantes (20), activityPage de columnas (10), ambas 1..10000. Total exacto y navegación conservan el otro paginador. Los promedios no cambian al paginar columnas. Respuesta privada no-store, lectura coherente en transacción read-only repeatable-read, número acotado de consultas sin N+1. No se modifica ni duplica el esquema de notas ni XP.

## Contratos
GET /api/v1/teacher/courses/{courseId}/gradebook?page=1&activityPage=1 devuelve course, activities, rows, page, pageSize=20, totalStudents, activityPage, activityPageSize=10, totalActivities. Cada actividad: activityId, lessonId, title. Cada fila: studentId, alias, cells y summary. Cada celda: activityId, state, score nullable y submissionId nullable; sin feedback ni respuesta completa.

GET /api/v1/teacher/activities/{activityId}/submissions admite submissionId opcional positivo. Se aplica dentro de los permisos existentes: nunca permite acceder a entrega de otra actividad ni estudiante no vinculado. Si no coincide con una entrega enviada y autorizada devuelve lista vacía. Filtrado no concede permisos nuevos.

401 sin sesión; 403 rol estudiante; 404 curso ajeno o inexistente; 400 identificadores/paginación/filtro inválidos; 503 DB. Parámetros de paginación fuera de rango no se corrigen silenciosamente en frontend. Mantener OpenAPI y tipos generados.

## Interfaz
Ruta /teacher/courses/{courseId}/gradebook. Enlaces desde listado de cursos y editor del curso. Tabla semántica con caption, headers y estados textuales; desplazamiento horizontal dentro de una región con foco y etiqueta. Alias de fila visible, celdas con enlaces de al menos 44 px al corrector filtrado; navegación por teclado, tablet y reduced motion. El promedio indica explícitamente cuántas notas publicadas cubre del total. Estados vacíos diferenciados: sin tareas publicadas, sin estudiantes vinculados, página fuera de resultados. Caída API muestra error/recuperación, jamás ceros inventados.

## Aceptación y trazabilidad
- GB1 / LMS-018: matriz distingue sin entregar, enviada, calificada oculta y publicada, incluye cero y no revela borradores de respuesta ni títulos editados sin publicar.
- GB2 / LMS-018: media de 80 y 0 = 40.00; una nota oculta 100 no cuenta; ninguna publicada da null; redondeo 2/3 = 0.67. No cambia entre páginas de actividades.
- GB3 / LMS-021: docente ajeno y estudiante rechazados; Sol sin vínculo no aparece ni afecta totales; alumno vinculado no inscrito tampoco; filtro de entrega no elude curso/actividad/vínculo.
- GB4 / LMS-018/022: más de 20 estudiantes y más de 10 actividades se recuperan sin duplicaciones; resumen incorpora columnas no visibles. Datos de otro curso, lecturas y borradores no cuentan.
- GB5 / LMS-022: navegador abre libro, muestra notas y pendientes, abre la entrega correcta, conserva páginas, permite teclado/tablet y recuperación ante fallo real de lectura API.

No incluye edición masiva, CSV, notas de otras actividades aún sin evaluación, fórmulas configurables, nota final, exenciones, rúbricas ni vista global del estudiante. No certifica 98 %.
