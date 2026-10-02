# Diseño propuesto

## Experiencia
Docente: Mis cursos → Contenido / Entregas / Calificaciones / Banco de preguntas. Crear actividad: tipo → instrucciones y recursos → entrega/evaluación → fechas → vista previa → publicar. Guardar borrador debe funcionar sin publicar accidentalmente. Bandeja con filtros por curso, actividad, estudiante y estado; corrección con evidencia y rúbrica, seguida de devolución explícita.

Administrador: asignaciones de docentes, configuración y auditoría. El permiso administrativo no concede implícitamente permiso para cambiar notas: capacidad separada y auditada. Estudiante: actividad, Mi entrega y devolución; no recibe claves de respuesta ni notas ajenas. Formularios Astro; React solo para editor estructurado, rúbricas y tabla editable cuando sea necesario. Tabletas, teclado, foco visible, objetivos de 44 px y movimiento reducido.

## Backend y datos
Extender Go/Fiber, servicios y repositorios existentes; PostgreSQL autoritativo. No crear otro backend académico en Node. Migraciones nuevas, sin modificar SQL ya aplicado.

Tablas propuestas: course_staff(course_id,user_id,capability), activity_versions(lesson_id,version,status,config,published_at), submissions(lesson_id,user_id,attempt,activity_version_id,status,submitted_at), submission_files(submission_id,storage_key,mime,size), question_versions(question_id,version,kind,prompt,answer_key), quiz_items(activity_version_id,question_version_id,position), quiz_answers(submission_id,question_version_id,response), rubrics y rubric_criteria con versiones, grade_items(course_id,lesson_id,max_score,category_id), grade_categories(course_id,parent_id,weight), grades(grade_item_id,user_id,submission_id,score,status,version), grade_revisions(grade_id,actor_id,previous_value,new_value,reason,created_at).

FK obligatorias; UNIQUE(lesson_id,version), UNIQUE(lesson_id,user_id,attempt), UNIQUE(grade_item_id,user_id). Claves/validaciones compuestas impiden relacionar entregas, actividades y notas de cursos distintos. Índices de bandeja por curso/estado/fecha y usuario/actividad. La clave de respuestas nunca se serializa al estudiante. NUMERIC para notas; escala, precisión y redondeo explícitos, no float monetario ni total del cliente.

Publicar fija una versión. Entregar fija contenido y versión de actividad. Reabrir genera otro intento y conserva el anterior. Guardar nota usa control optimista del campo version; edición obsoleta falla, no sobrescribe. Publicar nota y registrar historial ocurre en una transacción. Recalificar conserva autor, motivo y valor anterior. Nota ausente no es cero; distinguir pendiente, exento, calificado y publicado. Agregación propuesta: media ponderada normalizada de ítems incluidos; política de faltantes obligatoria por curso, validación de pesos y categorías sin ciclos. La equivalencia con cada agregación Moodle exige pruebas adicionales.

Notas y XP siguen separados. Un cambio de nota no vuelve a premiar la lección. H5P client_reported no se convierte en evaluación verificada. MockRunner continúa sin ejecutar código.

## REST del primer incremento y extensiones previstas
Prefijo /api/v1. Mantener contracts/openapi.json fiel a lo ejecutable; incorporar cada endpoint allí junto con su implementación y tipos.

El contrato ejecutable del incremento está en `contracts/openapi.json` (1.1.0), con el detalle de campos/transiciones en `increment-1.md`. Utiliza PUT para reemplazar borradores, version en JSON para control optimista y 409 para revisiones obsoletas. Incluye cursos asignados, actividades, publicación, entrega propia, bandeja docente y guardado/publicación de notas por submissionId. Las listas usan page/pageSize=20, no cursor en este incremento.

Los incrementos 2–6 implementan libro ponderado, rúbricas, archivos privados TXT/PNG/JPEG, plazos/prórrogas e intentos con reapertura manual e historial. Sus contratos vigentes están en OpenAPI 1.6.0 y las decisiones concretas en increment-2.md a increment-6.md; prevalecen sobre tablas y rutas orientativas de este diseño inicial. El incremento 7 incorpora banco privado versionado y evaluación objetiva docente. El incremento 8 publica cuestionarios con versiones fijas, intentos de alumno evaluados en Go y libro unificado; contrato vigente 1.8.0. Fechas, temporizador y revisión tras cierre continúan en T5c. Siguen previstos /me/grades, categorías, otras agregaciones y las demás ampliaciones del inventario. Sus endpoints requieren OpenAPI antes del código. No se conserva como contrato vigente el borrador previo con PATCH, If-Match o /teacher/grades/{id}.

Todas las escrituras comprueban Origin, sesión, rol y objeto. 400 entrada inválida; 401 sesión; 403 rol/Origin; 404 objeto no accesible; 409 conflicto; 413 más de 128 KiB; 429 límite; 503 DB. Publicación y envío definitivos repetidos con la misma revisión son idempotentes mediante transacciones, locks y restricciones; no dependen de una clave arbitraria del navegador.

## WebSocket
Sin cambio al contrato v1 de /ws/code. Autoría/calificaciones usan REST; no reutilizar eventos run.* para notas. No se necesita WebSocket para esta primera extensión.

## Seguridad y privacidad
Reutilizar cookie HttpOnly, comprobación Origin, no-store y autorización por objeto. HTML del editor sanitizado con lista permitida en servidor; no ejecutar scripts embebidos. Archivos privados, límites de tipo/tamaño, cuarentena y análisis antes de descarga; nombres de almacenamiento generados y descarga autorizada, nunca rutas del cliente. Importación CSV con vista previa, rechazo de filas ajenas y protección de fórmulas al exportar. Auditoría separada de logs HTTP, acceso restringido y política de conservación/borrado por definir antes de datos reales. No introducir chat, rankings ni evaluaciones punitivas que bloqueen lecturas.

## Entrega incremental
1. Permisos de curso + autoría de lectura y tarea textual + borrador/publicación.
2. Entrega textual + nota manual + devolución + historial.
3. Libro de calificaciones + rúbricas + archivos seguros.
4. Banco de preguntas + cuestionarios + evaluación en servidor.
5. Funciones avanzadas y auditoría de paridad sobre inventario congelado.

Cada incremento debe ejecutarse en Go/PostgreSQL y navegador. No atribuir a la API Go una función que solo existe en demo.
