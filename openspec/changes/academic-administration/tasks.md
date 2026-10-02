# Tareas por dependencia

- [x] P0 Inspeccionar estructura, panel admin, modelos, rutas y limitaciones existentes.
- [x] P1 Consultar documentación oficial Moodle y redactar comparación candidata.
- [x] P2 Redactar propuesta, diseño y escenarios LMS-015..023.
- [ ] T0 Cerrar versión, inventario atómico del alcance Moodle acordado, pesos y brechas del 98 %. LMS-023 / A9.
- [x] T1 Especificar OpenAPI del primer incremento; migraciones course_staff y versiones; autorización Go y autoría SSR. LMS-015/021 / A1/A7.
- [x] T2 Entregas textuales, transiciones, idempotencia y persistencia Go/PostgreSQL. Depende T1. LMS-016 / A2.
- [x] T3 Notas, devolución y auditoría transaccional con concurrencia. Depende T2. LMS-017 / A3/A7.
- [ ] T4 Categorías, agregación, rúbricas y archivos privados seguros; contratos adicionales. Depende T3. LMS-018/019 / A4/A5/A7.
- [ ] T5 Banco versionado, tipos de preguntas acordados, cuestionarios y evaluación en servidor. Depende T1/T3. LMS-020 / A6.
- [ ] T6 E2E, accesibilidad, recuperación, CSV e integración con flujos existentes; medir rendimiento. Depende T4/T5. LMS-022 / A8.
- [ ] T7 Completar funciones avanzadas del inventario; ejecutar auditoría contra Moodle y calcular cobertura sin redondeo. Depende T0/T6. LMS-023 / A9.

El usuario autorizó continuar con implementación. T1–T3 tienen código y pruebas de integración Go/PostgreSQL del incremento 1; los escenarios más amplios (reentregas, adjuntos, rúbricas y libro agregado) siguen pendientes en siguientes incrementos. La base existente se conserva. El usuario confirmó primero toda la plataforma y después excluyó foros, wikis, extensiones, respaldos y operación. Ya no existe bloqueo por la elección módulo/plataforma. Sigue pendiente cerrar el inventario y demostrar cobertura; no se declara el objetivo logrado.

- [x] P3 Desglosar 43 capacidades candidatas con criterios y fuentes en capabilities.csv; sin fijar pesos ni declarar cobertura.

- [x] P4 Registrar alcance inicial: toda la plataforma (revisado por P5); ampliar propuesta y dependencias en platform-scope.md.
- [ ] T8 Desglosar los dominios de plataforma en capacidades atómicas y contratos; no limitar T7 a las primeras 43 filas.
- [ ] T9 Implementar y verificar incrementos de plataforma PL1..PL7, conservando trazabilidad y brechas.

- [x] P5 Aplicar exclusiones explícitas: foros, wikis, gestor de extensiones, respaldos y módulo de operación; retirar PL8 y ajustar el denominador declarado.

- [x] I0 Especificación concreta previa al código: increment-1.md.
- [x] I1 Validar academic-administration con OpenSpec 1.14.0 --strict --no-interactive.
- [x] I2 Completar validación E2E, registrar resultados finales y capturas de T1–T3.

- [x] T2a Añadir apertura, vencimiento, cierre y prórrogas (MDL-003..005); DT1–DT5 verificados internamente, comparación Moodle pendiente.
- [ ] T2b Añadir reapertura y límite configurable de intentos (MDL-009..010).
- [ ] T2c Añadir entregas grupales y notas individuales de grupo (MDL-011..012).
- [ ] T3a Añadir anotación, anonimato y corrección por varios evaluadores (MDL-017..022).

- [ ] TH5P Autoría y biblioteca H5P con contenido real, aislamiento y atribución (MDL-042..043); fuera del incremento 1.

- [x] T4a Libro de calificaciones: contrato, consulta coherente y acotada, matriz SSR y enlaces a entrega exacta (GB1..GB5 / LMS-018/021/022).
- [x] T4b Validar libro con PostgreSQL, cálculo unitario, navegador y registrar evidencia; no marcar T4 completo por este subconjunto.

- [x] T4c Implementar rúbricas versionadas, corrección por niveles y pesos publicados (RB1..RB5 / LMS-018/019/021).
- [x] T4d Validar migración, cálculo, aislamiento, snapshots, promedio ponderado e interfaz E2E (LMS-022).

- [ ] TSEC1 Revisar límites por usuario y proxy para concurrencia real de aula; el límite global por IP produjo 429 al encadenar suites. No debilitar protección para pasar pruebas.
