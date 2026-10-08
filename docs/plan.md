# Nexo Chess Implementation Plan

> **For agentic workers:** Use development with isolated component ownership and independent final review. User authorizes autonomous execution; no intermediate approval gates.

**Goal:** Entregar un exe nativo de ajedrez para la red local, ligero y listo para jugar.
**Architecture:** Go HTTP con assets embebidos y reglas validadas en servidor. Estado y sesiones en memoria; sondeos del navegador para sincronización.
**Tech Stack:** Go portable de compilación; github.com/corentings/chess/v2; HTML/CSS/JS sin framework.
**Spec:** docs/design.md

## Global Constraints

- Windows 10/11 x64, sin instalar dependencias.
- No modificar sources ni recursos de Álvaro/ RICI.
- Sin datos persistentes de la app ni red externa en ejecución.
- Interfaz española, assets embebidos y reglas autoritativas en servidor.
- Respetar el contrato HTTP de design.md.

## Review Focus

- Servidor ocupado/puerto ocupado: error útil o búsqueda de puerto libre.
- Doble clic y solicitudes simultáneas: nunca duplicar ni saltar turnos.
- Recargar una pestaña: recuperar asiento y posición.
- Navegación al pasado durante jugadas: no mover desde una posición histórica.
- Contraste y orientación: piezas/selección visibles en ambos temas y con negras.

## Task 1: Reglas y salas

Files: game.go, game_test.go, api.go, api_test.go.
- [ ] Escribir y ejecutar primero pruebas de reglas/sesiones relevantes.
- [ ] Implementar reglas, estado, propuestas, legal moves, PGN y endpoints acordados.
- [ ] Ejecutar go test, incluir perft y concurrencia.

## Task 2: Interfaz

Files: web/index.html, web/style.css, web/app.js, web/pieces.js.
- [ ] Implementar lobby, partida, piezas SVG, promociones, historial y temas con el contrato.
- [ ] Comprobar JavaScript y flujos con servidor real.
- [ ] Verificar visualmente escritorio y móvil, temas y orientación.

## Task 3: Ejecutable y entrega

Files: main.go, main_test.go, build.ps1, README.txt, LICENSE, THIRD_PARTY_NOTICES.txt.
- [ ] Pruebas de ejecución, assets y cierre.
- [ ] Servidor HTTP, URL LAN, apertura de navegador y cierre ordenado.
- [ ] Compilar exe nativo con paths de compilación eliminados, empaquetar exe y fuente vendorizada.
- [ ] Revisar de forma independiente y corregir hallazgos; ejecutar suite completa.
- [ ] Guardar entregables y devolver enlaces.
