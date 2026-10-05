# Incremento 16: exportación, rendimiento y accesibilidad (T6)

Especificación previa al código. T6 / LMS-022 / A8, en tres partes. No acredita el 98 % ni equivalencia Moodle.

## Reglas y aceptación

- EX1: el docente del curso descarga el libro como **CSV** con los **mismos números** que ve en pantalla: se reutiliza la misma consulta y el mismo ensamblado, solo cambia la ventana (pantalla paginada, exportación completa). Una celda solo lleva nota si está **publicada**; los contadores de trabajo pendiente viajan en columnas propias para que el archivo cuente lo que la tabla cuenta sin filtrar calificaciones sin publicar.
- EX2: el archivo se entrega como descarga, con tipo `text/csv`, marca de orden de bytes para que una hoja de cálculo lea los acentos, y sin caché compartida. Un nombre que empiece por `=`, `+`, `-` o `@` se neutraliza: un alias de estudiante no puede convertirse en fórmula al abrir el archivo.
- EX3: una exportación que no quepa se **rechaza**, no se trunca: el libro se corta en un límite declarado (500 estudiantes y 100 actividades) y por encima responde 409 explicando que hace falta exportar por partes. Un archivo incompleto que parezca completo es peor que un error.
- EZ1: la autorización es la del libro: docente del curso; alumno 403 y docente ajeno 404.
- RP1: el rendimiento se **mide** en este entorno, no se afirma: consultas por pantalla (para probar que no crece con las filas) y latencia de las pantallas principales sobre un curso con 40 estudiantes y 12 actividades publicadas, con el método, el entorno y los números escritos.
- RP2: cuando una medición muestre un costo evitable, se corrige y **se vuelve a medir**; si una optimización no mejora nada, se revierte en lugar de dejarla por apariencia.
- AC1: la accesibilidad se revisa con criterios comprobables (etiquetas, orden de encabezados, foco visible, tamaño de objetivos táctiles, contraste, movimiento reducido) y los hallazgos se corrigen o se registran.

## Alcance de esta rebanada

Implementadas y verificadas: EX1–EX3, EZ1, RP1 y RP2. **AC1 queda sin ejecutar en esta ronda** y así se registra: no hay auditoría de accesibilidad hecha, solo la intención especificada.

## Fuera de alcance

Exportación en otros formatos (Excel, PDF), exportación por actividad, programación de informes, perfiles de carga con miles de estudiantes y pruebas de accesibilidad con lectores de pantalla reales.
