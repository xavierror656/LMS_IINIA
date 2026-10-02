## ADDED Requirements

### Requirement: LMS-015 Autoría y publicación
El docente SHALL crear y editar borradores de lectura y tarea textual en sus cursos y publicar una versión validada.

Errores y restricciones: Rechazar módulos de otro curso, títulos vacíos y publicación incompleta; conservar borrador ante error.

Fuera de alcance: Sin borrado de evidencia ni constructor H5P.

Trazabilidad: T1 → A1.

#### Scenario: Autoría y publicación
- **GIVEN** un docente asignado y una actividad en borrador
- **WHEN** publica y el estudiante inscrito vuelve a cargar el curso
- **THEN** la actividad aparece con la versión publicada; el borrador no era visible

### Requirement: LMS-016 Entregas
El estudiante SHALL guardar y enviar su entrega; el backend SHALL conservar versión y hora canónicas.

Errores y restricciones: Rechazar identidad ajena, adjuntos ajenos y envíos fuera del cierre sin extensión; no perder texto ante fallo.

Fuera de alcance: Solo texto en primer incremento; archivos requieren almacenamiento seguro; sin colaboración grupal implícita.

Trazabilidad: T2 → A2.

#### Scenario: Entregas
- **GIVEN** una tarea publicada y un estudiante inscrito
- **WHEN** envía dos veces la misma entrega con la misma clave
- **THEN** existe un único envío, recuperable tras recargar

### Requirement: LMS-017 Calificación y devolución
El docente autorizado SHALL guardar una nota y comentario y publicarlos explícitamente.

Errores y restricciones: Rechazar nota fuera de rango y escritura con versión obsoleta; edición concurrente no pisa cambios.

Fuera de alcance: No conceder XP arbitrario ni aceptar nota calculada por estudiante.

Trazabilidad: T3 → A3.

#### Scenario: Calificación y devolución
- **GIVEN** una entrega enviada y nota guardada como borrador
- **WHEN** el docente publica la devolución
- **THEN** solo su estudiante ve nota y comentario y se conserva la revisión

### Requirement: LMS-018 Libro de calificaciones
El sistema SHALL consolidar notas por curso y categoría con política explícita de faltantes.

Errores y restricciones: Rechazar pesos inválidos y ciclos; no revelar notas ocultas en totales del estudiante.

Fuera de alcance: Primera agregación media ponderada; otras fórmulas no se declaran equivalentes sin pruebas.

Trazabilidad: T4 → A4.

#### Scenario: Libro de calificaciones
- **GIVEN** ítems 80/100 y 30/50 de pesos 60 y 40
- **WHEN** se calcula el total con ambos incluidos
- **THEN** el total es 72/100 y se distinguen pendientes de ceros

### Requirement: LMS-019 Rúbricas
El docente SHALL definir criterios y niveles y calificar contra una versión fija.

Errores y restricciones: Rechazar niveles inexistentes, criterios sin puntaje y cambios retroactivos silenciosos.

Fuera de alcance: Guías de evaluación y moderación múltiple requieren ampliación de escenarios.

Trazabilidad: T4 → A5.

#### Scenario: Rúbricas
- **GIVEN** una rúbrica publicada y usada en una entrega
- **WHEN** se crea una nueva versión de la rúbrica
- **THEN** la nota anterior conserva criterios y valores originales

### Requirement: LMS-020 Cuestionarios
El sistema SHALL reutilizar preguntas versionadas y evaluar respuestas objetivas en servidor.

Errores y restricciones: Rechazar intentos agotados y preguntas ajenas; reintentos/concurrencia no duplican nota.

Fuera de alcance: Código continúa simulado; ensayos se evalúan manualmente; tipos exactos por cerrar en inventario.

Trazabilidad: T5 → A6.

#### Scenario: Cuestionarios
- **GIVEN** un cuestionario publicado con respuestas correctas privadas
- **WHEN** el estudiante envía respuestas y manipula un score en su petición
- **THEN** la API rechaza el campo indebido y no concede puntuación elegida por cliente

### Requirement: LMS-021 Autorización y auditoría
Toda operación académica SHALL validar rol, curso y objeto y registrar cambios de notas.

Errores y restricciones: Sesión revocada debe fallar; exportación y archivos aplican igual control; error no deja escritura parcial.

Fuera de alcance: El rol admin de la demo no acredita estas capacidades en Go.

Trazabilidad: T1 → A7.

#### Scenario: Autorización y auditoría
- **GIVEN** un docente de otro curso o un estudiante distinto
- **WHEN** intenta consultar o modificar una entrega o nota ajena
- **THEN** se deniega sin revelar contenido ni alterar registros

### Requirement: LMS-022 Experiencia y recuperación
Las pantallas SHALL admitir teclado, tablet y recuperación de errores sin afirmar guardados inexistentes.

Errores y restricciones: No ocultar errores de validación; proteger navegación con cambios pendientes.

Fuera de alcance: Sin modo offline persistente ni datos personales en localStorage.

Trazabilidad: T6 → A8.

#### Scenario: Experiencia y recuperación
- **GIVEN** un formulario con cambios y API caída
- **WHEN** se intenta guardar y luego se recupera el servicio
- **THEN** el error es accesible, el texto se conserva y solo la confirmación del servidor indica guardado

### Requirement: LMS-023 Paridad verificable
El informe SHALL calcular paridad exclusivamente sobre inventario acordado y evidencia ejecutada.

Errores y restricciones: Si falta versión, denominador o evidencia, informar no determinado; nunca anunciar 98 % por semejanza visual.

Fuera de alcance: No extrapolar una selección de funciones a toda la plataforma Moodle.

Trazabilidad: T0 → A9.

#### Scenario: Paridad verificable
- **GIVEN** un inventario congelado con pesos y pruebas
- **WHEN** se solicita el porcentaje de compatibilidad
- **THEN** se informa cobertura exacta, fallos y brechas sin contar pruebas omitidas

