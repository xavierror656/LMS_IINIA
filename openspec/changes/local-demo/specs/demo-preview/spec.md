## ADDED Requirements
### Requirement: LMS-011 Vista local con datos mock
El sistema SHALL permitir iniciar la aplicación con Node sin Docker, Go ni PostgreSQL mediante npm run demo. SHALL mostrar aviso de datos ficticios y ofrecer cuentas student y teacher. El progreso SHALL sobrevivir recarga/reinicio en archivo local y no duplicar recompensas. Los datos de producción SHALL permanecer separados.
Errores: puertos ocupados, Node incompatible o archivo corrupto producen diagnóstico sin sobrescribir datos; acceso ajeno se rechaza. Fuera de alcance: backend productivo Node, evaluaciones reales, sincronización y garantías distribuidas.
#### Scenario: Explorar sin infraestructura
- **GIVEN** dependencias frontend instaladas y Node >=22.12
- **WHEN** el usuario ejecuta npm run demo e inicia sesión como luna
- **THEN** ve cursos, mapa y HUD, completa una lectura y conserva la recompensa tras recargar
#### Scenario: Explorar contenido interactivo y docente
- **GIVEN** modo demo activo
- **WHEN** abre programación, H5P o inicia sesión como profe
- **THEN** recibe eventos simulados por WebSocket, ve H5P sin configurar y consulta estudiantes vinculados
#### Scenario: Progreso separado
- **GIVEN** una lectura completada en demo
- **WHEN** repite finalización o reinicia demo
- **THEN** conserva una única recompensa local y no modifica PostgreSQL
