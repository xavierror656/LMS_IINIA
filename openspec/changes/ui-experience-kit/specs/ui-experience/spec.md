## ADDED Requirements

### Requirement: UI-201 Progreso visible del curso
El mapa del curso SHALL mostrar el avance por módulo con `steps`/`progress` y el estado de cada lección sin cambiar rutas ni textos existentes.

Errores y restricciones: sin lecciones el progreso es 0 sin inventar divisiones; el estado por color no es la única señal (texto e icono acompañan); objetivos ≥44 px.

Fuera de alcance: nuevas métricas o consultas al backend; usa los datos del mapa actual.

Trazabilidad: F1a → accessibility.spec.ts.

#### Scenario: Módulo a medias
- **GIVEN** un módulo con dos lecciones completadas y una pendiente
- **WHEN** el estudiante abre el curso
- **THEN** el módulo muestra el avance 2/3 y la lección pendiente se distingue por texto, icono y color

### Requirement: UI-202 Contenido largo plegable
Las instrucciones, la rúbrica y el historial SHALL poder plegarse con `collapse` conservando el acceso al contenido y el foco por teclado.

Errores y restricciones: el contenido no se elimina, solo se oculta; el resumen tiene nombre accesible; funciona con movimiento reducido.

Fuera de alcance: plegar formularios en edición.

Trazabilidad: F1b → E2E rubric/attempts.

#### Scenario: Rúbrica plegada
- **GIVEN** una tarea con rúbrica publicada
- **WHEN** el estudiante abre la sección
- **THEN** ve el resumen y puede expandir criterios con teclado sin perder el foco

### Requirement: UI-203 Estados vacíos y de carga
Los listados SHALL mostrar un estado vacío con icono, explicación y acción, y las islas SHALL mostrar `skeleton` mientras cargan.

Errores y restricciones: nunca mostrar datos inventados ni un vacío ambiguo sin siguiente paso.

Fuera de alcance: ilustraciones nuevas de marca.

Trazabilidad: F1c/F1d → E2E learning y smoke.

#### Scenario: Curso sin grupos
- **GIVEN** un curso sin grupos creados
- **WHEN** el docente abre Grupos
- **THEN** ve un estado vacío claro con la acción Crear grupo y no una tabla en blanco

### Requirement: UI-204 Navegación uniforme
Las subpáginas docente SHALL mostrar migas y sus listados paginación con `join`, sin romper deep links.

Errores y restricciones: las migas reflejan la jerarquía real; la paginación conserva filtros y no pierde el foco.

Fuera de alcance: rediseñar la navegación principal.

Trazabilidad: F1e → E2E gradebook/groups.

#### Scenario: Volver desde el libro
- **GIVEN** el libro de calificaciones abierto
- **WHEN** el docente usa las migas
- **THEN** vuelve al curso y la página de estudiantes conserva su estado

### Requirement: UI-205 Acciones sin recarga en islas shadcn
Crear grupo, editar categoría y calificación rápida SHALL ocurrir en Dialog/Sheet de shadcn cuando la fase lo habilite, sin recargar la página y con foco atrapado.

Errores y restricciones: el error conserva lo escrito y se anuncia; Escape cancela sin enviar; el diálogo se carga bajo demanda; los roles y textos que usan las pruebas se conservan o se actualizan con evidencia.

Fuera de alcance: convertir a isla los formularios de entrega del alumno.

Trazabilidad: F3 → E2E grupos/categorías/libro.

#### Scenario: Cancelar creación
- **GIVEN** el diálogo de crear grupo abierto
- **WHEN** el docente pulsa Escape
- **THEN** no se envía nada y la lista queda igual

### Requirement: UI-206 Selección en listas largas
La elección de preguntas del banco y de estudiantes SHALL ofrecer búsqueda y selección con Combobox/Command cuando la fase lo habilite.

Errores y restricciones: la búsqueda no cambia el payload ni las versiones fijadas; la selección conserva las reglas actuales (una versión por pregunta, un estudiante por grupo); objetivos ≥44 px.

Fuera de alcance: reemplazar la paginación del banco.

Trazabilidad: F5 → E2E quizzes/groups.

#### Scenario: Elegir pregunta por nombre
- **GIVEN** un banco con muchas preguntas
- **WHEN** el docente escribe parte del nombre
- **THEN** la lista se filtra y la selección fija la versión mostrada

### Requirement: UI-207 Feedback sin recargar (Sonner)
Los guardados de formularios compartidos SHALL confirmar con Sonner y manejar errores conservando el estado cuando la fase lo habilite.

Errores y restricciones: un fallo de red no pierde lo escrito y muestra recuperación; el servidor sigue siendo la única confirmación de guardado.

Fuera de alcance: notificaciones push o correo.

Trazabilidad: F6 → E2E completa y AC1.

#### Scenario: Error de red al guardar
- **GIVEN** un formulario con cambios y la API caída
- **WHEN** se intenta guardar
- **THEN** aparece el error en un toast, el texto permanece y no se afirma guardado

### Requirement: UI-208 Fechas y reportes
El sistema SHALL usar Calendar para fechas manteniendo UTC y SHALL ofrecer reportes del docente con charts cuando esas fases se habiliten.

Errores y restricciones: las fechas siguen viajando en UTC y el backend sigue validando; los charts no inventan datos y respetan la autorización del docente por curso.

Fuera de alcance: analítica empresarial o exportaciones nuevas.

Trazabilidad: F7/F8 → E2E schedule/quiz-timing y pruebas de agregados.

#### Scenario: Reporte autorizado
- **GIVEN** un docente con cursos asignados
- **WHEN** abre reportes
- **THEN** solo ve sus cursos y las series coinciden con el libro y el progreso persistidos
