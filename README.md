# AulaQuest

LMS infantil con Astro SSR, islas React y API Go/Fiber. PostgreSQL es la fuente de verdad; GORM se usa para acceso a datos y las migraciones SQL versionadas gobiernan el esquema. Interfaz en español, marca centralizada en `frontend/src/config/brand.ts`.

## Qué incluye

Catálogo y mapa de cursos inscritos, lecturas con recompensa persistente, HUD, CodeMirror JavaScript/Python conectado a WebSocket real, sesiones HttpOnly, estudiantes/docente y panel de progreso por curso. El código del estudiante **no se ejecuta**: MockRunner emite eventos simulados. H5P está integrado y desactivado hasta proporcionar un paquete completo y confiable; muestra “Actividad aún no configurada”.

## Ver la aplicación sin Docker (modo demo)

Solo necesitas **Node 22.12 o superior** y npm. No requiere Go, Docker, PostgreSQL ni configurar archivos .env.

Desde la raíz:

```bash
npm --prefix frontend ci
npm run demo
```

Espera el mensaje `ready` de Astro y abre **http://localhost:4321/login**.

| Usuario demo | Rol | Contraseña |
|---|---|---|
| `luna` | Estudiante, comienza sin logros | `aulaquest-demo` |
| `sol` | Estudiante con avances de ejemplo | `aulaquest-demo` |
| `profe` | Docente de Luna | `aulaquest-demo` |

Incluye dos cursos, tres módulos y siete lecciones. Puedes completar lecturas, ver cambios en el HUD, cambiar de página, recargar, probar el laboratorio y consultar el progreso como docente. El aviso **Modo demo · Datos ficticios** distingue esta vista del backend real.

El progreso se guarda en `frontend/.demo/state.json`, solo en este equipo. Las sesiones se cierran al reiniciar. Para empezar otra demostración, detén con Ctrl+C y renombra `state.json` a `state.backup.json`; al volver a iniciar se cargarán los datos iniciales. Ese archivo no se sube a Git ni a imágenes Docker.

La API mock Node existe únicamente para esta vista local; el backend real sigue usando Go, GORM y migraciones PostgreSQL. La consola está simulada y H5P sigue sin paquete configurado. Si el puerto 4321 está ocupado se muestra un error para evitar abrir otra instancia por accidente. **Ctrl+C** detiene ambos servidores.

Pruebas del modo demo: `npm --prefix frontend run test:demo`. Para comprobar la interfaz con Playwright, deja la demo activa y usa `E2E_STUDENT_PASSWORD=aulaquest-demo E2E_TEACHER_PASSWORD=aulaquest-demo npm --prefix frontend run test:e2e` sobre el perfil Luna inicial (solo la primera lectura completada).

## Requisitos

Node >=22.12 (verificado 22.23.3), npm >=9.6.5 (usado npm 10), Go 1.27.1 y PostgreSQL 18. Docker Compose v2 para el modo contenedores. No utilizar datos reales de menores en esta demostración.

## A. Desarrollo con dos terminales

Desde la raíz (Bash/WSL):

```bash
cp .env.example .env
cp backend/.env.example backend/.env
cp frontend/.env.example frontend/.env
npm --prefix frontend ci
docker compose up -d db
```

Los archivos de ejemplo coinciden para desarrollo local. Si cambias la contraseña PostgreSQL, actualiza `.env` y `DATABASE_URL` en `backend/.env`. No se cargan archivos `.env` automáticamente en Go.

Terminal 1:

```bash
cd backend
set -a
source .env
set +a
go mod download
go run ./cmd/migrate
go run ./cmd/seed
go run ./cmd/api
```

Terminal 2, desde la raíz:

```bash
npm --prefix frontend run dev
```

Abre http://localhost:4321. API en http://localhost:8080; Astro reenvía `/api` y `/ws`. PostgreSQL solo se publica en 127.0.0.1:5432. Si ya tienes PostgreSQL, crea una base vacía y ajusta DATABASE_URL en lugar de iniciar `db`.

Astro 7 puede activar modo background en entornos de agentes. Si alcanza su timeout inicial, el comando verificado `npm --prefix frontend run dev -- --ignore-lock` mantiene el servidor en primer plano. No lo uses para iniciar una segunda instancia sobre el mismo puerto.

## B. Docker Compose

Desde la raíz:

```bash
cp .env.example .env
docker compose up --build
```

Compose espera PostgreSQL, ejecuta migraciones y seed, inicia API/frontend y sirve el origen único por Nginx en http://localhost:4321. Los datos quedan en el volumen `postgres_data`. Para detener conservando datos: `docker compose down` (sin `-v`).

Para aplicar una migración nueva:

```bash
docker compose run --rm migrate
```

El entorno Compose entregado es de desarrollo: no habilita TLS y ejecuta seed sintético. No es un despliegue de producción. No se pudo ejecutar Docker en el entorno de implementación; véase `docs/validation.md`.

## Cuentas sintéticas

| Usuario | Rol | Contraseña con los ejemplos sin modificar |
|---|---|---|
| `luna` | student, vinculada a profe | `change-this-student-password` |
| `sol` | student, no vinculada a profe | `change-this-student-password` |
| `profe` | teacher | `change-this-teacher-password` |

Se configuran mediante `SEED_STUDENT_PASSWORD` y `SEED_TEACHER_PASSWORD` antes del primer seed (12–72 bytes). El seed es idempotente: **no cambia contraseñas existentes ni borra datos**. Modificar la variable después no cambia una cuenta ya creada. Se rechaza el seed con APP_ENV=production; no hay registro público.

## Migraciones y reglas de datos

`backend/migrations/001_initial.sql` crea tablas, FK, índices, checks y restricciones únicas. `go run ./cmd/migrate` aplica SQL embebido pendiente en orden, bajo transacción y advisory lock; registra nombres en `schema_migrations`. Añadir migraciones nuevas con prefijos ordenados (`002_...sql`); nunca editar una migración ya aplicada. No se usa AutoMigrate ni hay rollback destructivo automático: restaurar un backup o escribir una migración compensatoria revisada.

La finalización de lectura es declarada por el estudiante. La transacción bloquea su perfil e inserta reward_events con UNIQUE(user_id,lesson_id): ni repetición, ni concurrencia, ni cambiar Idempotency-Key duplican 25 XP, 1 estrella y 2 gemas. Nivel = 1 + floor(XP/100); las 5 vidas no bloquean contenido. Code/H5P no tienen evaluación validada ni premios en este MVP. Los intentos H5P se guardan como client_reported sin actor ni declaración xAPI completa.

## Comprobaciones

```bash
npm --prefix frontend run types:api
npm --prefix frontend run check
npm --prefix frontend test
npm --prefix frontend run build
cd backend
go test ./...
go vet ./...
```

Sin TEST_DATABASE_URL, la prueba de integración se **omite explícitamente**. Usar una base exclusiva con nombre terminado en `_test`; crea un esquema único por ejecución y nunca borra datos previos:

```bash
TEST_DATABASE_URL='postgres://usuario:clave@localhost:5432/aulaquest_test?sslmode=disable' go test -count=1 ./...
```

Los esquemas de prueba se conservan para inspección; su limpieza corresponde al operador de esa base descartable.

Pruebas de navegador contra un entorno de demostración sembrado y servidores activos (no producción):

```bash
cd frontend
npx playwright install chromium
E2E_STUDENT_PASSWORD='change-this-student-password' E2E_TEACHER_PASSWORD='change-this-teacher-password' npm run test:e2e
```

Chromium necesita sus bibliotecas del sistema; en Linux puede ser necesario `npx playwright install --with-deps chromium` con permisos administrativos. Las E2E completan la primera lectura de Luna; pueden repetirse y esperan que ese perfil no haya completado otras lecturas. `E2E_BASE_URL` permite cambiar el origen.

## Contratos y estructura

- `openspec/changes/bootstrap-kids-lms/`: propuesta, requisitos LMS-001..010, diseño y tareas.
- `contracts/openapi.json`: REST; genera `frontend/src/lib/generated/api.d.ts` y los tipos usados por el cliente.
- `contracts/websocket.md`: protocolo v1, límites y errores.
- `backend/cmd/{api,migrate,seed}`: ejecutables separados.
- `backend/internal/{handlers,services,repositories,models,middleware,ws,runner}`: monolito modular.
- `frontend/src/pages/`: Astro, sin SPA global; `components/react/`: HUD, CodePlayground y H5PPlayer.
- `frontend/public/h5p/README.md`: instalación de contenido completo, bibliotecas, recursos y atribución.
- `infra/nginx.conf`, Dockerfiles y Compose: base local de ejecución.
- `docs/versions.md`, `docs/test-plan.md`, `docs/validation.md`: versiones y evidencia.

## Seguridad y privacidad

Cookie de 12 h HttpOnly, SameSite=Lax, Secure con APP_ENV=production; sesión aleatoria cuyo hash se guarda en DB. Logout revoca sesión y conexiones. Todos los POST y WS validan Origin exacto, incluido login; clientes de API deben enviar APP_ORIGIN en Origin. Sin CORS abierto. Las rutas privadas autorizan inscripción o vínculo docente, no un user_id enviado por cliente. Páginas/API privadas no-store. SSR reenvía solo aq_session al API_INTERNAL_URL, nunca secretos PUBLIC_*. La store se usa exclusivamente en cliente y se limpia al cerrar sesión.

Una conexión y ejecución WS por estudiante en esta única instancia; 16 KiB de código, 24 KiB por mensaje, cola 8, timeout 10 s, ping 20 s, lectura 45 s y salida visible limitada a 100 líneas. No hay reejecución automática. La API no usa shell/eval/os/exec ni socket Docker para ejecutar código. Sesiones expiradas se revisan durante la conexión y revocaciones cierran sockets.

Antes de producción faltan: TLS/WSS, gestión real de cuentas y vínculos, backups y restauración probados, retención/borrado de sesiones e intentos, revisión legal de privacidad, observabilidad, límites distribuidos si se añaden réplicas, pruebas de carga y auditoría independiente. No habilitar H5P no confiable en el mismo origen.

Runner real: implementar la interfaz Runner contra un servicio aislado, sin secretos/backend/host/red interna, cuotas CPU/RAM/procesos/tiempo/salida, FS efímero restringido, red denegada por defecto, auditoría y cancelación. Un contenedor por sí solo no garantiza aislamiento. Definir evaluación de ejercicios en servidor antes de dar recompensas por código.

## Alternativa: scaffolder para otro proyecto vacío

No ejecutar sobre el frontend ya generado. Opciones verificadas con create-astro 5.2.4 --help:

```bash
npm create astro@latest otra-aula-vacia -- --template minimal --no-install --no-git --yes
```

Esto solo crea un proyecto nuevo; no reproduce AulaQuest. Para reproducir esta entrega usa sus lockfiles y los comandos anteriores.

Para registrar capturas y mediciones del entorno local ya iniciado, desde frontend: `E2E_STUDENT_PASSWORD='change-this-student-password' node scripts/inspect.mjs`. El script está ajustado a los IDs del seed nuevo (lecciones 1/2/3); los artefactos se guardan en docs. No ejecutarlo contra datos reales.

### Editar plugins en la demo

Inicia con `npm run demo` (Node 22.12 o superior), abre `http://localhost:4321/login` y entra como `profe` con contraseña `aulaquest-demo`. En **Mi grupo → Editar plugins** (`/teacher/plugins`) puedes cambiar nombres e instrucciones de Código y H5P, y elegir el lenguaje inicial del laboratorio. Los estudiantes ven los cambios al abrir una lección. Los ajustes se guardan junto al progreso en `.demo/state.json` y sobreviven al reinicio.

Esta pantalla corresponde al servidor de demostración: aún no está implementada en Go/PostgreSQL. No instala plugins ni paquetes H5P; H5P sigue pendiente de contenido y el laboratorio sigue siendo una ejecución simulada. Contrato: `contracts/demo-plugins.md`.

Las entradas de tarjetas y las celebraciones de progreso son breves. Con movimiento reducido se conserva la información sin partículas ni desplazamientos. El HUD celebra aumentos confirmados por la API, no la carga inicial.


### Administrador de demostración

Ejecuta `npm run demo` desde la raíz con Node 22.12 o superior. En `http://localhost:4321/login`, entra como **admin**, contraseña **aulaquest-demo**. Se abrirá `/admin`: resumen de cuentas y roles, catálogo de cursos y acceso a **Editar plugins**. Estudiantes y docentes no pueden consultar este panel.

La administración usa páginas Astro SSR y el servidor local de demostración. Los ajustes de plugins se guardan en `frontend/.demo/state.json`; las cuentas y cursos son sintéticos. Crear usuarios, cambiar roles, editar cursos y trasladar la administración a Go/PostgreSQL quedan pendientes. Contrato: `contracts/demo-admin.md`. Código y H5P conservan sus límites: ejecución simulada y actividad todavía sin paquete configurado.


### Autoría, entregas y calificación (backend Go)

El espacio **Mi grupo → Crear actividades y calificar** permite al docente gestionar los cursos que tiene asignados: crear lecturas y tareas, guardar borradores, publicarlos, revisar entregas y guardar/publicar una devolución con nota entera de 0 a 100. Las notas no conceden XP. Las lecturas conservan su recompensa única existente.

El enlace **Ver calificaciones** en cada curso abre el libro: estudiantes vinculados e inscritos, tareas publicadas, pendientes, notas, total por categorías y total del curso con política de faltantes explícita. Permite revisar una entrega concreta; las notas ocultas no cuentan, una actividad sin publicar no participa y el cálculo no es un boletín oficial. Hay paginación independiente de estudiantes y tareas.

Esta función usa Go/PostgreSQL; `npm run demo` conserva la demostración anterior y no incluye autoría ni calificaciones. Para usarla, inicia el modo A o Docker Compose descritos arriba y aplica las nuevas migraciones. En una base de desarrollo con las variables ya configuradas, desde `backend`:

```bash
go run ./cmd/migrate
go run ./cmd/seed
go run ./cmd/api
```

En otra terminal, desde la raíz:

```bash
npm --prefix frontend run dev
```

Abre `http://localhost:4321/login`. El seed asigna `profe` a los dos cursos sintéticos y conserva cuentas/datos existentes. La contraseña es la configurada al crear esa cuenta (`change-this-teacher-password` en los ejemplos sin modificar); Luna usa `change-this-student-password` en esos mismos ejemplos. Las asignaciones de cursos reales se administrarán en otro incremento, no se infieren del vínculo de estudiantes. Para consultar o calificar entregas también se exige el vínculo teacher_students: asignar un curso no concede acceso a estudiantes ajenos.

Flujo: docente crea actividad → guarda → publica. El estudiante inscrito la encuentra en el mapa, guarda su respuesta y la envía. El docente entra en esa actividad, guarda la nota/comentario y publica la devolución. El alumno la consulta al volver a abrir su entrega.

Los borradores del docente no son visibles al estudiante; los borradores de respuestas no son visibles al docente. Los cambios publicados conservan snapshots; una entrega definitiva mantiene las instrucciones con las que fue guardada. La entrega no se puede editar tras enviarse en este incremento. Los conflictos de revisión devuelven 409 para evitar sobrescrituras; el formulario conserva el texto y pide revisar la versión antes de reintentar. Texto plano escapado, no HTML activo. Límite REST de 128 KiB; el límite WebSocket sigue siendo 24 KiB.

Todavía faltan más formatos de archivo, categorías anidadas, exenciones individuales y otros métodos de agregación de notas. Foros, wikis, gestor de extensiones, respaldos y panel de operación están excluidos del nuevo alcance por decisión del usuario. No se afirma paridad del 98 % con Moodle.

Prueba E2E académica contra una base sintética separada y servidores Go/Astro activos:

```bash
cd frontend
E2E_ACADEMIC=1 E2E_STUDENT_PASSWORD='change-this-student-password' E2E_TEACHER_PASSWORD='change-this-teacher-password' npm run test:e2e -- academic.spec.ts
```

La prueba crea una actividad y entrega nuevas; no borra datos. Referencias: `openspec/changes/academic-administration/increment-1.md`, `contracts/openapi.json`, `backend/migrations/002_academic_authoring.sql`.

### Rúbricas y pesos de tareas (Go/PostgreSQL)

En el editor de una tarea, abre **Configurar rúbrica y peso**. Define criterios y niveles, guarda el borrador y vuelve a publicar la actividad. Las entregas conservan su rúbrica histórica. El docente selecciona niveles y el servidor calcula la nota; el alumno ve los resultados al publicarse la devolución.

El peso es relativo (2 cuenta el doble que 1). El libro solo incluye notas publicadas y muestra la cobertura de pesos; no convierte ausencias en cero. Aplica `go run ./cmd/migrate` desde backend para instalar `003_rubrics_weights.sql` antes de iniciar la API actualizada. Evidencia, comandos E2E y límites: [docs/rubric-validation.md](docs/rubric-validation.md).

### Fechas y prórrogas (Go/PostgreSQL)

En cada tarea puedes **Configurar fechas** de apertura, entrega y cierre; guarda y publica la actividad. Vencer la fecha permite entregas tardías hasta el cierre. El alumno conserva acceso a instrucciones aunque no pueda enviar. **Gestionar prórrogas** amplía los plazos de un estudiante vinculado e inscrito, con motivo y revisión; vaciar ambas fechas revoca la excepción. El historial de la entrega no cambia al ajustar fechas.

Los formularios usan **UTC explícito**, sin convertir silenciosamente la hora local. Aplica `go run ./cmd/migrate` desde backend para `004_assignment_schedule.sql`. Evidencia y límites: [docs/schedule-validation.md](docs/schedule-validation.md).

### Adjuntos privados de entregas (Go/PostgreSQL)

Desde una tarea, el estudiante abre **Gestionar adjuntos**, sube TXT/PNG/JPEG, vuelve y envía la tarea. Puede entregar texto, archivos o ambos. Hasta 5 archivos por entrega, 2 MiB por archivo, 4 millones de píxeles por imagen y 50 MiB por estudiante. Las imágenes se recodifican sin metadatos. El docente solo descarga archivos de entregas enviadas y autorizadas; los archivos enviados quedan congelados.

Aplica `go run ./cmd/migrate` desde backend para `005_submission_attachments.sql`. Los archivos se almacenan en PostgreSQL, sin directorio público. JSON conserva 128 KiB; solo la ruta multipart admite 2 MiB + 64 KiB. PDF/ofimática y archivos de retroalimentación siguen pendientes. Evidencia y límites: [docs/attachments-validation.md](docs/attachments-validation.md).

### Reentregas e historial

En **Configurar intentos**, el docente establece de 1 a 10 intentos totales, guarda y publica. Desde una entrega enviada puede **Autorizar otro intento** con motivo. El estudiante recibe un borrador vacío y consulta los anteriores mediante **Ver mis intentos anteriores**. Se conservan texto, archivos y devoluciones; reabrir no modifica las fechas ni concede recompensas. El libro cuenta solo el último intento enviado, manteniendo la nota anterior mientras el nuevo sea borrador. Aplica la migración 006 con el comando de migración habitual. Evidencia y límites: [docs/attempts-validation.md](docs/attempts-validation.md). Prueba de navegador: `npm run test:e2e -- attempts.spec.ts` con las variables E2E anteriores.

### Banco de preguntas privado del docente

Desde un curso abre **Banco de preguntas** para crear selección única, selección múltiple, verdadero/falso o respuesta corta. Cada edición conserva una versión; puedes archivarla/restaurarla y consultar su historial. **Probar evaluación** calcula la respuesta en Go contra esa versión, sin crear calificaciones ni recompensas. Para usar preguntas con alumnos, crea una actividad **Cuestionario** como se describe abajo. Ejecuta `go run ./cmd/migrate` desde backend para aplicar la migración 007 antes de usar la API actualizada. Evidencia y pendientes: [docs/question-bank-validation.md](docs/question-bank-validation.md). Prueba de navegador: `npm run test:e2e -- questions.spec.ts` con las variables E2E anteriores.

### Cuestionarios del alumno y libro

Aplica también la migración 008 con `go run ./cmd/migrate`. Docente crea una actividad **Cuestionario**, abre **Configurar cuestionario**, selecciona preguntas/versiones/pesos, máximo de intentos y políticas de nota/revisión; guarda y publica. El alumno comienza, guarda respuestas y envía lo guardado. La calificación se calcula en Go y el libro incluye un solo ítem por cuestionario: primera, última, mejor nota o promedio. Se conservan intentos y claves privadas, sin premios por enviar. Docente revisa únicamente intentos enviados de alumnos vinculados e inscritos. Fechas, temporizador y revisión tras el cierre se añadieron en el incremento 9 (siguiente sección). Evidencia: [docs/quiz-validation.md](docs/quiz-validation.md). Prueba: `npm run test:e2e -- quizzes.spec.ts` con las variables E2E anteriores.

### Fechas, temporizador y revisión del cuestionario (Go/PostgreSQL)

En una actividad **Cuestionario**, **Configurar fechas** define apertura, vencimiento y cierre en UTC —igual que en las tareas— y **Configurar cuestionario** fija el tiempo máximo por intento (1 a 720 minutos; vacío para no limitar) y cuándo mostrar las soluciones: nunca, al enviar cada intento o solo después del cierre. El cuestionario solo se puede comenzar dentro de su calendario; después del vencimiento un intento en curso todavía se resuelve y se envía hasta el cierre.

El temporizador lo controla el servidor: cada intento congela su límite y su vencimiento al comenzar, y la cuenta atrás del alumno es solo informativa. Al agotarse el tiempo, el servidor califica las respuestas ya guardadas, marca la lección como completada y no concede XP, estrellas ni gemas. **Gestionar tiempos** concede a un estudiante vinculado e inscrito minutos extra y un cierre ampliado, con motivo y versión: solo amplía límites publicados y se revoca vaciando las fechas y escribiendo 0 minutos. Con revisión tras el cierre, cada intento conserva el cierre que tenía al comenzar, así que posponerlo no revela intentos anteriores; sin cierre registrado esa política no revela nada. Aplica `go run ./cmd/migrate` desde backend para `009_quiz_timing.sql`. Evidencia, entorno verificado y límites: [docs/quiz-timing-validation.md](docs/quiz-timing-validation.md). Prueba: `npm run test:e2e -- quiz-timing.spec.ts` con las variables E2E anteriores (2 pruebas, alrededor de 1.2 minutos: una espera el minuto real del temporizador).

### Límites de solicitudes y proxy confiable (TSEC1)

El tráfico autenticado se limita **por usuario** (600 solicitudes por minuto por defecto) y el no autenticado **por dirección real del cliente** (180), de modo que un aula completa detrás de una sola salida a Internet no comparte una cuota colectiva. Ajusta `RATE_LIMIT_USER` y `RATE_LIMIT_IP` en el entorno; un valor inválido detiene el arranque en lugar de relajar el límite en silencio. Se conservan el límite estricto de inicio de sesión (10 por minuto), los adjuntos (20 por minuto por usuario) y los intentos de actividad (30 por minuto por usuario), y toda respuesta 429 anuncia `Retry-After`.

La API solo cree `X-Forwarded-For` cuando el par directo está en `TRUSTED_PROXIES` (direcciones o rangos CIDR separados por comas). En desarrollo con dos terminales déjalo **vacío**: la API se alcanza directamente y ninguna cabecera es de fiar. Bajo Compose el rango interno ya viene configurado y Nginx sobrescribe la cabecera; en producción fija el rango exacto del ingress. `/healthz` y `/readyz` no están limitados a propósito. Evidencia, hallazgos y límites: [docs/ratelimit-validation.md](docs/ratelimit-validation.md).

### Grupos de curso (Go/PostgreSQL)

Desde un curso, **Grupos del curso** permite al docente crear grupos, renombrarlos, eliminarlos y asignar estudiantes vinculados e inscritos. El nombre es único dentro del curso y un estudiante pertenece **como máximo a un grupo por curso**: intentar meterlo en un segundo devuelve 409 con un mensaje propio. Solo el docente con el curso asignado ve y gestiona sus grupos; el roster que aparece es el suyo, nunca estudiantes ajenos. Eliminar un grupo quita sus pertenencias y no toca entregas, notas ni progreso; si el grupo **ya tiene entregas del equipo**, el borrado se rechaza con 409 y hay que vaciarlo, para no perder trabajo calificado. Aplica `go run ./cmd/migrate` desde backend para `010_course_groups.sql`. Evidencia y límites: [docs/groups-validation.md](docs/groups-validation.md). Prueba: `npm run test:e2e -- groups.spec.ts` con las variables E2E anteriores.

### Entrega grupal compartida (Go/PostgreSQL)

En una tarea, **Trabajo en equipo** activa la entrega grupal. Cualquier miembro del grupo guarda, adjunta, envía y consulta el mismo trabajo; los intentos y las reaperturas cuentan por grupo y el libro muestra esa entrega a **todos** los miembros. Un alumno inscrito sin grupo recibe un aviso en la tarea y 409 con su propio mensaje, sin crear ninguna entrega a medias. Los adjuntos son del equipo y su cuota se descuenta a quien sube el archivo. Requiere `011_group_submissions.sql` (`go run ./cmd/migrate`). Evidencia, hallazgos y límites: [docs/group-submission-validation.md](docs/group-submission-validation.md). Prueba: `npm run test:e2e -- group-submission.spec.ts`.

### Notas individuales por miembro (Go/PostgreSQL)

En una entrega de equipo, cada miembro recibe **su propia nota y su propia devolución**: el docente califica uno por uno desde la bandeja de la actividad, con revisión optimista, rúbrica e idempotencia por miembro, y publicar la de uno **no** publica las demás. Solo puede calificar a estudiantes vinculados e inscritos; a un miembro que no pertenece a la entrega se le responde 404. El alumno ve únicamente **su** nota; el listado de compañeros solo lo recibe el docente. En el libro, cada miembro recibe la entrega del grupo con su nota, y quien no tiene nota queda pendiente sin **heredar** la de un compañero. Las rutas individuales siguen valiendo para tareas individuales y rechazan una entrega de equipo con 409. Requiere `012_member_grades.sql` (`go run ./cmd/migrate`). Evidencia y límites: [docs/member-grades-validation.md](docs/member-grades-validation.md).

En una tanda larga de pruebas de navegador, el límite de inicio de sesión (10 por minuto y por dirección) puede agotarse y las últimas suites fallan al entrar. No es un defecto: ejecuta `npm run test:e2e` por tandas o espera un minuto entre ellas. **No relajes los límites para pasar las pruebas.**

Si acabas de cambiar archivos `.astro` (o de ejecutar `astro check` o `astro build`) mientras el servidor de desarrollo estaba levantado, Vite reoptimiza dependencias y la primera página puede quedar **sin hidratar**: el formulario de acceso no responde y las suites fallan al iniciar sesión. Reinicia el servidor de desarrollo, espera a que responda `/login` y navega una vez antes de lanzar las pruebas.

### Formatos, archivos de instrucciones y retención (Go/PostgreSQL)

Los adjuntos admiten **texto, imagen y PDF**, siempre por análisis del contenido y no por la extensión: el PDF se comprueba de verdad (cabecera, tabla de referencias, cola, páginas y rechazo de cifrado y contenido activo declarado). Es una criba estructural, no un saneador, así que **todo se descarga, nunca se abre en línea**. El docente añade **archivos de instrucciones** a la actividad; publicar congela una copia por versión, igual que el enunciado, de modo que quien entregó contra unas instrucciones puede seguir consultándolas, y el alumno ve las de la versión con la que trabaja. La **retención** solo borra archivos de borradores abandonados: `go run ./cmd/retention` informa sin tocar nada y `-confirm` es lo que borra, devolviendo los bytes a quien los subió y conservando los textos. Requiere `013_instruction_attachments.sql` (`go run ./cmd/migrate`). Evidencia, hallazgos y límites —incluido que Nginx y Docker **no** pudieron ejecutarse aquí—: [docs/instruction-attachments-validation.md](docs/instruction-attachments-validation.md). Prueba: `npm run test:e2e -- attachments.spec.ts`.

### Adjuntos de devolución (Go/PostgreSQL)

Cada nota admite hasta cinco archivos de devolución (2 MiB cada uno), **por estudiante**: en una entrega de equipo cada miembro tiene los suyos. Los archivos cuelgan de la nota, así que adjuntar exige haberla guardado antes (si no, 409) y la propia base impide que existan huérfanos. El alumno descarga **solo los suyos y solo cuando la nota está publicada**; un compañero recibe 404 y una devolución en borrador no se filtra. La autorización es literalmente la misma que la de calificar, y la cuota se descuenta a quien sube y se le devuelve al borrar. Requiere `014_grade_attachments.sql` (`go run ./cmd/migrate`). Evidencia y límites: [docs/grade-attachments-validation.md](docs/grade-attachments-validation.md).

T4g queda cerrado con este incremento (formatos, instrucciones, retención y devoluciones).

### Exportación del libro y rendimiento (Go/PostgreSQL)

En el libro de calificaciones, **Descargar CSV** entrega el libro completo con los **mismos números** que la tabla: la exportación reutiliza la misma consulta y el mismo ensamblado, solo cambia la ventana. Solo viajan las notas **publicadas**; los pendientes aparecen como contadores. Un archivo que no quepa (más de 500 estudiantes o 100 actividades) se **rechaza con 409** en lugar de truncarse, y un alias que parezca una fórmula se neutraliza. Rendimiento medido con 40 estudiantes, 12 actividades y 480 entregas: el libro pasó de **549 ms a 46 ms** al dejar de reevaluar la vista del libro 9.600 veces, y una prueba fija en **6 sentencias** el número de consultas para impedir N+1. Método, entorno y límites: [docs/gradebook-export-validation.md](docs/gradebook-export-validation.md).

La **accesibilidad (AC1)** de T6 quedó auditada con criterios comprobables (etiquetas, encabezados, foco visible, objetivos táctiles, contraste y movimiento reducido): dos correcciones de tamaño táctil y cero hallazgos en diez páginas ([docs/accessibility-validation.md](docs/accessibility-validation.md)). Queda PL1 (administración real con capacidades por contexto) en el catálogo pendiente.

### Categorías de calificación y total del curso (Go/PostgreSQL)

El libro incorpora **categorías planas con peso** (nombre único por curso, 1..1000, máximo cien), la **política de faltantes** `exclude` o `zero` y el **total del curso** como media ponderada de los totales de categoría. Cada tarea o cuestionario publicado pertenece a una categoría; publicar sin elegir una usa la primera del curso y crea «General» si no existe. Una nota sin publicar nunca participa ni viaja al alumno, y el alumno consulta «Mis notas» en su curso con solo las notas publicadas. La interfaz permite crear, renombrar, ponderar, reordenar y eliminar categorías (409 si tienen actividades) y elegir la categoría al configurar la tarea o el cuestionario. Requiere `015_grade_categories.sql` (`go run ./cmd/migrate`). Contrato OpenAPI 1.10.0. Evidencia y límites: [docs/grade-categories-validation.md](docs/grade-categories-validation.md).
