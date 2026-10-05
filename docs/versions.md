# Versiones y fuentes
Consultadas el 1 de octubre de 2026 mediante metadatos npm, proxy.golang.org y documentación oficial. Se conservan package-lock.json y go.sum. Dependencias directas exactas en package.json/go.mod.

- Node 22.23.3; npm 10 local para validación. Astro 7.3.5 exige Node >=22.12 y npm >=9.6.5; el Node 20 del entorno no bastaba.
- Astro 7.3.5, @astrojs/react 7.0.0, @astrojs/node 11.1.6, React 19.3.0; peers verificados.
- Tailwind CSS/@tailwindcss/vite 4.3.3, CSS-first; [documentación oficial](https://tailwindcss.com/docs/installation/framework-guides/astro).
- ClientRouter y transition:persist: [Astro](https://docs.astro.build/en/guides/view-transitions/).
- TypeScript 5.9.3: elegido por compatibilidad con openapi-typescript 7.13.0 (peer ^5.x); no se forzó TypeScript 6.
- Nano Stores 1.5.4, @nanostores/react 2.0.1, Framer Motion 13.5.0, Lucide React 1.49.0. CodeMirror 6; versiones exactas por módulo en package.json.
- h5p-standalone 3.8.2: API revisada en README instalado y [repositorio oficial](https://github.com/tunapanda/h5p-standalone). Su preinstall exige Yarn. .npmrc usa ignore-scripts=true; se usan artefactos dist publicados, y scripts/copy-h5p.mjs copia el reproductor. Compilación y reproducción del estado no configurado verificadas; paquete educativo real pendiente.
- Lexend @fontsource/lexend 5.3.0, servido localmente. Licencia SIL OFL conservada en node_modules y docs/licenses.
- Go 1.27.1 descargado de [go.dev](https://go.dev/dl/) y verificado con SHA256 oficial.
- Fiber v2.52.15, contrib/websocket v1.3.4, GORM v1.31.2, driver PostgreSQL v1.6.3 y x/crypto v0.57.0. Se elige la línea Fiber v2 compatible con [su adaptador WebSocket](https://docs.gofiber.io/contrib/websocket/); no se mezclan APIs v2/v3.
- PostgreSQL real local para pruebas mediante embedded-postgres 18.4.0-beta.17 en /tmp. Esta herramienta de prueba no es dependencia del producto. Compose usa postgres:18.4-alpine; imágenes Docker no ejecutadas en este entorno sin Docker.
- Incremento 9: PostgreSQL 18.6 dentro de WSL2 (Ubuntu 26.04), instalado sin privilegios de administrador con `apt-get download` y `dpkg -x` en un prefijo del usuario (`postgresql-18`, `postgresql-client-18`, `libpq5`, `libicu78`, `libnuma1`, `liburing2`) y arrancado con un directorio de socket propio. Se usó para migraciones, seed y pruebas de integración y de navegador. No es dependencia del producto ni se publica fuera de la VM.
- Incremento 17: WSL2 Debian 13 con Go 1.27.1 oficial verificado por SHA256, Node 22.23.3 y PostgreSQL 18.6 arrancado mediante binarios zonky (18.6.0) en un prefijo del usuario; Chromium headless de Playwright con las bibliotecas del sistema extraídas de paquetes .deb sin root. OpenSpec CLI 1.14.0 (@fission-ai/openspec) instalada en un prefijo de /tmp: validación estricta del cambio aprobada.
- OpenSpec CLI 1.14.0: init/new/instructions/validate consultados mediante --help. No se instaló otro framework SpecOps.
- create-astro 5.2.4 --help consultado: admite --template, --no-install, --no-git y --yes. No se ejecutó scaffolding sobre frontend.

Revisión de seguridad: se fijaron fasthttp v1.70.0 y klauspost/compress v1.18.7 para resolver GO-2026-4950 y GO-2026-5841 en dependencias transitivas. Regresión PostgreSQL y WS aprobada después del cambio. Govulncheck conserva únicamente el aviso del módulo x/crypto sobre OpenPGP (no importado ni invocado).
