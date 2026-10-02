# Banco de preguntas y evaluador docente

Incremento 7, QB1–QB6 / LMS-020/021/022 / T5a. Especificación previa en `openspec/changes/academic-administration/increment-7.md`. No completa T5. El incremento 8 incorpora publicación de cuestionarios e intentos/notas del alumno; véase `docs/quiz-validation.md`. No acredita paridad Moodle ni 98 %.

## Implementación

Cuatro tipos: selección única, selección múltiple, verdadero/falso y respuesta corta. Autoría privada por curso asignado, revisiones inmutables con autor/fecha, edición optimista, archivo/restauración y listas/historial paginados a veinte. ID de pregunta estable y referencia `(question_id,version)` preparada para que los futuros cuestionarios conserven su evaluación original.

Migración 007 crea questions y question_versions con claves foráneas, PK compuesta, referencia diferida a versión actual e índice de curso. Cambiar pregunta inserta una revisión y actualiza su puntero dentro de la misma transacción; las revisiones anteriores no se actualizan. No copia soluciones a lessons.config ni datos de alumno.

El evaluador Go compara selección múltiple como conjunto exacto, sin crédito parcial ni penalización. Respuesta corta recorta espacios Unicode exteriores y puede ignorar mayúsculas con EqualFold; conserva acentos y espacios internos. No interpreta regex ni ejecuta código. Respuestas vacías puntúan cero; estructuras incompatibles, duplicados o índices inexistentes se rechazan. La prueba de evaluación es exclusivamente docente, calcula 0/100 contra una versión fija y no genera intentos, notas ni recompensas.

OpenAPI 1.7.0 y tipos generados. Rutas `/teacher/courses/{courseId}/questions`, `/teacher/questions/{questionId}`, `/versions`, `/versions/{version}`, `/archive` y `/versions/{version}/preview`. Todas requieren sesión teacher y course_staff vigente. La revocación aplica también a versiones históricas y evaluación de prueba. Las listas entregan metadatos, no todas las soluciones. Se reutilizan límites, Origin y no-store existentes.

Astro conserva el editor sin React: formulario para los cuatro tipos, solución sin JSON manual, historial de solo lectura, prueba contra backend y archivo/restauración. Error de API conserva entradas y navegación advierte cambios. No se permite archivar desde el editor con cambios sin guardar. No se instalaron nuevas dependencias.

## Validación ejecutada

- `go test ./... -count=1` con PostgreSQL de prueba: aprobado, handlers 6.087 s. Esquema nuevo y migraciones/seed repetidos; datos existentes no se borran. Comprueba permisos, claves ausentes en curso del alumno, versiones, carrera de dos guardados (éxito/409), conservación del evaluador antiguo, archivo/restauración, paginación, revocación y ausencia de notas/progreso por preview.
- Unitarias de los cuatro tipos: vacíos, tipos/tamaños/rangos, duplicados, UTF-8/NUL, incompatibilidad de campos, conjunto exacto, respuestas vacías, mayúsculas, acentos y espacios. El algoritmo es explícito; no se promete corrección semántica de lenguaje natural.
- Go vet aprobado, API compilada. Migración aplicada a academic_preview conservando datos.
- Astro check final: 51 archivos, cero errores/warnings/hints. En la primera ejecución detectó un acceso de array potencialmente undefined; corregido antes de continuar.
- Build SSR final aprobado en 17.69 s; se conserva aviso de chunks superiores a 500 kB.
- QB6 Playwright: una prueba aprobada en 22.1 s (escenario 20.4 s). Crea los cuatro tipos desde la interfaz, prueba respuestas correctas/incorrectas contra Go, simula caída de API sin perder entradas, verifica archivo con cambios pendientes bloqueado, edita/restaura/consulta historial y comprueba ausencia de desbordamiento horizontal en tablet. Capturas `question-history-tablet.png` y `question-editor-tablet.png` en docs/screenshots.
- Ambas capturas fueron inspeccionadas: historial de solo lectura separado del editor y evaluación docente identificada. Regresión posterior: academic/gradebook/learning, siete pruebas aprobadas en 16.2 s. Ocho escenarios de navegador aprobados en ejecuciones separadas para este incremento.
- OpenSpec strict válido. Sin comparación Moodle, auditoría completa WCAG, prueba de carga ni despliegue Docker ejecutados en este incremento.

## Uso y pendientes

Demo local `http://localhost:4323/login`, cuenta `profe` / `academic-teacher-pass`: Mis cursos → curso → **Banco de preguntas**. Crear → guardar → **Probar evaluación**; **Historial de versiones** conserva el contenido anterior. Archivo/restauración no borra evidencia. El banco privado no aparece al alumnado; las preguntas seleccionadas se usan en los cuestionarios publicados del incremento 8.

Con el entorno del README, ejecutar migraciones desde backend (`go run ./cmd/migrate`), iniciar API/Astro y desde frontend: `E2E_ACADEMIC=1 E2E_TEACHER_PASSWORD='change-this-teacher-password' npm run test:e2e -- questions.spec.ts`. La demo temporal usa el puerto 4323; configurar `E2E_BASE_URL` cuando no se usa el 4321 del README.

El incremento 8 resuelve publicación, intentos del alumno, evaluación y libro con revisión never/after_attempt. Pendientes del objetivo: tiempo/fechas, revisión tras cierre, categorías/búsqueda/importación/exportación, aleatoriedad, otros tipos, permisos compartidos y equivalencia Moodle. Continúan los pendientes de grupos, plataforma, H5P válido y runner aislado. No ocultar estas brechas tras la prueba docente.
