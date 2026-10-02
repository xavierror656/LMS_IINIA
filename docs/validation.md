# Validación de bootstrap-kids-lms

Fecha: 2026-10-01. Linux x86_64, repositorio bajo montaje de Windows/OneDrive. Node 22.23.3, npm 10, Go 1.27.1, PostgreSQL 18.4 independiente en /tmp, Chromium 153 mediante Playwright 1.63.0. No se tocó ninguna base existente. El entorno original no tenía Go, PostgreSQL, Docker ni bibliotecas de Chromium; SDK y herramientas se prepararon en /tmp. No se realizó push ni despliegue externo.

## Resultados ejecutados

| Comprobación | Resultado |
|---|---|
| OpenSpec 1.14.0 validate bootstrap-kids-lms --strict | Válido |
| npm run check | 0 errores, 0 warnings, 0 hints en 25 archivos |
| npm run build | Compilación SSR y prerender de inicio correctos; aviso de chunk CodeMirror >500 kB |
| npm test | 4 pruebas de store/API aprobadas |
| go test -count=1 ./... con TEST_DATABASE_URL | Aprobado, incluyendo PostgreSQL y WebSocket reales |
| go vet ./... | Sin diagnósticos tras corregir ownership del cancel del runner |
| Playwright sobre servidor compilado | 5 pruebas aprobadas, 18.3 s en esa ejecución |
| npm audit --omit=dev | 0 vulnerabilidades reportadas |
| govulncheck -show verbose ./... | 0 vulnerabilidades alcanzables y 0 en paquetes importados; 1 aviso a nivel módulo para OpenPGP no usado |

La prueba Go de integración crea un esquema nuevo en una base cuyo nombre termina en _test. Ejecuta migraciones y seed dos veces, confirma tres usuarios, comprueba sesión/Origin/roles, catálogo, ocho finalizaciones simultáneas con claves distintas (25 XP, una estrella y dos gemas en total), payload de recompensa rechazado, inscripción y vínculos docentes, evento H5P sin premio, WS accepted/stdout/cancelled y reconexión tras desconexión, y revocación de sesión.

Las E2E verifican login, catálogo/mapa, lectura, actualización del HUD, conservación del mismo elemento de isla al navegar, recarga, logout, consola con eventos reales simulados, cancelación, H5P ausente, docente vinculado, teclado/tablet/movimiento reducido y caída/recuperación del endpoint de progreso. Se ejecutaron primero en desarrollo y después sobre dist/server/entry.mjs con proxy temporal Node a Go. Este proxy de pruebas NO valida Nginx ni Docker.

## Fallos encontrados y corregidos

- Node 20/npm 9.2 iniciales no satisfacían Astro 7: herramientas locales actualizadas.
- H5P preinstall obliga a Yarn: artefactos dist con ignore-scripts y copia explícita de recursos; no se afirma reproducción de paquete real.
- Símbolo Fiber incorrecto y middleware student que interceptaba teacher: corregidos y cubiertos por pruebas.
- go vet señaló el ciclo de cancelación: cada goroutine ahora libera su contexto.
- Primera E2E saltaba el click por consultar count antes de terminar navegación: espera explícita del botón.
- SSR/React funcionaban pero faltaban glifos emoji en Chromium: SVG locales, capturas actualizadas.
- Login dependía del controlador JS para impedir GET nativo. Formulario ahora POST y botón inactivo hasta registrar controlador; comprobado en build.
- Servidor dev conservó módulos anteriores tras build/reinicio por cambios: validación final sobre servidor compilado recién iniciado.
- Dependencias transitivas fasthttp/compress actualizadas a versiones corregidas. Go test/vet y navegador se ejecutaron después.

## Rendimiento medido

Método: 20 peticiones secuenciales calientes desde Chromium por proxy local, backend Go y DB en la misma máquina. Catálogo: mediana 1.6 ms, p95 2.1 ms. Progreso: mediana 1.7 ms, p95 2.2 ms. Son muestras locales, no SLA ni prueba de carga. Datos exactos en browser-measurements.json; al ejecutar de nuevo pueden variar.

JavaScript solicitado en contextos de navegador nuevos sobre build, Resource Timing decodedBodySize (sin compresión); inline medido aparte. Incluye dependencias compartidas y HUD cuando corresponde:

| Ruta | JS externo inicial | Inline |
|---|---|---|
| `/` | 18.5 KiB | 0 B |
| `/courses` | 363.1 KiB | 4510 B |
| `/lessons/1` | 363.7 KiB | 4510 B |
| `/lessons/2` | 899.3 KiB | 4510 B |
| `/lessons/3` | 367.0 KiB | 4640 B |

CodePlayground: 548496 bytes (184090 gzip), cargado únicamente en programación. H5PPlayer solo se solicita en H5P; el reproductor dinámico no se solicita cuando falta configuración. No se carga CodeMirror en catálogo, lectura o H5P. Inventario exacto en bundles.json. El bundle del editor provoca la advertencia Vite >500 kB; reducirlo/lazy-load por lenguaje es trabajo posterior, no se ocultó el warning.

No hubo errores JavaScript de página en la inspección. Se verificó liberación funcional de ejecuciones y conexión mediante cancelación/desconexión/reconexión; no se midió retención de heap a largo plazo. No se promete ausencia de fugas en bibliotecas H5P sin contenido real.

## Accesibilidad y evidencia visual

Capturas en screenshots/courses-desktop.png y courses-tablet.png; inspección visual realizada. Playwright verifica 820×1180, ausencia de overflow en login, foco de skip link, tabulación y objetivo táctil >=44 px. Se respeta prefers-reduced-motion. Contrastes matemáticos WCAG de seis pares de tokens en contrast.json: todos superan 4.5:1 (mínimo 6.24:1). No equivale a auditoría WCAG completa ni prueba en todos los dispositivos físicos.

## Pendientes explícitos

- Docker y Nginx: archivos entregados; no había Docker para construir imágenes ni ejecutar Compose.
- H5P educativo real: falta paquete completo revisado y con licencia/atribución; no verificado E2E. Solo estado ausente e integración de API cliente reportada.
- Runner real: no implementado deliberadamente; no se ejecuta código arbitrario. Ver requisitos de aislamiento en README y diseño.
- Producción: TLS, cuentas operativas, privacidad/retención, backups, proxy confiable, límites distribuidos y auditoría/carga sostenida.
- Govulncheck informa GO-2026-5932 en el módulo x/crypto/OpenPGP, sin corrección publicada. La aplicación solo importa bcrypt; no usa OpenPGP y el análisis no encuentra rutas alcanzables.

## Reproducción en este entorno temporal

Los paths /tmp son de esta sesión y no requisitos del repositorio. Se usaron GOPATH=/tmp/aulaquest-gopath, GOCACHE=/tmp/aulaquest-gocache, SDK /tmp/aulaquest-sdk/go, Node /tmp/aulaquest-tools/node_modules/node/bin y PLAYWRIGHT_BROWSERS_PATH=/tmp/aulaquest-browsers. Bibliotecas de Chromium extraídas de paquetes Debian a /tmp/aulaquest-browser-libs; no se instalaron sobre el sistema. Para otro equipo seguir README con herramientas normales.
