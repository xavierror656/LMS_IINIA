# Proposal
## Why
El usuario no tiene Docker y necesita ver y probar la aplicación con datos mock sin instalar PostgreSQL ni Go.
## What Changes
- Arranque único con Node para Astro y API de demostración local.
- Datos sintéticos, sesiones de demostración y progreso en archivo local separado.
- Mismo frontend, REST y WebSocket; aviso visible y credenciales demo en login.
## Capabilities
### New Capabilities
- `demo-preview`: exploración local sin servicios externos.
### Modified Capabilities
Ninguna; la API Go/GORM y sus migraciones conservan su función para datos reales.
## Impact
Scripts de desarrollo, datos mock, README y aviso condicional. Sin despliegue externo ni cambios al contrato de producción.
