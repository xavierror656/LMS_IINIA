# Diseño: daisyUI 5 como sistema de interfaz

## Integración

CSS-first, sin `tailwind.config.mjs`:

```css
@import "tailwindcss";
@plugin "daisyui" { themes: false; }
@plugin "daisyui/theme" {
  name: "aulaquest";
  default: true;
  prefersdark: false;
  color-scheme: light;
  /* variables mapeadas abajo */
}
```

Se apagan los temas incluidos para no cargar CSS muerto. Los componentes solo se
emiten si se usan. Versión exacta `daisyui@5.7.47` con
`npm install --save-exact`, `package-lock.json` actualizado.

## Tema `aulaquest`

Mapeo de los tokens vigentes (hex permitido en daisyUI 5). Los valores de
`*-content` se eligen para superar 4.5:1; el cálculo se adjunta en
`docs/contrast.json` tras la fase 1.

| Variable daisyUI | Valor | Origen |
|---|---|---|
| `--color-base-100` | `#ffffff` | superficies |
| `--color-base-200` | `#f8f7fc` | canvas (`--color-canvas`) |
| `--color-base-300` | `#e8e3f3` | línea (`--color-line`) |
| `--color-base-content` | `#292442` | tinta (`--color-ink`) |
| `--color-primary` | `#6534cf` | morado eléctrico |
| `--color-primary-content` | `#ffffff` | texto sobre morado |
| `--color-secondary` | `#d5f578` | verde lima |
| `--color-secondary-content` | `#292442` | tinta sobre lima |
| `--color-accent` | `#ffdb66` | amarillo sol |
| `--color-accent-content` | `#292442` | tinta sobre sol |
| `--color-neutral` | `#635d78` | gris violáceo (`--color-muted`) |
| `--color-neutral-content` | `#ffffff` | texto sobre neutral |
| `--color-info` | `#d9efff` | azul cielo |
| `--color-info-content` | `#292442` | tinta sobre cielo |
| `--color-success` | `#d5f578` | progreso positivo (lima) |
| `--color-success-content` | `#292442` | tinta sobre lima |
| `--color-warning` | `#ffdb66` | recompensas (sol) |
| `--color-warning-content` | `#292442` | tinta sobre sol |
| `--color-error` | `#a12240` | peligro (`--color-danger`) |
| `--color-error-content` | `#ffffff` | texto sobre peligro |
| `--radius-box` | `1rem` | tarjetas y paneles |
| `--radius-field` | `0.75rem` | botones y campos |
| `--radius-selector` | `1rem` | controles de selección |
| `--border` | `2px` | bordes visibles actuales |
| `--depth` / `--noise` | `0` / `0` | superficies limpias |

## Mapa de clases

Los nombres definitivos se confirman contra la documentación instalada de la
versión 5.7.47 antes de escribir marcado (la lista de v4 no aplica).

| Actual | daisyUI 5 | Nota |
|---|---|---|
| `.button` | `btn btn-primary` | conservar borde inferior/bloque presionable |
| `.button.quiet` | `btn btn-outline` o `btn-ghost` | decidir por contraste |
| campos HTML | `input`, `select`, `textarea`, `checkbox` | alto ≥44 px y foco visible |
| `.panel` | `card` + `card-body`, o composición documentada `panel` sobre variables del tema | se decide por estructura en la fase 2 |
| tablas del libro | `table` | contenedor con `overflow-x-auto` |
| `.alert`/`ServiceError` | `alert alert-error` | texto comprensible ya existente |
| `.crumb` | `link` con objetivo ≥44 px | navegación de retorno |
| `.toolbar` | utilidades `flex flex-wrap gap-2 items-center` | daisyUI recomienda utilidades para layout |
| `.helper` | utilidad de texto atenuado | mismo tono ya auditado |
| `.eyebrow` | `badge`/utilidad | etiqueta de sección |
| HUD y progreso | `progress`, `badge`, `avatar` | dentro de la isla React, sin cambiar lógica |

Regla: si daisyUI no cubre algo, se usan utilidades Tailwind; solo se conserva
CSS propio documentado para composiciones que no son un componente de daisyUI
(por ejemplo el mapa de aprendizaje). No se permite reimplementar botones,
campos, alertas ni tablas con CSS propio.

## Estrategia por fases

1. **Tema base**: instalar, configurar tema, verificar coexistencia sin tocar
   páginas; contrastes y check/build.
2. **Shell y compartidos**: Layout, header, HUD, footer, ServiceError, botones,
   formularios compartidos, login.
3. **Docente**: cursos, actividades, libro, grupos, banco, cuestionarios.
4. **Alumno y público**: inicio, catálogo, curso, lecciones, admin demo.
5. **Limpieza**: retirar CSS y clases heredadas, medir CSS emitido, correr todas
   las suites, actualizar capturas y documentación.

Cada fase termina con `astro check`, `astro build`, la suite E2E del área,
`accessibility.spec.ts` y recálculo de contraste si cambian colores. La revisión
Oracle es obligatoria antes de la siguiente fase.

## Riesgos y mitigación

- **HMR no detecta cambios en este filesystem**: reiniciar `npm run dev` antes de
  cada validación o validar contra build SSR.
- **Límite de login 10/min por IP**: ejecutar una suite por corrida o reiniciar
  la API para limpiar contadores; no se debilita el límite.
- **Selectores E2E**: se conservan roles, textos y atributos `data-*`; las suites
  dependen de etiquetas accesibles, no de clases.
- **Capturas binarias**: se regeneran al final; se documenta cuáles cambian.
- **Contraste**: los pares actuales pasan 4.5:1; cualquier par nuevo (por
  ejemplo `secondary`) se recalcula antes de aceptar.
- **Doble sistema temporal**: prohibido añadir CSS nuevo fuera del tema durante
  las fases; la limpieza final elimina lo heredado.
