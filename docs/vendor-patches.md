# Parche de reglas incorporado

La copia vendorizada de github.com/corentings/chess/v2 v2.6.0 contiene un cambio en position.go: samePosition compara las posiciones conforme a FIDE 9.2.3 sin utilizar el hash Polyglot como filtro, y relevantEnPassantSquare sólo tiene en cuenta capturas al paso legales. Los hashes Polyglot conservan su comportamiento original.

Una captura al paso que deja al rey en jaque no altera los movimientos posibles y no debe distinguir dos apariciones de la misma posición. La versión original comprobaba peones adyacentes sin comprobar la clavada, retrasando la reclamación por triple repetición y las tablas automáticas por cinco repeticiones.

Reproducción: FEN 1n2r1k1/8/8/3pP3/8/8/8/4K1N1 w - d6 0 1; repetir dos veces g1f3 b8c6 f3g1 c6b8. TestAdditionalRulesPinnedEnPassantRepetition falla con la dependencia original y pasa con el parche.

Compilar siempre con -mod=vendor (build.ps1 lo hace). Volver a ejecutar go mod vendor reemplaza este parche; en ese caso hay que reaplicarlo y ejecutar las pruebas.

Un segundo cambio en game.go contempla la excepción de rendición contra un rival que sólo conserva el rey: el resultado es tablas, porque el rey solo no puede dar jaque mate (FIDE 5.1.2). TestAdditionalRulesResignationBareKingException prueba ambos colores.

Referencia oficial: https://handbook.fide.com/chapter/E012023

Alcance de tablas: mate y ahogado se determinan mediante movimientos legales. El material insuficiente se detecta en los casos habituales; no se realiza una búsqueda exhaustiva de todas las posiciones muertas posibles con peones bloqueados. Para esos casos los jugadores pueden acordar tablas. La app está orientada a partidas casuales en LAN, sin reloj ni arbitraje de torneo.
