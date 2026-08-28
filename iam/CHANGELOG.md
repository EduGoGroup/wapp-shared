# Changelog — iam

El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.0.0/)
y [Versionado Semantico](https://semver.org/lang/es/).

## [Unreleased]

## [0.1.0] - 2026-08-28

### Added

- Version inicial del modulo `iam`: el cliente del **plano de identidad** de
  wApp —login, refresh y logout contra identity-api, y canje del Identity Token
  por el Context Token de la plataforma— reconciliando las dos implementaciones
  que vivian copiadas en `wapp-guardian-bff/internal/apiclient` y en
  `wapp-platform-console/internal/authclient`.
  - `Options` / `NewClient`: el **`system` es un campo del cliente**, no una
    constante del binario. El mismo codigo sirve a `wapp.bff`, a `wapp.platform`
    y a cualquier otra clave del catalogo **sin una sola rama por su valor**.
    `NewClient` **valida las opciones** y devuelve `ErrInvalidOptions`: un
    `system` vacio o una URL sin esquema fallan al construir, no dentro del
    primer login.
  - `Client.Login` / `Client.Refresh`: los dos saltos server-to-server en un
    movimiento; devuelven un `AuthResult` cuyo `AccessToken` es **siempre** el
    Context Token. El Identity Token muere dentro del modulo.
  - `Client.Logout`: cierra en identity **solo** la sesion de esta aplicacion,
    sin `Authorization` y presentando solo el refresh; **propaga el error** del
    upstream en vez de devolver `nil`.
  - `Client.IdentityLogin` / `Client.IdentityRefresh` / `Client.Exchange`: los
    tres escalones sueltos, para quien necesite componerlos de otra forma.
  - Errores con nombre: `ErrUnauthorized` (401), `ErrForbidden` (403 del System
    Gate), `ErrDualModeOff` (503 del canje), `ErrInvalidOptions`, mas `APIError`
    y `StatusCodeOf`. El sentinela viaja **dentro** del `APIError`, asi que el
    mismo error responde a `errors.Is(err, ErrUnauthorized)` **y** a
    `StatusCodeOf(err) == 401`.
  - `Timeout` configurable con caida a `DefaultTimeout` (15s) cuando es `<=0`, e
    inyeccion opcional de un `http.Client` propio. Todas las llamadas reciben y
    respetan el `context.Context`.
  - **Los secretos no salen en los errores**: `APIError` guarda operacion y
    status, nunca el cuerpo del upstream. Un test recorre todos los caminos de
    error contra un emisor que hace eco de la peticion y comprueba que ni la
    contraseña, ni el refresh, ni el Identity Token aparecen en el texto.
  - **Nivel 0 y sin dependencias**: no importa ningun otro modulo de
    `wapp-shared` —tampoco `auth`, que es logica pura— ni ninguna dependencia
    externa; los claims del Context Token se decodifican con la stdlib, sin
    verificar la firma, y solo para la traza.

