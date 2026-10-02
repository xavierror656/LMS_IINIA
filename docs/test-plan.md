# Plan y trazabilidad
| Requisitos / tareas | Pruebas |
|---|---|
| LMS-001/002 T02 | login/logout, Origin, 401, alumno ajeno y docente no vinculado |
| LMS-003 T02 | catálogo, detalle y configuración de tipos |
| LMS-004 T04 | lectura, repetición, claves distintas, concurrencia en PostgreSQL separado, persistencia |
| LMS-005 T07 | lista y detalle de alumnos vinculados |
| LMS-006 T05 | WS real, origen, accepted/stdout/finished, cancelación, desconexión, límites |
| LMS-007 T06 | H5P ausente controlado, intento manipulado sin premio |
| LMS-008 T03 | store, continuidad, recarga, logout, error API y reintento |
| LMS-009 T03 | teclado, tablet, reduced-motion y contraste |
| LMS-010 T01/T08 | migración/seed repetidos, build/check, go test/vet, readiness |

Unitarias: Go runner/config y store/API frontend. Integración: base exclusiva TEST_DATABASE_URL, nunca limpiar base existente. E2E: Playwright con cuentas sintéticas. Medidas: bundles build, tiempos endpoint y concurrencia; registrar entorno y resultados reales en validation.md. Prueba no ejecutada nunca cuenta como aprobada.
