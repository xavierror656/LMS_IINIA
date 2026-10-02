# Reentregas e historial: incremento 6

Especificación previa: `openspec/changes/academic-administration/increment-6.md`, AT1–AT6 / LMS-016/017/018/021/022. La implementación interna no acredita equivalencia Moodle ni el objetivo del 98 %.

## Comportamiento

Docente configura de uno a diez intentos totales, guarda y publica la actividad. Desde una entrega enviada autoriza un nuevo intento con motivo obligatorio. La reapertura crea un borrador vacío; no cambia fechas ni prórrogas. Reducir el máximo conserva borradores ya autorizados. Solo docentes asignados con vínculo y matrícula vigente pueden hacerlo.

Texto, archivos, instrucciones, rúbrica, puntualidad y notas anteriores permanecen en su intento. No se copian archivos ni se duplica su cuota. Alumno consulta historial propio y únicamente notas publicadas. Docente sigue sin poder leer borradores; reintentar reapertura devuelve solo id/número de intento.

Libro usa el último intento enviado por tarea y estudiante. Mientras el nuevo intento sea borrador se conserva la nota anterior; enviarlo la sustituye por pendiente. Cada tarea cuenta una sola vez. No hay elección de mejor nota ni promedio de intentos, reapertura automática, intentos ilimitados o borrado de historial en este incremento.

## Datos, contratos y seguridad

- Migración `006_submission_attempts.sql`: máximo de actividad/lección, número de intento, autor/motivo/fecha de reapertura y enlace al anterior. UNIQUE por alumno/lección/intento, anterior único, un borrador por alumno/lección y FK compuesta que impide enlazar otra persona o tarea. Registros existentes conservan su id como intento 1.
- La transacción bloquea lección, matrícula y entrega en ese orden. Las calificaciones usan ahora el mismo orden y verifican matrícula vigente; listar entregas también verifica matrícula. El libro continúa con consultas por lote e instantánea repeatable-read.
- Nueva revisión de borrador comienza por encima de la anterior. Formularios obsoletos no modifican ni envían un intento posterior aunque apunten a la misma lección. Reapertura concurrente/repetida del mismo origen devuelve un recibo único y conserva el primer motivo auditado.
- OpenAPI 1.6.0 y tipos generados. PUT `/teacher/activities/{activityId}/attempt-policy`, POST `/teacher/submissions/{submissionId}/reopen`, GET `/lessons/{lessonId}/submission/history`; consulta histórica propia con `submissionId` en GET submission. Historial acotado a diez metadatos, detalle y archivos bajo autorización habitual.
- Páginas Astro para configurar intentos e historial de solo lectura. Formularios existentes conservan entradas tras errores y limpian listeners al navegar. No se añadieron dependencias ni islas React.

## Evidencia ejecutada

- Go/PostgreSQL: `go test ./... -count=1` aprobado; última ejecución de handlers 4.989 s. Se aplican migraciones y seed dos veces en esquema de prueba separado. AT prueba permisos, rango, política no publicada, reapertura simultánea e idempotente, ausencia de filtración del borrador, versiones obsoletas, conservación/descarga y prohibición de borrar el archivo anterior, cierre, reducción del máximo, historial y notas, agregación sin duplicar, auditoría y revocación por matrícula.
- `go vet ./...` sin diagnósticos y API final compilada. Migración aplicada a la demo preservando datos.
- Astro check: 47 archivos, cero errores/warnings/hints. Build SSR final aprobado en 18.90 s; aviso existente de chunks mayores de 500 kB continúa.
- Vitest con `--pool=threads`: cuatro pruebas aprobadas, 23.95 s.
- AT6 Playwright aprobado en primera ejecución: 4.7 s, configuración/publicación, nota, reapertura, historial, segundo envío, límite y celda del libro. Captura tablet `screenshots/attempt-history-tablet.png` inspeccionada: texto legible, HUD reconciliado e historial separado del formulario editable.
- En la misma primera ejecución FL5 falló antes de iniciar sesión: el trace registra 504 de Vite al cargar `nanostores.js` y el botón permaneció deshabilitado. No fue una prueba aprobada de adjuntos. Repetición con dependencias estabilizadas y API final: FL5 y AT6 aprobados, dos pruebas en 9.3 s (4.0 s y 3.7 s).
- Regresión posterior al build final y reinicio de Astro: `academic.spec.ts`, `gradebook.spec.ts` y `learning.spec.ts`, siete pruebas aprobadas en 31.5 s. Con AT6/FL5, nueve escenarios de navegador aprobados en ejecuciones separadas. No se ejecutaron build/check en paralelo con esta regresión.
- OpenSpec strict válido. No se ejecutó comparación contra Moodle, carga sostenida, Docker ni auditoría completa de accesibilidad.

## Uso de la demo

`http://localhost:4323/login`: `profe` / `academic-teacher-pass`, `luna` / `academic-student-pass`. Docente abre tarea → **Configurar intentos** → guarda → vuelve y publica. Después de un envío, abre **Autorizar otro intento**, escribe motivo y confirma. Alumno vuelve a la tarea y entra a **Ver mis intentos anteriores** para consultar el trabajo conservado.

Para reproducción estable se mantienen los comandos de migración, seed, Go y Astro del README. Esta demo temporal depende de procesos locales y herramientas en `/tmp`; no es un despliegue. Runner continúa simulado; H5P real y resto de capacidades del inventario siguen pendientes.
