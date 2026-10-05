# Tareas por dependencia

- [x] F0 Especificar el cambio, verificar versiones y validar con OpenSpec --strict. Dueño: orquestador. Compuerta: oracle (sustituida por auto-revisión por decisión del usuario).
- [x] F0a Publicar el contrato de migración: mapa de tokens, mapa de clases y criterios de aceptación (UI-001..UI-006). Depende F0.
- [x] F1 Instalar `daisyui@5.7.47` exacto y registrar el tema `aulaquest` en `global.css` con `@plugin "daisyui" { themes: false; }`. Sin tocar páginas. Dueño: designer. Compuerta: oracle (auto-revisión).
- [x] F1a Calcular contraste de los pares del tema, actualizar `docs/contrast.json` y ejecutar check/build + auditoría AC1 sobre la base sin cambios visuales. Depende F1.
- [x] F2 Migrar shell y componentes compartidos: Layout, navegación, HUD, footer, `ServiceError`, botones, `AcademicForm`, `RubricFields`, `GradeCategories`, `QuizComposer` y login. Dueño: designer + fixer. Compuerta: oracle (auto-revisión).
- [x] F2a Validar E2E de login/rúbrica/categorías, check/build, AC1 y capturas del shell. Depende F2.
- [x] F3 Migrar pantallas docente: cursos, actividades y sus editores, libro de calificaciones, grupos, banco de preguntas y cuestionarios. Dueño: fixer + designer. Compuerta: oracle (auto-revisión).
- [x] F3a Validar E2E de libro, rúbrica, categorías, grupos y cuestionarios; check/build y AC1. Depende F3.
- [x] F4 Migrar pantallas de alumno y públicas: inicio, login, catálogo, mapa del curso, lecciones (lectura, código, H5P, tarea, cuestionario) y admin de demostración. Dueño: fixer + designer. Compuerta: oracle (auto-revisión).
- [x] F4a Validar E2E de aprendizaje, consola, H5P ausente y intentos; check/build y AC1. Depende F4.
- [x] F5 Retirar CSS y clases heredadas sin referencias; medir CSS emitido y comparar con la base. Dueño: orquestador. Compuerta: oracle final.
- [x] F5a Correr la batería completa (Go, E2E con ventanas, AC1, contraste, build), actualizar capturas, README, `docs/versions.md` y registro de validación. Depende F5.
- [x] F5b Verificar que no queda doble sistema: búsqueda de clases heredadas en `frontend/src` sin resultados y `global.css` limitado a tema, fuentes y layout. Depende F5.

Sin despliegues, sin commits automáticos y sin debilitar límites de seguridad para
las pruebas. Cada fase conserva la API y el backend intactos.
