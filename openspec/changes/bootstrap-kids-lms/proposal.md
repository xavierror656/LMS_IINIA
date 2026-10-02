# Proposal
## Why
AulaQuest ofrece aprendizaje infantil con progreso verificable y una interfaz cómoda en tablet, sin complejidad de administración escolar.
## What Changes
- Monorepo Astro SSR y monolito Go/PostgreSQL con sesiones, cursos, recompensas, consola simulada y panel docente.
- H5P confiable opcional, con ausencia explícita de contenido.
- Contratos, migraciones, pruebas y entorno reproducible.
## Capabilities
### New Capabilities
- `identity`: sesiones y autorización por objeto.
- `learning`: catálogo, lecciones, progreso y docente.
- `interactive-content`: consola WebSocket y reproductor H5P.
- `experience`: HUD, accesibilidad y operación.
### Modified Capabilities
Ninguna: repositorio nuevo, únicamente Agents.md.
## Impact
Nuevas carpetas frontend, backend, contracts, docs y configuración Docker. Sin despliegue externo ni cambios a datos existentes.
