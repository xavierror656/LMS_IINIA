## ADDED Requirements

### Requirement: UI-101 Confirmación destructiva con AlertDialog
Las acciones destructivas SHALL pedir confirmación con un `AlertDialog` de shadcn, en español, con título, descripción, botón destructivo y foco atrapado.

Errores y restricciones: Cancelar, cerrar con Escape o pulsar fuera resuelve `false` y no ejecuta la acción; la variante destructiva usa el color de peligro del tema; el diálogo tiene `AlertDialogTitle` y `AlertDialogDescription` accesibles.

Fuera de alcance: Guards de cambios sin guardar y `beforeunload`, que siguen nativos.

Trazabilidad: S1–S4 → docs/shadcn-validation.md.

#### Scenario: Cancelar un borrado
- **GIVEN** una categoría en el libro de calificaciones
- **WHEN** el docente pulsa Eliminar y cancela en el diálogo
- **THEN** la categoría sigue existiendo y no se envió ninguna petición

#### Scenario: Confirmar un borrado
- **GIVEN** un grupo de curso
- **WHEN** el docente pulsa Eliminar y confirma en el diálogo
- **THEN** se llama a la API con DELETE y el grupo desaparece de la lista

### Requirement: UI-102 Respaldo sin host
Si la isla de confirmación no está montada, la acción SHALL usar `window.confirm` nativo para no quedar bloqueada.

Errores y restricciones: el respaldo no cambia la semántica de la acción ni omite la confirmación.

Fuera de alcance: Mensajes personalizados en el respaldo.

Trazabilidad: S3.

#### Scenario: Página sin isla
- **GIVEN** una página sin la isla de confirmación cargada
- **WHEN** se ejecuta una acción destructiva
- **THEN** aparece el diálogo nativo y la acción respeta la respuesta

### Requirement: UI-103 Sin regresión
La migración de confirmaciones SHALL conservar autorización, estados de ocupado y accesibilidad existentes.

Errores y restricciones: no se debilita ninguna validación del servidor; las suites de adjuntos, grupos, categorías, intentos y accesibilidad pasan; el JS añadido se mide y documenta.

Fuera de alcance: Sustituir Sonner o el selector de fechas.

Trazabilidad: S5–S6.

#### Scenario: Borrado de adjunto propio
- **GIVEN** una entrega con un archivo
- **WHEN** el estudiante confirma quitarlo
- **THEN** el archivo se borra en el servidor y desaparece de la lista sin afectar a otros usuarios
