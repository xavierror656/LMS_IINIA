# Validación del incremento académico 1

Fecha: 2026-10-02. Linux/WSL x86_64, repositorio en montaje Windows/OneDrive. Node 22.23.3, Go 1.27.1, PostgreSQL 18.4, OpenSpec 1.14.0 y Playwright/Chromium disponibles en el entorno. PostgreSQL de prueba se inició en 127.0.0.1:55439 con archivos nuevos en /tmp/aulaquest-academic-db; no se usó una base del usuario. academic_test crea un esquema nuevo por prueba; academic_preview conserva datos sintéticos del navegador. No se hizo push ni despliegue externo.

## Entregado y comprobado

- Migración 002 aditiva: asignación de cursos, borradores, publicaciones inmutables, entregas, notas e historial.
- Cursos asignados, creación y edición de lecturas/tareas, publicación explícita e idempotente.
- Entrega textual privada mientras es borrador; envío definitivo conserva snapshot de instrucciones.
- Calificación entera 0–100, comentarios, guardado oculto y publicación explícita. Historial SQL con actor, revisión y estado.
- Control de sesión, Origin, rol y curso; 409 en revisiones obsoletas; notas separadas de XP.
- Páginas Astro con formularios nativos y controlador del navegador; sin nuevas islas React ni dependencias de producción.
- Contrato OpenAPI 1.1.0 y tipos generados coherentes con el flujo.

## Resultados reales

| Comprobación | Resultado |
|---|---|
| OpenSpec validate academic-administration --strict --no-interactive | Válido |
| npm --prefix frontend run types:api | Tipos generados |
| npm --prefix frontend run check, con telemetría desactivada | 35 archivos, 0 errores, 0 warnings y 0 hints en la última ejecución |
| npm --prefix frontend run build | SSR compilado; permanece aviso de bundle mayor de 500 kB |
| npm --prefix frontend test -- --pool=threads | 4 pruebas aprobadas |
| go test -count=1 ./... con TEST_DATABASE_URL | Aprobado incluyendo ambas integraciones reales PostgreSQL y WebSocket |
| go vet ./... desde backend | Sin diagnósticos |
| academic.spec.ts contra Go/Astro/PostgreSQL | 1 prueba integral aprobada; última ejecución de 5.9 s, total 7.4 s, incluyendo entrega enviada por Sol y rechazo de su consulta/calificación por un docente sin vínculo |
| academic.spec.ts + learning.spec.ts | 6 aprobadas tras reiniciar Astro, 39.8 s en esa ejecución. La primera pasada dio 5 aprobadas y 1 fallo por referencias Vite obsoletas de CodeMirror. |

Go crea esquema aislado, ejecuta migraciones y seed dos veces y comprueba roles, módulo ajeno, borrador invisible, publicación concurrente sin duplicados, conflicto entre guardados, snapshots, aislamiento de respuestas y de estudiantes no vinculados aunque compartan curso, dos envíos simultáneos, nota oculta/publicada, conflicto al recalificar, cuatro revisiones de auditoría y ausencia de recompensa por notas. No hay pruebas omitidas por falta de TEST_DATABASE_URL en la ejecución de integración registrada.

La E2E académica crea una tarea mediante formulario, comprueba invisibilidad antes de publicación, HTML mostrado como texto (sin ejecutarlo), recuperación ante API inaccesible sin perder respuesta, guardado por teclado, recarga persistente, borrador privado, aislamiento de Sol incluso después de enviar su trabajo y rechazo 404 al intentar calificarlo por API, envío de Luna, calificación oculta y devolución visible, aviso de cambios sin guardar y logout. Se ejecuta con viewport tablet y reduced motion. Capturas: screenshots/academic-grading-tablet.png y screenshots/academic-student-tablet.png. Revisión visual: controles legibles y sin desbordamiento horizontal en las capturas. Esto no es auditoría completa WCAG.

## Incidencias de validación

- El sandbox impidió registros de paquetes y conexiones locales: se solicitaron y obtuvieron permisos para herramientas de /tmp, DB y servidores locales.
- El primer arranque PostgreSQL inicializó archivos pero no pudo abrir sockets; se reutilizó el clúster inicializado sin borrar datos.
- Vitest con pool forks no arrancó el worker dentro del tiempo permitido en este montaje. Con --pool=threads ejecutó las cuatro pruebas correctamente; no se modificó la configuración del proyecto para ocultar el fallo.
- La primera E2E académica capturaba el ID del curso antes de terminar navegación; la prueba se corrigió con espera explícita de URL y pasó.
- Ejecutar build mientras estaba activo el dev server invalidó referencias de optimización de CodeMirror; la regresión de laboratorio devolvió 504 Outdated Optimize Dep. Tras reiniciar el servidor de desarrollo sin compilar en paralelo, las seis pruebas pasaron. No se modificó CodePlayground ni se relajó su prueba para ocultar el fallo.

## Límites y pendientes

Este es el incremento 1, no los siete incrementos completos ni una certificación de 98 %. La comparación del inventario con una instalación Moodle no se ejecutó. Las cuatro capacidades académicas con evidencia interna permanecen parciales a efectos de certificación.

Pendientes: administración Go de usuarios/roles y asignaciones desde UI, fechas/prórrogas, adjuntos, reentregas, entregas grupales, rúbricas, banco de preguntas, cuestionarios, libro agregado de calificaciones y resto del alcance acordado. Sin implementación de las exclusiones (foros, wikis, gestor de extensiones, respaldos y panel de operación). Las actividades antiguas del seed permanecen publicadas y no se migran automáticamente al editor.

CodePlayground conserva MockRunner y muestra ejecución simulada; no ejecuta código. H5P sigue sin paquete válido verificado. No se ejecutaron Docker, una auditoría de privacidad, restauraciones, pruebas de carga ni evaluación de runner aislado en este incremento. No se declara listo para producción.

## Reproducir

Seguir README, modo A con PostgreSQL o Compose (Compose no fue ejecutado aquí), aplicar migraciones y seed. Con Go/Astro activos y cuentas sintéticas en base separada:

```bash
# Desde backend; sustituir por la conexión a una base terminada en _test.
TEST_DATABASE_URL='postgres://usuario:clave@localhost:5432/aulaquest_test?sslmode=disable' go test -count=1 ./...
go vet ./...
# Desde la raíz.
npm --prefix frontend run check
npm --prefix frontend run build
npm --prefix frontend test -- --pool=threads
# Desde frontend, servidor activo y credenciales de seed ya configuradas.
E2E_ACADEMIC=1 E2E_STUDENT_PASSWORD='change-this-student-password' E2E_TEACHER_PASSWORD='change-this-teacher-password' npm run test:e2e -- academic.spec.ts learning.spec.ts
```

No ejecutar build en paralelo con las pruebas del servidor dev. Las credenciales del ejemplo no sustituyen las de cuentas previamente creadas. Las E2E agregan datos sintéticos y no borran datos existentes.

## Vista local preparada en esta sesión

`http://localhost:4323/login`, con Astro en 127.0.0.1:4323 y Go en 127.0.0.1:8083. Cuenta docente `profe` / `academic-teacher-pass`; estudiante `luna` / `academic-student-pass`. Son credenciales sintéticas de academic_preview, distintas de la demo Node y de los .env de ejemplo. Los procesos son temporales de esta sesión; para reproducir después se usan los comandos del README y las credenciales configuradas allí.

Resultado final del ajuste de permisos: go test -count=1 ./... con PostgreSQL y go vet ./... aprobados; E2E académica repetida con el backend final aprobada. La suite de seis pruebas ya había pasado tras reiniciar Astro; el último ajuste se verificó específicamente en la prueba académica y en integración.
