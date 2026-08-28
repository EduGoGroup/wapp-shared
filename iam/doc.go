// Package iam es el cliente del PLANO DE IDENTIDAD de wApp: login, refresh y
// logout contra identity-api (el SSO del grupo) y el canje del Identity Token
// por el Context Token que emite la plataforma de wApp.
//
// Reconcilia las dos implementaciones que vivían copiadas en dos repos —el
// `apiclient` del BFF del cliente y el `authclient` de la consola de
// operadores—, que habían divergido en cosas que importan: la una distinguía el
// 403 del System Gate del 401 de credenciales y la otra los colapsaba; la una
// exigía el par de tokens completo y la otra aceptaba un 200 vacío; la una tenía
// el timeout clavado y la otra lo hacía configurable.
//
// # Las dos credenciales, y por qué son dos
//
//   - El **Identity Token** dice QUIÉN ERES y lo emite identity. No lleva claims
//     de negocio y no puede llevarlos: el tenant no está ahí.
//   - El **Context Token** dice QUÉ PUEDES HACER EN WAPP y lo emite la
//     plataforma, a cambio de un Identity Token válido.
//
// [Client.Login] y [Client.Refresh] hacen los dos saltos server-to-server de una
// vez y devuelven un [AuthResult] cuyo AccessToken es SIEMPRE el Context Token:
// el Identity Token muere aquí dentro, no vuelve al llamante y no se registra.
// Lo que no sale, no se filtra. Quien necesite los dos escalones por separado
// tiene [Client.IdentityLogin], [Client.IdentityRefresh] y [Client.Exchange].
//
// # El `system` es un CAMPO, no una constante
//
// El System Gate de identity autoriza aplicaciones, no ecosistemas: la clave es
// namespaced (`wapp.bff`, `wapp.platform`) y "wapp" a secas no vale. Aquí viaja
// en [Options] y el mismo código sirve a cualquier aplicación del catálogo sin
// una sola rama por valor. Solo el login lo declara: el refresh NO lo lleva —la
// aplicación sale de la fila de la sesión en identity—, porque aceptarlo
// permitiría canjear el refresh de una aplicación por el token de otra.
//
// # Nivel 0 y sin dependencias
//
// El módulo no importa ningún otro módulo de `wapp-shared` —tampoco `auth`, que
// es lógica pura y no debe cargar con un cliente HTTP— ni ninguna dependencia
// externa: los claims del Context Token se leen decodificando el payload del JWT
// con la stdlib, SIN verificar la firma, y solo para alimentar la traza (quien
// valida de verdad es la plataforma en cada llamada).
package iam
