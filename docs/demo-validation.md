# Validación del modo demo (LMS-011)

El cambio local-demo permite explorar AulaQuest sin Docker, Go ni PostgreSQL. Se conserva el backend real y no se altera su contrato REST. La API Node de demostración está en frontend/scripts/demo/ y solo escucha en loopback; no se inicia con NODE_ENV=production.

## Datos y acceso

Dos cursos, tres módulos y siete lecciones. Usuarios luna/sol (student) y profe (teacher); contraseña pública ficticia aulaquest-demo. Luna comienza con cero XP y Sol tiene dos lecturas de muestra. Profe solo consulta a Luna, manteniendo el caso de alumno no vinculado.

Ejecutar desde la raíz con Node >=22.12:

```bash
npm --prefix frontend ci
npm run demo
```

URL: http://localhost:4321/login (también se admite 127.0.0.1). Ctrl+C cierra API y Astro. El puerto mock se elige automáticamente; solo el frontend usa 4321. Si ese puerto está ocupado, no se selecciona otro silenciosamente.

## Persistencia

frontend/.demo/state.json conserva progreso ficticio y hasta 100 intentos H5P; no contiene contraseñas ni tokens de sesión. Escritura a archivo temporal y rename, actualización de memoria solo tras guardar. Idempotencia de lectura dentro de un proceso Node. Estas garantías de demostración no sustituyen transacciones PostgreSQL ni sirven para múltiples instancias. Archivo ignorado por Git y Docker.

Las sesiones se mantienen en memoria y se revocan al reiniciar. Cookie HttpOnly/SameSite Lax, Origin exacto y autorización de rol y vínculo. Datos corruptos provocan error sin sobrescribirlos. Para empezar de nuevo, parar demo y renombrar state.json; nunca se borra ni modifica la DB real.

## Pruebas ejecutadas

- OpenSpec validate local-demo --strict: válido.
- Build normal sin modo demo: compilación SSR y prerender aprobados; se mantiene el aviso conocido de CodeMirror >500 kB.
- Astro check: 25 archivos, 0 errores, 0 warnings y 0 hints.
- Node test API mock: **7/7 aprobadas**. Sesiones, catálogo/mapa, roles, alumno ajeno, Origin, logout, 8 finalizaciones concurrentes con premio único, reinicio con progreso conservado, H5P sin recompensas, payload de premios rechazado, WebSocket real/stdout/cancelación/nueva ejecución, archivo corrupto conservado, rechazo de producción y los dos orígenes loopback exactos.
- Playwright sobre npm run demo: **5/5 aprobadas, 29.0 s**. Se verificó aviso demo y credenciales, login, lectura, HUD persistente, recarga/logout, consola simulada/cancelación, H5P ausente, panel docente, teclado/tablet/reduced-motion y error/recuperación de API.
- Entorno: Node 22.23.3 en Linux, Chromium local; ningún proceso Go/PostgreSQL/Docker activo durante esta validación.

Comandos:

```bash
npm --prefix frontend run test:demo
E2E_DEMO=1 E2E_STUDENT_PASSWORD=aulaquest-demo E2E_TEACHER_PASSWORD=aulaquest-demo npm --prefix frontend run test:e2e
```

Las E2E completaron la primera lectura de Luna y dejaron ese avance visible en la demostración de esta sesión. Para repetir sobre otro perfil, usar datos iniciales o conservar solo esa lectura; la suite existente espera 25 XP.

## Límites visibles

La UI muestra «Modo demo · Datos ficticios». El código no se ejecuta y la consola lo indica; H5P real permanece sin configurar. Esta API no implementa una alternativa productiva a Go/GORM y sus migraciones. Las cuentas públicas solo deben usarse para explorar localmente.
