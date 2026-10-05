# Formatos, archivos de instrucciones y retención: incremento 14 (T4g)

Especificación previa `openspec/changes/academic-administration/increment-14.md`, FA1–FA3, FI1–FI3 y FR1–FR3. Amplía T4e/T4f (adjuntos privados, ya verificados). Cubre **tres de las cuatro** partes de T4g: los adjuntos de devolución quedan pendientes y así se registran. No acredita el 98 % ni equivalencia Moodle.

## Implementación y decisiones

**PDF con análisis real.** El formato entra solo tras examinar el contenido, no la extensión: cabecera `%PDF-1.x`, tabla de referencias (`xref` o `/XRef`), `%%EOF` en la cola, al menos una página y como mucho 200, y rechazo de cifrado y de contenido activo declarado (`/Encrypt`, `/JavaScript`, `/JS`, `/Launch`, `/RichMedia`, `/EmbeddedFile`), exigiendo un delimitador tras el nombre para no confundirlo con un nombre más largo. La prueba construye un PDF **de verdad** —con su tabla de referencias— y comprueba que cada desplazamiento apunta a su objeto, así que el analizador se ejercita contra un archivo real y no contra una cabecera falsa.

Honestidad sobre el alcance: es una **criba estructural, no un saneador**. Los flujos de objetos van comprimidos, de modo que un marcador puede esconderse dentro y la búsqueda no puede demostrar su ausencia. Por eso un PDF se entrega **siempre como descarga opaca** (`application/octet-stream`, `Content-Security-Policy: default-src 'none'; sandbox`, `nosniff`), nunca en línea.

**Archivos de instrucciones.** El docente sube hasta cinco archivos (2 MiB cada uno) a la actividad en borrador; la cuota se descuenta a quien sube y se le devuelve al borrar. Publicar **congela una copia** en la publicación, igual que el enunciado y la rúbrica: una versión anterior conserva los suyos aunque el borrador cambie o los borre, de modo que quien entregó contra esas instrucciones puede seguir consultándolas. El alumno inscrito ve los archivos de la versión con la que trabaja —la de su propia entrega si la tiene, y si no la última publicada— y nunca los de un borrador ni los de otra lección.

**Retención.** Se limita a lo que puede borrarse sin destruir trabajo evaluado: archivos de **borradores abandonados** (sin envío, sin intento derivado y sin cambios en más de N días, 180 por omisión). El comando `cmd/retention` informa antes de borrar, **no borra nada sin `-confirm`**, devuelve los bytes a quien los subió y conserva intactos el texto del borrador y la entrega. Nunca toca archivos de entregas enviadas o calificadas ni los de instrucciones publicadas, e informa de cuántos archivos publicados examina sin eliminarlos.

## Hallazgos y correcciones

- **El proxy habría rechazado la subida nueva.** `infra/nginx.conf` solo ampliaba `client_max_body_size` para la ruta del alumno, así que la ruta del docente habría fallado con 413 antes de llegar a la aplicación. Ambas rutas comparten ahora la misma expresión y el mismo límite (2 MiB + 64 KiB), coherente con el límite de cuerpo de la aplicación y con la comprobación de 128 KiB para el resto de solicitudes.
- **GORM vacía el destino en cada `Scan`.** El informe de retención leía los dos recuentos en el mismo struct y el segundo borraba el primero: informaba de cero archivos abandonados. Lo cazó la prueba de integración; ahora cada lectura tiene su destino.
- **`submissions` no tenía marca de tiempo propia.** Sin ella la regla de abandono sería una suposición. La migración añade `updated_at` y todas las escrituras (guardar, enviar, adjuntar, borrar adjunto) la mantienen al día.

## Archivos y contratos

- `013_instruction_attachments.sql`: `application/pdf` en la restricción de tipo; tablas `activity_attachments` y `publication_attachments` con la misma forma que los adjuntos de entrega; `submissions.updated_at` con índice parcial de borradores.
- `services/attachment_content.go` con el analizador; `services/instruction_attachments.go` con archivos, congelado y retención; `handlers/instruction_attachments.go`; `cmd/retention`.
- Cinco rutas nuevas (listar, subir y borrar instrucciones del docente; listar y descargar del alumno) y `Activity.attachments`; OpenAPI con `AttachmentList` y el PDF en el enum (60 rutas, 87 esquemas) y tipos regenerados.
- Interfaz: sección **Archivos de instrucciones** en la actividad con subida, listado y borrado; en la tarea del alumno, lista de descarga de la versión que le corresponde.

## Evidencia ejecutada

- `gofmt` sobre copia normalizada y `go vet ./...`: sin hallazgos.
- `go test ./... -count=1` con `TEST_DATABASE_URL` contra PostgreSQL 18.6 real dentro de WSL: **todo aprobado**, `handlers` 31.3 s.
- Unitarias del analizador: PDF válido aceptado y almacenado sin cambios con su tipo; sin cabecera, sin `xref`, sin `%%EOF`, cifrado, `JavaScript`, `EmbeddedFile` y `Launch` rechazados; más de 200 páginas rechazado como demasiado grande; `/Type /Pages` no cuenta como página; nombres más largos no coinciden; texto e imagen conservan sus reglas y una imagen con extensión falsa se rechaza.
- Integración: el docente sube, lista y borra; ajeno 404 y alumno 403; publicar congela la copia; borrar del borrador no altera la versión publicada; el alumno ve y **descarga el mismo archivo** con tipo opaco y política de sandbox; la cuota se descuenta al subir y se devuelve al borrar; sin cupo, 409. Retención: el informe enumera un archivo abandonado **sin borrar nada** e informa de los publicados que nunca toca; la purga borra ese archivo, devuelve los bytes, **conserva el texto del borrador** y no toca el archivo de una entrega enviada.
- Navegador: `attachments.spec.ts` ampliada y **aprobada** (7.8 s): la guía en PDF se sube antes de publicar, el alumno la ve y la descarga, y quitarla del borrador no rompe la versión publicada. Regresión `quizzes`, `academic`, `group-submission` y `groups` aprobada.
- `astro check`: 60 archivos, 0 errores. `vitest`: 4 aprobadas. `astro build`: correcto.

## Nginx y Docker: lo que no pudo ejecutarse

Se revisó la aritmética de límites y se corrigió el hueco descrito. **Nginx y Docker no están instalados en este entorno** (`nginx` y `docker` no existen en WSL), así que no se ejecutó `nginx -t`, ni una subida real a través del proxy, ni el arranque por Compose. Queda declarado como no validado, no como aprobado.

## Reproducción

```bash
go run ./cmd/migrate && go run ./cmd/api           # backend
npm --prefix frontend run dev                       # frontend
go run ./cmd/retention                              # informe, sin borrar
go run ./cmd/retention -days 180 -confirm           # borra lo enumerado
```

En la actividad: **Archivos de instrucciones** → elegir texto, imagen o PDF → **Subir archivo** → publicar para congelarlos. En la tarea del alumno aparecen como descargas.

## Límites pendientes de T4g

Los **adjuntos de devolución** (archivos del docente junto a la nota de un estudiante) siguen sin implementarse: es la cuarta parte de T4g y queda como siguiente rebanada. No hay análisis antivirus ni conversión o previsualización de documentos, solo texto, imagen y PDF; no hay cuotas por curso; la retención se ejecuta a mano y con confirmación, sin borrado programado; y la validación con Nginx y Docker sigue pendiente de un entorno que los tenga.
