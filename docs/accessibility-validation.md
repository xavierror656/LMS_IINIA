# Auditoría de accesibilidad: evidencia de AC1 (T6-2)

Especificación previa: `openspec/changes/academic-administration/increment-16.md`, AC1; T6-2 / LMS-022 / A8. Ejecutada tras cerrar el incremento 17 con la interfaz de categorías ya montada.

## Método

Suite Playwright `frontend/tests/accessibility.spec.ts` con navegador real, viewport de tablet 820×1180 y `prefers-reduced-motion: reduce`, contra Astro SSR y Go/PostgreSQL con datos sintéticos. En cada página se evalúan criterios comprobables en el DOM:

- Exactamente un `h1` y sin saltos de nivel en el orden de encabezados.
- Todo `input`, `select` y `textarea` visible tiene etiqueta asociada (`label[for]`, `label` envolvente o `aria-label`/`aria-labelledby`).
- Todo botón o enlace tiene nombre accesible (texto o `aria-label`).
- Objetivos táctiles de 44×44 CSS px como mínimo; para un control dentro de su etiqueta se mide la unión con el área clicable de la etiqueta, que es el objetivo real.
- Foco visible en los primeros ocho destinos de tabulación de la página más densa, comprobando contorno o sombra calculados.
- Con movimiento reducido activo no queda ninguna animación infinita en ejecución.
- Contraste: se conservan los seis pares de tokens verificados matemáticamente en `docs/contrast.json` (mínimo 6.24:1, WCAG AA). La interfaz nueva reutiliza esos mismos tokens.

Páginas auditadas: `/login`; docente `/teacher`, curso, libro de calificaciones, editor de evaluación y banco de preguntas; alumno `/courses`, detalle de curso y lección de lectura.

## Hallazgos y correcciones

Primera pasada: 27 hallazgos, todos de tamaño táctil. Dos correcciones reales:

- El enlace de marca `✦AulaQuest` medía 42 px de alto: ahora declara `min-height: 44px` en `.brand` (`global.css`).
- Los campos del gestor de categorías (nombre, peso y selector de política) medían 22–24 px de alto porque no usan `.academic-form`: ahora tienen `min-height: 44px` en `GradeCategories.astro`.

Los checkboxes de 24 px de ancho quedaron correctos al medir la unión con su etiqueta, que es el área que recibe el toque. Segunda pasada: cero hallazgos en las diez páginas, foco visible en los ocho destinos de tabulación y sin animaciones infinitas con movimiento reducido.

## Archivos

- `frontend/tests/accessibility.spec.ts`: suite AC1 condicionada a `E2E_ACADEMIC=1`.
- `frontend/src/styles/global.css` y `frontend/src/components/astro/GradeCategories.astro`: correcciones de tamaño táctil.
- `docs/contrast.json`: pares de contraste vigentes; `docs/validation.md` conserva la evidencia original de foco y objetivos del MVP.

## Validación ejecutada

- Playwright AC1: aprobada en las diez páginas, 0 hallazgos (5.0 s).
- `astro check`: 61 archivos, 0 errores. Build SSR completado.
- Regresión E2E de `grade-categories.spec.ts`, `rubric.spec.ts` y `gradebook.spec.ts`: aprobada en la misma corrida de validación del incremento 17.
- Entorno: WSL2 Debian 13, Node 22.23.3, Chromium headless de Playwright, Astro dev y API Go sobre PostgreSQL 18.6.

## Límites

No es una auditoría WCAG completa: no se probaron lectores de pantalla reales (NVDA, VoiceOver), navegación por voz, dispositivos físicos ni todos los estados dinámicos de la aplicación. La comprobación de foco cubre los primeros destinos de una página; los flujos con modales o menús emergentes quedan para una revisión futura. El contraste se verificó sobre los tokens, no sobre cada combinación de estado; los estados nuevos usan los mismos tokens auditados.
