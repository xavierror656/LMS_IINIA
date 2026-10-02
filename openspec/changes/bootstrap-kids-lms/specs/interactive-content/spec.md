## ADDED Requirements

### Requirement: LMS-006 Consola
El sistema SHALL conectar CodeMirror JavaScript/Python mediante WebSocket real a un Runner simulado, identificado como Ejecución simulada.

Errores, restricciones y fuera de alcance: límites, timeout, heartbeat, Origin, autenticación; sin ejecución arbitraria ni reenvío automático.

#### Scenario: LMS-006 aceptación
- **GIVEN** sesión e inscripción válidas
- **WHEN** ejecuta y cancela
- **THEN** recibe accepted, stdout y finished con secuencia y libera recursos

### Requirement: LMS-007 H5P
El sistema SHALL cargar h5p-standalone únicamente en navegador con rutas confiables y registrar resultados mínimos como client_reported sin premios.

Errores, restricciones y fuera de alcance: sin subidas ni editor; paquete real y su licencia necesarios para validar reproducción.

#### Scenario: LMS-007 aceptación
- **GIVEN** contenido ausente o evento manipulado
- **WHEN** abre actividad o reporta resultado
- **THEN** muestra Actividad aún no configurada o guarda intento sin modificar XP
