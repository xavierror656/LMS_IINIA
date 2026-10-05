# Diseño: shadcn AlertDialog para acciones destructivas

## Base shadcn en Astro (Tailwind 4)

- `tsconfig.json`: `baseUrl: "."` y `paths: { "@/*": ["./src/*"] }`.
- `astro.config.mjs`: alias Vite `"@" -> ./src` para que Astro resuelva lo mismo que TypeScript.
- `frontend/components.json`: estilo `new-york`, `rsc: false`, `tsx: true`, CSS `src/styles/global.css`, alias `@/components`, `@/components/ui`, `@/lib/utils`.
- `src/lib/utils.ts` con `cn()` (`clsx` + `tailwind-merge`).
- Los componentes de shadcn son fuente copiada en `src/components/ui/`; se añaden con `npx shadcn@latest add` o manualmente si el CLI no está disponible.

## Tokens

shadcn usa variables semánticas (`--background`, `--foreground`, `--card`, `--popover`, `--primary`, `--secondary`, `--muted`, `--accent`, `--destructive`, `--border`, `--input`, `--ring`, `--radius`). Se definen en `:root` con los valores de AulaQuest y se exponen con `@theme inline` para las utilidades (`bg-background`, `text-foreground`, `border-border`, `bg-destructive`…). No se redefine `--color-primary`/`--color-secondary`/`--color-accent` porque ya existen en el `@theme` de daisyUI con el mismo valor; se mapean las que faltan. Sin modo oscuro.

| Variable shadcn | Valor AulaQuest |
|---|---|
| `--background` | `#f8f7fc` (canvas) |
| `--foreground` | `#292442` (ink) |
| `--card` / `--popover` | `#ffffff` |
| `--muted` | `#f1ecfa` |
| `--muted-foreground` | `#635d78` |
| `--destructive` | `#a12240` |
| `--border` / `--input` | `#e8e3f3` |
| `--ring` | `#6534cf` |
| `--radius` | `1rem` |

## Host de confirmación

`src/components/react/ConfirmHost.tsx` se monta en `Layout.astro` cuando hay usuario (`client:load`). Al montar registra `window.aulaquestConfirm`. La API acepta un string o `{title, description, confirmLabel, cancelLabel, destructive}` y devuelve una promesa que resuelve `true/false` al pulsar confirmar/cancelar o cerrar con Escape/fuera. Estados: sin diálogo devuelve `null`; con diálogo renderiza `AlertDialog` con `AlertDialogTitle` y `AlertDialogDescription` (accesibles), foco atrapado por Radix y variante destructiva. Si el host no está montado, `window.confirm` nativo como respaldo para no bloquear la acción.

## Reemplazos

| Archivo | Acción | Confirmación |
|---|---|---|
| `components/astro/GradeCategories.astro` | eliminar categoría | `{title:"Eliminar categoría", description:"¿Eliminar la categoría «X»? Solo se puede si no tiene actividades.", confirmLabel:"Eliminar", destructive:true}` |
| `components/astro/AttachmentManager.astro` | quitar archivo de la entrega | `{title:"Quitar archivo", description:"¿Quieres quitar «X» del borrador?", confirmLabel:"Quitar", destructive:true}` |
| `pages/teacher/courses/[courseId]/groups.astro` | eliminar grupo | `{title:"Eliminar grupo", description:"¿Eliminar el grupo «X»? Sus estudiantes quedarán sin grupo.", confirmLabel:"Eliminar", destructive:true}` |
| `pages/teacher/activities/[activityId].astro` | quitar adjunto docente | `{title:"Quitar adjunto", description:"¿Quitar «X» del borrador? Las versiones publicadas conservan su copia.", confirmLabel:"Quitar", destructive:true}` |

Los handlers ya son asíncronos o usan `void run(...)`; se añade `await` donde corresponde y se conserva el estado de ocupado. Los guards de cambios sin guardar siguen con `window.confirm` (síncrono) y `beforeunload` nativo.

## Validación

- `astro check`, build SSR, Vitest.
- E2E actualizadas: `attachments.spec.ts` (dos borrados) y `groups.spec.ts` pulsan el botón del diálogo en lugar de aceptar el diálogo nativo. Regresión de `learning`, `grade-categories`, `quiz-timing` (guards nativos intactos) y `accessibility`.
- Medición de bundles JS antes/después (el host añade Radix a las páginas autenticadas).
- Riesgos: el nuevo JS se carga en todas las páginas autenticadas; se mitiga con `client:load` de un único componente y sin cargarlo en páginas públicas.
