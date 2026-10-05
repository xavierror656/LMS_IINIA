# Incremento 15: adjuntos de devolución (T4g-2)

Especificación previa al código. T4g-2 / LMS-019, la cuarta parte de T4g, tras el incremento 14 (formatos, instrucciones y retención). No acredita el 98 % ni equivalencia Moodle.

## Reglas y aceptación

- FD1: el docente adjunta hasta cinco archivos (2 MiB cada uno) a la devolución de **un estudiante concreto**, no a la entrega en abstracto: en una entrega de equipo, la devolución de cada miembro lleva sus propios archivos. Formatos, análisis de contenido y límites son los mismos que ya valen para el resto de adjuntos.
- FD2: adjuntar exige que **la calificación exista** (borrador o publicada): la devolución es de una nota y no puede haber archivos huérfanos. Sin calificación se responde 409 con un mensaje que explica que primero hay que guardarla. La cuota se descuenta a quien sube y se le devuelve al borrar, igual que en el resto del sistema.
- FD3: el alumno ve y descarga **solo** los archivos de su propia devolución y **solo cuando está publicada**, con las mismas garantías que el resto de descargas: tipo opaco, política de sandbox y nunca en línea. Un archivo de una devolución en borrador no se filtra por ninguna ruta, y un compañero de equipo nunca ve los archivos de otro miembro.
- FD4: la autorización es la misma que la de calificar: docente del curso, vinculado e inscrito con el miembro. Docente ajeno 404, alumno 403, miembro que no pertenece a la entrega 404. Se reutiliza la resolución del miembro en lugar de duplicar la regla.
- FD5: reabrir el grupo conserva la devolución anterior con sus archivos en el historial, y el nuevo intento empieza sin ellos. Publicar la devolución de un miembro no publica ni revela los archivos de otro.

## Datos y contratos

Migración 014: tabla `grade_attachments` con clave foránea compuesta a `submission_grades(submission_id, student_id)`, ranura 1..5, nombre, tipo, tamaño, contenido y cargador, con unicidad por miembro y ranura e índice por cargador. Los archivos cuelgan de la nota, de modo que la propia base impide que existan sin ella.

Rutas: listar, subir y borrar archivos de devolución de un miembro (`/teacher/submissions/{submissionId}/grades/{studentId}/attachments...`) y descarga del alumno por su propia devolución. OpenAPI antes del código.

## Validación prevista

- Integración PostgreSQL: sin calificación, 409 y ningún archivo; con calificación, subida, listado, cuota descontada y devuelta al borrar; archivo de devolución en borrador invisible para el alumno y visible para el docente; tras publicar, el alumno lo descarga y **el otro miembro del equipo no**; docente ajeno 404 y alumno 403; miembro ajeno a la entrega 404; el límite de ranura responde 409.
- Navegador: el docente adjunta una devolución con archivo al calificar, publica y el alumno la descarga desde su tarea.

## Fuera de alcance

Análisis antivirus, previsualización, archivos de devolución en cuestionarios (la devolución de un intento no admite adjuntos), cuotas por curso y borrado automático de devoluciones antiguas: la retención del incremento 14 solo toca archivos de borradores abandonados y **no** toca devoluciones publicadas.
