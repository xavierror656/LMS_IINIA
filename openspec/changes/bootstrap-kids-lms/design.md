# Design
## Context
Repositorio vacío; Node 20 disponible, Go/Docker ausentes inicialmente. OpenSpec instalado en /tmp, sin integración que modifique Agents.md.
## Decisions
Astro SSR con adaptador oficial Node y ClientRouter; páginas públicas prerenderizadas. React solo HUD, editor y H5P. Tailwind v4 Vite CSS-first, Lexend local. API_INTERNAL_URL solo servidor; navegador mismo origen mediante proxy. Cookies reenviadas selectivamente; no-store en páginas/API privadas. Store sin estado personal SSR.
Go Fiber v2 + contrib/websocket compatible, GORM PostgreSQL. Capas handlers/services/repositories, interfaz Runner. SQL explícito; bloqueo de perfil y UNIQUE(user_id,lesson_id) hacen finalización concurrente idempotente. No AutoMigrate.
## Data model
users -> sessions, gamification_profiles; teacher_students relaciona dos users; courses -> modules -> lessons; enrollments(user,course); lesson_progress(user,lesson,status); reward_events UNIQUE(user,lesson); activity_attempts con trust=client_reported. FK, checks, índices y UTC en migración 001.
## Security and privacy
Sesiones aleatorias de 256 bits, hash SHA256 almacenado, bcrypt y cookies HttpOnly/Lax/Secure producción; Origin exacto protege mutaciones y WS. Sin CORS abierto. JSON estricto y cuerpos limitados. Consultas autorizadas por inscripción/vínculo. No logs de código, contraseñas o cookies. Solo alias sintéticos. Retención de sesiones expiradas requiere tarea operativa antes de producción; definir conservación y borrado de datos con responsable escolar.
H5P ejecuta contenido activo: únicamente revisado; aislar contenido no confiable en otro origen con política de comunicación restringida antes de admitirlo. xAPI cliente nunca es validación de evaluación.
Runner real: servicio separado con aislamiento para código hostil, sin host/red interna/secretos; cuotas CPU, RAM, procesos, tiempo y salida; FS restringido y efímero, red denegada por defecto, auditoría y cancelación. Un contenedor solo no garantiza aislamiento.
## Risks
Sin Docker local al inspeccionar: intentar pruebas PostgreSQL independientes; registrar pendientes reales. Sin paquete H5P licenciado disponible: integración explícitamente no verificada E2E. Nunca presentar mock como ejecución.
## Contracts
Ver contracts/openapi.json y contracts/websocket.md. Versionado REST /api/v1 y WS v=1.
