# Nexo Chess: verificación independiente de reglas

Fecha: 6 de octubre de 2026. Base: `github.com/corentings/chess/v2 v2.6.0`, incluida en `vendor`, con los parches de Nexo descritos en `vendor-patches.md`.

La verificación cubre la generación de movimientos y la integración de `game.go`. Se añadieron 14 familias de pruebas en `additional_rules_test.go`, independientes de los cuatro tests originales. Las comprobaciones de legalidad contrastan la lista de movimientos ofrecidos y la ejecución real; también exigen que rechazar una jugada conserve FEN, resultado, motivo e historial.

## Pruebas añadidas

| Test | Comprobaciones concretas |
| --- | --- |
| `TestAdditionalRulesPawnMovementAndPins` | Avance simple y doble de ambos colores; bloqueo del camino y del destino; captura diagonal; prohibición de captura hacia delante, diagonal vacía, retroceso y doble avance posterior; torre y caballo clavados; captura del atacante; reyes adyacentes; rey entrando en ataque de una pieza clavada. |
| `TestAdditionalRulesCheckUsesAttacks` | El avance de peón no produce jaque; sus diagonales sí, para ambos colores; torre bloqueada y libre; ataques de torre y caballo clavados; ausencia de salto entre columnas a/h. |
| `TestAdditionalRulesCastlingAttackConditions` | Enroque desde, a través de y hacia jaque; enroque permitido con torre atacada; b1 atacada no impide enroque largo, pero b1 ocupada sí; derechos perdidos tras devolver rey o torre a su origen; ejemplo equivalente de negras. |
| `TestAdditionalRulesCastlingMovesBothPieces` | Los cuatro enroques colocan rey y torre en las casillas correctas. |
| `TestAdditionalRulesEnPassantRemovalExpiryAndKingSafety` | Captura al paso blanca y negra retira el peón adecuado; caduca tras una jugada distinta; se rechaza si descubre una torre en la columna o en la fila del rey. |
| `TestAdditionalRulesAllPromotionChoicesBothColors` | Dama, torre, alfil y caballo; blancas y negras; avance y captura; rechazo de coronación omitida o a rey/peón. |
| `TestAdditionalRulesCheckMateAndStalemateAreDistinct` | Mate blanco y negro; ahogado blanco y negro; jaque con escapes; prioridad del mate sobre 150 semijugadas. |
| `TestAdditionalRulesMaterialOutcomes` | K/K, KB/K, KN/K y alfiles de un solo color producen tablas; alfiles de colores distintos, KNN/K, KN/KN, KBN/K y un peón no producen tablas prematuras; posición de mate con dos caballos. |
| `TestAdditionalRulesFiftyMovesAndResets` | Reclamación al llegar a 100 semijugadas; no antes; avance de peón y captura reinician el contador. |
| `TestAdditionalRulesResignationBareKingException` | La rendición blanca o negra contra un rey rival sin ninguna otra pieza produce tablas. |
| `TestAdditionalRulesPinnedEnPassantRepetition` | Triple repetición con captura al paso ilegal; reclamación y motivo; deshacer elimina el resultado y la repetición retirada; repetir la jugada restaura la reclamación. |
| `TestAdditionalRulesEnPassantIdentityAndFivefold` | Una captura al paso legal diferencia posiciones; un objetivo sin peón adyacente o con captura clavada no; la triple repetición requiere reclamación; cinco repeticiones finalizan automáticamente; motivo correcto, rechazo de jugada tras el final y recuperación por deshacer. |
| `TestAdditionalRulesCastlingRightsSeparateRepetitions` | Mismo tablero y turno con diferentes derechos de enroque no constituyen una repetición. |
| `TestAdditionalRulesPerftIndependentFixtures` | Posiciones estándar 4, 5 y 6 a profundidades 1, 2 y 3; división exacta por los 20 movimientos iniciales a profundidad 2. |

El cálculo de `inCheck` usa `Board.AttacksFrom`, no los movimientos legales del rival. Las posiciones específicas de peones y piezas clavadas comprueban esa distinción.

## Perft

Los valores se contrastaron con [las pruebas de Chess-Plisco](https://github.com/gflohr/Chess-Plisco/blob/main/t/05perft.t). Las mismas posiciones aparecen en [la suite oficial de Stockfish](https://github.com/official-stockfish/Stockfish/blob/master/tests/perft.sh), a mayor profundidad.

| Posición | Profundidad 1 | Profundidad 2 | Profundidad 3 |
| --- | ---: | ---: | ---: |
| 4: promociones, enroques y clavadas | 6 | 264 | 9467 |
| 5: promoción con jaque | 44 | 1486 | 62379 |
| 6: medio juego | 46 | 2079 | 89890 |

FEN exactas conservadas en el test. La suite original añade posición inicial (8902), Kiwipete (97862) y final de torre/peones (2812), a profundidad 3. Perft verifica movimientos, sin finalizar anticipadamente por reclamaciones o tablas de material.

## Defectos reproducidos y corregidos

**Repetición y captura al paso clavada.** FEN inicial:

```text
1n2r1k1/8/8/3pP3/8/8/8/4K1N1 w - d6 0 1
```

Repetir dos veces `g1f3 b8c6 f3g1 c6b8`. La jugada `e5d6` es ilegal porque descubre la torre e8 contra el rey e1. Antes del parche, la biblioteca devolvía `EligibleDraws=[DrawOffer]` y `claimable=false`: diferenciaba el campo FEN d6 aunque no existía captura legal. La prueba falló primero y pasa tras el parche de igualdad de posiciones en `vendor/.../position.go`, incluyendo triple y quíntuple repetición. Referencia normativa: [FIDE 9.2.3](https://handbook.fide.com/chapter/E012023).

**Rendición ante un rey solo.** Con `7k/8/8/8/8/8/R7/K7 w - - 0 1`, `Resign(White)` devolvía `0-1`. Con `7k/7r/8/8/8/8/8/K7 b - - 0 1`, `Resign(Black)` devolvía `1-0`. El rival carece de posibilidad de dar mate; corresponde tablas. Ambos tests fallaron primero y pasan tras el parche de `vendor/.../game.go`. El motivo conserva `Resignation`. Referencia: [FIDE 5.1.2](https://handbook.fide.com/chapter/E012023).

## Límites comprobados

La detección automática de material cubre los finales básicos de la tabla. No resuelve todas las posiciones muertas de FIDE 5.2.2. Sondeo ejecutado:

```text
7k/8/p1p1p1p1/PpPpPpPp/1P1P1P1P/8/8/7K w - - 0 1
Resultado actual: outcome=* method=NoMethod
```

Aquí los peones no pueden avanzar ni capturar y separan a los reyes mediante barreras infranqueables. Ningún rey puede capturar el muro. La biblioteca continúa porque su comprobación de material considera suficiente cualquier peón. El parche de rendición solo añade la excepción demostrable del rey rival solo; tampoco pretende resolver toda imposibilidad de mate en posiciones bloqueadas.

Las reclamaciones se ofrecen por la posición y el contador actuales. La API no permite indicar previamente la jugada que producirá la tercera repetición o las 100 semijugadas (FIDE 9.2.1 y 9.3.1). Sondeos ejecutados:

- Desde el inicio, `g1f3 g8f6 f3g1 f6g8 g1f3 g8f6 f3g1`: `f6g8` es legal y produciría la tercera posición inicial; antes de jugarla, `claimable=false`.
- `7k/8/8/8/8/8/R7/K7 w - - 99 51`: `a2b2` es legal y completaría 100 semijugadas; antes de jugarla, `claimable=false`.

Estos límites se documentan explícitamente y no se presentan como conformidad FIDE exhaustiva. Los sondeos temporales se retiraron del archivo final para no convertir comportamientos incompletos en expectativas de regresión.

## Comandos de validación

```powershell
$env:GOCACHE = 'D:\Temp\ajedrez-build\cache'
$env:GOPATH = 'D:\Temp\ajedrez-build\gopath'
$env:GOTOOLCHAIN = 'local'
& 'D:\Temp\ajedrez-build\go\bin\go.exe' test -mod=vendor -count=1 game.go game_test.go additional_rules_test.go
& 'D:\Temp\ajedrez-build\go\bin\go.exe' test -mod=vendor -count=1 ./...
```

Ambos comandos terminaron con código 0 después de los parches. Resultado observado: `ok command-line-arguments 0.642s` y `ok nexo-chess 0.727s`. El primer comando cubre 18 familias de pruebas, 14 de ellas añadidas por esta revisión. El segundo verifica también las pruebas HTTP y de arranque presentes en el proyecto. Esta revisión no modificó código de producto ni la dependencia; las correcciones fueron aplicadas por el responsable del proyecto.
