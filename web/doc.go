// Package web es el middleware web endurecido de wApp: la capa transversal que
// toda consola SSR del ecosistema necesita —nonce y CSP, CSRF double-submit,
// rate-limit por clave, allowlist CORS, política de cookies, deadline por
// petición, single-flight de refresh y traducción de flashes— extraída a un solo
// sitio.
//
// Nace RECONCILIANDO dos copias que ya habían divergido: la del BFF del cliente
// (`wapp-guardian-bff/internal/web`) y la de la consola de operadores
// (`wapp-platform-console/internal/web`). Ninguna de las dos ganó entera: la
// política que se sirve aquí es la unión de lo mejor de cada una (por ejemplo la
// CSP lleva el `object-src 'none'` y el `font-src 'self'` del BFF Y el
// `base-uri 'none'` de la consola, que era más estricto). Cada divergencia está
// anotada en el CHANGELOG del módulo.
//
// # La partición en dos paquetes
//
// Este paquete es SOLO stdlib (+ golang.org/x/time/rate): decide, no sirve.
// Trabaja sobre http.Header, cadenas y relojes, y no conoce ningún framework.
// El adaptador a Gin vive aparte, en el paquete `web/gin` (nombre de paquete
// `webgin`), porque es el primer sitio del monorepo que trae Gin y quien solo
// quiere el algoritmo no tiene por qué arrastrar los handlers.
//
// Es un módulo de NIVEL 0: no importa ningún otro módulo de wapp-shared. Lo que
// necesitaría de `auth` (los claims del JWT) entra como parámetro —SessionValid
// y RefreshDue reciben el `exp` ya extraído, no un tipo de otro módulo—.
//
// # Restricciones de orden (no son negociables)
//
//   - BodyLimit va ANTES que CSRF. El middleware CSRF lee el formulario para
//     comparar el token, y con eso consume el cuerpo entero (a memoria hasta
//     MaxMultipartMemory y a disco lo que sobre). Un tope montado después llega
//     cuando el daño ya está hecho.
//   - El CSRF VALIDA ANTES DE SEMBRAR. Si primero se siembra la cookie y luego
//     se rechaza, el 403 sale con un Set-Cookie que el atacante puede provocar a
//     voluntad.
//   - Las cabeceras de seguridad van antes que cualquier handler que renderice:
//     el nonce lo siembra ese middleware y el renderizador lo lee.
//
// # Fail-closed
//
// La entropía es un único punto de fallo compartido: Nonce y NewCSRFToken leen
// del mismo io.Reader (crypto/rand por defecto, inyectable en los tests). Si
// falla, no se sirve la página: 500 y sin CSP, nunca una respuesta con inline
// sin nonce.
package web
