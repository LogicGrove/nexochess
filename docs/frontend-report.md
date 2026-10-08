# Interfaz Nexo Chess

## Archivos

- `web/index.html`: estructura en español, lobby, sala, invitación, historial, diálogos y etiquetas accesibles.
- `web/style.css`: diseño adaptable de marfil y verde, tema oscuro, tablero, indicadores y foco de teclado. Sin fuentes ni recursos externos.
- `web/app.js`: sesión de pestaña, API local, cola de peticiones, un único sondeo programado, reconexión, creación/entrada/salida, promociones, propuestas, rendición, cierre del host y descarga PGN explícita.
- `web/pieces.js`: SVG originales para los seis tipos de pieza en ambos colores, integrados y con siluetas diferenciadas.

## Comportamiento verificado en fuente

- `a1` oscuro, coordenadas orientadas, negras abajo automáticamente y giro manual.
- El servidor es la única fuente de movimientos legales. Selección, puntos, aros de captura —incluida captura al paso—, última jugada y rey en jaque.
- Elección expresa de dama, torre, alfil o caballo. El movimiento conserva la versión al abrir el diálogo.
- Historial independiente del directo; no permite mover ni responder acciones mientras se revisa.
- Espectadores sin acciones de jugador. Presencia y resultado/motivo visibles.
- Acciones con versión esperada. No se repiten mutaciones tras errores; un rechazo HTTP provoca una resincronización inmediata. Se descartan snapshots anteriores y se actualiza presencia en versiones iguales.
- No hay solicitudes simultáneas: las peticiones se serializan. El sondeo se vuelve a programar tras finalizar y los dobles clics quedan bloqueados.
- Token únicamente en `sessionStorage`; URL con código de sala, sin token. Ningún uso de cookies, `localStorage` ni `crypto.randomUUID`.
- La navegación manual del hash mantiene la sala activa hasta una salida explícita. `lobby.activeRoom` autenticado permite recuperar una creación o entrada que se confirmó en servidor pero cuya respuesta se perdió, sin repetirla.
- Invitación con dirección LAN para el host. Al copiar en HTTP sin API de portapapeles, selecciona el enlace y explica cómo copiarlo.

## Validación

- `node --check web/app.js`: pasa.
- `node --check web/pieces.js`: pasa.
- Comprobación estática de todas las referencias DOM literales: pasa.
- Comprobación de ausencia de recursos externos y almacenamiento persistente del navegador: pasa.
- Comprobación de SVG para los doce códigos FEN y rechazo de entradas inválidas: pasa.
- Validación dinámica independiente con JSDOM y API simulada: 25/25 pruebas pasan. Incluye orientaciones, jugadas de ambos colores, captura, cuatro promociones y cancelación, jaque/mate/ahogado, historial con sondeo, dobles clics, presencia, versiones anteriores, temas, espectadores, token y recarga, propuestas, rechazo 409 con una resincronización, pérdida de membresía, portapapeles HTTP, texto de formularios seguro, enroque, captura al paso, serialización sondeo/acción, teclado y protección de la sala ante cambios del hash.
- Contraste matemático WCAG de puntos y aros: color sólido `#183d2c` en casillas claras y resaltados, `#fffceb` en casillas oscuras. Mínimo 3,23:1 frente a los ocho fondos de tablero en ambos temas; resaltados de última jugada mínimo 3,97:1. Cálculo sRGB, sin afirmación de evaluación visual en navegador.
- No se afirma verificación visual en navegador: la apertura del navegador local fue denegada por la política de seguridad de la sesión.

## Limitación de privacidad del navegador

La aplicación no guarda ajustes ni partidas automáticamente. `sessionStorage` permite recuperar el asiento al recargar la misma pestaña. Las cachés, el historial y las descargas explícitas pertenecen al navegador; el ejecutable no puede garantizar su eliminación.
