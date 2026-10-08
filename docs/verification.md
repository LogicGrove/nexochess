# Verificación de entrega 1.0.0

- go test -mod=vendor ./... -count=1: PASS.
- go vet -mod=vendor ./...: PASS.
- build.ps1: PASS; ejecutable Windows x64 con CGO_ENABLED=0, trimpath y assets incrustados.
- node --check web/app.js y web/pieces.js: PASS.
- Node/JSDOM: 27 pruebas PASS, ninguna red real ni navegador. Orientaciones, casillas, puntos, capturas, enroque, EP, cuatro promociones, historial, turnos, espectador, presencia, reconexión, temas, propuestas, solicitudes serializadas, hash y creación con respuesta perdida.
- Reglas: perft de posiciones conocidas, 14 suites independientes de casos especiales y regresiones de dos defectos de dependencia; detalles en rules-qa-report.md.
- Concurrencia de la API repetida 50 veces: PASS. Se garantiza liberación del mutex antes de escribir a la red; respuestas atrasadas se rechazan por versión.
- EXE nativo iniciado desde carpeta vacía: PASS. Prueba JSON con dos identidades y un espectador, mate del loco, SAN/PGN, deshacer acordado, recuperación de estado, activeRoom y apagado: PASS.
- Al cerrar, carpeta de trabajo vacía y puerto libre: PASS. El ejecutable no escribe configuración ni partidas. No se afirma ausencia de huellas administradas por Windows/navegador.
- Assets del EXE comparados byte a byte con los cuatro archivos actuales de web: PASS.
- Revisión independiente del servidor y revisión final: aprobadas sin incidencias críticas o importantes abiertas; informes incluidos.

## Límites comprobados

El navegador real rechazó abrir la vista previa por permiso denegado; no se realizó revisión visual renderizada, prueba real de móvil/tacto ni descarga PGN desde navegador. La partida nativa se probó en un ordenador con sesiones HTTP separadas; no se dispone de un segundo ordenador físico para comprobar su cortafuegos/router. Los tests DOM utilizan snapshots y fetch simulado.

El detector Go -race requiere un compilador C que no está disponible; se ejecutaron pruebas concurrentes ordinarias. La aplicación final no necesita compilador C, Go, Node, JSDOM ni Python.

Los límites de posiciones muertas excepcionales y reclamaciones con jugada prevista están explicados en README.txt y vendor-patches.md. La app está orientada a partidas casuales en LAN.

Para repetir la prueba nativa: iniciar dist/NexoChess.exe --no-browser --port 18088 desde una carpeta vacía y ejecutar python tests/native_smoke.py. La prueba apaga el servidor al finalizar.

Para los tests DOM, instalar JSDOM 27 como herramienta de desarrollo y ejecutar node --test tests/frontend.test.cjs (NEXO_JSDOM_PATH puede indicar su ruta). Estas herramientas no forman parte del ZIP de la aplicación.
