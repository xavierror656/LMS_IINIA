# Tareas por dependencia

- [x] P0 Inspeccionar estructura, panel admin, modelos, rutas y limitaciones existentes.
- [x] P1 Consultar documentación oficial Moodle y redactar comparación candidata.
- [x] P2 Redactar propuesta, diseño y escenarios LMS-015..023.
- [ ] T0 Cerrar versión, inventario atómico del alcance Moodle acordado, pesos y brechas del 98 %; validar OpenSpec con CLI disponible. LMS-023 / A9.
- [ ] T1 Especificar OpenAPI del primer incremento; migraciones course_staff y versiones; autorización Go y autoría SSR. LMS-015/021 / A1/A7.
- [ ] T2 Entregas textuales, transiciones, idempotencia y persistencia Go/PostgreSQL. Depende T1. LMS-016 / A2.
- [ ] T3 Notas, devolución y auditoría transaccional con concurrencia. Depende T2. LMS-017 / A3/A7.
- [ ] T4 Categorías, agregación, rúbricas y archivos privados seguros; contratos adicionales. Depende T3. LMS-018/019 / A4/A5/A7.
- [ ] T5 Banco versionado, tipos de preguntas acordados, cuestionarios y evaluación en servidor. Depende T1/T3. LMS-020 / A6.
- [ ] T6 E2E, accesibilidad, recuperación, CSV e integración con flujos existentes; medir rendimiento. Depende T4/T5. LMS-022 / A8.
- [ ] T7 Completar funciones avanzadas del inventario; ejecutar auditoría contra Moodle y calcular cobertura sin redondeo. Depende T0/T6. LMS-023 / A9.

No hay tareas de implementación marcadas como terminadas. La solicitud inmediata es proponer el módulo; la base existente se conserva. El usuario confirmó primero toda la plataforma y después excluyó foros, wikis, extensiones, respaldos y operación. Ya no existe bloqueo por la elección módulo/plataforma. Sigue pendiente cerrar el inventario y demostrar cobertura; no se declara el objetivo logrado.

- [x] P3 Desglosar 43 capacidades candidatas con criterios y fuentes en capabilities.csv; sin fijar pesos ni declarar cobertura.

- [x] P4 Registrar alcance inicial: toda la plataforma (revisado por P5); ampliar propuesta y dependencias en platform-scope.md.
- [ ] T8 Desglosar los dominios de plataforma en capacidades atómicas y contratos; no limitar T7 a las primeras 43 filas.
- [ ] T9 Implementar y verificar incrementos de plataforma PL1..PL7, conservando trazabilidad y brechas.

- [x] P5 Aplicar exclusiones explícitas: foros, wikis, gestor de extensiones, respaldos y módulo de operación; retirar PL8 y ajustar el denominador declarado.
