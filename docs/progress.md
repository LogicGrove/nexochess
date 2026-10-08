# Progreso: docs/plan.md

El usuario autoriza trabajo autónomo. Se crea un proyecto separado sin modificar archivos sincronizados. Implementaciones de interfaz y servidor usan archivos distintos y el contrato escrito para permitir trabajo paralelo.

| Interfaz compartida | Productor | Consumidor | Comprobación |
| --- | --- | --- | --- |
| Snapshot y rutas JSON | api.go/game.go | web/app.js | Contrato en design.md |
| http.Handler NewApp() | api.go | main.go | main inicia listener y sirve assets |
| web/* | frontend | main.go | go:embed web/* |

Estado: servidor e interfaz completados; revisión independiente y pruebas de reglas/red/DOM realizadas. Entrega final en preparación.

- Servidor: tests de autorización, concurrencia, propuestas, versiones, sesiones, JSON y desconexiones pasan; writes en red fuera de mutex.
- Reglas: perft y 14 suites independientes; dos errores de dependencia corregidos en vendor (EP en repetición y rendición contra rey solo).
- Interfaz: pruebas DOM con fetch simulado; código, temas, promociones, historial, marcadores y sesiones revisados de forma independiente.
- Navegador real: intento de abrir preview CUA rechazado por permiso de usuario denegado; no se intenta eludir. No se afirma revisión visual renderizada.
- Race detector: no disponible por falta de compilador C; pruebas concurrentes ordinarias repetidas50 veces pasan.
- Recuperación de respuesta perdida: tests backend y DOM pasan.
- Revisión final: aprobada sin críticos/importantes abiertos.
- EXE final recompilado, prueba nativa JSON y cierre verificados; sin archivos creados en carpeta de trabajo.
- Pendiente: ZIPs y guardar entregables.
