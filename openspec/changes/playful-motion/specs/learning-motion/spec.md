## ADDED Requirements
### Requirement: LMS-012 Movimiento contextual accesible
La interfaz SHALL animar brevemente entradas e interacciones sin saltos de layout ni movimiento continuo. El HUD SHALL celebrar exclusivamente aumentos confirmados de progreso del mismo estudiante y SHALL conservar la información sin movimiento cuando prefers-reduced-motion está activo.
Errores: API caída o progreso no confirmado no producen celebración; se limpia el timer al desmontar. Restricciones: sin nuevas islas para contenido simple, sin sonidos ni dependencia del color. Fuera de alcance: efectos permanentes, recompensas cliente o rediseño de lecciones.
#### Scenario: Explorar
- **GIVEN** movimiento normal y catálogo o mapa
- **WHEN** abre la página e interactúa con una tarjeta
- **THEN** observa entradas escalonadas y respuesta visual finita sin cambiar dimensiones
#### Scenario: Celebrar progreso canónico
- **GIVEN** un progreso previamente cargado del estudiante
- **WHEN** el servidor confirma un aumento
- **THEN** el HUD muestra el incremento y una celebración breve; recargar o repetir no vuelve a celebrar
#### Scenario: Movimiento reducido
- **GIVEN** prefers-reduced-motion activo
- **WHEN** navega o recibe recompensa
- **THEN** conserva texto accesible y controles sin desplazamientos ni partículas animadas
