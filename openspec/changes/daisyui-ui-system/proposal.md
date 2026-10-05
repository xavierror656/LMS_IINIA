# Cambio: sistema de interfaz daisyUI 5

## Why

La interfaz actual usa un sistema propio (`global.css`, 1061 líneas) con clases
caseras (`.button`, `.panel`, `.crumb`, `.toolbar`, formularios ad hoc) repetidas
por página. Cada incremento añade más CSS y más variantes; mantener coherencia,
accesibilidad y estados cuesta cada vez más. El usuario decidió adoptar daisyUI
como capa de componentes y migrar el frontend completo ahora, conservando la
identidad infantil de AulaQuest.

## What Changes

- Integrar daisyUI 5.7.47 con Tailwind CSS 4.3.3 en modo CSS-first (`@plugin`),
  sin `tailwind.config.mjs`.
- Definir el tema `aulaquest` con los tokens actuales (morado eléctrico, verde
  lima, amarillo sol, azul cielo, fondos claros), radios grandes y contraste AA.
- Migrar todas las pantallas Astro a componentes daisyUI (`btn`, `card`, `alert`,
  `badge`, `table`, `input`, `select`, `textarea`, `link`, `menu`, etc.).
- Retirar las clases propias duplicadas y el CSS que daisyUI cubra; conservar
  solo tokens, layout global y lo que ningún componente resuelva.
- Revalidar accesibilidad (AC1), contraste, E2E y capturas después de cada fase.

## Capabilities

- ui-system: sistema visual y de componentes de toda la interfaz infantil.

## Versiones verificadas (2026-10-05)

- daisyUI 5.7.47 (latest) — plugin CSS-first para Tailwind 4, temas por
  `@plugin "daisyui/theme"` con variables CSS; valores hex admitidos.
- Tailwind CSS y `@tailwindcss/vite` 4.3.3 (ya instalados), Astro 7.3.5,
  React 19.3.0. Sin dependencias JavaScript nuevas en el cliente.
- Node 22.23.3 para compilar; el paquete se fija exacto en `package.json` y
  `package-lock.json`.

## Fuera de alcance

- Cambiar la estructura de navegación, los flujos o la UX acordada.
- Modo oscuro: el producto es claro; no se activa `prefersdark`.
- Reemplazar CodeMirror, H5P ni la lógica interna de las islas React; solo sus
  contenedores y estados visuales.
- Rediseñar la identidad de marca (nombre, logo, colores base).
- Publicar, desplegar o tocar datos reales.

## Impact

Frontend completo (~61 archivos Astro, islas React del HUD y formularios
compartidos). Sin cambios de API, base de datos ni backend. Requiere recalcular
`docs/contrast.json`, volver a ejecutar la auditoría AC1 y las suites E2E, y
actualizar capturas. La migración se ejecuta por fases con compuerta de revisión
para no dejar el frontend roto entre pasos.
