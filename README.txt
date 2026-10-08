NEXO CHESS 1.0 · AJEDREZ EN TU RED
Windows 10/11 de 64 bits. No necesita instalar Python, Node, .NET ni Go.

PARA JUGAR
1. Extrae el ZIP en una carpeta y abre NexoChess.exe.
2. Se abre tu navegador. Escribe tu apodo, crea una mesa y elige color.
3. Comparte la dirección de red que aparece en la web o en la ventana.
   Ejemplo: http://192.168.1.20:8088 (usa la tuya, no esta de ejemplo).
4. En el otro ordenador, abre esa dirección y entra en la misma mesa.
5. Mantén el programa abierto en el ordenador anfitrión.

No necesita internet. Solo el anfitrión ejecuta el EXE; el otro equipo
necesita un navegador actual. También puedes entrar desde móvil/tableta.

SI EL OTRO EQUIPO NO PUEDE ENTRAR
- Ambos deben estar en la misma red. Evita el Wi-Fi de invitados: algunos
  routers aíslan sus dispositivos aunque tengan el mismo nombre de red.
- Si Windows pregunta, permite Nexo Chess en redes privadas.
- Puede haber varias direcciones si usas VPN o adaptadores virtuales.
  Prueba la del adaptador que conecta con tu Wi-Fi o Ethernet.
- Si cambias la red mientras el programa está abierto, recarga la web del
  anfitrión para actualizar las direcciones que puedes compartir.
- No abras puertos del router: el programa está destinado a una LAN.
- Si el puerto habitual está ocupado, el programa busca el siguiente libre.

EL TABLERO
Pulsa una pieza para ver puntos con sus destinos legales y un aro en las
capturas. Pulsa un destino para mover. La coronación siempre permite elegir
dama, torre, alfil o caballo. Se validan enroque y captura al paso.
Se muestran jaque, mate, ahogado, tablas y resultado de la partida.

REBOBINAR
Las flechas del historial permiten revisar jugadas sin cambiar la partida.
Vuelve a 'En directo' para mover. 'Deshacer' solicita retirar la última
media jugada y necesita la aceptación del rival. Reiniciar y acordar tablas
también requieren aceptación. Puedes exportar un archivo PGN cuando quieras.

CIERRE Y DATOS
Apaga el servidor desde la web del anfitrión, pulsa Ctrl+C en su ventana o
cierra la ventana. Al cerrar se pierden las partidas; exporta el PGN si
quieres conservarlas. Recargar una pestaña mantiene tu asiento durante la
sesión del servidor. Cerrar la pestaña elimina su sesión temporal.

Si cierras una pestaña sin salir de la mesa, el asiento se reserva durante
dos minutos de desconexión. Después otro jugador puede ocuparlo. El botón
de salir lo libera inmediatamente.

El programa no instala servicios, no modifica el registro, no crea archivos
de configuración, no guarda partidas ni descarga recursos al ejecutarse.
Todo lo necesario va dentro del EXE. Para quitarlo, cierra la app y borra
la carpeta que extrajiste.

Windows y tu navegador pueden conservar sus propias huellas (historial,
archivos recientes, etc.). Si autorizas una regla de cortafuegos, Windows
puede conservarla después de borrar el EXE; puedes quitarla en 'Firewall de
Windows Defender > Permitir una aplicación'. Borrar un ejecutable no permite
garantizar la eliminación de todas las huellas que gestione el sistema.
El programa no añade ni elimina reglas de cortafuegos por su cuenta.

Se detectan los casos habituales de material insuficiente. Las posiciones
muertas excepcionales con peones totalmente bloqueados pueden requerir
acordar tablas. La reclamación de repetición/50 jugadas está disponible una
vez alcanzada la posición; no se declara una jugada prevista ante árbitro.

Opciones: NexoChess.exe --no-browser / --port 8099 / --version
Proyecto y licencia: MIT. Dependencias y avisos en THIRD_PARTY_NOTICES.txt.
