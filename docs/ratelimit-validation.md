# Límites por usuario y proxy confiable: incremento 10 (TSEC1)

Especificación previa `openspec/changes/academic-administration/increment-10.md`, RL1–RL7 / TSEC1, relacionado con LMS-021/022. No debilita ninguna protección y no acredita el 98 %.

## Problema y decisión

El limitador global anterior usaba la IP del par como clave con 180 solicitudes por minuto. Una página privada se despliega en varias llamadas al API, así que un aula con treinta estudiantes tras una sola salida a Internet agotaba una cuota colectiva y recibía 429 en tráfico legítimo. Además, `infra/nginx.conf` no reenviaba `X-Forwarded-For`, de modo que bajo Compose todos los clientes compartían la IP del contenedor proxy.

Ahora el tráfico autenticado se cuenta **por usuario** y el no autenticado **por dirección real del cliente**. El presupuesto por usuario es de 600 solicitudes por minuto y el de dirección de 180, ambos configurables (`RATE_LIMIT_USER`, `RATE_LIMIT_IP`). Los límites específicos conservan su magnitud —inicio de sesión 10 por minuto por dirección, adjuntos 20 por minuto por usuario, intentos de actividad 30 por minuto— y ahora comparten la clave saneada y la respuesta informada; los intentos de actividad pasan a contar por usuario en lugar de por dirección compartida, que era el mismo defecto de reparto. `/healthz` y `/readyz` quedan sin límite a propósito, para que un sondeo de orquestación no consuma la cuota de nadie ni provoque falsos no saludables, y la ruta WebSocket conserva un límite por dirección.

La IP real se toma de `X-Forwarded-For` **solo** cuando el par directo pertenece a `TRUSTED_PROXIES` (direcciones o rangos CIDR, vacío por defecto). La clave se valida siempre como IP real y, ante un valor ausente, múltiple o no analizable, cae al par directo: rotar la cabecera desde un cliente no confiable no cambia la cuota ni permite eludirla. `infra/nginx.conf` reenvía `$remote_addr` **sobrescribiendo** la cabecera, nunca añadiendo, para que el cliente no pueda inyectar valores. Toda respuesta 429 incluye `Retry-After`.

## Hallazgos durante la implementación

- **Fiber convierte un presupuesto en cero en su propio valor por defecto.** Su limitador trata `Max <= 0` como cinco solicitudes por minuto, así que un `Config` construido sin límites —como hacen las pruebas— endurecía el límite hasta romper la suite con 429. Se añadió `Config.Budgets()`, que sustituye valores no positivos por los predeterminados documentados, y `Load()` rechaza un valor de entorno inválido en lugar de silenciarlo.
- **Fiber considera confiable a cualquier par cuando la verificación está desactivada.** `IsProxyTrusted()` devuelve verdadero si `EnableTrustedProxyCheck` es falso, de modo que fijar `ProxyHeader` de forma permanente permitía a cualquier cliente rotar la cabecera y estrenar presupuesto en cada intento. Lo detectó la prueba nueva antes de commitear; la corrección consiste en no establecer `ProxyHeader` salvo que exista al menos un proxy confiable.
- **Fragilidad de la prueba de navegador del incremento 9.** Al encadenar suites, un clic de navegación podía caer mientras el formulario de publicación seguía guardando: el guardián de «cambios sin guardar» mostraba un diálogo, Playwright lo descartaba y la navegación se cancelaba en silencio. La especificación ahora espera la versión publicada, señal que solo existe tras la recarga. No era un defecto del producto.

## Archivos y contratos

- `internal/config/config.go`: presupuestos y proxies confiables con validación al arrancar, y `Budgets()` como red de seguridad.
- `internal/middleware/ratelimit.go`: limitadores por usuario y por dirección, resolución saneada de la IP del cliente y respuesta 429 con `Retry-After`.
- `internal/handlers/app.go` y `api.go`: el limitador global se sustituye por los de grupo; la ruta WebSocket recibe el suyo. Sin cambios de esquema ni de OpenAPI: son límites de transporte.
- `infra/nginx.conf`, `docker-compose.yml`, `.env.example` y `backend/.env.example` alineados con la nueva configuración.

## Evidencia ejecutada

- `go vet ./...`: sin diagnósticos.
- `go test ./... -count=1` con `TEST_DATABASE_URL` contra PostgreSQL 18.6 real dentro de WSL: todas las pruebas aprobadas, `handlers` 24.8 s. Incluye las suites de configuración, middleware y los dos casos de integración nuevos.
- Pruebas nuevas de middleware: dos usuarios distintos mantienen presupuestos independientes y agotar uno no afecta al otro; sin proxy confiable, rotar `X-Forwarded-For` no crea cuotas nuevas; con proxy confiable, direcciones distintas son clientes distintos y cada uno se limita; una cabecera múltiple, vacía o no analizable cae al par directo; el 429 anuncia `Retry-After: 60`.
- Pruebas nuevas de integración con la aplicación real: tres cuentas con sesión comparten dirección y cada una conserva su presupuesto mientras la primera lo agota; el límite estricto de inicio de sesión sigue vigente y se alcanza antes que el de dirección; ocho sondeos de `/healthz` responden 200; sin proxy confiable la cabecera se ignora y con proxy confiable se honra, incluyendo tráfico autenticado.
- Aceptación de navegador: `quiz-timing.spec.ts`, `quizzes.spec.ts` y `schedule.spec.ts` encadenadas en una sola ejecución contra una única instancia de la API —4 pruebas aprobadas, 1.7 m— sin ningún 429. Antes de este incremento la misma secuencia agotaba el limitador por dirección y fallaba; el registro de la API no muestra ninguna respuesta 429.
- `astro check`: 58 archivos, 0 errores, 0 warnings, 0 hints. `vitest`: 4 aprobadas. `astro build` aprobado con el aviso existente de chunks superiores a 500 kB.
- No se ejecutaron Docker ni Nginx reales: el cambio de cabecera del proxy se verifica con pruebas y revisión, no levantando Compose.

## Reproducción

```bash
# backend: configuración y límites
RATE_LIMIT_USER=600 RATE_LIMIT_IP=180 TRUSTED_PROXIES='172.16.0.0/12' go run ./cmd/api
# pruebas
TEST_DATABASE_URL='postgres://aulaquest:aulaquest@127.0.0.1:5432/aulaquest_test?sslmode=disable' go test ./... -count=1
# aceptación de navegador: las suites intensivas se encadenan sin reiniciar la API
E2E_ACADEMIC=1 E2E_STUDENT_PASSWORD='change-this-student-password' E2E_TEACHER_PASSWORD='change-this-teacher-password' \
  npm run test:e2e -- quiz-timing.spec.ts quizzes.spec.ts schedule.spec.ts
```

En dos terminales, `TRUSTED_PROXIES` debe quedar **vacío**: la API se alcanza directamente y ninguna cabecera es de fiar. Bajo Compose se configura el rango de la red interna; en producción debe fijarse el rango exacto del ingress.

## Límites pendientes

El almacén del limitador es en memoria por proceso, igual que antes: con varias réplicas cada una aplicaría su propio presupuesto, así que un almacén compartido sigue pendiente. No hay defensa contra un atacante con muchas cuentas válidas —el límite de inicio de sesión lo acota, pero no lo elimina— ni sustituye a un WAF o a límites perimetrales. Tampoco se midió el coste del limitador bajo carga real. Docker, Nginx y el despliegue en producción siguen sin ejecutarse en este entorno.
