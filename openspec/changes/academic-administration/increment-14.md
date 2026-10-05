# Incremento 14: formatos, adjuntos de instrucciones y retención (T4g)

Especificación previa al código. T4g / LMS-016/019, ampliación de T4e/T4f (adjuntos privados, ya verificados). No acredita el 98 % ni equivalencia Moodle.

## Alcance de esta rebanada

T4g pedía cuatro cosas. Esta rebanada cubre tres y deja la cuarta documentada como pendiente:

1. **Más formatos con análisis de contenido real**: PDF.
2. **Adjuntos de instrucciones**: archivos del docente junto al enunciado, congelados por versión publicada.
3. **Gestión de retención**: política explícita y comando con informe previo.
4. *(Pendiente)* **Adjuntos de devolución**: archivos del docente junto a la nota de un estudiante.

## Reglas y aceptación

- FA1: el PDF entra en los formatos admitidos **solo tras un análisis estructural real** del contenido, no por su extensión: cabecera `%PDF-1.x`, presencia de tabla de referencias (`xref` o `/XRef`) y de `%%EOF` en la cola, al menos una página `/Type /Page` y como mucho 200, y rechazo de cifrado y de contenido activo declarado (`/Encrypt`, `/JavaScript`, `/JS`, `/Launch`, `/RichMedia`, `/EmbeddedFile`). Se aplican las mismas reglas que ya valen para texto e imágenes: tamaño máximo, cuota por cargador y nombre normalizado.
- FA2: el análisis es una **criba estructural, no un saneador**. Los flujos comprimidos pueden ocultar marcadores, así que el PDF nunca se sirve en línea: la descarga mantiene `application/octet-stream` y `Content-Security-Policy: default-src 'none'; sandbox`. Esta limitación se documenta y no se presenta como aislamiento suficiente.
- FI1: el docente adjunta archivos de instrucciones a una actividad en borrador (hasta 5, 2 MiB cada uno). La cuota se descuenta a quien sube y se le devuelve al borrar; solo el docente del curso puede subir, listar y borrar los de su actividad.
- FI2: **publicar congela una copia** de esos archivos en la publicación, igual que el enunciado y la rúbrica. Una publicación anterior conserva los suyos aunque el borrador cambie o los borre, de modo que quien entregó contra esas instrucciones puede seguir consultándolas. Retirar un archivo del borrador no altera publicaciones ya emitidas.
- FI3: el alumno inscrito ve y descarga los archivos de instrucciones de la versión con la que trabaja: la de su propia entrega si la tiene, y si no la última publicada. Nunca ve los de un borrador ni los de otra lección.
- FR1: la retención se limita a lo que puede borrarse sin destruir trabajo evaluado: **adjuntos de borradores abandonados** (entregas en estado borrador sin envío posterior, sin cambios en más de N días, por omisión 180). Nunca toca archivos de entregas enviadas, calificadas o con intento posterior, ni los de instrucciones publicadas.
- FR2: el comando informa antes de borrar: por omisión **no borra nada** y solo enumera qué haría; borrar exige confirmación explícita. Al borrar devuelve los bytes a la cuota de quien los subió y conserva el texto del borrador y la entrega. Cada ejecución registra cuántos archivos y bytes se examinaron, se borrarían o se borraron.
- FR3: la política queda escrita, incluido lo que **no** se borra automáticamente y por qué.

## Datos y contratos

Migración 013: `application/pdf` añadido a la restricción de tipo de `submission_attachments`; tablas `activity_attachments` (borrador) y `publication_attachments` (congelado por versión), con la misma forma que `submission_attachments` (ranura 1..5, nombre, tipo, tamaño, contenido, cargador, unicidad por contenedor y ranura) e índices por cargador. Publicar copia las filas del borrador a la publicación.

Rutas nuevas: subir, listar y borrar archivos de instrucciones del docente, y listar y descargar los de la publicación para el alumno. OpenAPI antes del código.

## Validación prevista

- Unitarias del analizador: PDF válido aceptado; sin cabecera, sin `xref`, sin `%%EOF`, con `/Encrypt` o con `/JavaScript` rechazados; más de 200 páginas rechazado; extensión falsa rechazada; texto e imagen siguen igual.
- Integración PostgreSQL: docente sube, lista y borra; cuota descontada y devuelta; alumno ajeno 403 y docente de otro curso 404; publicar congela la copia y una segunda publicación no borra la primera; el alumno ve la versión que le corresponde; borrar del borrador no altera la publicación.
- Integración de retención: borrador viejo con archivo se vacía, el texto permanece, la cuota vuelve y una entrega enviada o calificada no se toca; el informe sin confirmación no borra nada.
- Navegador: el docente sube un archivo de instrucciones y lo ve publicado; el alumno lo ve y lo descarga.
- Nginx/Docker: se revisa la aritmética de límites (`client_max_body_size` frente al límite de cuerpo de la aplicación) y se declara con honestidad qué no puede ejecutarse en este entorno.

## Fuera de alcance

Adjuntos de devolución por estudiante (siguiente rebanada de T4g), análisis antivirus, contenido distinto de texto, imagen y PDF, conversión o previsualización de documentos, cuotas por curso y cualquier borrado automático programado: la retención se ejecuta a mano y con confirmación.
