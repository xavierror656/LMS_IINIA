# Adjuntos privados: incremento 5

Especificación previa: `openspec/changes/academic-administration/increment-5.md`, FL1–FL5 / LMS-016/019/021/022. No acredita 98 % ni equivalencia con Moodle; conserva pendientes de formatos y otros usos de archivos.

## Implementación y decisiones

Se admiten TXT UTF-8, PNG y JPEG en borradores propios, con 2 MiB por archivo (antes y después de normalizar), cinco archivos por entrega, cuatro millones de píxeles por imagen y 50 MiB por estudiante entre cursos. Se valida nombre/extensión/contenido; las imágenes se decodifican y recodifican sin metadatos, JPEG a calidad 85. El proceso permite dos normalizaciones simultáneas. Esto no es un antivirus; formatos activos y no admitidos se rechazan, incluyendo PDF, ofimática, SVG y ZIP.

Archivos bytea en PostgreSQL y metadatos privados permiten confirmar blob, cuota y revisión en la misma transacción. No hay URL pública ni ruta de disco proporcionada por usuario. Las cinco posiciones tienen restricción UNIQUE por entrega; una cuenta de bytes bloqueada serializa cuotas entre cursos. Si una subida falla, se revierte también un borrador vacío creado por ese intento. Eliminar un adjunto de borrador libera su espacio.

Subir/eliminar incrementa versión de borrador; enviar congela texto y archivos. Se acepta entrega con solo archivos y se rechaza envío completamente vacío. Fechas y prórrogas también se aplican a operaciones de archivos. Los reintentos con revisión antigua dan 409 para evitar duplicación; tras fallo de conexión la interfaz indica recargar antes de repetir.

Estudiante necesita propiedad e inscripción vigente para descargar. Docente necesita curso asignado, vínculo e inscripción del estudiante, y entrega ya enviada. Descarga forzada con Content-Disposition seguro, octet-stream, private/no-store, nosniff y CSP sandbox. Listados de estudiantes y docente incluyen únicamente metadatos; nunca blobs en JSON. La consulta de archivos de la bandeja docente se realiza por lote, no una consulta por entrega.

## Archivos

- `backend/migrations/005_submission_attachments.sql`: archivos, cuenta de cuota, posiciones únicas y texto de borrador opcional.
- `backend/internal/services/attachment_content.go`, `attachments.go`, `handlers/attachments.go`, modelo de metadata y cambios de envío/listado.
- OpenAPI 1.5.0 y tipos generados; API JSON conserva 128 KiB, ruta multipart 2 MiB + 64 KiB, WS conserva 24 KiB. Subidas limitadas además a 20/min por estudiante.
- `frontend/src/pages/lessons/[lessonId]/files.astro`, componentes `AttachmentManager.astro` y `AttachmentList.astro`; lección y corrector integrados. Sin dependencias adicionales ni React nuevo.
- `infra/nginx.conf`: JSON 128 KiB y excepción regex de subida de 2112 KiB. El límite anterior de 24 KiB en Nginx era incoherente con JSON de 128 KiB y se corrigió. No se ejecutó Nginx ni Docker: `command -v nginx` no encontró ejecutable. La configuración está revisada, no validada en contenedor.

## Pruebas ejecutadas

- Go test con PostgreSQL separado: aprobado, handlers 3.935 s en ejecución final. Migraciones/seed repetidos en esquema nuevo. Valida formatos falsos/nombres peligrosos, archivo grande, borrador invisible, descarga propia byte a byte y headers, rechazo de alumno ajeno/docente ajeno, listado docente después del envío, cierre, envío solo con archivo, bloqueo de cambios después de enviar y JSON ordinario sobre 128 KiB rechazado.
- Dos subidas con la misma versión generan una sola inserción. Cinco posiciones impiden sexto archivo. Cuota simulada al límite rechaza sin alterar revisión. Dos subidas simultáneas en cursos distintos, con espacio para una, producen 201/409 y un solo borrador: comprueba bloqueo de cuenta y rollback.
- Unitarias: UTF-8/control/NUL, nombre con separadores y controles, tipos no admitidos, extensión falsa, tamaño recibido, imagen inválida, límite de píxeles y eliminación de bytes extra al recodificar PNG.
- Go vet sin diagnósticos; API compilada. Migración 005 aplicada a academic_preview conservando datos existentes.
- Astro check: 45 archivos, cero errores/warnings/hints. Build SSR aprobado; permanece aviso previo de chunks superiores a 500 kB.
- Vitest `--pool=threads`: cuatro pruebas aprobadas, 28.29 s.
- E2E de adjuntos aprobada: 22.6 s en primera ejecución y 6.5 s en la última, que además espera al HUD reconciliado antes de capturar. Comprueba rechazo de PNG falso conservando selección, creación de borrador por subida, descarga real y contenido exacto, recarga, borrado, nueva subida, envío sin texto y acceso docente después de enviar. Capturas tablet inspeccionadas: `screenshots/attachments-student-tablet.png`, `screenshots/attachments-teacher-tablet.png`.
- OpenSpec strict válido y `git diff --check` sin errores.

Incidencia: al agregar archivos al final de la integración académica, su instancia acumuló más de 180 solicitudes/min y respondió 429 antes de concluir permisos. Las pruebas de archivos usan ahora una instancia de transporte separada con la misma configuración y limitadores activos, conservando base y sesiones de prueba. No se aumentó ni desactivó el límite de producción.

## Demo y reproducción

Demo temporal: `http://localhost:4323/login`, `luna` / `academic-student-pass` para subir y `profe` / `academic-teacher-pass` para revisar. El alumno abre una tarea → **Gestionar adjuntos** → adjunta → vuelve a la tarea → **Enviar al docente**. El docente abre **Entregas para revisar** y descarga desde la entrega. Esto utiliza Go/PostgreSQL, no la API Node de `npm run demo`.

Con variables y PostgreSQL del README:

```bash
# backend, terminal 1
go run ./cmd/migrate
go run ./cmd/api
# raíz, terminal 2
npm --prefix frontend run dev
# frontend, entorno sintético con servidores activos
E2E_ACADEMIC=1 E2E_STUDENT_PASSWORD='change-this-student-password' E2E_TEACHER_PASSWORD='change-this-teacher-password' npm run test:e2e -- attachments.spec.ts
```

No compilar en paralelo con E2E del servidor dev. Separar suites intensivas según la ventana del limitador. Las pruebas agregan datos sintéticos y no borran datos ajenos.

## Pendientes reales

PDF/ofimática con análisis externo, otros formatos, archivos de instrucciones/devoluciones, almacenamiento de objetos, interfaz administrativa de cuota/retención y borrado de archivos enviados. No modificar archivos congelados para recuperar cuota sin un flujo autorizado. No se ejecutaron análisis antivirus, auditoría WCAG, pruebas de carga, Docker ni equivalencia Moodle. Runner sigue simulado y H5P sin contenido válido verificado. El objetivo general continúa activo.
