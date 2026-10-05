# Tareas por dependencia

- [x] P0 Inspeccionar estructura, panel admin, modelos, rutas y limitaciones existentes.
- [x] P1 Consultar documentación oficial Moodle y redactar comparación candidata.
- [x] P2 Redactar propuesta, diseño y escenarios LMS-015..023.
- [ ] T0 Cerrar versión, inventario atómico del alcance Moodle acordado, pesos y brechas del 98 %. LMS-023 / A9.
- [x] T1 Especificar OpenAPI del primer incremento; migraciones course_staff y versiones; autorización Go y autoría SSR. LMS-015/021 / A1/A7.
- [x] T2 Entregas textuales, transiciones, idempotencia y persistencia Go/PostgreSQL. Depende T1. LMS-016 / A2.
- [x] T3 Notas, devolución y auditoría transaccional con concurrencia. Depende T2. LMS-017 / A3/A7.
- [x] T4 Categorías, agregación, rúbricas y archivos privados seguros; contratos adicionales. Depende T3. LMS-018/019 / A4/A5/A7. T4a–T4h cerrados: categorías planas con peso, agregación por categoría, política de faltantes y vista del alumno verificadas con PostgreSQL 18.6 real, navegador y migración de retroceso.
- [ ] T5 Banco versionado, tipos de preguntas acordados, cuestionarios y evaluación en servidor. Depende T1/T3. LMS-020 / A6.
- [x] T5a Banco privado versionado y evaluación docente en Go para selección única/múltiple, verdadero-falso y respuesta corta. QB1–QB6 internos; no acredita cuestionarios del alumno ni paridad Moodle.
- [x] T5b Cuestionarios publicados con versiones de preguntas fijas, resolución e intentos del alumno, evaluación canónica, revisión por intento y cuatro políticas de nota en libro. QT1–QT6 verificados internamente; equivalencia Moodle pendiente.
- [x] T5c Fechas de cuestionario, temporizador del servidor, revisión tras cierre y excepciones de tiempo (QT7–QT13) verificados con PostgreSQL 18.6 real y navegador; comparación Moodle pendiente. Depende T5b.
- [x] T6-1 Exportación CSV del libro con el mismo cálculo que la tabla, medición real de rendimiento y consultas acotadas (EX1–EX3, EZ1, RP1–RP2). Verificado con PostgreSQL real; el libro pasó de 549 ms a 46 ms.
- [x] T6-2 Auditoría de accesibilidad (AC1): etiquetas, orden de encabezados, foco visible, tamaños táctiles y contraste revisados con criterios comprobables. Ejecutada con Playwright sobre diez páginas; dos correcciones de tamaño táctil y cero hallazgos finales. Evidencia: docs/accessibility-validation.md; suite frontend/tests/accessibility.spec.ts.
- [ ] T7 Completar funciones avanzadas del inventario; ejecutar auditoría contra Moodle y calcular cobertura sin redondeo. Depende T0/T6. LMS-023 / A9.

El usuario autorizó continuar con implementación. T1–T3 tienen código y pruebas Go/PostgreSQL del incremento 1; los incrementos 2–6 añaden libro, rúbricas/pesos, fechas/prórrogas, adjuntos privados y reentregas con evidencia interna. Los incrementos 7–9 añaden banco de preguntas, cuestionarios publicados y fechas/temporizador/revisión del cuestionario, con pruebas de integración y navegador ejecutadas contra PostgreSQL 18.6 real. El incremento 10 (TSEC1) reparte los límites por usuario y por dirección real con proxy confiable, verificado encadenando las suites intensivas sin 429. El incremento 11 añade grupos de curso (T2c-1) y descubre que el manejador de errores descartaba los mensajes específicos, ya corregido. El incremento 12 añade la entrega grupal compartida con adjuntos por cargador y libro por miembro, y el 13 las notas individuales por miembro con publicación selectiva. El incremento 17 cierra T4 con categorías planas, agregación por categoría, política de faltantes y vista «Mis notas» del alumno, verificadas con PostgreSQL 18.6 real, navegador y prueba de retroceso de la migración 015. La auditoría AC1 (T6-2) se ejecutó después con Playwright sobre diez páginas: dos correcciones de tamaño táctil y cero hallazgos finales. La validación estricta del cambio con OpenSpec 1.14.0 pasó. Las tareas desglosadas distinguen lo implementado de las ampliaciones pendientes. La base existente se conserva. El usuario confirmó primero toda la plataforma y después excluyó foros, wikis, extensiones, respaldos y operación. Ya no existe bloqueo por la elección módulo/plataforma. Sigue pendiente cerrar el inventario y demostrar cobertura; no se declara el objetivo logrado.

- [x] P3 Desglosar 43 capacidades candidatas con criterios y fuentes en capabilities.csv; sin fijar pesos ni declarar cobertura.

- [x] P4 Registrar alcance inicial: toda la plataforma (revisado por P5); ampliar propuesta y dependencias en platform-scope.md.
- [ ] T8 Desglosar los dominios de plataforma en capacidades atómicas y contratos; no limitar T7 a las primeras 43 filas.
- [ ] T9 Implementar y verificar incrementos de plataforma PL1..PL7, conservando trazabilidad y brechas.

- [x] P5 Aplicar exclusiones explícitas: foros, wikis, gestor de extensiones, respaldos y módulo de operación; retirar PL8 y ajustar el denominador declarado.

- [x] I0 Especificación concreta previa al código: increment-1.md.
- [x] I1 Validar academic-administration con OpenSpec 1.14.0 --strict --no-interactive.
- [x] I2 Completar validación E2E, registrar resultados finales y capturas de T1–T3.
- [x] I3 Revalidar academic-administration con OpenSpec --strict tras el incremento 17: CLI 1.14.0 (@fission-ai/openspec) instalada en /tmp y validación estricta aprobada.

- [x] T2a Añadir apertura, vencimiento, cierre y prórrogas (MDL-003..005); DT1–DT5 verificados internamente, comparación Moodle pendiente.
- [x] T2b Añadir reapertura y límite configurable de intentos (MDL-009..010); AT1–AT6 verificados internamente, historial y libro coherentes. Comparación Moodle pendiente.
- [x] T2c-1 Grupos de curso y pertenencia, con un estudiante por grupo y curso (CG1–CG7 / MDL-011). Verificado con PostgreSQL real y navegador; no cambia entregas ni notas.
- [x] T2c-2 Entrega grupal compartida y libro por miembro (GS1–GS6 / MDL-011). Verificado con PostgreSQL real y navegador; la nota sigue siendo una por entrega.
- [x] T2c-3 Notas individuales por miembro con publicación selectiva (GI1–GI6 / MDL-012). Verificado con PostgreSQL real y navegador.
- [ ] T3a Añadir anotación, anonimato y corrección por varios evaluadores (MDL-017..022).

- [ ] TH5P Autoría y biblioteca H5P con contenido real, aislamiento y atribución (MDL-042..043); fuera del incremento 1.

- [x] T4a Libro de calificaciones: contrato, consulta coherente y acotada, matriz SSR y enlaces a entrega exacta (GB1..GB5 / LMS-018/021/022).
- [x] T4b Validar libro con PostgreSQL, cálculo unitario, navegador y registrar evidencia; no marcar T4 completo por este subconjunto.

- [x] T4c Implementar rúbricas versionadas, corrección por niveles y pesos publicados (RB1..RB5 / LMS-018/019/021).
- [x] T4d Validar migración, cálculo, aislamiento, snapshots, promedio ponderado e interfaz E2E (LMS-022).

- [x] TSEC1 Límites por usuario y por dirección real, con proxy confiable, verificados con PostgreSQL real y suites de navegador encadenadas sin 429 (RL1–RL7). Queda pendiente el almacén compartido entre réplicas y la ejecución de Nginx/Docker.

- [x] T4e Adjuntos privados de borradores/envíos, normalización, cuotas y descarga autorizada (FL1–FL5 / LMS-016/019/021).
- [x] T4f Validar archivos, límites HTTP/proxy, concurrencia, permisos y recorrido E2E (LMS-022).

- [x] T4g-1 PDF con análisis estructural real, archivos de instrucciones congelados por publicación y retención con informe previo (FA1–FA3, FI1–FI3, FR1–FR3). Verificado con PostgreSQL real y navegador; Nginx y Docker no pudieron ejecutarse en este entorno.
- [x] T4g-2 Adjuntos de devolución por estudiante, colgados de la nota y visibles solo al publicarla (FD1–FD5). Verificado con PostgreSQL real y navegador.

- [x] T4h Categorías de calificación planas, agregación por categoría, política de faltantes y vista «Mis notas» del alumno (GC1–GC8 / LMS-018 / A4); implementadas y verificadas con PostgreSQL 18.6 real, navegador, migración de retroceso y regresión. Evidencia: docs/grade-categories-validation.md. Contrato OpenAPI 1.10.0, migración 015 y tipos regenerados.
