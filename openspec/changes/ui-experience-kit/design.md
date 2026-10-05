# Diseño: fases del kit de experiencia

Regla permanente: daisyUI para SSR (sin JS), shadcn/Radix solo en islas React y con carga bajo demanda. Cada fase cierra con `astro check`, build, E2E del área, `accessibility.spec.ts` y medición de bundle cuando añada JS.

## F1 — Patrones daisyUI sin dependencias

| Qué | Dónde | Detalle |
|---|---|---|
| `steps` + `progress` | mapa del curso, módulos | estado por lección y avance del módulo; sin cambiar rutas |
| `collapse` | instrucciones, rúbrica, historial de intentos | contenido largo plegable en tablet |
| Estados vacíos | catálogo, entregas, grupos, banco, intentos | tarjeta con icono, texto y CTA |
| `skeleton` | HUD, consola, H5P | reemplaza textos de carga |
| `breadcrumbs` | subpáginas docente | migas consistentes |
| `join` | paginación del libro y listados | botones unidos accesibles |
| tooltips de texto | iconos y HUD | `tooltip` daisyUI; el HUD es isla, se revisa no romper motion |

Riesgo: cambios de markup en páginas con E2E. Mitigación: conservar roles, textos y `data-*`.

## F2 — Menú de usuario

Isla `UserMenu` con DropdownMenu de shadcn (Radix) cargada bajo demanda al primer clic, o inmediata si el costo medido es bajo. Acciones: “Mi perfil” (cuando exista), “Salir” (reutiliza el logout actual). Conserva `#logout` para la prueba existente.

## F3 — Dialog/Sheet de shadcn

- `Dialog`: crear grupo y editar categoría sin recargar.
- `Sheet`: calificación rápida desde el libro (score + feedback) reutilizando el flujo de la API.
- Reutiliza la infraestructura lazy del ConfirmHost (`@/components/ui/dialog`, `sheet`).
- Dependencias nuevas: `@radix-ui/react-dialog` (el Sheet usa `react-dialog`).
- Validación: E2E grupos, categorías y libro; AC1 con foco atrapado y Escape.

## F4 — Tabs del curso docente

Navegación con `tabs` de daisyUI sobre enlaces reales (Actividades · Grupos · Calificaciones · Banco). Sin JS; deep links intactos; E2E de navegación.

## F5 — Combobox/Command de shadcn

- Selector de preguntas del banco en QuizComposer (búsqueda por nombre/tipo, versión visible).
- Selector de estudiante en grupos y excepciones.
- Dependencias: `@radix-ui/react-popover`, `cmdk`.
- Carga bajo demanda; el payload y el contrato no cambian.

## F6 — Sonner y guardado sin recarga

Convertir los guardados de los formularios compartidos a fetch sin `location.reload()`, usando Sonner para confirmaciones y errores. Depende de F3. Es la fase más riesgosa: E2E completa y AC1; se conserva el estado ante error.

## F7 — Calendar para fechas

Reemplazar los `datetime-local` de programación y excepciones por Calendar+Popover de shadcn con entrada UTC explícita, solo si F6 dejó esos formularios como islas. Contrato UTC y pruebas existentes se conservan; validación schedule y quiz-timing.

## F8 — Charts de reportes

Página de reportes del docente (participación, finalización, notas por curso) con Recharts (charts de shadcn). Requiere definir series y paginación; pruebas unitarias de agregados y E2E. Depende de datos existentes; no toca calificaciones.

## Orden y compuertas

F1 → F2 → F3 → F4 → F5 → F6 → F7 → F8. Cada fase es un incremento verificable; no se inicia la siguiente sin check/build/E2E/AC1 en verde y medición registrada. Sin commits automáticos. Las dependencias se fijan exactas y se verifica su API contra la versión instalada antes de usarla.
