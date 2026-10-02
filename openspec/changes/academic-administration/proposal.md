# Plataforma AulaQuest: propuesta de paridad funcional con Moodle

Estado: implementación incremental en curso; autoría, entregas textuales y notas manuales implementadas en Go/PostgreSQL. No es certificación de paridad.
Fecha de investigación: 2026-10-02. Relacionada con bootstrap-kids-lms y demo-admin.

## Why
El administrador actual consulta cuentas/cursos sintéticos y modifica ajustes de plugins. El docente consulta progreso. Ninguna de esas funciones equivale a crear actividades o calificar entregas. Se necesita cerrar el ciclo crear → publicar → entregar → evaluar → devolver retroalimentación.

## What Changes
Crear un espacio docente con cursos editables, biblioteca de recursos, tareas, cuestionarios, entregas, rúbricas y libro de calificaciones. El administrador asignará docentes a cursos y permisos; el docente trabajará únicamente en sus cursos y estudiantes autorizados. Mantener una experiencia infantil sencilla para el estudiante.

Se propone usar las funciones documentadas de Moodle 5.2 como referencia de comparación, no como dependencia ni garantía de compatibilidad de archivos. Véanse matrix.md y design.md.

## Alcance confirmado
La respuesta «todo» amplía esta iniciativa a administración del sitio, identidad, matrículas, cursos, recursos, actividades, evaluación, comunicación, seguimiento, reportes, integraciones y móvil, con las exclusiones posteriores documentadas en platform-scope.md. El módulo académico es el primer incremento de esa plataforma, no el objetivo final reducido.

## Capabilities
- academic-administration: autoría, publicación, entregas, evaluación y auditoría.
- Reutilizar sesiones, cursos, lecciones y progreso sin confundir nota académica con XP.

## Impact
Extensión de Astro SSR y del monolito Go/PostgreSQL. Nuevas migraciones aditivas y contratos versionados. La demo Node no será evidencia de persistencia o autorización del backend Go. Sin despliegues ni modificaciones a datos existentes durante esta propuesta.

## Objetivo del 98 %
Solicitud original: cumplir exactamente el 98 % respecto de Moodle. No existe un porcentaje verificable sin un inventario acordado, versión, pesos y pruebas. La matriz adjunta es una base candidata, NO el inventario completo de Moodle. El usuario confirmó «todo»: el alcance inicial abarcaba toda la plataforma Moodle. Su instrucción posterior excluye foros, wikis, extensiones/plugins, respaldos y el módulo de operación; esas exclusiones se aplican al denominador y quedan registradas en platform-scope.md. La versión de referencia documental sigue siendo Moodle 5.2. Falta cerrar el inventario completo y sus criterios; ya no falta decidir entre módulo y plataforma.

Para un inventario congelado, cobertura = 100 × suma de pesos de capacidades verificadas / suma de todos los pesos incluidos. Una capacidad requiere todas sus pruebas obligatorias; documentación, pantallas, mocks y pruebas omitidas no suman. No redondear 97.6 a 98. No cambiar el denominador para ocultar pendientes.

La igualdad exacta se evaluaría como suma_verificada × 100 = suma_total × 98. Por ejemplo, 49 de 50 capacidades de igual peso representa 98 %, pero seleccionar arbitrariamente 50 funciones no prueba paridad con Moodle. Se recomienda un umbral de al menos 98 %; este cambio a la solicitud requiere acuerdo, no se asume. Seguridad, autorización y conservación de notas deben cumplir el 100 % aunque la cobertura funcional llegue a 98 %.

## Límites que deben quedar visibles
El MVP excluye SCORM, videoconferencia, chat entre menores, importación de respaldos y constructor H5P. Moodle posee capacidades fuera de este alcance: no se pueden descontar silenciosamente ni declarar que todas juntas representan solo el 2 %. Al comparar el alcance acordado, las funciones no excluidas entran en el análisis de brechas; los respaldos ya fueron excluidos expresamente. No se consideran automáticamente parte del 2 % permitido. La propuesta ampliada está en platform-scope.md. La inclusión en la comparación no habilita automáticamente pagos, chat entre menores ni subida de código activo en el producto.

## Evidencia de la inspección inicial (antes del incremento 1)
- frontend/src/pages/admin/index.astro: panel condicionado a PUBLIC_DEMO_MODE; creación de usuarios/edición de cursos pendientes.
- backend/internal/models/models.go: reading/code/h5p; sin modelos de entregas, rúbricas o calificaciones.
- backend/internal/handlers/api.go y contracts/openapi.json: API de aprendizaje/progreso; no constituyen contrato de autoría y evaluación.
- README.md: administración solo demo, MockRunner y H5P sin paquete validado.

La inspección no equivale a ejecutar pruebas funcionales. OpenSpec no estaba instalado inicialmente; posteriormente se instaló 1.14.0 en /tmp y la validación estricta del cambio pasó. Git no reconoció este directorio como repositorio utilizable; no se modificó .git.

## Revisión de alcance solicitada por el usuario
Foros, wikis, gestor de extensiones, respaldos y módulo de operación quedan fuera. Se conserva el resto de la propuesta. La meta es exactamente 98 % del alcance Moodle acordado, no del producto Moodle completo. No hay porcentaje verificado todavía.

## Evidencia del incremento 1
Implementación en `backend/internal/{models,repositories,services,handlers}/academic.go`, migración 002 y páginas `frontend/src/pages/teacher/{courses,activities}`. Entregas y notas tienen flujo Go/PostgreSQL y navegador; resultados y pendientes en `docs/academic-validation.md`.
