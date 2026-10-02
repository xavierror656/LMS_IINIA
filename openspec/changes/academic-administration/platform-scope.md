# Alcance confirmado: Moodle con exclusiones acordadas

Estado: implementación incremental iniciada; primer flujo de autoría, entregas y notas en Go/PostgreSQL. Los demás dominios continúan como propuesta. El usuario respondió «todo» a la elección explícita entre módulo académico y toda la plataforma. Posteriormente el usuario excluyó expresamente foros, wikis, extensiones/plugins, respaldos y el módulo de operación. Esta revisión prevalece sobre el alcance global anterior. No sustituye la necesidad de comprobar cada función ni autoriza despliegues.

## Base de comparación

Referencia documental: Moodle LMS 5.2, consultada el 2026-10-02. Comparar el alcance acordado exige recorrer administración y experiencia de administrador, docente y estudiante, además de componentes estándar e integraciones documentadas. El catálogo inicial de 43 capacidades académicas es solo una parte.

## Exclusiones autorizadas

Fuera de implementación y del denominador acordado: foros, wikis, gestor de extensiones/plugins, copias/restauración de sitio o cursos (incluidos respaldos Moodle) y panel de operación/mantenimiento. No son el 2 % pendiente ni capacidades cumplidas: son exclusiones de alcance solicitadas por el usuario. El resultado deberá titularse «cobertura del alcance Moodle acordado», nunca «98 % de todo Moodle».

Se conservan las integraciones concretas previstas, como H5P, SCORM, LTI y SSO; excluir el gestor de extensiones no elimina esas funciones. Las prórrogas de entrega tampoco se eliminan: no son extensiones de software. Glosarios, talleres y las demás actividades no mencionadas siguen incluidas.

Se conserva la infraestructura básica necesaria para ejecutar la aplicación y proteger sus datos: configuración, migraciones, archivos privados, límites, logs técnicos y healthchecks. No se propone un módulo de operación ni herramientas de respaldo. No se borran archivos ni funciones existentes de la base como parte de esta revisión documental.

No mezclar Moodle LMS con otros productos comerciales. Registrar las versiones de los componentes y servicios externos de las integraciones incluidas. No se promete compatibilidad binaria con plugins PHP.

La igualdad solicitada continúa siendo exactamente 98 %, sin sustituirla por ≥98 %. Completar más funciones no justifica eliminar funcionalidades para bajar al porcentaje: informar el resultado real y resolver la diferencia con el usuario. No fijar ahora un 2 % arbitrario. Seguridad, privacidad y conservación de datos son requisitos de calidad obligatorios, no funciones sacrificables para cuadrar la cifra.

## Mapa funcional y brechas

| Dominio | Funciones a especificar y contrastar | Evidencia actual en AulaQuest | Incremento |
|---|---|---|---|
| Identidad y cuentas | Alta administrativa, edición, suspensión, recuperación, importación masiva, perfil, métodos de autenticación y sesiones | Sesiones y cuentas sintéticas; gestión completa pendiente | PL1 |
| Roles y permisos | Capacidades por contexto de sitio, categoría, curso y actividad; delegación y auditoría | Roles limitados; admin solo demo | PL1 |
| Matrículas | Matrícula manual, cohortes, grupos, sincronización, bajas, caducidad y métodos adicionales | Inscripciones de seed; administración pendiente | PL1 |
| Sitio y organización | Categorías, configuración, marca, idioma, zona horaria, búsqueda y navegación | Marca centralizada y español; administración amplia pendiente | PL1 |
| Cursos | Crear, editar, duplicar, ordenar secciones, visibilidad, formatos, carga masiva, reinicio y archivo | Catálogo y mapa de cursos existentes; autoría de lecturas/tareas disponible; gestión completa de cursos pendiente | PL2 |
| Recursos | Página, libro, archivo, carpeta, URL, texto/multimedia, editor y repositorios | Lectura precargada; edición y biblioteca pendientes | PL2 |
| Tareas | Texto/archivo, plazos, prórrogas, reenvíos, grupos y entrega definitiva | Entregas de texto y TXT/PNG/JPEG privados, calendario UTC, prórrogas, reentregas e historial; otros formatos y grupos pendientes | PL3 |
| Calificación | Notas, rúbricas, guías, anotaciones, anonimato, moderación, publicación y libro de calificaciones | Notas manuales y por rúbrica versionada; libro con media ponderada de tareas publicada; categorías y otros métodos pendientes | PL3 |
| Cuestionarios | Tipos de preguntas, banco/versiones, importación/exportación, aleatoriedad, tiempo, intentos, revisión y recalificación | Banco versionado, cuestionarios publicados, intentos del alumno y cuatro evaluadores en Go conectados al libro; temporizador/fechas/aleatoriedad/importación y otras ampliaciones pendientes | PL4 |
| Actividades colaborativas | Glosario, base de datos y taller de evaluación por pares | Pendiente; definir controles adecuados para menores | PL5 |
| Otras actividades | Consulta, encuesta, feedback y lección ramificada | Pendiente; lectura lineal no acredita lección ramificada | PL5 |
| H5P | Banco, autoría, carga, bibliotecas, reproducción, intentos y relación con notas | Reproductor integrado sin contenido validado; constructor pendiente | PL5 |
| Seguimiento | Finalización de actividad/curso, restricciones condicionales, competencias, planes e insignias | Lecturas completadas y gamificación parcial; no equivalencia completa | PL5 |
| Comunicación | Calendario, avisos, preferencias de notificación, mensajería y moderación | Pendiente; chat libre entre menores sigue sin habilitarse | PL6 |
| Reportes | Participación, actividad, finalización, notas, logs, informes de sitio y analítica | Resumen docente limitado | PL6 |
| Interoperabilidad | SCORM, LTI, servicios web, repositorios, SSO y herramientas externas | API propia no acredita protocolos Moodle | PL7 |
| Videoconferencia | Integración BigBlueButton y controles de acceso; servicio externo y grabaciones | Pendiente; no servicio contratado ni desplegado | PL7 |
| Privacidad y seguridad | Solicitudes de datos, retención, eliminación, políticas, permisos y trazabilidad | Controles base; no ciclo administrativo completo | Transversal |
| Accesibilidad y móvil | Teclado, lectores, idiomas, tablet, app, descargas, trabajo sin conexión y sincronización | Web adaptable; móvil nativo/offline no acreditados | Transversal |

Fuentes de estructura: [Features](https://docs.moodle.org/502/en/Features), [Managing a Moodle site](https://docs.moodle.org/502/en/Managing_a_Moodle_site), [Managing a Moodle course](https://docs.moodle.org/502/en/Managing_a_Moodle_course), [Activities](https://docs.moodle.org/502/en/Activities). Las funciones específicas de cada dominio requieren revisar su documentación enlazada y probar una instalación de referencia; esta tabla es un mapa de investigación y producto, no certificación de cada detalle.

## Arquitectura propuesta

Conservar Astro SSR y monolito modular Go/PostgreSQL. Organizar responsabilidades internas en identity, access, enrollment, authoring, assessment, gradebook, communication, reporting e integrations. Son límites internos del monolito. Mantener contratos REST versionados, datos privados sin caché compartida, permisos por objeto y migraciones SQL explícitas.

Añadir almacenamiento privado de archivos con análisis y cuotas; cola persistente con reintentos idempotentes para notificaciones, importaciones e informes; auditoría para operaciones administrativas. Los servicios externos se integran mediante adaptadores. SCORM/H5P/LTI requieren separación de confianza y políticas de origen. Nunca incorporar un plugin arbitrario al proceso API ni confundir compatibilidad funcional con compatibilidad binaria.

Los permisos dejan de depender solo de tres roles globales: modelar capacidades y asignaciones por contexto. Mantener defaults simples para la experiencia infantil. Administrar y calificar son permisos separados; toda suplantación administrativa, si se incorpora, necesita alcance, razón y auditoría.

## Organización de la interfaz

Administrador: Inicio, Personas, Matrículas, Cursos, Contenido, Evaluación, Comunicación, Reportes, Integraciones y Configuración. Docente: Mis cursos, Crear contenido, Entregas, Calificaciones, Banco de preguntas y Mi grupo. Estudiante: Mis aventuras, Actividad, Mi entrega, Mi progreso y Devoluciones.

Mostrar solo herramientas disponibles y autorizadas. Funciones pendientes permanecen en el plan, no como botones que aparenten funcionar. Cada pantalla debe contemplar vacío, carga, error recuperable, permisos y confirmación de operaciones con consecuencias.

## Cambios frente al MVP original

SCORM, constructor H5P, videoconferencia y analítica entran ahora en la propuesta global y en la evaluación de brechas. El MVP original ya no describe todo el alcance deseado. No cambiar retrospectivamente sus pruebas ni marcarlo incompleto por requisitos que son nuevos: mantener trazabilidad del incremento.

Las protecciones infantiles permanecen: sin publicidad, rankings públicos ni recompensas compradas; las vidas no bloquean aprendizaje. La comparación debe documentar diferencias deliberadas como ausencia de chat libre entre menores o registro público. Esas diferencias no equivalen automáticamente al 2 %. Pagos y métodos de matrícula de pago, cuando figuren en la referencia, se registrarán como brecha de producto; no activarlos sin una decisión concreta de producto y operación.

## Dependencias y aceptación por incremento

- PL1: identidad, capacidades y matrículas. Probar aislamiento entre cursos y suspensión efectiva de sesiones.
- PL2, depende PL1: cursos, recursos y biblioteca. Crear y publicar desde interfaz; estudiante autorizado recupera contenido tras recarga.
- PL3, depende PL2: entregas y calificación. Probar concurrencia, historial y publicación de notas; nunca duplicar recompensas.
- PL4, depende PL3: cuestionarios y banco. Probar cada tipo incluido, secretos de respuesta y cálculo canónico.
- PL5, depende PL2/PL3: resto de actividades y seguimiento. Cada tipo requiere su propio flujo completo y controles de confianza.
- PL6, depende PL1/PL3/PL5: comunicación y reportes. Aislamiento, preferencias y datos reconciliados con origen.
- PL7, depende PL1/PL2/PL3: estándares e integraciones. Pruebas con proveedor/paquete de referencia, revocación y fallos externos.

Sin estimación cerrada de calendario: requiere inventario detallado, recursos y validación técnica. Este alcance es una plataforma completa, no un único panel CRUD.

## Qué falta para demostrar exactamente 98 %

1. Completar el inventario por dominio incluyendo configuraciones y errores, con IDs estables; no fragmentar solo las funciones fáciles para inflar cobertura.
2. Fijar release y componentes de referencia; registrar versiones de componentes y servicios externos de las integraciones incluidas.
3. Definir pesos y pruebas de aceptación antes de medir implementación. Mantener las exclusiones autorizadas fuera del denominador y las funciones incluidas pendientes dentro del total; no confundir ambas categorías.
4. Ejecutar flujos equivalentes con datos sintéticos en Moodle y AulaQuest; conservar salidas y evidencia por capacidad.
5. Publicar cálculo exacto sin redondeo, pendientes, fallos y alcance de la auditoría.

Hoy no hay porcentaje acreditado. El bloqueo anterior de módulo versus plataforma está resuelto; el trabajo siguiente es especificación detallada e implementación incremental del alcance acordado. PL8 queda retirado por exclusión explícita; los IDs PL1..PL7 se conservan.
