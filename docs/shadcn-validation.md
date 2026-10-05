# shadcn/ui en AulaQuest: evidencia del cambio `shadcn-alert-dialog`

Especificación: `openspec/changes/shadcn-alert-dialog/` (UI-101..UI-103). Se incorporaron primitivas React de shadcn/ui **solo donde aportan**: confirmación destructiva con `AlertDialog`. daisyUI sigue siendo la única capa de estilo de las páginas Astro.

## Qué se hizo

- Base shadcn para islas React: alias `@/*` (tsconfig + Vite), `components.json`, `cn()` (`clsx` + `tailwind-merge`) y variables semánticas de shadcn mapeadas a los tokens de AulaQuest en `global.css` (`--background`, `--foreground`, `--card`, `--popover`, `--muted-foreground`, `--destructive`, `--border`, `--input`, `--ring`; radios `--radius-*`). No se redefine `--color-muted` para no pisar el texto atenuado propio.
- `AlertDialog` y `Button` de shadcn en `src/components/ui/`, adaptados a las convenciones del proyecto: imports desde `@/lib/utils` y paquetes Radix individuales, variantes `dark:` retiradas (la app es clara), bordes con `border-border`/`border-input`, y botones del diálogo con `min-h-11` (44 px).
- `ConfirmHost` se monta una vez cuando hay sesión y expone `window.aulaquestConfirm`. **El diálogo se descarga solo al primer uso** (`import()` diferido); si el chunk no carga, cae al `window.confirm` nativo. Los guards de cambios sin guardar siguen nativos por ser síncronos.
- Cuatro borrados usan el diálogo: eliminar categoría, quitar archivo de la entrega, eliminar grupo y quitar adjunto docente. Textos en español, título/descripción accesibles y variante destructiva.
- Pulido de tablas con daisyUI: las cinco tablas llevan `table-zebra` y ya estaban en contenedores con scroll.

## Dependencias (exactas)

`@radix-ui/react-alert-dialog` 1.1.23, `@radix-ui/react-slot` 1.3.3, `class-variance-authority` 0.7.1, `clsx` 2.1.1, `tailwind-merge` 3.7.0. Componentes generados con la CLI shadcn 4.21.2 y adaptados. Sin `radix-ui` unificado ni el paquete `cn`.

## Medición (build de producción)

| Escenario | JS total | Diferencia |
|---|---|---|
| Sin la isla (referencia) | 976 712 B · 318 632 B gzip | — |
| Isla estática (descartado) | 1 049 625 B · 341 830 B gzip | +23.2 KB gzip en cada página autenticada |
| Isla diferida (entregado) | carga inicial igual a la referencia; chunk `ConfirmDialog` 72 536 B · **22 989 B gzip** solo al primer confirmar | costo pagado donde se usa |

Verificado en navegador: la carga inicial de `/teacher` no descarga Radix; al abrir el primer diálogo se incorporan `@radix-ui_react-alert-dialog`, `react_jsx-runtime`, `clsx`, `tailwind-merge`, `class-variance-authority` y `@radix-ui_react-slot`. Sin JavaScript de Radix en páginas públicas.

## Validación ejecutada

- `astro check` 0 errores (66 archivos), build SSR completo, Vitest 4/4.
- E2E: `attachments` (dos borrados por diálogo), `groups` (dos eliminaciones), `accessibility` (AC1 0 hallazgos en 10 páginas), `learning` (5), `gradebook`, `grade-categories`, `academic` y `quiz-timing` (2) aprobadas.
- Las pruebas de adjuntos y grupos ya no aceptan diálogos nativos: pulsan el botón del `AlertDialog`.
- OpenSpec `shadcn-alert-dialog` validado `--strict`.

## Fuera de alcance con motivo

- **Sonner**: la celebración de recompensas del HUD es diseño aceptado y probado (`motion.spec.ts`); sustituirla degradaría la experiencia y añadiría dependencia sin un caso genérico de notificaciones.
- **Calendario/date picker**: los `datetime-local` nativos están cubiertos por pruebas y el contrato UTC; migrarlos obligaría a reescribir formularios SSR a islas React.
- **shadcn Table**: es un componente React; las tablas son SSR de Astro y ya usan daisyUI `table`. Se pulieron con `table-zebra` sin añadir JS.
- **Guards de cambios sin guardar**: seguirán nativos porque un diálogo asíncrono no puede bloquear de forma fiable la navegación.

## Reproducir

```bash
npm --prefix frontend run types:api
npm --prefix frontend run check && npm --prefix frontend run build
# Con Go/PostgreSQL/Astro activos:
E2E_ACADEMIC=1 E2E_STUDENT_PASSWORD='...' E2E_TEACHER_PASSWORD='...' npm --prefix frontend run test:e2e -- attachments.spec.ts groups.spec.ts
```

Sin commit; el trabajo queda en el árbol para revisión.
