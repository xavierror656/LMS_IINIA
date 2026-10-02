## ADDED Requirements

### Requirement: LMS-003 Cursos y lecciones
El sistema SHALL mostrar cursos inscritos, módulos ordenados y lecciones reading, code y h5p con configuración validada.

Errores, restricciones y fuera de alcance: 404 sin inscripción; listas paginadas; sin marketplace.

#### Scenario: LMS-003 aceptación
- **GIVEN** una inscripción sintética
- **WHEN** abre catálogo y mapa
- **THEN** se muestran título, descripción, tipo y estado

### Requirement: LMS-004 Recompensas
El sistema SHALL completar lecturas declaradas en una transacción, otorgando 25 XP, 1 estrella y 2 gemas una sola vez por usuario y lección. Nivel = 1 + XP/100; vidas 5 no bloqueantes.

Errores, restricciones y fuera de alcance: 409 para tipos sin evaluación validada; sin premios por scores cliente.

#### Scenario: LMS-004 aceptación
- **GIVEN** una lectura sin completar
- **WHEN** llegan dos finalizaciones concurrentes o con claves diferentes
- **THEN** hay un solo reward_event y una sola recompensa persistente

### Requirement: LMS-005 Docente
El sistema SHALL listar estudiantes vinculados y resumen por curso con detalle individual.

Errores, restricciones y fuera de alcance: 403 estudiante, 404 alumno no vinculado; sin CRUD escolar.

#### Scenario: LMS-005 aceptación
- **GIVEN** un docente autenticado
- **WHEN** consulta un alumno vinculado
- **THEN** obtiene totales y lecciones completadas por curso
