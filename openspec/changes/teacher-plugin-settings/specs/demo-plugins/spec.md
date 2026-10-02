## ADDED Requirements
### Requirement: LMS-013 Edición docente de extensiones demo
La demo SHALL permitir al docente editar nombre e instrucciones de code y h5p y lenguaje inicial de code. SHALL conservar ajustes tras reinicio y mostrarlos al abrir lecciones. SHALL rechazar estudiantes, orígenes ajenos, campos desconocidos y textos vacíos o excesivos. Fuera de alcance: instalación de paquetes, edición de código del plugin, contenido H5P, API Go y configuración de producción.
#### Scenario: Guardar y consultar
- **GIVEN** un docente autenticado en demo
- **WHEN** guarda un nombre de hasta 60 caracteres, instrucciones de hasta 300 y lenguaje válido
- **THEN** los cambios se muestran en lecciones y persisten tras reiniciar sin alterar progreso
#### Scenario: Acceso inválido
- **GIVEN** un estudiante o una solicitud inválida
- **WHEN** intenta modificar ajustes
- **THEN** recibe error y los ajustes anteriores permanecen
#### Scenario: Actividad pendiente
- **GIVEN** H5P sin paquete revisado
- **WHEN** se cambia su nombre
- **THEN** continúa mostrando Actividad aún no configurada
