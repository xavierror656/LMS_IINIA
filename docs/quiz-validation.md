# Cuestionarios publicados: incremento 8

Especificación previa `openspec/changes/academic-administration/increment-8.md`, QT1–QT6 / LMS-015/018/020/021/022. Continúa T5; no acredita el 98 % ni equivalencia Moodle.

## Implementación y decisiones

Actividad quiz en el monolito Go, con páginas Astro y formularios sin nuevas islas React. Docente selecciona de una a veinte preguntas de su curso, cada una con versión y peso fijos; guarda configuración y publica. El selector pagina el banco conservando selecciones y sus nombres históricos. No es necesario escribir IDs ni JSON. La publicación falla si falta configuración o se usan preguntas ajenas/archivadas; archivar después no altera intentos anteriores.

El alumno inicia, guarda respuestas completas, recarga y envía lo guardado. Backend valida tipo/rango y calcula con los cuatro evaluadores del banco. Nota del intento = promedio ponderado redondeado a entero, mitad hacia arriba. No se acepta score ni identidad del navegador. Enviar congela respuestas/resultado/fecha/publicación. Marcar completado significa terminar un intento, no aprobar; no concede XP, estrellas ni gemas ni consume vidas.

Máximo publicado de uno a diez intentos. Comenzar usa afterAttempt y bloqueos lección → matrícula; repetir o concurrir con el mismo comienzo devuelve el mismo intento sin consumir otro. Guardar controla revisión; enviar es idempotente con la revisión guardada. Un límite reducido conserva intentos ya iniciados. Historial propio acotado a diez; acceso docente solo a intentos terminados de vinculados e inscritos.

Antes de enviar, DTO público contiene pregunta/opciones/peso y respuestas propias, sin claves ni explicaciones. Revisión never o after_attempt queda fijada en la publicación usada al empezar. Nota total siempre visible tras enviar; soluciones individuales solo cuando se permitió. Cambiar revisión no revela retroactivamente intentos anteriores. After_attempt junto a reintentos permite estudiar las soluciones antes del siguiente intento, advertido al docente.

El libro unifica tareas/cuestionarios sin mezclar IDs. Cada cuestionario cuenta un ítem/peso. Política actualmente publicada first/last/highest/average usa solo intentos terminados; promedio redondeado a entero con mitad hacia arriba. Un intento en curso conserva notas anteriores. La vista SQL calcula el agregado sin modificar notas históricas. En promedio, el enlace abre el último intento para inspección, no presenta ese intento como la nota agregada. Paginación y consulta por lote del libro se mantienen.

## Archivos y contratos

- `008_quizzes.sql`: enum quiz, configuración privada/publicación, referencias inmutables de preguntas, tabla quiz_attempts independiente de entregas y vista gradebook_entries. Índices, FK compuesta publicación/lección, unicidad de número y un intento activo por estudiante/lección.
- `models/quiz.go`, `services/quiz.go`, `handlers/quiz.go`, publicación académica y consulta de libro. El DTO interno con claves nunca se serializa en rutas de alumno.
- OpenAPI 1.8.0/tipos generados: configuración/selector/resultados docente, vista/inicio/guardado/envío propios y detalle autorizado docente. Cada ruta conserva sesiones, Origin, límites y no-store.
- `QuizComposer.astro`, `Quiz.astro`, `QuizAttempt.astro`, páginas docentes de configuración/resultados/detalle y conexión con lección/mapa/libro.

## Evidencia ejecutada

- Go/PostgreSQL: `go test ./... -count=1` aprobado, handlers 7.133 s en ejecución final. Esquemas de prueba aislados, migraciones/seed repetidos. QT valida publicación incompleta/ajena, inicio/envío simultáneos, guardado obsoleto, campos de nota rechazados, privacidad entre alumnos/docentes, snapshots tras editar/archivar, reducción del límite, revisión no retroactiva, cuatro políticas y libro mixto. El ajuste final prueba que publicar instrucciones nuevas no altera las del intento iniciado; el DTO y la interfaz muestran las originales con aviso cuando difieren.
- Caso numérico: intentos 25 y 100 producen primera 25, última 100, mejor 100, promedio 63. Cuestionario promedio 63 con peso 2 y tarea 40 con peso 1 producen 55.33 en el libro. Cada actividad cuenta una vez.
- Unitarias de configuración y redondeo ponderado añadidas; suite de evaluadores del banco sigue aprobada. Go vet sin diagnósticos y API compilada. Migración aplicada a academic_preview sin borrar datos.
- Astro check: 57 archivos, cero errores/warnings/hints. Build SSR final aprobado en 23.62 s; continúa aviso previo de chunks superiores a 500 kB.
- QT6 Playwright aprobado: 31.5 s (escenario 29.4 s). Autoría/publicación, claves ausentes, fallo de red conservando respuestas, guardado/recarga, dos intentos con notas 25/100, límite, recompensas sin cambios y libro/detalle docente. Capturas tablet `quiz-student-tablet.png` y `quiz-teacher-tablet.png` inspeccionadas: respuestas conservadas, alumno sin soluciones y docente con revisión autorizada.
- Regresión academic/gradebook/learning: siete escenarios aprobados en 14.9 s antes del ajuste aislado de instrucciones históricas. No se modificó el código de esos recorridos con ese ajuste; QT6 ampliado comprobará el cambio final.
- OpenSpec strict válido y diff check sin errores. No se ejecutaron equivalencia Moodle, carga, auditoría WCAG completa ni Docker.

## Reproducción y pendientes

Aplicar migraciones con `go run ./cmd/migrate` desde backend y arrancar Go/Astro según README. Demo temporal en `http://localhost:4323/login`: profe/academic-teacher-pass y luna/academic-student-pass. Docente crea actividad **Cuestionario**, abre **Configurar cuestionario**, selecciona preguntas/versión/pesos/intentos/políticas, guarda y vuelve a publicar. Alumno abre curso → cuestionario → comienza → guarda → envía. Docente consulta **Ver intentos del cuestionario** y libro.

Prueba desde frontend, con entorno sintético activo: `E2E_ACADEMIC=1 E2E_BASE_URL=http://localhost:4323 E2E_TEACHER_PASSWORD=academic-teacher-pass E2E_STUDENT_PASSWORD=academic-student-pass npm run test:e2e -- quizzes.spec.ts`.

Pendientes explícitos: apertura/cierre, temporizador del servidor, revisión tras cierre, excepciones de tiempo, aleatoriedad, importación/exportación, categorías, más tipos y recalificación. Se mantienen en T5/T5c y el inventario; no son capacidades cumplidas. H5P real y runner aislado siguen pendientes, junto con grupos y resto de plataforma.
