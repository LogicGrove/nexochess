# Revisión final independiente de Nexo Chess

6 de octubre de 2026. Inspección de design.md, plan.md, server-review.md, vendor-patches.md, main.go, api.go, game.go, web/*, pruebas e informes de los implementadores. Esta revisión no modifica código de producción.

**Requisitos: aprobado para empaquetar.** La arquitectura satisface el uso casual en LAN de Windows x64: ejecutable Go con recursos embebidos, sin runtimes adicionales, instalaciones, servicios ni archivos de aplicación creados automáticamente. Reglas y turnos autoritativos, salas, espectadores, consentimiento, promociones, historial, exportación y reconexión son coherentes entre API e interfaz. La entrega conserva las salvedades documentadas sobre huellas de Windows/navegador y descargas explícitas. La compilación y prueba de arranque/cierre del ejecutable definitivo corresponden a la integración de entrega.

**Calidad: aprobado, sin incidencias críticas o importantes abiertas.** Las dos incidencias importantes encontradas durante esta revisión se corrigieron y se volvieron a inspeccionar. La actualización de direcciones tras un cambio de red queda como limitación menor documentada, sin bloquear la entrega.

## Hallazgos concretos y resolución

| Gravedad | Función y escenario | Resultado de la revisión |
| --- | --- | --- |
| Importante, corregida | `routeFromHash`: borrar el hash o abrir otra invitación ocultaba la sala actual sin liberar el asiento del servidor. Crear o entrar en otra sala devolvía 409 y desaparecía el control para salir. | Conserva la sala y restaura su hash hasta una salida explícita. Se comprobó que la protección ocurre también antes de la salida temprana por una mutación pendiente. La prueba DOM cubre ambos casos. |
| Importante, corregida | Creación confirmada en servidor cuya respuesta se pierde: el cliente permanecía sin sala conocida, mientras el token retenía un asiento y bloqueaba nuevos intentos. | `/api/lobby` devuelve `activeRoom` exclusivamente para la pertenencia válida del token autenticado. `refreshLobby` recupera el snapshot y el hash mediante lectura, sin repetir la mutación. La prueba HTTP reproduce una escritura fallida con `io.ErrClosedPipe`, comprueba recuperación, aislamiento entre tokens y limpieza al salir. |
| Menor, corregida | `.legal-dot` y `.legal-ring`: en casilla oscura del tema oscuro, los colores originales tenían contraste calculado de 1,24:1 y 1,36:1. | Colores sólidos claros/oscuros según la casilla y la última jugada. Cálculo independiente de los ocho pares: mínimo 3,23:1; tema oscuro/casilla oscura 4,61:1. |
| Menor, corregida | Confirmación de `resign-action`: prometía victoria rival incluso cuando solo quedaba su rey. | Texto neutral sobre la rendición; el servidor mantiene correctamente la excepción de tablas. |
| Menor, corregida | `THIRD_PARTY_NOTICES.txt`: separadores literales `\\r\\n` y descripción incompleta de cambios vendorizados. | Saltos de línea reales y mención de los parches de repetición y rendición. |
| Menor, documentada | `poll`/`renderInfo`: `state.info` se conserva tras el arranque. Si DHCP o el cambio de Wi-Fi altera la IPv4 del anfitrión, las direcciones e invitaciones mostradas siguen siendo antiguas hasta recargar. | README indica recargar la web del anfitrión después de cambiar de red para actualizar las direcciones compartidas. El backend admite las direcciones actuales; recargar conserva el token y recupera el asiento. |

## Comprobación de conjunto

La matemática del tablero es correcta: `a1` oscura; con blancas abajo la fila inferior es `a1…h1`; con negras abajo es `h8…a8`, con `a8` clara en la esquina inferior derecha. Las piezas permanecen verticales y sus SVG originales diferencian peón, torre, caballo, alfil, dama y rey mediante siluetas y detalles. Ambos colores tienen rellenos y contornos opuestos. El CSS usa ocho columnas iguales, proporción cuadrada y una sola columna de interfaz en móvil; no se encontró un error estático de dimensiones o de giro.

La selección histórica impide mutaciones; las promociones conservan la versión al abrirse; las respuestas antiguas se descartan y las de igual versión actualizan presencia. La cola de solicitudes y los bloqueos de interacción evitan duplicación de movimientos. Se revisaron roles, propuestas, salida, reconexión, descarga PGN, temas, pérdida de conexión, recursos locales, listener y cierre. No se encontraron nuevas incidencias importantes en esos recorridos.

## Evidencia y límites

El informe del servidor registra `go test ./... -count=1` aprobado en 0,723 s y `go vet ./...` aprobado, incluyendo la regresión de creación con respuesta perdida. El informe de interfaz registra 25 pruebas DOM aprobadas mediante JSDOM y API simulada; la integración está añadiendo la regresión DOM específica de respuesta de creación perdida. Esta revisión comprobó directamente el código final de ambos lados de esa recuperación y calculó los contrastes, sin repetir las suites completas.

No se utilizó un navegador real ni una captura visual: esa acción fue denegada por la herramienta. El análisis de SVG, CSS y DOM no certifica el aspecto renderizado ni una conexión entre dos dispositivos físicos. El detector de carreras no se ejecutó por ausencia de compilador C; las pruebas concurrentes existentes no lo sustituyen. Se aceptan los límites casuales documentados: sin reloj, sin reclamación por jugada anunciada y sin búsqueda exhaustiva de posiciones muertas con peones bloqueados.
