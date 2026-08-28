// Package webgin es el adaptador a Gin del middleware web endurecido de wApp.
// Es DELGADO a propósito: cada middleware traduce el gin.Context a lo que el
// paquete `web` entiende (una http.Header, un método, una ruta, una clave) y
// devuelve la respuesta. Las decisiones —la política CSP, la comparación del
// token CSRF, el algoritmo del limitador— viven en `web` y aquí no se repiten.
//
// # Por qué el paquete NO se llama `gin`
//
// El directorio es `gin/` (el import queda `.../wapp-shared/web/gin`, que es como
// se lee bien), pero el paquete se llama `webgin`. Si se llamara `gin`
// colisionaría con `github.com/gin-gonic/gin`, que TODO consumidor de este
// paquete importa también, y obligaría a cada uno de ellos a poner un alias en su
// bloque de imports para poder nombrar a los dos. El alias iría en N sitios; el
// nombre distinto va en uno.
//
// # Orden de los middlewares (no es negociable)
//
//	SetReleaseMode()                     // una vez por proceso
//	engine.Use(gin.Recovery())
//	engine.Use(webgin.SlogLogger())
//	engine.Use(webgin.SecurityHeaders(...))   // siembra el nonce CSP
//	engine.Use(webgin.CORS(...))
//	engine.Use(webgin.RateLimit(...))         // antes de auth: por user_id si hay, si no por IP
//	… rutas estáticas y /healthz (no renderizan formularios ni mutan) …
//	engine.Use(webgin.BodyLimit(...))         // 🔴 ANTES del CSRF
//	engine.Use(webgin.CSRF(...))
//
// BodyLimit ANTES del CSRF porque el CSRF lee el formulario para comparar el
// token y con eso consume el cuerpo entero: un tope montado después llegaría
// cuando el daño ya está hecho. Las cabeceras de seguridad antes de cualquier
// handler que renderice, porque el renderizador lee el nonce que siembran.
package webgin
