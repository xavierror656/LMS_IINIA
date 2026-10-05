# Fechas, temporizador y revisión del cuestionario: incremento 9

Especificación previa `openspec/changes/academic-administration/increment-9.md`, QT7–QT13 / T5c / LMS-015/020/021/022. Continúa T5b; no acredita el 98 % ni equivalencia Moodle.

## Implementación y decisiones

El calendario del cuestionario reutiliza las columnas de horario que la publicación ya aplicaba a `lessons`: guardar fechas deja de exigir una tarea y acepta tarea o cuestionario, mientras las lecturas siguen recibiendo 400. `GET /lessons/{lessonId}/availability` sirve ambos tipos con la misma forma y añade `timeLimitSeconds` y `extraSeconds`; un cuestionario informa `upcoming`, `open`, `late` o `closed` a partir del reloj de la base.

El temporizador vive en la configuración publicada: `timeLimitSeconds` entero entre 60 y 43200 segundos o nulo. Al publicar se copia a `lessons.quiz_time_limit_seconds` y queda dentro del snapshot `quiz_config` de la publicación. Comenzar un intento congela `time_limit_seconds`, `expires_at` y el cierre efectivo vigente (`closes_at`), de modo que editar después el límite, el calendario o la excepción no mueve un plazo ya iniciado.

El plazo efectivo es el menor entre `expires_at` y el cierre efectivo vigente. El vencimiento del temporizador es inmutable; el cierre se evalúa en vivo, así que una excepción concedida durante un intento en curso todavía permite guardar o enviar aunque no amplíe el tiempo de resolución. Al alcanzar el plazo el servidor finaliza el intento con las respuestas guardadas y las versiones de pregunta fijadas en su publicación, calcula la nota canónica y marca la lección como completada sin XP, estrellas, gemas ni vidas. La finalización se aplica al guardar, al enviar, al consultar y al comenzar el siguiente intento, así que ningún intento queda activo indefinidamente: después de vencer, guardar responde 409 y enviar devuelve el recibo del intento terminado.

La excepción individual por estudiante se guarda en `quiz_extensions` con auditoría en `quiz_extension_revisions`, versionada por par lección/estudiante y con motivo obligatorio. Solo amplía límites existentes: no crea un vencimiento donde no lo hay, no acorta plazos ni añade minutos si el cuestionario no tiene tiempo máximo. Revocar usa fechas nulas y cero minutos extra. Un alumno no puede configurarse fechas, límite ni excepciones.

La revisión añade la política `after_close`: las soluciones aparecen solo cuando el intento está terminado y su cierre congelado ya pasó, de modo que posponer el cierre general no revela intentos anteriores. Sin cierre registrado la política nunca revela y el compositor lo advierte.

Hallazgo y corrección: el contador del cliente truncaba los segundos, así que podía anunciar el vencimiento hasta un segundo antes de que el servidor rechazara escrituras. `QuizRemainingSeconds` ahora redondea hacia arriba —el aviso nunca muestra menos tiempo del concedido— y una prueba unitaria fija ese redondeo. Lo detectó la prueba de navegador de vencimiento real.

## Archivos y contratos

- `009_quiz_timing.sql`: `lessons.quiz_time_limit_seconds`, columnas temporales de `quiz_attempts` con índice parcial de intentos activos vencidos, y tablas `quiz_extensions`/`quiz_extension_revisions`. Sin borrado ni reinterpretación de datos previos: los intentos existentes quedan sin límite ni cierre congelado.
- `models/quiz.go` y `models/quiz_timing.go`: configuración con límite, snapshot temporal del intento, disponibilidad efectiva y decisiones puras de plazo, vencimiento, segundos restantes y revelación.
- `services/quiz.go`, `services/quiz_extension.go`, `services/schedule.go`, `services/academic.go`, `handlers/quiz.go`, `handlers/schedule.go`, `handlers/api.go`.
- OpenAPI 1.9.0 y tipos generados: esquemas `QuizExtension`, `QuizExtensionInput` y `QuizExtensionList`, campos temporales en `Availability`, `QuizAttempt` y `QuizOverview`, y las rutas de excepción de cuestionario. Se conservan sesiones, `Origin`, límites y `no-store`.
- `QuizComposer.astro` (minutos por intento y política de cierre), `schedule.astro`, `quiz-extensions.astro`, `Quiz.astro`, `QuizAttempt.astro`, `ScheduleView.astro` y `AcademicForm.astro`. Sin islas React nuevas ni dependencias añadidas.

## Evidencia ejecutada

Entorno real: Node 22.23.3, Go 1.27.1 (Windows y Linux), Chromium de Playwright 1.63 y PostgreSQL 18.6 en WSL2 (Ubuntu 26.04), extraído sin root desde los paquetes de Ubuntu y escuchando en `127.0.0.1:5432`. No se usó Docker.

- `go vet ./...`: sin diagnósticos en ambos entornos.
- `go test ./... -count=1` con `TEST_DATABASE_URL` contra PostgreSQL real: todas las pruebas aprobadas. La ejecución definitiva se hizo dentro de WSL (Go 1.27.1 linux, `handlers` 9.7 s) con el modo de módulos por defecto, sin modificar `go.mod` ni `go.sum`; la misma suite ya había aprobado desde Windows (`handlers` 12.4 s). Se aplicaron migraciones y seed dos veces sobre el mismo esquema; la nueva prueba de integración aprueba con 429/403/404/409 esperados y sin filtrar claves privadas.
- Incidencia del entorno: Smart App Control de Windows (en evaluación) denegó de forma intermitente la ejecución de binarios de prueba recién compilados —`fork/exec ...: An Application Control policy has blocked this file`— en paquetes distintos en cada intento, incluidos `services` y `config`. No es un fallo del código: la suite completa pasa dentro de WSL, y un `-buildid` distinto también la dejaba pasar en Windows. Se optó por verificar en WSL en lugar de alterar el código o la política del equipo.
- Integración QT7–QT13: calendario publicado y límite en `lessons`; `upcoming` rechaza comenzar; `open` concede el intento con 60 s restantes exactos; forzar el vencimiento finaliza con nota canónica 100, marca la lección completada y conserva el recibo idempotente del envío; guardar después responde 409; `after_close` no revela antes del cierre y sí después; la excepción rechaza acortar plazos, motivo vacío, alumno no vinculado (404) y alumno (403), guarda versión 1 y detecta la revisión obsoleta (409); el intento siguiente recibe un minuto publicado más cinco concedidos (360 s); sin límite publicado, los minutos extra se rechazan y la disponibilidad no informa límite; revocar y cerrar el calendario rechaza comenzar sin retirar la lectura de intentos anteriores, cuyo cierre congelado sobrevive al cambio general.
- Unitarias nuevas: rango del límite y de la excepción, política `after_close`, transiciones de estado, plazo mínimo, vencimiento, redondeo hacia arriba de los segundos restantes y decisión de revelar soluciones.
- E2E `quiz-timing.spec.ts`: 2 aprobadas, 1.2 m. La primera (7.4 s) recorre compositor con tiempo máximo, fechas de cuestionario en UTC, publicación, cronómetro de 60 s visible, guardado, rechazo 400 de una nota declarada por el navegador, nota 100 calculada en servidor, ausencia de soluciones antes del cierre, 403 al intentarse una excepción como alumno, excepción de tiempo del docente (versión 1), 360 s en el segundo intento, revocación y estados `upcoming` y `closed` sin botón de comienzo. La segunda (1.0 m) espera el minuto real: el contador del cliente muestra `0:00`, deshabilita guardar y enviar, anuncia el vencimiento y, tras reconciliar, el servidor presenta la nota 100 calculada con las respuestas guardadas.
- Regresión con API reiniciada entre suites: `schedule.spec.ts` aprobada (5.9 s) y `quizzes.spec.ts` aprobada (7.0 s).
- `astro check`: 58 archivos, 0 errores, 0 warnings, 0 hints. `vitest`: 4 aprobadas. `astro build` aprobado conservando el aviso existente de chunks superiores a 500 kB.
- Capturas inspeccionadas: `screenshots/quiz-timing-student-tablet.png`, `screenshots/quiz-timing-extensions-tablet.png` y `screenshots/quiz-timing-expired-tablet.png`.
- Incidencia del limitador: encadenar la especificación completa con otras suites produjo 429 del límite por IP ya documentado en TSEC1. Se reinició la API para limpiar su ventana en lugar de debilitar la protección; TSEC1 sigue pendiente.
- No se ejecutaron Docker, pruebas de carga, auditoría WCAG completa ni comparación con Moodle.

## Reproducción

PostgreSQL real sin privilegios de administrador, dentro de WSL:

```bash
wsl -d Ubuntu
mkdir -p ~/pgdownload && cd ~/pgdownload
apt-get download postgresql-18 postgresql-client-18 libpq5 libicu78 libnuma1 liburing2
mkdir -p ~/pg && for d in *.deb; do dpkg -x "$d" "$HOME/pg"; done
export LD_LIBRARY_PATH="$HOME/pg/usr/lib/x86_64-linux-gnu"
B="$HOME/pg/usr/lib/postgresql/18/bin"; mkdir -p "$HOME/pgrun"
"$B/initdb" -D "$HOME/pgdata" -U postgres -E UTF8 --locale=C.UTF-8 --auth-local=trust --auth-host=trust
"$B/pg_ctl" -D "$HOME/pgdata" -l "$HOME/pg.log" -o "-c listen_addresses=0.0.0.0 -p 5432 -c unix_socket_directories=$HOME/pgrun" -w start
"$B/psql" -U postgres -h 127.0.0.1 -c "CREATE ROLE aulaquest LOGIN PASSWORD 'aulaquest'"
"$B/psql" -U postgres -h 127.0.0.1 -c "CREATE DATABASE aulaquest OWNER aulaquest"
"$B/psql" -U postgres -h 127.0.0.1 -c "CREATE DATABASE aulaquest_test OWNER aulaquest"
```

Desde Windows, con la API y Astro activos y `DATABASE_URL` apuntando a esa base:

```bash
# backend: la suite tambien puede ejecutarse dentro de WSL, donde el control de
# aplicaciones de Windows no bloquea los binarios de prueba recien compilados
wsl -d Ubuntu
export PATH="$HOME/golang/go/bin:$PATH"
cd /mnt/c/Users/javie/Documents/deepseek-harness/default-workspace/LMS_IINIA/backend
TEST_DATABASE_URL='postgres://aulaquest:aulaquest@127.0.0.1:5432/aulaquest_test?sslmode=disable' go test ./... -count=1
# frontend, entorno sintético
E2E_ACADEMIC=1 E2E_STUDENT_PASSWORD='change-this-student-password' E2E_TEACHER_PASSWORD='change-this-teacher-password' npm run test:e2e -- quiz-timing.spec.ts
```

En el editor de una actividad **Cuestionario**: **Configurar cuestionario** fija intentos, tiempo máximo y política de soluciones; **Configurar fechas** define apertura, vencimiento y cierre en UTC; **Gestionar tiempos** concede minutos extra y cierre ampliado por estudiante. Publicar aplica todo. Reiniciar la API entre suites intensivas evita el 429 por IP; las pruebas agregan datos sintéticos y no borran avances.

## Límites pendientes

Sin selección aleatoria de preguntas, importación/exportación, recalificación, categorías ni más tipos de pregunta. La revisión tras el cierre exige un cierre registrado: si el docente lo elimina, esa política deja de revelar nada y así se advierte. No hay pausa ni suspensión del temporizador, ni excepción de apertura individual, ni prórroga de tiempo sobre un intento ya iniciado. La cuenta atrás del cliente es solo informativa; el servidor es la autoridad. Docker, carga, WCAG completo y equivalencia Moodle siguen sin ejecutarse, y el objetivo del 98 % continúa sin acreditar.
