## ADDED Requirements

### Requirement: LMS-001 Sesiones
El sistema SHALL autenticar con cookie HttpOnly, SameSite Lax, expiración de 12 horas y revocación. Contraseñas bcrypt; Origin exacto en mutaciones, incluyendo login.

Errores, restricciones y fuera de alcance: 401 genérico, 403 para Origin incorrecto; sin registro público.

#### Scenario: LMS-001 aceptación
- **GIVEN** credenciales válidas
- **WHEN** inicia sesión y la cierra
- **THEN** la sesión se crea y después devuelve 401

### Requirement: LMS-002 Autorización
El sistema SHALL derivar identidad de sesión y comprobar inscripción o vínculo docente por objeto.

Errores, restricciones y fuera de alcance: sin administración escolar ni cambio de roles desde cliente.

#### Scenario: LMS-002 aceptación
- **GIVEN** dos estudiantes sin vínculo común
- **WHEN** se consulta un objeto ajeno
- **THEN** responde 404 o 403 sin datos personales
