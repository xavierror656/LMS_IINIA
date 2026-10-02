Actúa como Arquitecto de Software Full Stack, Ingeniero de Plataforma y Diseñador UI/UX especializado en EdTech infantil.

Debes diseñar e implementar desde cero un LMS moderno, rápido y lúdico para niños, alternativo a Moodle, siguiendo un flujo SpecOps: especificaciones verificables antes del código, implementación incremental y validación mediante pruebas.

Trabaja directamente sobre los archivos del repositorio. No te limites a responder con recomendaciones o bloques de código.

======================================================================
1. OBJETIVO DEL PROYECTO
======================================================================

Construir un MVP ejecutable de un LMS infantil con:

- Catálogo de cursos.
- Mapa de aprendizaje por curso.
- Lecciones de lectura, programación y actividades H5P.
- Progreso persistente por estudiante.
- HUD con avatar, experiencia, nivel, estrellas, gemas y vidas.
- Editor de código con consola conectada al backend mediante WebSocket.
- Vista mínima del docente para consultar el progreso de sus estudiantes.
- Interfaz accesible y cómoda en tablets.
- Arquitectura preparada para incorporar un runner aislado de código.

Nombre provisional del proyecto: “AulaQuest”.

El nombre, logotipo y textos de marca deben poder cambiarse desde una
configuración central. Usa español para la interfaz inicial e inglés para
identificadores, rutas de código, tablas y campos de la API.

El resultado debe ser una base real y mantenible, no un clon visual de
Moodle ni una plataforma empresarial sobredimensionada.

======================================================================
2. FORMA DE TRABAJO: SPECOPS
======================================================================

Primero inspecciona el repositorio:

- Lee AGENTS.md y las instrucciones existentes.
- Identifica archivos, dependencias y herramientas ya configuradas.
- Conserva cambios ajenos y evita sobrescribir trabajo existente.
- Determina si existe OpenSpec/OPSX, Spec Kit u otro flujo de
  especificaciones establecido.

Elección del flujo:

1. Si existe OpenSpec/OPSX, utiliza sus convenciones y herramientas.
2. Si existe Spec Kit, respeta ese flujo en lugar de instalar otro.
3. Si el repositorio es nuevo, utiliza OpenSpec como opción predeterminada.
4. Si no puedes instalar o ejecutar su herramienta, crea los documentos
   equivalentes y registra que la validación mediante CLI está pendiente.

No inventes comandos de OpenSpec, OPSX, Spec Kit o Codex.
Consulta las instrucciones disponibles y la ayuda de la versión instalada.
No mezcles dos frameworks de especificación en el mismo proyecto.

Crea un cambio o iniciativa llamado:

bootstrap-kids-lms

Antes de implementar, produce los siguientes artefactos conceptuales,
adaptados al formato del framework elegido:

- Propuesta y alcance.
- Requisitos funcionales y no funcionales.
- Escenarios de aceptación.
- Diseño técnico.
- Contrato REST.
- Contrato WebSocket.
- Modelo de datos.
- Análisis de seguridad y privacidad.
- Plan de pruebas.
- Tareas de implementación ordenadas por dependencias.

Asigna identificadores estables a los requisitos:

LMS-001, LMS-002, LMS-003...

Mantén trazabilidad entre requisito, tarea y prueba.

Cada requisito debe expresar:

- Comportamiento esperado.
- Criterio de aceptación observable.
- Casos de error.
- Restricciones relevantes.
- Qué queda explícitamente fuera de alcance.

Usa escenarios Given/When/Then cuando aporten claridad.

Después de crear las especificaciones, continúa con la implementación.
No te detengas únicamente porque terminaste la documentación.

Respeta las aprobaciones que exija el entorno. No publiques, despliegues
en servicios externos, hagas push ni ejecutes operaciones destructivas
sin autorización.

======================================================================
3. DECISIONES ARQUITECTÓNICAS OBLIGATORIAS
======================================================================

Frontend:

- Astro.
- React únicamente para islas interactivas complejas.
- Tailwind CSS v4.
- Lucide React.
- Framer Motion dentro de las islas de React.
- Nano Stores y @nanostores/react.
- CodeMirror 6.
- h5p-standalone.
- TypeScript estricto.
- Lexend o Fredoka como tipografía.

Backend:

- Go, con una versión mantenida compatible con las dependencias elegidas.
  El requisito mínimo original es Go 1.22.
- Go Fiber.
- Integración WebSocket compatible con la versión elegida de Fiber.
- PostgreSQL.
- GORM.
- Migraciones SQL explícitas y versionadas.

Arquitectura:

- Monorepositorio con /frontend y /backend.
- Backend como monolito modular.
- No crear microservicios innecesarios.
- Separar el runner de código mediante una interfaz.
- No ejecutar código arbitrario dentro del proceso API.
- El backend es la fuente de verdad del progreso y las recompensas.
- Nano Stores conserva una representación del estado para la interfaz,
  no la autoridad sobre puntuaciones o permisos.

Versiones:

Antes de instalar, verifica las versiones estables y la compatibilidad
entre Astro, React, Tailwind, Fiber, WebSocket y H5P usando documentación
disponible y metadatos de paquetes.

No asumas que APIs de tutoriales antiguos siguen vigentes.

Registra las versiones realmente utilizadas.
Genera y conserva package-lock.json y go.sum.
No dejes dependencias de producción con versiones “latest” sin resolver.

Si no tienes acceso a la red, identifica claramente qué no pudiste
verificar y no afirmes haber instalado o probado dependencias.

======================================================================
4. ALCANCE FUNCIONAL DEL MVP
======================================================================

Implementa:

A. Acceso

- Inicio y cierre de sesión.
- Sesiones seguras mediante cookies HttpOnly.
- Roles student y teacher.
- Cuentas sintéticas de desarrollo.
- Sin registro público de menores en esta primera versión.

B. Cursos

- Catálogo con tarjetas.
- Detalle del curso.
- Módulos ordenados.
- Lecciones con estado disponible, en progreso y completado.
- Inscripción del estudiante a cursos mediante datos de demostración.

C. Lecciones

Tipos iniciales:

- reading
- code
- h5p

Cada lección debe tener título, descripción, posición, tipo y
configuración específica validada.

D. Progreso y gamificación

- Experiencia.
- Nivel.
- Estrellas.
- Gemas.
- Vidas.
- Lecciones completadas.

Las vidas no deben bloquear el acceso al contenido educativo.
No implementar mecánicas de pago, publicidad, rankings públicos de niños
ni penalizaciones que impidan seguir aprendiendo.

E. Docente

- Lista de estudiantes vinculados al docente.
- Resumen de progreso por curso.
- Vista de detalle por estudiante.

Una tabla clara es suficiente para el MVP.
No construir todavía un sistema completo de administración escolar.

Fuera de alcance:

- Pagos y suscripciones.
- Marketplace.
- Chat entre menores.
- Videoconferencias.
- SCORM.
- Editor o constructor de contenido H5P.
- Importación de respaldos Moodle.
- Aprovisionamiento real de máquinas virtuales.
- Ejecución de código arbitrario en producción.
- Analítica empresarial avanzada.

======================================================================
5. FRONTEND ASTRO: PÁGINAS E ISLAS
======================================================================

Implementa páginas Astro para:

- Inicio.
- Inicio de sesión.
- Catálogo.
- Detalle de curso.
- Lección.
- Panel del docente.

No conviertas toda la aplicación en una SPA de React.

Utiliza Astro para estructura, contenido y componentes simples.
Usa React para:

- HUD.
- CodePlayground.
- H5PPlayer.
- Otras interacciones solamente cuando exista una justificación concreta.

Configura navegación con las transiciones y el router de cliente
correspondientes a la versión instalada de Astro.

Utiliza ClientRouter cuando sea la API vigente.
No copies una integración obsoleta de ViewTransitions sin verificarla.

Elige y documenta el modo de renderizado:

- Preferentemente SSR para páginas privadas.
- Prerenderizado para páginas públicas cuando resulte apropiado.
- Adaptador oficial compatible con el despliegue seleccionado.

No prerenderices información personalizada de estudiantes.
No permitas caché compartida de respuestas privadas.

Si utilizas SSR:

- Distingue API_INTERNAL_URL del origen visible por el navegador.
- Reenvía únicamente las credenciales necesarias al backend.
- No expongas secretos mediante variables PUBLIC_*.
- No escribas estado de usuarios en singletons globales del servidor.
- No compartas Nano Stores con datos personales entre solicitudes SSR.

======================================================================
6. DISEÑO VISUAL Y ACCESIBILIDAD
======================================================================

Diseña una interfaz infantil cuidada, no una página empresarial con
colores saturados añadidos.

Sistema visual:

- Morado eléctrico como color principal.
- Verde lima para avances y acciones positivas.
- Amarillo sol para recompensas.
- Azul cielo para información.
- Fondos claros y superficies limpias.
- Bordes redondeados grandes.
- Tarjetas con profundidad suave.
- Jerarquía tipográfica clara.
- Iconos comprensibles acompañados de texto cuando sea necesario.

Define colores como tokens de diseño, no como valores repetidos en
componentes.

Usa Tailwind CSS v4 con configuración CSS-first:

- @import "tailwindcss".
- @theme para colores, fuentes y tokens.
- Plugin de Vite compatible con Astro.

No crees tailwind.config.mjs por costumbre si no es necesario.
No utilices una integración antigua de Tailwind incompatible con v4.

Botones:

- Grandes y cómodos para interacción táctil.
- Efecto de bloque presionable.
- Sombra o borde inferior visible.
- Estado activo con desplazamiento corto.
- Sin cambios de altura que produzcan saltos de layout.

Accesibilidad:

- Objetivos táctiles de al menos 44 × 44 CSS px.
- Foco visible.
- Navegación completa por teclado.
- Etiquetas accesibles.
- Información que no dependa solo del color.
- Contraste comprobado.
- Compatibilidad con prefers-reduced-motion.
- Sin animaciones constantes que distraigan de la actividad.
- Estados de carga, vacío, éxito y error con texto comprensible.

Tipografía:

Utiliza Lexend o Fredoka, provenientes de Google Fonts.
Prefiere servir la fuente localmente cuando sea viable y compatible con
su licencia, para reducir solicitudes de terceros.

No añadas trackers, publicidad ni scripts analíticos externos.

======================================================================
7. LAYOUT Y HUD PERSISTENTE
======================================================================

Crea:

frontend/src/layouts/Layout.astro
frontend/src/components/react/HUD.tsx
frontend/src/store/progress.ts

Layout.astro debe incorporar:

- Metadatos básicos.
- CSS global.
- Tipografía.
- Navegación.
- Router de cliente y transiciones.
- HUD persistente.
- Región principal accesible.

Renderiza el HUD como isla React con client:load.

Configura su persistencia entre navegaciones mediante el mecanismo
oficial de Astro, con identidad estable.

El HUD debe mostrar:

- Avatar.
- Nombre corto o alias.
- Nivel.
- Experiencia.
- Estrellas.
- Gemas.
- Vidas.

La store debe contemplar:

- Estado de hidratación.
- Estado de carga.
- Progreso actual.
- Errores.
- Actualización a partir de respuestas autorizadas del backend.
- Limpieza al cerrar sesión o cambiar de usuario.

No agregues funciones públicas que permitan otorgar recompensas
persistentes arbitrariamente desde el navegador.

Una animación optimista puede existir, pero debe reconciliarse con la
respuesta del servidor.

Al recargar la página, recupera el progreso del backend.
No uses localStorage como fuente de verdad de puntuaciones o sesiones.

======================================================================
8. CODEPLAYGROUND Y WEBSOCKET
======================================================================

Crea:

frontend/src/components/react/CodePlayground.tsx

Requisitos:

- Isla React con client:load.
- CodeMirror 6.
- Soporte inicial de edición para JavaScript y Python.
- Botón Ejecutar.
- Botón Detener.
- Botón Limpiar consola.
- Indicador de conexión.
- Consola con stdout, stderr, estado final y errores.
- Interfaz usable con teclado y tablet.

Implementa un cliente WebSocket tipado.

El MVP utilizará un MockRunner en Go que emita eventos simulados.
Debe existir una conexión WebSocket real entre frontend y backend.

Importante:

- El mock no ejecuta el código escrito.
- La interfaz debe mostrar “Ejecución simulada”.
- No fabriques resultados presentándolos como ejecución real.
- No utilices eval, os/exec, shell, docker exec ni equivalentes.
- El backend API no debe montar el socket de Docker.

Define un contrato WebSocket versionado y documentado.

Eventos sugeridos del cliente:

- run.start
- run.cancel

Eventos sugeridos del servidor:

- run.accepted
- run.stdout
- run.stderr
- run.finished
- run.failed

Define para cada evento:

- Tipo.
- requestId.
- runId cuando corresponda.
- Secuencia cuando corresponda.
- Payload validado.
- Semántica de errores.

El servidor genera runId.
No confíes en identificadores de usuario enviados por el cliente.

Implementa:

- Autenticación al conectar.
- Validación explícita del encabezado Origin.
- Autorización por sesión y ejecución.
- Límite de tamaño de mensajes y código.
- Límite de ejecuciones concurrentes por usuario.
- Timeout.
- Cancelación.
- Heartbeats.
- Buffers acotados.
- Cierre de conexiones al apagar el servidor.
- Limpieza de listeners al desmontar componentes.

Deriva ws:// o wss:// del entorno.
No envíes credenciales en query strings.

No reenvíes automáticamente una ejecución al reconectar:
podrías duplicar el trabajo. Explica el estado al usuario.

Crea una interfaz Runner que permita sustituir el mock por un servicio
aislado en el futuro.

Documenta requisitos del runner real:

- Aislamiento adecuado para código no confiable.
- Sin acceso al host.
- Sin acceso a la red interna.
- Sin secretos del backend.
- Cuotas de CPU, memoria, procesos, tiempo y salida.
- Entornos efímeros.
- Filesystem restringido.
- Política explícita de red.
- Auditoría y cancelación.

No declares que un contenedor por sí solo resuelve todos los riesgos.

======================================================================
9. H5PPLAYER
======================================================================

Crea:

frontend/src/components/react/H5PPlayer.tsx

Utiliza h5p-standalone dentro de una isla:

client:only="react"

Verifica la API de la versión instalada antes de implementar.

Requisitos:

- Carga únicamente en el navegador.
- No acceder a window o document durante SSR.
- No incluir H5P en páginas que no lo utilizan.
- Estados de carga y error.
- Dimensiones adaptables.
- Limpieza compatible con la API real de la librería.
- Rutas de contenido controladas por configuración, no URLs arbitrarias
  aportadas por el estudiante.

Distingue entre:

1. Archivo .h5p.
2. Contenido extraído.
3. Librerías necesarias.
4. Recursos JavaScript y CSS del reproductor.
5. Datos de actividad.

No asumas que proporcionar cualquier ZIP o un h5p.json aislado basta
para reproducir una actividad.

Incluye una actividad H5P real de ejemplo solo si puedes obtener todos
sus recursos legalmente, conservar la atribución y verificar su carga.

Si no hay contenido válido disponible:

- Implementa la integración.
- Incluye un manifiesto de configuración de ejemplo.
- Documenta dónde colocar los recursos.
- Presenta un estado “Actividad aún no configurada”.
- No sustituyas H5P por HTML y lo presentes como integración terminada.

H5P ejecuta contenido activo:

- En el MVP solo se admiten paquetes revisados y confiables.
- No habilites subida libre de paquetes.
- Documenta aislamiento en otro origen para contenido no confiable.

Integra la captura de eventos xAPI relevantes cuando la API lo permita.

Los eventos del navegador son evidencia no confiable:

- No concedas recompensas únicamente porque llegue un score del cliente.
- Guarda resultados reportados con su nivel de confianza.
- Diferencia actividad reportada de evaluación validada por el servidor.
- Evita almacenar datos personales innecesarios en los eventos.

======================================================================
10. BACKEND GO
======================================================================

Estructura mínima:

backend/
  cmd/
    api/
      main.go
    seed/
      main.go
  internal/
    config/
    database/
    models/
    middleware/
    handlers/
    services/
    repositories/
    runner/
    ws/
  migrations/
  go.mod
  go.sum
  .env.example

Responsabilidades:

- handlers: HTTP, validación de entrada y respuestas.
- services: reglas de negocio y transacciones.
- repositories: acceso a PostgreSQL.
- ws: protocolo, conexiones y transporte.
- runner: ejecución simulada e interfaz del runner.
- middleware: sesiones, autorización, límites y trazabilidad.

Mantén las capas proporcionadas al MVP.
No introduzcas abstracciones genéricas sin un uso concreto.

main.go debe configurar:

- Variables de entorno validadas.
- Conexión y pool de PostgreSQL.
- Servicios.
- Middleware.
- Rutas REST.
- WebSocket.
- Healthcheck.
- Readiness.
- Apagado ordenado.

Implementa logs estructurados y request IDs.
No registres contraseñas, cookies, código completo del estudiante ni
datos personales innecesarios.

Utiliza migraciones versionadas.
No dependas de AutoMigrate para cambios de esquema en producción.

El seed debe ser explícito, repetible e idempotente.
Nunca borres datos para volver a cargarlo.

======================================================================
11. DATOS Y REGLAS DE GAMIFICACIÓN
======================================================================

Modelo inicial orientativo:

- users
- sessions
- teacher_students
- courses
- modules
- lessons
- enrollments
- lesson_progress
- activity_attempts
- gamification_profiles
- reward_events

Adapta el modelo a las especificaciones, sin duplicar información sin
necesidad.

Incluye:

- Claves foráneas.
- Índices de consulta.
- Restricciones únicas.
- Fechas consistentes.
- Validación de relaciones entre curso, módulo, lección e inscripción.

La API no debe aceptar del estudiante campos como:

- user_id para decidir sobre quién operar.
- gems_to_add.
- xp_to_add.
- new_level.
- is_teacher.

Obtén la identidad de la sesión y calcula las recompensas en servicios.

La finalización de una lección debe ser idempotente:

- Dos solicitudes iguales no duplican premios.
- Dos solicitudes concurrentes tampoco.
- Una nueva Idempotency-Key no permite volver a cobrar el mismo premio.

Usa transacciones y restricciones de base de datos, no solo comprobaciones
en memoria.

Para lecciones de lectura, puedes aceptar una finalización declarada,
dejando esa semántica documentada.

Para evaluaciones, distingue los resultados validados por el backend de
los resultados simplemente reportados por el navegador.

Devuelve el progreso canónico actualizado después de cada operación.

======================================================================
12. CONTRATO REST Y SESIONES
======================================================================

Define y mantiene un contrato OpenAPI coherente con la implementación.

Rutas iniciales orientativas:

GET  /healthz
GET  /readyz

POST /api/v1/auth/login
POST /api/v1/auth/logout
GET  /api/v1/auth/session

GET  /api/v1/courses
GET  /api/v1/courses/:courseId
GET  /api/v1/lessons/:lessonId

GET  /api/v1/me/progress
POST /api/v1/lessons/:lessonId/complete
POST /api/v1/lessons/:lessonId/attempts

GET  /api/v1/teacher/students
GET  /api/v1/teacher/students/:studentId/progress

WS   /ws/code

Puedes ajustar las rutas durante el diseño, pero actualiza contratos,
frontend y pruebas conjuntamente.

Implementa:

- Validación de payloads.
- Errores estructurados.
- Códigos HTTP apropiados.
- Límites de cuerpo.
- Paginación donde corresponda.
- Autorización por objeto, no solo por rol.

Sesiones:

- Cookies HttpOnly.
- Secure en producción.
- Política SameSite documentada.
- Expiración.
- Revocación.
- Protección CSRF en operaciones con estado.
- Contraseñas almacenadas con un algoritmo adecuado, nunca en texto plano.

No guardes tokens de sesión en localStorage.

Las cuentas de desarrollo deben usar datos sintéticos y credenciales
configurables. No establezcas credenciales predeterminadas de producción.

======================================================================
13. ESTRUCTURA FRONTEND ESPERADA
======================================================================

frontend/
  astro.config.mjs
  package.json
  package-lock.json
  tsconfig.json
  .env.example
  public/
    h5p/
      README.md
  src/
    components/
      astro/
      react/
        HUD.tsx
        CodePlayground.tsx
        H5PPlayer.tsx
    layouts/
      Layout.astro
    pages/
      index.astro
      login.astro
      courses/
      lessons/
      teacher/
    lib/
      api.ts
      websocket.ts
    store/
      progress.ts
    styles/
      global.css

Evita fetch dispersos con contratos distintos.
Centraliza errores, credenciales y tipado de la API.

Comparte o genera tipos a partir del contrato cuando sea viable.
No mantengas interfaces incompatibles entre backend y frontend.

======================================================================
14. DESARROLLO LOCAL Y DESPLIEGUE BASE
======================================================================

Entrega:

- Dockerfile del backend.
- Dockerfile del frontend.
- docker-compose.yml.
- PostgreSQL con volumen y healthcheck.
- Variables de entorno de ejemplo.
- Configuración de proxy inverso.
- README con pasos verificables.
- Comandos de migración y seed.

Configuración local:

- Frontend en http://localhost:4321.
- Backend en http://localhost:8080.
- Proxy de /api hacia Go.
- Proxy de /ws hacia Go con soporte WebSocket.

El navegador debe poder utilizar un único origen.
En producción configura HTTPS y WSS mediante el proxy inverso.

No expongas PostgreSQL públicamente.
Si publicas su puerto para desarrollo, limita el enlace a localhost.

Documenta dos modos:

A. Desarrollo con dos terminales:

- Instalar dependencias del frontend.
- Iniciar PostgreSQL.
- Ejecutar migraciones.
- Ejecutar seed.
- Iniciar Go.
- Iniciar Astro.

B. Arranque mediante Docker Compose.

Incluye npm create astro@latest como procedimiento alternativo para
crear un proyecto vacío, con opciones verificadas.

No indiques ejecutar el scaffolder encima del frontend ya generado.

No asegures que un despliegue está listo para producción si siguen
pendientes autenticación, H5P real, aislamiento del runner o pruebas.

======================================================================
15. PRUEBAS Y CRITERIOS DE ACEPTACIÓN
======================================================================

Frontend:

- Comprobación de Astro.
- TypeScript estricto.
- Pruebas unitarias de stores y componentes relevantes.
- Pruebas de integración de API.
- Pruebas E2E con Playwright o equivalente justificado.

Backend:

- go test ./...
- go vet ./...
- Pruebas de servicios.
- Pruebas de handlers.
- Pruebas de autorización.
- Pruebas de idempotencia y concurrencia.
- Integración con PostgreSQL de prueba separado.
- Pruebas del protocolo WebSocket.

Escenarios obligatorios:

1. Un estudiante inicia sesión y visualiza sus cursos.

2. Completa una lección de lectura y recibe la recompensa calculada por
   el servidor.

3. Repetir la solicitud no duplica experiencia, estrellas ni gemas.

4. Dos finalizaciones concurrentes producen una sola recompensa.

5. El HUD mantiene continuidad visual entre páginas y se reconcilia con
   el progreso del backend.

6. Una recarga completa conserva el progreso persistido.

7. Cerrar sesión elimina el estado personal de Nano Stores.

8. Un estudiante no consulta ni modifica el progreso de otro.

9. Un docente solo consulta estudiantes vinculados a su cuenta.

10. La consola recibe eventos reales del WebSocket y muestra claramente
    que la ejecución está simulada.

11. Cancelar o desconectar una ejecución libera recursos.

12. Un evento H5P manipulado no concede puntuación validada.

13. Un paquete H5P ausente produce un estado controlado, no una pantalla
    rota.

14. Una API caída muestra error y recuperación; no presenta progreso
    inventado como guardado.

15. La interfaz funciona con teclado, viewport de tablet y movimiento
    reducido.

No marques una prueba como exitosa si no fue ejecutada.
No ejecutes pruebas destructivas contra una base existente.

======================================================================
16. RENDIMIENTO
======================================================================

Evita usar “ultrarrápido” como afirmación sin medición.

Implementa medidas concretas:

- Contenido simple sin hidratación React innecesaria.
- CodeMirror solo en lecciones de programación.
- H5P solo en lecciones H5P.
- División de bundles.
- Recursos estáticos con caché apropiada.
- Consultas indexadas.
- Evitar N+1 en el backend.
- Pool de conexiones acotado.
- Payloads pequeños.
- Consola con salida y memoria limitadas.

Mide lo que permita el entorno:

- Tamaño de bundles.
- JavaScript inicial por tipo de página.
- Tiempo de respuesta de endpoints principales.
- Comportamiento bajo concurrencia básica.
- Posibles fugas tras navegar o abrir y cerrar WebSockets.

Documenta método, entorno y resultados.
No inventes métricas ni garantías de latencia.

======================================================================
17. ORDEN DE IMPLEMENTACIÓN
======================================================================

Trabaja por incrementos verificables:

Fase 0:
Inspección, decisiones y especificaciones.

Fase 1:
Monorepositorio, dependencias, configuración, PostgreSQL, migraciones,
healthchecks y estructura de aplicación.

Fase 2:
Sesiones, autorización, cursos, lecciones y datos sintéticos.

Fase 3:
Diseño visual, páginas Astro, navegación y HUD con Nano Stores.

Fase 4:
Progreso persistente, gamificación y pruebas de idempotencia.

Fase 5:
CodeMirror, protocolo WebSocket y MockRunner.

Fase 6:
Integración H5P y registro de intentos con confianza explícita.

Fase 7:
Panel mínimo del docente.

Fase 8:
Pruebas, revisión de seguridad, rendimiento, Docker y documentación.

Dentro de cada fase:

- Actualiza las tareas.
- Implementa una porción funcional.
- Ejecuta las validaciones disponibles.
- Corrige errores antes de continuar.
- Registra bloqueos reales y decisiones.

No generes primero decenas de archivos vacíos para aparentar avance.

======================================================================
18. DEFINICIÓN DE TERMINADO
======================================================================

El trabajo se considera terminado cuando:

- Existe código real en /frontend y /backend.
- La configuración es coherente con las versiones instaladas.
- El frontend compila.
- El backend compila.
- Las migraciones y el seed funcionan.
- El flujo de estudiante funciona de extremo a extremo.
- El progreso persiste en PostgreSQL.
- Las recompensas son calculadas e idempotentes en el servidor.
- El HUD refleja el progreso correcto.
- CodeMirror se conecta al WebSocket.
- La ejecución simulada está identificada como tal.
- H5P funciona con contenido válido o su falta queda explícitamente
  documentada como integración no verificada de extremo a extremo.
- Las rutas privadas aplican autorización.
- Las pruebas críticas se ejecutaron o figuran como pendientes con causa.
- El README permite reproducir el entorno.
- Las especificaciones reflejan el código entregado.

No uses “listo para producción” como sinónimo de “compila”.

======================================================================
19. RESPUESTA FINAL QUE DEBES ENTREGAR
======================================================================

Al finalizar, resume:

1. Qué implementaste.
2. Qué decisiones arquitectónicas tomaste.
3. Qué archivos y carpetas principales creaste.
4. Cómo iniciar el proyecto con comandos exactos.
5. Cómo acceder con las cuentas sintéticas.
6. Qué pruebas ejecutaste y sus resultados reales.
7. Qué está simulado.
8. Qué no pudo verificarse.
9. Qué falta para incorporar un runner real y habilitar producción.

No declares éxito de herramientas que no ejecutaste.
No ocultes TODOs críticos detrás de interfaces bonitas.

Empieza inspeccionando el repositorio, establece el flujo SpecOps y
continúa hasta dejar una base funcional verificable dentro de las
capacidades y permisos disponibles.
