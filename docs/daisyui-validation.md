# Migración a daisyUI 5: evidencia del cambio `daisyui-ui-system`

Especificación: `openspec/changes/daisyui-ui-system/` (UI-001..UI-006). Frontend Astro 7.3.5 + Tailwind 4.3.3 + islas React migrado por completo a daisyUI 5.7.47 con tema propio. Sin cambios de API, backend ni base de datos.

## Qué se hizo

Se integró daisyUI en modo CSS-first (`@plugin "daisyui" { themes: false; }` y `@plugin "daisyui/theme"`), sin `tailwind.config.mjs`, con el tema `aulaquest` que conserva los tokens de AulaQuest (morado, lima, sol, cielo, fondos claros), radios grandes y borde de 2 px. Se migraron shell, componentes compartidos, pantallas docente, alumno y públicas a componentes daisyUI (`btn`, `card`, `input`, `select`, `textarea`, `checkbox`, `file-input`, `alert`, `badge`, `link`, `table`, `hero`, `avatar`, `progress`). Los nombres de clase se verificaron contra `node_modules/daisyui@5.7.47/components/*/object.js` (p. ej. `card-border` en v5, sin `input-bordered`).

Se retiraron 242 líneas de CSS heredado y todas las clases propias que daisyUI cubre: `.button`, `.quiet`, `.lime`, `.crumb`, `.eyebrow`, `.helper`, `.toolbar`, `.tag`, `.path`, `.features`, `.course-visual`, `.lesson-link`, `.step`, `.login-wrap`, `.plugin-intro`, `.section-heading`, `.demo-notice`, `.grid`, `.status` y los alias temporales. Se conservan como composiciones documentadas: `.panel` (superficie), `.hero`/`.hero-art` (portada), `.course-card` (tarjeta animada; su selector lo usa `motion.spec.ts`), `.notice`, `.demo-access`, `.reading-body`, `.playground`, `.console`, `.editor`, `.cm-*`, `.h5p-container`, `.xp-track` y el efecto presionable global de botones.

## Requisitos y evidencia

- **UI-001 Tema**: el CSS emitido contiene solo `[data-theme=aulaquest]`; no se emiten los temas incluidos. `docs/contrast.json` ampliado con 10 pares del tema: todos ≥6.24:1 (mínimo exigido 4.5:1).
- **UI-002 Componentes**: migración completa de las 61 vistas; los E2E existentes pasan sin cambios de contrato. La auditoría AC1 (`accessibility.spec.ts`) recorre 10 páginas con 0 hallazgos de etiquetas, encabezados, nombres, objetivos táctiles (≥44 px), foco visible y movimiento reducido.
- **UI-003 Un solo sistema**: búsqueda de las clases retiradas en `frontend/src` sin referencias; `global.css` bajó de 1061 a 819 líneas y ya no reimplementa botones, campos, alertas ni tablas.
- **UI-004 Accesibilidad**: suite AC1 aprobada tras la migración; se corrigieron dos tamaños táctiles (marca y controles del gestor de categorías) y una carrera de la prueba de categorías.
- **UI-005 Sin regresión**: batería E2E completa aprobada suite por suite (API reiniciada o ventana del limitador respetada): academic, attachments, attempts, gradebook, groups, group-submission, learning, motion, questions, quiz-timing, quizzes, rubric, schedule, grade-categories, accessibility. `admin` se salta sin `PUBLIC_DEMO_MODE`, como estaba previsto. `astro check` 0 errores; build SSR completo; Vitest 4/4.
- **UI-006 Rendimiento medido**: mismo árbol, con y sin el plugin:

| Artefacto | Sin daisyUI | Con daisyUI (final) | Delta |
|---|---|---|---|
| CSS de la app (sin comprimir) | 27 723 B | 91 239 B | +63 516 B |
| CSS de la app (gzip) | 6 923 B | 15 809 B | +8 886 B |

El crecimiento comprimido es de ~8.9 KB a cambio del sistema completo de componentes; no se añadió JavaScript de cliente (daisyUI es solo CSS). La mayor parte del CSS sin comprimir corresponde a componentes que sí se usan; daisyUI también emite componentes cuyos nombres coinciden con etiquetas HTML (`input`, `select`, `table`) presentes en el marcado.

## Hallazgos durante la validación

- `@apply` dentro de `<style scoped>` sin `@reference` rompía la página `/courses` con HTTP 500 en runtime; `astro check` y el build no lo detectan. Se sustituyó por CSS plano y se añadió un smoke de rutas con curl a la validación. Reincidencia posible: no volver a usar `@apply` en estilos scoped.
- daisyUI 5 define `.status` como punto indicador y colisionaba con el `.status` del CodePlayground; se sustituyó por utilidades conservando `role="status"`.
- Los botones sin clase y los campos `datetime-local`/`file` se completaron con `btn`, `input` y `file-input`.
- Cambio visible a revisar por el usuario: el `h1` del catálogo pasó a un saludo personalizado y "Mis aventuras" quedó como `h2`. Los E2E pasan porque buscan el encabezado por rol; es una decisión de copy de la fase de diseño.

## Reproducir

```bash
npm --prefix frontend run types:api
npm --prefix frontend run check
npm --prefix frontend run build
npm --prefix frontend test
# Con Go/PostgreSQL/Astro activos y E2E_ACADEMIC=1:
E2E_ACADEMIC=1 E2E_STUDENT_PASSWORD='...' E2E_TEACHER_PASSWORD='...' npm --prefix frontend run test:e2e
```

Validación estricta del cambio OpenSpec: aprobada con la CLI 1.14.0. Entorno: WSL2 Debian 13, Node 22.23.3, PostgreSQL 18.6, Chromium headless de Playwright.

## Límites

No es una auditoría WCAG completa (sin lectores de pantalla reales). Nginx/Docker no se ejecutaron. La comparación con Moodle y la cobertura del 98 % siguen fuera de este cambio, al igual que el inventario de plataforma. No se hizo commit; el trabajo queda en el árbol para revisión.
