# iam

Cliente del **plano de identidad** de wApp: login, refresh y logout contra
identity-api (el SSO del grupo) y el **canje** del Identity Token por el Context
Token que emite la plataforma de wApp. Solo stdlib —cero dependencias, tambien en
los tests— y no depende de ningun otro modulo de wapp-shared.

Reconcilia las dos implementaciones que vivian copiadas en dos repos (el
`apiclient` del BFF del cliente y el `authclient` de la consola de operadores) y
que habian divergido en cosas que importan: una distinguia el 403 del System Gate
del 401 de credenciales y la otra los colapsaba; una exigia el par de tokens
completo y la otra aceptaba un 200 vacio; una tenia el timeout clavado y la otra
lo hacia configurable.

## Instalacion

```bash
go get github.com/EduGoGroup/wapp-shared/iam
```

## Las dos credenciales, y por que son dos

| Credencial | Que dice | Quien la emite | Lleva tenant |
| --- | --- | --- | --- |
| Identity Token | QUIEN ERES | identity-core | no, y no puede |
| Context Token | QUE PUEDES HACER EN WAPP | la plataforma de wApp | si |

`Login` y `Refresh` hacen los dos saltos server-to-server de una vez. El
`AuthResult` que devuelven lleva **siempre** el Context Token en `AccessToken`: el
Identity Token **muere dentro del modulo**, no vuelve al llamante, no entra en
ninguna cookie y no se registra. Lo que no sale, no se filtra.

## El `system` es un CAMPO, no una constante

El System Gate de identity autoriza **aplicaciones**, no ecosistemas: la clave es
namespaced (`wapp.bff`, `wapp.platform`) y `wapp` a secas no vale. Aqui viaja en
`Options` y el mismo codigo sirve a cualquier aplicacion del catalogo **sin una
sola rama por su valor** (hay un test con un `system` inventado que lo custodia).

Solo el **login** lo declara. El **refresh NO**: la aplicacion sale de la fila de
la sesion en identity, y aceptarlo del cliente permitiria canjear el refresh de
una aplicacion por el token de otra.

## Uso

```go
c, err := iam.NewClient(iam.Options{
	System:          "wapp.platform",           // o "wapp.bff"
	IdentityBaseURL: "http://127.0.0.1:8200",
	PlatformBaseURL: "http://127.0.0.1:8103",
	Timeout:         20 * time.Second,          // <=0 cae a iam.DefaultTimeout (15s)
})
if err != nil {
	return err // opciones que no pueden funcionar: fallan aqui, no dentro del login
}

sesion, err := c.Login(ctx, email, password)  // identity + canje, en un movimiento
sesion, err = c.Refresh(ctx, sesion.RefreshToken)
err = c.Logout(ctx, sesion.RefreshToken)
```

Los tres escalones sueltos, para quien necesite componerlos de otra forma:

```go
tokens, err := c.IdentityLogin(ctx, email, password)   // *IdentityTokens
tokens, err = c.IdentityRefresh(ctx, refreshToken)
canjeado, err := c.Exchange(ctx, tokens.IdentityToken) // *ExchangeResult
```

## Errores con nombre

| Situacion | Como se reconoce |
| --- | --- |
| Credencial invalida o vencida (401) | `errors.Is(err, iam.ErrUnauthorized)` |
| System Gate: contraseña CORRECTA pero la aplicacion no es suya (403) | `errors.Is(err, iam.ErrForbidden)` |
| Canje apagado en la plataforma (503) | `errors.Is(err, iam.ErrDualModeOff)` |
| Opciones que no pueden funcionar | `errors.Is(err, iam.ErrInvalidOptions)` |
| Cualquier otro status del upstream | `iam.StatusCodeOf(err)` |

Dos detalles que no son cosmeticos:

- **401 y 403 no se colapsan.** Son diagnosticos distintos —«no eres tu» y «eres
  tu, pero esta aplicacion no es tuya»— y de distinguirlos depende que quien lo
  lea sepa si tiene que cambiar la contraseña o pedir un alta en el catalogo.
- **El sentinela viaja DENTRO del `APIError`**, no envolviendolo por fuera, para
  que el mismo error responda a las dos preguntas: `errors.Is(err,
  ErrUnauthorized)` **y** `StatusCodeOf(err) == 401`.

## Los secretos no salen en los errores

`APIError` guarda la operacion y el status, y **nunca el cuerpo de la respuesta**.
Por este plano solo viajan credenciales, y un emisor que haga eco de la peticion
en su mensaje de error meteria la contraseña en el log del llamante. Hay un test
que recorre **todos** los caminos de error contra un upstream hostil —uno que
devuelve de vuelta lo que se le mando— y comprueba que ni la contraseña, ni el
refresh, ni el Identity Token aparecen en el texto del error.

## Nivel 0: sin dependencias

El modulo no importa ningun otro modulo de wapp-shared, **tampoco `auth`**: ese es
logica pura («sin base de datos ni HTTP») y un cliente HTTP con endpoints y
timeouts dentro lo contradiria. Los claims del Context Token se leen
decodificando el payload del JWT con la stdlib, **sin verificar la firma**, y solo
para alimentar la traza: quien lo valida de verdad es la plataforma en cada
llamada. Un token ilegible da contexto vacio en vez de tumbar el login.
