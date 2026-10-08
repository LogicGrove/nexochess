# Validación dinámica del frontend

Fecha: 6 de octubre de 2026. Resultado: **27 pruebas, 27 correctas, 0 fallos**. La ejecución final terminó con código 0, sin errores DOM observados. No se detectaron defectos en los escenarios cubiertos.

Se ejecutó el HTML y los JavaScript reales de `web/` en Node.js 24.19.0 y JSDOM 27.0.0. Se cargaron los archivos mediante `fs` y se usó `runScripts: "outside-only"`, sin habilitar carga de recursos. Todas las peticiones fueron respuestas JSON simuladas de `window.fetch`; no hubo servidor real, sockets, navegador, automatización de interfaz ni solicitudes externas. El sondeo y los tiempos de espera se controlaron de forma manual.

Las pruebas interactúan con botones, formularios, teclado y atributos DOM públicos. Las posiciones FEN y SAN son ejemplos concretos de apertura, captura, enroque, captura al paso, coronación, jaque, mate y ahogado. No se copió la lógica interna de renderizado ni de generación de movimientos de la aplicación.

## Cobertura

| Área | Comprobaciones |
| --- | --- |
| Tablero y piezas | 64 coordenadas únicas; orientación automática para ambos colores; giro; a1 oscura antes y después; piezas SVG blancas/negras; jugador propio abajo. |
| Movimientos | Destinos enviados por el servidor; rechazo local de destinos ilegales; movimiento blanco y negro; cuerpo con sala, origen, destino y versión; autenticación y cabeceras JSON. |
| Casos de ajedrez | Captura con anillo; enroque con reubicación de rey y torre; captura al paso sobre destino vacío con eliminación del peón; cuatro coronaciones reales, incluidas subcoronaciones; cancelación sin mutación. |
| Estado de partida | Jaque en texto y casilla del rey; resultado de mate; tablas por ahogado; controles y tablero bloqueados al terminar. |
| Historial | Posiciones FEN anteriores reales; inicio, atrás, adelante y directo; imposibilidad de mover durante revisión; sondeo conserva revisión; vuelta al directo recupera destinos legales. |
| Concurrencia | Doble clic envía una sola mutación; controles bloqueados durante petición; sondeo detenido y reiniciado una sola vez; clic durante sondeo pendiente queda en cola; máximo una petición activa. |
| Versiones y propuestas | Presencia actualizada con versión igual; respuestas antiguas descartadas; aceptar/rechazar propuesta con versión exacta; propuesta propia sin respuesta; oferta de tablas versionada; HTTP 409 refresca una sola vez sin repetir la jugada. |
| Sesión y conexión | Reconexión de pestaña con `sessionStorage`; recarga conserva color y sala sin sesión extra; desconexión bloquea juego y recuperación restaura controles; pérdida de pertenencia prepara formulario para reentrar; creación ya confirmada con respuesta perdida recupera sala, asiento, orientación y hash mediante `lobby.activeRoom` sin repetir create/join; lista de salas sin `activeRoom` no incorpora al usuario a otra partida. |
| Navegación y permisos | Espectador sin movimientos ni acciones; cambio/borrado manual del hash conserva pertenencia incluso con mutación pendiente; nombres se muestran como texto; formularios envían nombre y color elegidos. |
| Interacción | Tema y etiqueta accesible; flechas según orientación visual, límites, Home/End y Escape; un único foco de tablero; selección exacta del enlace al copiar por HTTP sin contexto seguro. |

## Reproducción

Desde la raíz del proyecto:

```powershell
node --test tests/frontend.test.cjs
```

JSDOM es una dependencia exclusiva de estas pruebas de desarrollo. Puede encontrarse con la resolución habitual de Node o mediante `NEXO_JSDOM_PATH` apuntando a su módulo instalado. En esta validación se utilizó `D:\Temp\ajedrez-build\dom\node_modules\jsdom`. No se añadió JSDOM al código Go, al ejecutable portátil ni a sus recursos.

## Límites

Esta validación no acredita apariencia visual, tamaño móvil, geometría CSS, colores renderizados, interacción táctil, foco nativo de diálogos, descarga PGN real ni comportamiento de portapapeles/almacenamiento de navegadores concretos. El acceso al navegador para la vista local estaba denegado; no se intentó una alternativa que eludiera esa restricción. Las reglas de ajedrez y el servidor HTTP real requieren sus propias pruebas y no quedan validados por los snapshots simulados.

El reloj del sondeo es manual: se verificó su programación, cancelación y serialización, no el tiempo de pared de una red real. La reconexión cubierta es la recuperación de un token válido, de un fallo temporal de conexión y de una creación confirmada cuyo snapshot no llega al cliente. La confirmación del servidor en este último caso es una simulación; no se ejercitó un fallo real de transporte. No se simularon cierres reales del proceso, caducidad prolongada de plazas ni todos los errores posibles del servidor.

## Archivos examinados

SHA-256 de los archivos del frontend presentes durante la ejecución. El CSS queda identificado, aunque JSDOM no lo cargó ni renderizó:

| Archivo | SHA-256 |
| --- | --- |
| `web/app.js` | `D869CDA6C177B7A82EF929E8D27E3A58D2300DBFC54960A24C3D0637ACAE4395` |
| `web/index.html` | `F1A20FE0707B6C88A07DCD080A3DB40C605AA98DA4383B9420A5D90F48ECFBB3` |
| `web/pieces.js` | `1B574B867A7A4313BF7B08C0CDE1D6DF8ADBB80217B49C689057A84BC5F169D0` |
| `web/style.css` | `12C6FAFEC926F54786095D9EDB2A061F1D6F0455DD89795AEDCF8B3C54FE179D` |
