## ADDED Requirements

### Requirement: UI-001 Tema AulaQuest sobre daisyUI
El frontend SHALL definir un único tema daisyUI llamado `aulaquest` con los
colores y radios de AulaQuest, cargado en CSS-first sin `tailwind.config.mjs`.

Errores y restricciones: Rechazar cualquier tema por defecto de daisyUI como
predeterminado; no activar modo oscuro. Un color que no alcance contraste AA
contra su `*-content` no se acepta: se ajusta el valor y se recalcula.

Fuera de alcance: Cambiar la paleta acordada o introducir degradados ajenos a la
identidad.

Trazabilidad: F1 → docs/contrast.json.

#### Scenario: Tema cargado
- **GIVEN** el proyecto compilando con Tailwind 4
- **WHEN** se inspecciona el CSS emitido
- **THEN** existe el tema `aulaquest` como predeterminado y no se emiten los
  temas incluidos de daisyUI

### Requirement: UI-002 Componentes daisyUI en toda la interfaz
Toda pantalla SHALL usar componentes daisyUI para botones, campos, tarjetas,
alertas, insignias, tablas y enlaces, con estados de foco, hover y activo.

Errores y restricciones: Conservar los roles, textos y atributos `data-*` que
usan las pruebas; objetivos táctiles ≥44×44 px; el botón principal mantiene el
efecto presionable sin cambiar la altura del layout. No reimplementar con CSS
propio un componente que daisyUI ya ofrece.

Fuera de alcance: Reemplazar CodeMirror, H5P ni la lógica de las islas React;
solo sus contenedores y estados visuales.

Trazabilidad: F2/F3/F4 → frontend/tests/accessibility.spec.ts.

#### Scenario: Botón y campo migrados
- **GIVEN** una página migrada con un formulario
- **WHEN** se navega con teclado y se enfoca el botón y el campo
- **THEN** ambos muestran foco visible, el botón responde como `btn` y el campo
  conserva etiqueta accesible y alto ≥44 px

### Requirement: UI-003 Un solo sistema
El proyecto SHALL eliminar las clases y el CSS heredados que daisyUI cubra,
conservando solo tema, fuentes, layout global y composiciones documentadas.

Errores y restricciones: Ninguna página puede depender de selectores retirados;
`global.css` no puede contener reimplementaciones de botones, campos, alertas ni
tablas. Las composiciones propias usan variables del tema.

Fuera de alcance: Reescribir utilidades de layout de Tailwind.

Trazabilidad: F5 → búsqueda de clases huérfanas.

#### Scenario: Limpieza verificable
- **GIVEN** la migración terminada
- **WHEN** se busca en `frontend/src` una clase heredada retirada
- **THEN** no hay referencias y el build SSR sigue pasando

### Requirement: UI-004 Accesibilidad y movimiento
La interfaz migrada SHALL conservar la auditoría AC1 en cero hallazgos y respetar
`prefers-reduced-motion`.

Errores y restricciones: Encabezados sin saltos, etiquetas asociadas, foco
visible, contraste AA y sin animaciones infinitas con movimiento reducido.

Fuera de alcance: Auditoría con lectores de pantalla reales, ya fuera del
alcance registrado.

Trazabilidad: F2a/F3a/F4a → docs/accessibility-validation.md.

#### Scenario: Auditoría tras migrar
- **GIVEN** la aplicación migrada en ejecución
- **WHEN** se ejecuta la suite de accesibilidad sobre las páginas auditadas
- **THEN** no aparecen hallazgos de etiquetas, encabezados, nombres, objetivos ni
  movimiento

### Requirement: UI-005 Sin regresión funcional
Los flujos existentes SHALL seguir funcionando sin cambios de contrato: autoría,
publicación, entregas, calificación, categorías, cuestionarios, grupos y
progreso.

Errores y restricciones: Las suites E2E existentes pasan con sus aserciones de
rol y texto; no se debilita ningún límite de seguridad por comodidad de prueba.

Fuera de alcance: Añadir funciones nuevas.

Trazabilidad: F2a/F3a/F4a/F5a → frontend/tests.

#### Scenario: E2E de regresión
- **GIVEN** el entorno sintético con Go/PostgreSQL
- **WHEN** se ejecutan las suites académicas y de aprendizaje
- **THEN** todas pasan sin cambios de comportamiento observable

### Requirement: UI-006 Rendimiento medido
El cambio SHALL registrar el tamaño del CSS emitido antes y después y no
aumentarlo sin justificarlo.

Errores y restricciones: No añadir JavaScript de cliente; daisyUI es solo CSS.
Si el CSS crece, se documenta la causa y se recorta con `themes: false` y
componentes usados.

Fuera de alcance: Optimizaciones ajenas a la migración.

Trazabilidad: F5 → docs/daisyui-validation.md.

#### Scenario: Medición de CSS
- **GIVEN** el build de producción antes y después
- **WHEN** se comparan los tamaños de los archivos CSS emitidos
- **THEN** la diferencia está documentada con método y números reales
