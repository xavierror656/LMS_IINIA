## ADDED Requirements

### Requirement: LMS-008 HUD
El sistema SHALL persistir visualmente HUD React entre navegaciones y recuperar progreso autorizado al hidratar, navegar y completar.

Errores, restricciones y fuera de alcance: API caída muestra error y reintento, sin progreso inventado; sin localStorage.

#### Scenario: LMS-008 aceptación
- **GIVEN** un estudiante con XP guardado
- **WHEN** recarga o cierra sesión
- **THEN** conserva XP del backend o limpia todo estado personal

### Requirement: LMS-009 Accesibilidad
El sistema SHALL usar español, tokens, Lexend local, foco visible, controles >=44px y reduced-motion.

Errores, restricciones y fuera de alcance: sin animación constante ni trackers.

#### Scenario: LMS-009 aceptación
- **GIVEN** tablet y teclado
- **WHEN** navega por cursos y lecciones
- **THEN** puede operar sin ratón y entender estados sin color

### Requirement: LMS-010 Operación
El sistema SHALL ofrecer migraciones SQL versionadas, seed idempotente, health/readiness, apagado ordenado, logs sin secretos y configuración local reproducible.

Errores, restricciones y fuera de alcance: fallo explícito por configuración inválida; sin promesa de producción ni runner aislado real.

#### Scenario: LMS-010 aceptación
- **GIVEN** PostgreSQL separado y variables válidas
- **WHEN** migra dos veces y ejecuta seed dos veces
- **THEN** no duplica datos y readiness confirma DB
