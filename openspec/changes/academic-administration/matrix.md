# Comparación candidata con Moodle

Referencia: documentación Moodle 5.2 consultada el 2026-10-02. No se ha probado una instalación Moodle ni afirmado equivalencia detallada de todos sus ajustes. Cada grupo requiere descomposición en casos atómicos antes de fijar pesos. Estado de AulaQuest basado en inspección, no en pruebas nuevas.

| Área | Referencia Moodle | AulaQuest observado | Propuesta y brecha |
|---|---|---|---|
| Recursos | [Resources](https://docs.moodle.org/502/en/Resources): páginas, archivos, carpetas, URL y libros | Lectura precargada | Editor de recursos, adjuntos y organización; libro multipágina por especificar |
| Tareas | [Assignment](https://docs.moodle.org/502/en/Assignment_activity): entrega de texto/archivos, individual o grupal | No hay entregas | Entrega versionada, bandeja docente, grupos y permisos explícitos |
| Disponibilidad | [Assignment settings](https://docs.moodle.org/502/en/Assignment_settings): apertura, vencimiento, cierre | Sin calendario de entregas | Fechas UTC, visualización local, extensiones y estado tardío |
| Reentrega | [Assignment settings](https://docs.moodle.org/502/en/Assignment_settings): borradores, envío y nuevos intentos | No disponible | Borrador, envío inmutable y reapertura autorizada |
| Corrección | [Assignment](https://docs.moodle.org/502/en/Assignment_activity): notas y comentarios | Progreso de lectura; no nota académica | Calificación manual, devolución y archivos de retroalimentación |
| Evaluación avanzada | [Grades](https://docs.moodle.org/502/en/Grades): rúbricas y guías | No disponible | Criterios y niveles versionados; guías de evaluación |
| Flujo de corrección | [Assignment settings](https://docs.moodle.org/502/en/Assignment_settings): revisión y publicación | No disponible | Borrador de nota, revisión, publicación y auditoría; asignación de correctores por detallar |
| Cuestionarios | [Quiz](https://docs.moodle.org/502/en/Quiz_activity): preguntas reutilizables y evaluación | No disponible | Cuestionarios con evaluación en servidor |
| Configuración de examen | [Quiz settings](https://docs.moodle.org/502/en/Quiz_settings) | No disponible | Fechas, tiempo, intentos, orden aleatorio y política de revisión |
| Banco de preguntas | [Question banks](https://docs.moodle.org/502/en/Question_bank) | No disponible | Biblioteca reutilizable y versiones; fijar tipos y formatos compatibles |
| Libro de notas | [Grades](https://docs.moodle.org/502/en/Grades): ítems, escalas e historial | Resumen de lecciones completadas | Notas independientes de recompensas y estados visibles |
| Agregación | [Grade categories](https://docs.moodle.org/502/en/Grade_categories) | No disponible | Categorías, ponderaciones, exclusiones y reglas de faltantes |
| Intercambio de notas | [Grades](https://docs.moodle.org/502/en/Grades): importación/exportación | No disponible | CSV con prevalidación, simulación y auditoría; otros formatos por inventariar |
| H5P | [Content bank](https://docs.moodle.org/502/en/Content_bank): creación y gestión de contenido | Reproductor sin paquete validado | Selección de paquetes revisados; constructor sigue excluido del MVP y es brecha de paridad |

## Inventario pendiente para evitar una comparación sesgada
Desglosar también anotación PDF, corrección anónima, varios correctores, calificación rápida, entregas grupales, extensiones individuales, notificaciones, resultados/competencias, cálculos de notas y tipos de preguntas. No tratarlos como implementados por tener una fila general parecida.

El objetivo revisado comprende Moodle con exclusiones explícitas: inventariar además talleres, glosarios, base de datos, consultas, encuestas, lección ramificada, LTI, SCORM, roles/capacidades, matrícula, grupos, reportes, integraciones y administración del sitio. Foros, wikis, gestor de extensiones, respaldos y módulo de operación están excluidos por el usuario. Esta lista orienta la investigación; tampoco es un inventario exhaustivo.

## Registro de certificación requerido
Por capacidad: ID estable, versión/página Moodle, comportamiento atómico, peso acordado, incluida/excluida con motivo, requisito LMS, tarea, prueba, resultado, evidencia y fecha. Estados: pendiente, parcial, verificada, fallida, excluida. Solo verificada cuenta. Porcentaje actual: **no determinado**; no debe publicarse 98 %.

## Inventario atómico candidato

`capabilities.csv` desglosa inicialmente 43 capacidades académicas con fuente, criterio observable e identificador de prueba PAR. El alcance con exclusiones ya fue confirmado; los pesos quedan sin fijar hasta completar el inventario. Ninguna fila suma cobertura: todas están `unverified`, sin evidencia funcional. Estos casos son condiciones propuestas para AulaQuest, no resultados de una ejecución en Moodle. Los requisitos LMS agrupan la intención; para funciones avanzadas, ampliar los escenarios y contratos antes de implementar.

El inventario sigue abierto: aún deben desglosarse tipos de preguntas, recursos, permisos avanzados y otras funciones del alcance acordado. No se eligieron 50 filas para hacer coincidir aritméticamente el 98 %. La ausencia de una capacidad en este archivo no equivale a una exclusión aprobada.

## Ampliación confirmada
El mapa de la plataforma con exclusiones está en `platform-scope.md`. Las 43 filas académicas no son el denominador de la plataforma. El porcentaje global permanece no determinado hasta completar y verificar todos los dominios incluidos.
