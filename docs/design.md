# Nexo Chess: diseño

App portátil para Windows 10/11 de 64 bits que sirve ajedrez por HTTP dentro de una red local. El usuario pide desarrollo autónomo, pocas preguntas, ligereza, cero instalaciones y ausencia de datos persistentes creados por la aplicación.

## Decisiones

Go compila un ejecutable nativo con HTML, CSS y JavaScript incrustados. No instala runtimes, no extrae recursos, no escribe registros, partidas ni ajustes. Utiliza una biblioteca MIT de reglas contrastada y pruebas de posiciones conocidas. El navegador utiliza sessionStorage para recuperar la sesión de la pestaña; se elimina al cerrar esa pestaña. Las huellas administradas por Windows o el navegador y las reglas de cortafuegos que acepte el usuario no se pueden prometer eliminar al borrar el exe.

Partidas sin reloj, varias salas con códigos cortos, blancas/negras, espectadores, turnos validados por el servidor. Aceptación mutua para deshacer una media jugada, reiniciar y acordar tablas. Coronación a dama, torre, alfil o caballo. Enroque y captura al paso. Jaque, mate, ahogado, material insuficiente, reclamaciones de triple repetición y 50 jugadas; final automático de cinco repeticiones/75 jugadas cuando la biblioteca lo permita. Historial SAN, posiciones FEN, navegación independiente y exportación PGN por descarga explícita.

Interfaz española adaptable a escritorio y móvil, tipografía del sistema, piezas SVG incluidas, coordenadas, selección y puntos de movimientos legales, última jugada y rey en jaque, temas claro/oscuro, orientación automática y botón de giro. Ningún recurso externo ni CDN.

## Contrato HTTP

Todas las respuestas JSON. Errores: {error:"mensaje"}. Las mutaciones requieren Content-Type application/json y X-Nexo-Client: 1, y validan Origin cuando exista. Authorization: Bearer TOKEN para lobby, salas y acciones. GET /api/info devuelve {name,version,urls:[],isHost}; POST /api/session {} devuelve {token}; GET /api/lobby devuelve {rooms:[{id,name,white,black,ply,status}],activeRoom:string|null}. activeRoom sólo informa de la membresía de la sesión solicitante y permite recuperar una creación confirmada si se perdió su respuesta, sin repetir mutaciones. POST /api/rooms {name,playerName,color:"w"|"b"} devuelve snapshot. POST /api/join {room,playerName,color:"w"|"b"|"spectator"} devuelve snapshot. GET /api/state?room=ID devuelve snapshot. POST /api/action {room,action,from?,to?,promotion?,accept?,version?} devuelve snapshot. Actions: move, undo, restart, draw, respond, resign, leave, claim. POST /api/shutdown únicamente cliente loopback autenticado.

Snapshot: {id,name,you:"w"|"b"|"spectator",white:{name,online}|null,black:{name,online}|null,fen,turn:"w"|"b",check:boolean,outcome:"*"|"1-0"|"0-1"|"1/2-1/2",reason:string,legal:[{from,to,promotion?}],history:[{san,from,to,fen}],initialFen,version:number,pending:{kind:"undo"|"restart"|"draw",by:"w"|"b"}|null,canClaim:boolean,pgn:string}. room IDs and session tokens never confer host privilege. Name limit 32 Unicode runes, room cap 32. API handlers serialize mutations under a mutex, release it before network writes. State polling 1000 ms; no duplicate polling or events while rebuilding UI. Presence expires after 15 seconds; seat is retained for reconnection for 2 minutes offline, then reclaimable, and can be explicitly left. Frontend supplies snapshot.version in action bodies; stale version returns HTTP 409 without mutating the room. Leave does not require a matching version.

## Validación

Pruebas de reglas con perft, mate, ahogado, enroque prohibido a través de jaque, captura al paso clavada, cuatro promociones, repetición y undo. Pruebas HTTP con dos sesiones y espectadores, movimientos ilegales/fuera de turno, respuestas a propuestas, reconexión, concurrencia y entradas malformadas. Navegadores reales para partida completa, puntos, orientación, historial, temas y tamaño móvil. Compilar exe, ejecutarlo desde carpeta vacía, comprobar que todos los assets están embebidos y que cerrar libera el puerto.
