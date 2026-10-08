# Nexo Chess · Guía avanzada

**Español** | [English](README.advanced.en.md) · [Volver al inicio](../README.md)

Esta publicación contiene **Nexo Chess 1.0.0**, la versión original para partidas entre personas en una LAN, sin reloj ni IA. El código de la aplicación y el ZIP ejecutable originales se conservan; se añade la presentación para GitHub.

## Uso y conexión

El anfitrión ejecuta `NexoChess.exe` en Windows 10/11 x64. La app abre el navegador local y muestra las direcciones de red que pueden compartir los jugadores. Los otros dispositivos necesitan únicamente un navegador actual con JavaScript.

El puerto predeterminado es **8088**; si está ocupado, el programa busca el siguiente disponible. Usa siempre la dirección mostrada por tu instancia. Por ejemplo, `http://192.168.1.20:8088` es solo un ejemplo y no la dirección de todos los equipos.

Opciones desde PowerShell, en la carpeta del ejecutable:

```powershell
.\NexoChess.exe --no-browser
.\NexoChess.exe --port 8099
.\NexoChess.exe --version
```

Si no conecta, comprueba que ambos dispositivos estén en la misma red, que el router no aísle los clientes y que Windows permita la app en redes privadas. Las VPN y adaptadores virtuales pueden añadir direcciones que no sirven para tu Wi-Fi. Tras cambiar de red, recarga la página del anfitrión para actualizar las direcciones.

La app usa HTTP local y está pensada para una red de confianza. No requiere internet ni abrir puertos en el router. Solo el anfitrión, desde su conexión local, puede apagar el servidor con el botón de la web; también puede pulsar Ctrl+C o cerrar la ventana.

## Mesas y tablero

Puedes crear varias mesas, elegir color, unirte a una plaza libre o mirar como espectador. Al seleccionar una pieza aparecen puntos en los destinos legales y aros en las capturas. La coronación ofrece cuatro piezas.

El historial permite revisar posiciones con sus flechas y volver a **En directo** para mover. Rebobinar no modifica la partida. **Deshacer** retira una media jugada tras la aceptación del rival; reiniciar y acordar tablas también requieren su aceptación.

Al recargar la pestaña se conserva el asiento durante la sesión del servidor. Si se cierra sin salir de la mesa, el asiento se reserva durante dos minutos de desconexión; después puede ocuparlo otro jugador. Salir de la mesa lo libera inmediatamente.

## Datos y portabilidad

Las mesas y partidas viven en memoria. Se pierden al cerrar el servidor y no se recuperan al abrirlo de nuevo. Exporta el PGN antes de terminar para guardar o estudiar una partida.

Todo lo necesario para ejecutar la app va dentro del EXE. No instala servicios, modifica el registro, guarda configuración ni descarga recursos al ejecutarse. Para quitarla, cierra el programa y borra su carpeta.

Windows y el navegador pueden conservar historial, archivos recientes u otras huellas propias. Si autorizaste una regla de cortafuegos, Windows puede conservarla; se gestiona desde «Firewall de Windows Defender → Permitir una aplicación». La app no crea ni elimina esas reglas por su cuenta.

## Reglas y límites

Se validan enroque, captura al paso, coronación, jaque, mate y ahogado. Se detectan los casos habituales de material insuficiente; las posiciones muertas excepcionales con peones bloqueados pueden requerir acordar tablas. Las reclamaciones por repetición o 50 jugadas se hacen cuando ya se alcanzó la posición, sin declaración de una jugada prevista ante árbitro.

La dependencia de reglas incluida incorpora dos correcciones locales: identidad de posición para repeticiones con captura al paso legal y rendición contra un rival que solo tiene rey. Consulta [los parches originales](vendor-patches.md). La app está orientada a partidas informales.

## Código y compilación

| Ruta | Contenido |
| --- | --- |
| `main.go` | Arranque Windows, recursos incrustados y servidor HTTP local |
| `api.go` | Sesiones, mesas, solicitudes y estado en memoria |
| `game.go` | Reglas, historial y exportación PGN |
| `web/` | Interfaz HTML, CSS y JavaScript; piezas SVG propias |
| `vendor/` | Dependencia de ajedrez y sus correcciones incluidas |
| `*_test.go`, `tests/` | Pruebas del servidor, reglas e interfaz |
| `docs/` | Esta guía e informes originales de desarrollo y verificación |

Para desarrollar necesitas **Go compatible con 1.25 o posterior** y PowerShell en Windows. Para jugar no necesitas Go, Python, Node ni .NET.

Desde la raíz del proyecto:

```powershell
.\build.ps1
```

El script ejecuta las pruebas y genera `dist\NexoChess.exe` para Windows x64 con recursos incrustados y sin CGO. Usa `-mod=vendor`; **`go mod vendor` reemplazaría las correcciones locales**, por lo que tendrías que reaplicarlas y comprobarlas. El código de arranque es específico de Windows.

Comprobaciones adicionales:

```powershell
go test -mod=vendor ./... -count=1
go vet -mod=vendor ./...
node --check web/app.js
node --check web/pieces.js
# Instala JSDOM 27 solo como herramienta de desarrollo:
npm install --no-save --package-lock=false jsdom@27
node --test tests/frontend.test.cjs
```

`tests/native_smoke.py` permite comprobar la API con el EXE iniciado con `--no-browser --port 18088`; la prueba apaga el servidor al terminar. Consulta [la verificación original](verification.md) para los resultados y límites de aquellas pruebas, incluida la ausencia de una prueba entre dos ordenadores físicos. Son informes de la entrega original, no garantías para todas las redes.

## Licencias

El proyecto conserva su [licencia MIT](../LICENSE) y [avisos de terceros](../THIRD_PARTY_NOTICES.txt). La dependencia vendorizada conserva su licencia propia. El ZIP de la release se publica tal como se recibió en las fuentes originales.
