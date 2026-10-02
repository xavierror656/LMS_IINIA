# Recursos H5P
La actividad demo está desactivada: no se ha incluido un paquete con licencia y reproducción verificadas.
1. Revisar licencia/atribución y seguridad del archivo .h5p (ZIP).
2. Extraer en demo/: h5p.json, content/content.json y sus datos multimedia, TODAS las carpetas de bibliotecas y dependencias.
3. El build copia los recursos del reproductor (frame.bundle.js, styles y fonts) desde h5p-standalone/dist a player/. No son el contenido de la actividad.
4. Conservar atribución en demo/ATTRIBUTION.md. Comprobar rutas y reproducción antes de activar src/config/h5p.ts.
5. El manifiesto es configuración del operador, nunca URL del estudiante. No se admiten subidas.
La API real 3.8.2 inicializa con await new H5P(element, options). No documenta destroy(): se retira listener xAPI y DOM; para contenido activo no confiable se requiere otro origen y un ciclo de vida aislado. La liberación interna de bibliotecas H5P depende de estas y debe medirse con paquete real.
