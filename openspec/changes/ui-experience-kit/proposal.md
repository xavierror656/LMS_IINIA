# Cambio: kit de experiencia UI por fases

## Why

La interfaz ya tiene un sistema visual consistente (daisyUI + shadcn en islas), pero le faltan patrones de experiencia que hoy se notan: el mapa del curso no muestra pasos ni avance por módulo, las instrucciones y rúbricas largas obligan a mucho scroll en tablet, los estados vacíos y de carga son textos planos, la navegación docente no tiene migas ni paginación uniforme, el menú de usuario es un botón suelto, crear grupos/categorías/calificar recarga la página entera, y elegir preguntas o estudiantes en listas largas es tedioso. El usuario pidió incorporar todos estos componentes, sin límite de costo.

## What Changes

- Componentes daisyUI sin JS nuevo: `steps`/`progress`, `collapse`, `skeleton`, `breadcrumbs`, paginación con `join`, tooltips de texto, estados vacíos consistentes.
- Menú de usuario con DropdownMenu (shadcn) dentro de isla.
- Dialog/Sheet de shadcn para crear grupo, editar categoría y calificación rápida sin recargar.
- Tabs de curso docente con enlaces (daisyUI), conservando rutas y deep links.
- Combobox/Command de shadcn para elegir pregunta del banco y estudiante en listas largas.
- Sonner y guardado sin recarga en formularios clave.
- Date picker (Calendar) en fechas de tarea/cuestionario.
- Charts para reportes del docente.

## Fuera de alcance

- Chat/foros entre menores, rankings públicos, publicidad o recompensas compradas.
- shadcn Table o Tabs como componentes React en páginas SSR: se usan los equivalentes daisyUI.
- Cambiar el contrato UTC de fechas o la autoridad del servidor en calificaciones.

## Impact

Solo frontend e islas React; contratos REST/WS y backend se conservan salvo que una fase requiera un endpoint nuevo (se especificará en su incremento). Cada fase se valida con check/build, E2E del área, auditoría AC1 y medición de JS/CSS; las islas nuevas se cargan bajo demanda como el ConfirmHost. Dependencias nuevas solo donde el plan las declara (DropdownMenu, Dialog/Sheet, Combobox, Sonner, Calendar, charts), fijadas exactas y verificadas contra su documentación vigente.
