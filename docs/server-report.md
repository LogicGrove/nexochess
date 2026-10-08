# Informe del servidor de Nexo Chess

Implementados `game.go`, `game_test.go`, `api.go` y `api_test.go`. El estado de sesiones, asientos, espectadores y partidas permanece en memoria. La aplicación no escribe archivos durante su ejecución.

## Comportamiento

- API JSON autoritativa con turnos, movimientos legales, jaque, finales, historial SAN/FEN, PGN y cuatro promociones.
- Deshacer reconstruye el motor desde la secuencia UCI completa y elimina la jugada acordada: conserva las repeticiones correctas y elimina las variantes descartadas.
- Deshacer, reiniciar y acordar tablas requieren la respuesta del otro jugador. Mover, rendirse o abandonar cancela las propuestas pendientes.
- Las acciones pueden incluir `version`; un estado obsoleto devuelve 409, excepto al abandonar. El cliente incluye la versión para impedir que una respuesta antigua acepte una propuesta posterior.
- Un espectador no puede realizar acciones de jugador. Un token conserva su asiento y nombre al reconectar; no puede ocupar ambos colores ni aceptar su propia propuesta.
- Presencia durante 15 segundos. Los asientos se reservan durante dos minutos desde la última actividad; después otro token puede ocuparlos. Abandonar libera el asiento inmediatamente.
- Límites de 32 salas, 512 sesiones y 64 espectadores por sala. Sesiones y salas inactivas caducan a las 24 horas. Una sala vacía se elimina inmediatamente.
- Nombres limitados a 32 runas Unicode, sin caracteres de control y con espacios exteriores eliminados.
- Mutaciones exigen MIME JSON, `X-Nexo-Client: 1` y un `Origin` coincidente con esquema, host y puerto cuando existe. Bearer identifica la sesión y se valida la pertenencia a la sala. El servidor principal aplica además la lista de hosts locales permitidos.
- Cuerpos limitados a 4096 bytes; se rechazan campos desconocidos, JSON múltiple, inválido y `null`.
- El mutex protege tanto las lecturas como las mutaciones del motor. La respuesta se serializa en memoria y se escribe a la red después de liberar el mutex, para que un cliente lento no bloquee las otras salas.
- Los arrays vacíos se codifican como `[]`.
- El lobby incluye `activeRoom: string|null` de la sesión autenticada que lo consulta. Permite recuperar una creación de sala confirmada aunque se pierda su respuesta HTTP, mediante una lectura de estado y sin repetir la mutación. Otra sesión sin sala recibe `null`; la consulta sin autenticación sigue devolviendo 401.

## Verificación

Se escribieron pruebas antes de implementar los endpoints; su primera ejecución falló por ausencia de `App`/`Snapshot`. Las regresiones de JSON `null`, recuperación de asiento, respuesta de red lenta y versión obsoleta se observaron antes de aplicar las correcciones.

Pruebas de reglas: perft de inicio (8902 nodos), Kiwipete (97862) y una posición de torres y peones (2812), todas a profundidad 3. También mate, ahogado, material insuficiente, enroque a través de jaque, captura al paso clavada, cuatro promociones, triple/cinco repeticiones, reglas de 50/75 jugadas y undo.

Pruebas HTTP: dos jugadores y espectadores, identidad y autorización, legalidad y turnos, consentimiento, rechazo, reinicio, rendición, salida, nombres Unicode, arrays vacíos, presencia, recuperación y reclamación de asientos, caducidad, capacidad, entradas malformadas, cliente lento y versiones antiguas.

Las pruebas de concurrencia comprueban un único movimiento ante 20 solicitudes iguales, una única aceptación de undo ante 20 respuestas y 200 lecturas concurrentes. El grupo de concurrencia, respuesta lenta, versiones y reloj se repitió 50 veces: pasó en 1,037 s. `go vet game.go api.go` terminó con código 0.

La suite global final `go test ./... -count=1` pasó en 0,723 s, con código de salida 0. Incluye las regresiones independientes, la explicación JSON de tablas por rendición en ambos colores y la recuperación de una creación cuya conexión falla al escribir la respuesta. La prueba de recuperación comprobó primero el fallo por ausencia de `activeRoom`, y verifica también el aislamiento entre tokens, la sesión sin sala y la limpieza al abandonar. `go vet ./...` terminó también con código 0.

## Dependencia y limitaciones de verificación

La revisión independiente detectó errores del motor vendorizado en la identidad FIDE de posiciones con captura al paso ilegal y en la rendición contra un rey solo. Los parches y sus pruebas se documentan en `docs/vendor-patches.md`; la aplicación usa los métodos corregidos de la dependencia directamente.

No se pudo ejecutar el detector de carreras: `go test -race` exige cgo y, al activarlo, el entorno respondió `C compiler "gcc" not found`. Las pruebas de concurrencia sí se ejecutaron y pasaron, pero no sustituyen el detector de carreras.
