# Exportación del libro y rendimiento medido: incremento 16 (T6, parcial)

Especificación previa `openspec/changes/academic-administration/increment-16.md`, EX1–EX3, EZ1, RP1 y RP2. **AC1 (accesibilidad) queda sin ejecutar** y así se declara. No acredita el 98 % ni equivalencia Moodle.

## Exportación CSV

El docente descarga el libro desde la propia página del libro. La exportación **reutiliza la misma consulta y el mismo ensamblado** que la tabla: el repositorio recibió una ventana parametrizada (página y tamaño de estudiantes y de actividades) y el servicio tiene un único ensamblado, así que la pantalla pide 20×10 y la exportación pide todo hasta sus topes. Duplicar el cálculo habría garantizado que el archivo y la tabla divergieran con el tiempo; ahora es imposible por construcción.

Decisiones que importan:

- **Solo viajan notas publicadas.** Una calificación en borrador se queda en la pantalla del docente; el archivo incluye columnas de estado (entregadas, calificadas, publicadas, pendientes de revisión, pendientes de publicación, sin entregar) para que el docente sepa qué falta sin que el archivo filtre lo que aún no se ha devuelto.
- **Un archivo que no cabe se rechaza.** Por encima de 500 estudiantes o 100 actividades responde 409 explicando el límite. Truncar en silencio produciría un archivo que parece completo y no lo es.
- **Neutralización de fórmulas.** Un alias que empiece por `=`, `+`, `-` o `@` se prefija con una comilla para que una hoja de cálculo no lo ejecute.
- **Marca de orden de bytes** al principio, para que Excel y Sheets lean bien los acentos, y `Cache-Control: private, no-store`.

## Rendimiento: método y resultados

Método: prueba de referencia de Go (`go test -bench`) contra PostgreSQL 18.6 real en WSL, con un curso preparado por la propia prueba: **40 estudiantes** vinculados e inscritos, **12 actividades publicadas** y **480 entregas enviadas con su nota publicada**. Se mide la aplicación completa (`app.Test`, enrutado, sesión, middleware, consultas y serialización), no la consulta aislada. Comando: `go test ./internal/handlers/ -run '^$' -bench . -benchtime 30x` con `TEST_DATABASE_URL`. Máquina: Windows con WSL2, Go 1.27, PostgreSQL 18.6, 30 iteraciones por caso.

| Pantalla | Antes | Después |
|---|---|---|
| Libro de calificaciones (20×10 sobre 480 entregas) | 549 ms | **46 ms** |
| Lección de un estudiante | — | **0,77 ms** |

**Defecto real encontrado y corregido.** El libro tardaba 549 ms porque su consulta de resumen reevaluaba la vista `gradebook_entries` en **9.600 bucles anidados**: el plan ejecutaba las subconsultas correlacionadas de la vista una vez por cada combinación de estudiante y entrega. Se midió con `EXPLAIN (ANALYZE)`: 839 ms de ejecución en la base. La corrección acota la vista a las lecciones del curso en una CTE **materializada**, de modo que se evalúa una sola vez; la medición volvió a ejecutarse y dio 46 ms.

También probé la misma materialización en la consulta de la rejilla visible: **no mejoró nada** (46,6 ms frente a 45,9 ms, dentro del ruido), así que **la revertí** en lugar de dejarla por apariencia.

**Consultas acotadas.** Una prueba cuenta las sentencias que el libro ejecuta con un registrador de GORM: **6 sentencias** con 40 estudiantes y 12 actividades, y el límite está fijado en 12. Es una barrera contra N+1: si alguien introduce una consulta por fila, la prueba falla.

## Evidencia ejecutada

- `go test ./... -count=1` con PostgreSQL 18.6 real: **todo aprobado**, `handlers` 40 s (incluye las dos pruebas nuevas).
- `TestGradebookCSVIntegration`: 200 con `text/csv` y descarga; marca de orden de bytes; encabezado con el título de la actividad; la nota **en borrador no viaja** y el contador de pendientes lo dice; un estudiante sin trabajo sale con celdas vacías y «Sin entregar» 1; al publicar viajan la nota (85) y el **promedio ponderado 85.00**, y el libro por API coincide (8500 centésimas); un alias `=SUM(1)` sale neutralizado; alumno 403 y docente ajeno 404; y por encima del límite, 409 con mensaje.
- `TestGradebookUsesBoundedQueries`: 6 sentencias (registrado).
- `BenchmarkGradebookPage` y `BenchmarkLessonRead`: los números de la tabla.
- `gofmt` y `go vet ./...`: sin hallazgos. `astro check`: 60 archivos, 0 errores.

## Lo que NO se hizo

**La auditoría de accesibilidad (AC1) no se ejecutó**: no hay revisión de etiquetas, orden de encabezados, foco, tamaños táctiles ni contraste más allá de lo ya verificado en incrementos anteriores, y no se ejecutó ninguna herramienta de análisis automático. Queda pendiente y así se registra en `tasks.md`.

## Reproducción

```bash
go run ./cmd/migrate && go run ./cmd/api
npm --prefix frontend run dev
# medición (requiere TEST_DATABASE_URL terminada en _test)
cd backend && TEST_DATABASE_URL='postgres://aulaquest:aulaquest@127.0.0.1:5432/aulaquest_test?sslmode=disable' \
  go test ./internal/handlers/ -run '^$' -bench . -benchtime 30x
```

En el libro de calificaciones, **Descargar CSV**.

## Límites

Un solo formato (CSV) y sin exportación por actividad. La neutralización de fórmulas cubre los caracteres habituales, no todos los casos imaginables. Los topes de exportación (500×100) están probados por unidad y en su rechazo, no con un curso real de ese tamaño. La medición se hizo con un curso de 40 estudiantes y 12 actividades: no es un perfil de carga y no se ejecutó con miles de estudiantes ni con carga concurrente.
