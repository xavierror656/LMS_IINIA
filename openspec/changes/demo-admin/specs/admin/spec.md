## ADDED Requirements
### Requirement: LMS-014 Administración local
La demo SHALL ofrecer una sesión admin con panel de consulta de cuentas sintéticas y cursos y edición de plugins. SHALL dirigir a cada rol a su espacio y restringir endpoints admin al administrador. Errores de API SHALL mostrar recuperación; datos privados SHALL usar no-store. Fuera de alcance: producción Go, CRUD de usuarios/cursos, cambios de roles, paquetes y contraseñas reales.
#### Scenario: Administrar
- **GIVEN** cuenta admin autenticada
- **WHEN** entra en /admin
- **THEN** ve resumen, usuarios, cursos y acceso a editar plugins con guardado local
#### Scenario: Separar roles
- **GIVEN** estudiante o docente
- **WHEN** consulta endpoints admin o intenta editar sus plugins
- **THEN** recibe 403; páginas admin redirigen a su espacio sin revelar datos
#### Scenario: Salir
- **GIVEN** sesión admin activa
- **WHEN** cierra sesión
- **THEN** no puede recuperar el panel sin autenticarse
