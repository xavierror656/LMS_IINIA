# Cambio: confirmaciones destructivas con shadcn AlertDialog

## Why

Cuatro acciones destructivas usan `window.confirm`: eliminar una categoría, quitar un archivo de una entrega, eliminar un grupo y quitar un adjunto docente. El diálogo nativo no se puede estilar, ignora el sistema visual, bloquea el hilo y no coincide con la apariencia infantil de AulaQuest. El usuario pidió incorporar componentes de shadcn/ui donde aporten valor.

## What Changes

- Base shadcn/ui en el frontend para islas React: alias `@/*`, `components.json`, `cn()` y variables de tema mapeadas a `aulaquest` (sin duplicar la paleta).
- Componente `AlertDialog` de shadcn (Radix) con variante destructiva.
- Isla `ConfirmHost` montada una vez cuando hay sesión, que expone `window.aulaquestConfirm(options): Promise<boolean>` y cae a `window.confirm` si el host no está montado.
- Los cuatro borrados anteriores usan el nuevo diálogo.

## Fuera de alcance

- Guards de cambios sin guardar (`beforeunload`/navegación): seguirán nativos porque un diálogo asíncrono no puede bloquear la navegación de forma fiable.
- Sonner: la celebración de recompensas del HUD es diseño aceptado y probado; no se sustituye.
- Calendario/selector de fechas: los `datetime-local` nativos están cubiertos por pruebas y el contrato UTC; no se reescriben formularios a React por esto.
- Migrar el resto de daisyUI a shadcn: daisyUI sigue siendo la única capa CSS; shadcn aporta solo primitivas React donde hacen falta.

## Impact

Solo frontend. Dependencias nuevas exactas: `@radix-ui/react-alert-dialog`, `class-variance-authority`, `clsx`, `tailwind-merge`. Las suites E2E de adjuntos y grupos usan hoy eventos de diálogo nativo y se actualizan para pulsar el botón del nuevo diálogo. Se revalidan check/build, E2E afectadas, AC1 y el tamaño de bundles. Sin cambios de API ni backend.
