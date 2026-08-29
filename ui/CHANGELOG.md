# Changelog — ui

El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.0.0/)
y [Versionado Semantico](https://semver.org/lang/es/).

## [Unreleased]

## [0.4.1] - 2026-08-29

### Fixed

- **El enlace dentro de un snackbar hereda el color del contenedor, y se
  subraya.** La regla base `a { color: var(--wapp-color-link) }` que estreno la
  `0.4.0` le da al enlace un color que SIGUE AL TEMA, pero el fondo de un
  snackbar es de tono FIJO. En oscuro eso ponia el teal claro `#53DBC9` sobre el
  verde claro `#DCFCE7`: **1,55:1**, medido en Chrome contra UAT el mismo dia de
  publicar la `0.4.0`.
  > Es el **mismo par mixto** —color sensible al tema sobre fondo que no lo es—
  > que ya costo el `--wapp-color-on-success-container` de la `0.3.0` y el
  > `.link-list__link` de la `0.4.0`. La diferencia es que esta vez **lo
  > introdujimos nosotros** al anadir la regla base. Cuando se detecto **no habia
  > ni un `<a>` dentro de un snackbar** en las tres consolas: se corrige la
  > trampa antes de que tenga victima, no un sintoma.
  >
  > El subrayado **no es adorno**: al heredar, el enlace queda exactamente del
  > color del texto que lo rodea, y sin subrayar dejaria de distinguirse por otra
  > cosa que no fuera el color — justo lo que prohibe **WCAG 1.4.1**. Es lo que
  > hace legitimo el `inherit`, y por eso el test vigila los dos juntos: quitar
  > uno sin el otro cae en rojo.

## [0.4.0] - 2026-08-29

### Added

- **Token `--wapp-color-link`, y esta vez SI en los dos temas** (`#006A60` en
  `:root`, `#53DBC9` en `@media (prefers-color-scheme: dark)`). Nace de un
  defecto medido en campo, no de una revision de estilo: `.link-list__link` de
  `wapp-client-console` usaba `--wapp-color-brand-primary` como color de TEXTO, y
  ese token **no se redefine en oscuro**, asi que daba **2,64:1** sobre
  `surface-card`, **2,85:1** sobre `bg` y **1,43:1** sobre `surface-variant` — por
  debajo incluso del 3:1 de texto grande. Los valores nuevos se eligieron
  midiendo contra las superficies reales de cada tema: claro 6,50 / 6,35 / 5,04;
  oscuro 10,07 / 10,87 / 5,48. Todas AA.
  > **`--wapp-color-brand-primary` NO se toca**, a proposito. Los botones
  > rellenos lo usan de FONDO con `--wapp-color-brand-on-primary` encima y miden
  > 6,50:1 en los dos temas: cambiarlo en oscuro obligaria a mover tambien
  > `on-primary` e invertiria el boton. La distincion que importa la enseño este
  > defecto: `brand-primary` es seguro como texto en `.wapp-brand-badge`, que
  > declara su PROPIO fondo fijo, e inseguro en `.link-list__link`, que no
  > declaraba fondo y flotaba sobre la superficie del tema.

- **`wapp-components.css`: regla base para el elemento `a`** — `color:
  var(--wapp-color-link)`. Es el UNICO selector de elemento del fichero, y es
  deliberado: una clase `.wapp-link` no puede alcanzar al enlace que nadie
  clasifico, que era justo el defecto (dos `<a href>` sin clase en la consola de
  cliente caian al azul por defecto del navegador, `#0000EE`, **1,82:1** sobre el
  fondo oscuro). Su especificidad (0-0-1) es la minima posible: no pisa ninguna
  clase, y en los tres layouts el `app.css` de cada consola carga despues.

- **`wapp-components.css`: `.wapp-snackbar :is(h1, h2, h3, h4, h5, h6) { color:
  inherit; }`** — un encabezado dentro de un snackbar toma el color del
  contenedor. Cierra una CLASE de defecto, no una instancia: vale para
  `--success` y `--error` y para las tres consolas. El caso que lo destapo era
  `<h2 class="section-title">` dentro de un `.wapp-snackbar--success`, donde
  `.section-title` traia su propio `color: var(--wapp-color-on-surface)` y pisaba
  al heredado — **1,18:1** en modo oscuro. La especificidad de `:is()` (0-1-1)
  gana a una clase suelta (0-1-0), asi que **no hubo que tocar ningun markup**.

### Tests

- **`TestGetCSS_TokensDeTextoDefinidosEnLosDosTemas`** — el candado que faltaba.
  No comprueba valores: **deriva del propio CSS** que tokens se usan como valor
  de una declaracion `color:` y exige que esten definidos en los DOS temas, con
  una lista de excepciones EXPLICITA para los siete que son monotema a proposito
  (los `on-*` de error y exito, `brand-on-primary`, …), cada uno con su razon.
  La lista se vigila a si misma: si una excepcion deja de usarse, o pasa a estar
  en los dos temas, el test falla y la lista encoge. Es el mismo fallo que ya
  costo el `--wapp-color-on-success-container` de la `0.3.0`; ahora no puede
  repetirse en silencio.
- **`TestGetCSS_EncabezadoDentroDeSnackbarHereda`** — vigila que la regla de
  encabezados siga existiendo.

## [0.3.0] - 2026-08-28

### Added

- **Token `--wapp-color-on-success-container` (`#166534`) en `wapp-tokens.css`** —
  cierra el cuarteto semantico de exito (`success` / `on-success` /
  `success-container` / `on-success-container`), que hasta ahora solo estaba
  completo en el de error. Es el color de texto para ir **sobre el container
  claro**; `--wapp-color-on-success` (`#FFFFFF`) es el de ir sobre el verde
  fuerte, y no son intercambiables. Va solo en `:root`: el bloque de modo oscuro
  no redefine el par de error ni el container de exito, asi que tampoco este.
  > Es **aditivo** (una variable nueva, ningun valor anterior cambia), pero lo
  > estrena una version que ya es MINOR por el bloque de secretos.

- **`wapp-components.css`: el bloque «secreto de un solo uso»** — `.wapp-secret-box`
  (la caja monoespaciada con borde discontinuo donde se muestra el valor) y
  `.wapp-secret-expiry` (el pie con su vencimiento). Nace de promover al modulo
  compartido las clases locales `enrollment-code-box` / `enrollment-code-expiry`
  de `wapp-platform-console`: el componente deja de ser «de enrolamiento» porque
  ya tiene dos consumidores con secretos distintos —los codigos de enrolamiento
  del Edge (consola de plataforma) y los enlaces de invitacion a personas
  (consola de cliente, Plan 047)—, asi que el nombre no menciona ninguno de los
  dos. Todo va por tokens de `wapp-tokens.css`; ningun valor en crudo salvo los
  dos `font-size` en `rem`, como en el resto del fichero.
  > **Conserva exactamente el tono de la copia local que sustituye**, valor a
  > valor: texto y borde van por `var(--wapp-color-on-success-container)`, que es
  > el `#166534` original, y el fondo sigue siendo el **blanco literal**
  > `#FFFFFF`. Da **7,13:1**, AA para texto normal, **y el mismo numero en los dos
  > temas**: nada del componente depende de `prefers-color-scheme`.
  > 🔴 **El fondo es un literal A PROPOSITO — no lo tokenices.** `.wapp-secret-box`
  > es de tono fijo de arriba abajo: se sirve dentro de un
  > `.wapp-snackbar--success`, cuyo fondo (`--wapp-color-success-container`) no
  > cambia con el tema, y su texto (`--wapp-color-on-success-container`) tampoco.
  > Basta con que el fondo si lo siga para romper el par. Paso en una version
  > intermedia de este mismo cambio: ese `#FFFFFF` se tokenizo como
  > `var(--wapp-color-surface-card)` —el unico blanco de superficie que hay, y de
  > los que el bloque oscuro **si** redefine—, y en modo oscuro el fondo se iba a
  > `#161D1B` dejando el texto en **2,40:1**, por debajo incluso del 3:1 de texto
  > grande. Tokenizar un literal que estaba deliberadamente fijo le cambio el
  > comportamiento en silencio. Lo vigila `TestGetCSS_SecretBoxDeTonoFijo`.
- **`.wapp-snackbar--roomy`** — modificador de `.wapp-snackbar` con el doble de
  aire (`--wapp-space-6` / `--wapp-space-8`). Tambien vivia solo en el CSS local
  de `wapp-platform-console` y lo necesita el bloque anterior, que se sirve
  dentro de un snackbar de exito.

- **`wapp-components.css`: el chip de estado** — `.wapp-chip` y sus cuatro
  variantes `--success` / `--danger` / `--info` / `--neutral`. Promueve al modulo
  las **dos** copias locales que ya existian y **no eran equivalentes**: `.badge`
  de `wapp-platform-console` (con los colores en hexadecimal a mano) y `.chip` de
  `wapp-client-console`. Las variantes se nombran por el **estado que comunican**,
  no por el color, y cada una toma el par `*-container` / `on-*-container`
  **completo** de su familia de tokens.
  > 🔴 **Arregla un chip ilegible que estaba en campo.** `.chip--ok` de
  > `wapp-client-console` pintaba `var(--wapp-color-on-success)` (`#FFFFFF`) sobre
  > `var(--wapp-color-success-container)` (`#DCFCE7`): **1,10:1** —el mismo par de
  > colores que el snackbar, asi que literalmente el mismo numero—, texto
  > practicamente invisible, en la portada y en la pantalla de roles. Es el mismo
  > fallo que `.wapp-snackbar--success` y la misma causa raiz —faltaba el token
  > `on-*-container` de exito—, asi que `.wapp-chip--success` estrena el
  > `--wapp-color-on-success-container` de esta misma version: **6,49:1**, AA.
  > El gemelo de `wapp-platform-console` no lo tenia porque usaba un `#15803D`
  > escrito a mano.
  > Contrastes de las cuatro, **medidos en los dos temas**: `--success` 6,49:1 ·
  > `--danger` 13,26:1 · `--info` 13,30:1, los tres identicos en claro y oscuro
  > porque ninguno de sus tokens se redefine en `@media (prefers-color-scheme:
  > dark)`; `--neutral` 7,22:1 en claro y 5,48:1 en oscuro, porque sus **dos**
  > tokens (`surface-variant` / `on-surface-variant`) se redefinen **juntos**. Lo
  > que rompe un componente es mezclar un token sensible al tema con un valor
  > **fijo** —eso es lo que vigila `TestGetCSS_SecretBoxDeTonoFijo`—, no usar dos
  > sensibles que viajan en pareja.

- **`wapp-components.css`: `.wapp-btn--auto`, `.wapp-btn--compact` y
  `.wapp-btn--danger`** — `.wapp-btn` nace con `width: 100%` (es el boton de un
  formulario de login, que ocupa su tarjeta entera), y eso lo hacia inservible
  para una accion **por fila**; las dos consolas resolvieron lo mismo por
  separado. `--auto` suelta el ancho y `--compact` baja la talla, separados
  porque son **ortogonales**; los nombres son los que `wapp-platform-console` ya
  tiene en local, para que su migracion sea borrar lineas y no renombrar en sus
  plantillas.
  > `.wapp-btn--danger` va en el par **fuerte** (`--wapp-color-error` de fondo con
  > `--wapp-color-on-error` de texto, **6,46:1**, AA, igual en los dos temas) y
  > **no** en el par de container, que es el de `.wapp-chip--danger`. Es
  > deliberado: en la tabla de invitaciones el boton «revocar» y el chip
  > «revocada» comparten fila, y con los mismos colores no se distinguiria la
  > accion del estado. Sustituye el `#DC2626` en crudo de la copia de
  > `wapp-platform-console`, que **no** se migra en esta version (ver abajo).

  > Es **aditivo**: no toca ninguna regla anterior, no anade ningun asset nuevo y
  > `Assets`, `FS()` y `GetCSS` no cambian de firma. Aun asi amplia la superficie
  > publica del modulo (once clases que los
  > consumidores pasan a exigir: tres del bloque de secretos, cinco del chip y tres
  > modificadores de boton) ⇒ **bump
  > MINOR** (`0.3.0`), no patch.

### Changed

- 🔴 **`.wapp-snackbar--success` deja de ser ilegible**: su texto pasa de
  `var(--wapp-color-on-success)` (`#FFFFFF`) a
  `var(--wapp-color-on-success-container)` (`#166534`). Sobre su propio fondo
  (`--wapp-color-success-container`, `#DCFCE7`) el blanco daba **1,10:1** —texto
  invisible— y el verde oscuro da **6,49:1**, AA para texto normal. La causa raiz
  era el token que faltaba: sin gemelo `on-*-container` de exito, la regla echo
  mano del `on-success` que si existia. Afecta a los avisos de exito de
  `wapp-client-console` (`partials/flashes.html`, `pages/login.html`), que hoy
  salen en blanco sobre verde claro. `wapp-platform-console` no lo notaba porque
  lo tapaba con un override local, que se retira con este cambio.

- `ui_test.go` estrena `TestGetCSS_ComponentesPublicados`, que assertea la
  presencia de las tres clases en el contenido del CSS. Es el unico gate posible:
  una clase que desaparece no rompe la compilacion de ningun consumidor —solo
  sirve la pagina sin estilo, en silencio. Se le suman tres gates mas,
  todos sobre el mismo hecho —el CSS no lo compila nadie—:
  `TestGetCSS_ParSemanticoDeExitoCompleto` (los cuatro tokens de exito existen),
  `TestGetCSS_SnackbarSuccessLegible` (el snackbar no vuelve al blanco) y
  `TestGetCSS_SecretBoxDeTonoFijo` (el fondo de la caja de secreto no depende de
  ningun token que el modo oscuro redefina; la lista de prohibidos se deriva del
  propio bloque `@media` de `wapp-tokens.css`, no se escribe a mano). Los tres se
  verificaron mutando el CSS: caen.

- `ui_test.go` suma `TestGetCSS_ParesDeColorPorComponente`, que comprueba **regla
  a regla** que cada componente use el par de color completo de su familia y no lo
  cruce: el `prohibido` de cada fila es el `on-*` del **otro** par, que es justo el
  que se cuela cuando alguien tira del token que le suena. Cubre las cuatro
  variantes del chip y `.wapp-btn--danger`. Verificado con **tres mutaciones
  ejecutadas** —devolver `.wapp-chip--success` a `--wapp-color-on-success`, borrar
  el bloque `.wapp-chip--neutral` y cruzar el texto de `.wapp-btn--danger` a
  `--wapp-color-on-error-container`—: las tres lo ponen rojo.

- ⚠️ **`wapp-platform-console` NO se migra en esta version.** Sus `.badge--*` (11
  usos) y sus `.wapp-btn--danger` / `--auto` / `--compact` locales siguen en su
  `app.css`, que se carga **despues** del compartido, asi que su aspecto no cambia.
  Cuando se migre, `.wapp-btn--danger` pasara de `#DC2626` a
  `var(--wapp-color-error)` (`#BA1A1A`) y `--compact` de `0.375rem 0.75rem` a los
  tokens: es un cambio de tono, no de semantica. Su `.badge--warning` es lo unico
  que **no** tiene sitio todavia: el cuarteto de warning sigue incompleto (falta
  `--wapp-color-on-warning-container`), asi que no se promovio ninguna variante de
  aviso.

## [0.2.0] - 2026-08-15

### Added

- **`theme-platform.css`** — tema de la Consola de Plataforma Admin (paleta
  indigo / deep navy), el cuarto tema del ecosistema junto a `theme-bff.css`
  (teal) y `theme-edge.css` (slate/blue). Redefine sobre los tokens base de
  `wapp-tokens.css` la marca (`--wapp-color-brand-*`), el acento de la
  aplicacion y la cabecera (`--wapp-app-header-*`), sin tocar ningun otro token.
  Lo consume `wapp-platform-console` con `ui.GetCSS("theme-platform.css")`: sin
  esta version su pantalla de login se sirve sin estilos.
  > Es **aditivo**: entra por el `//go:embed css/*.css` ya existente, asi que
  > `Assets`, `FS()` y `GetCSS` no cambian de firma ni de comportamiento y
  > ningun CSS anterior se modifica. Aun asi amplia la superficie publica del
  > modulo (un asset nuevo direccionable por nombre) ⇒ **bump MINOR** (`0.2.0`),
  > no patch.

### Changed

- `ui_test.go` cubre el asset nuevo: `TestGetCSS_Success` lo incluye en su tabla
  y `TestFS_Subdirectory` sube su cota de 4 a 5 archivos en el FS sub-enrutado.

## [0.1.0] - 2026-08-01

### Added

- Version inicial del modulo `ui`: los design tokens y componentes CSS del
  ecosistema wApp, servidos desde Go con `embed.FS` (Plan 038). Fuente unica de
  verdad de la estetica de las interfaces web del ecosistema; sustituye a las
  copias de CSS que cada servicio arrastraba.
- Assets embebidos (`//go:embed css/*.css`):
  - `wapp-tokens.css` — identidad visual base en `:root`: variables `--wapp-*`
    de marca, color semantico, superficie, bordes, radios y sombras, con su
    variante en modo oscuro.
  - `wapp-components.css` — componentes estandar sobre esos tokens
    (`.wapp-card`, `.wapp-btn`, `.wapp-field`, `.wapp-snackbar`, bloque de
    login).
  - `theme-bff.css` — tema de la Consola Cloud BFF (paleta teal).
  - `theme-edge.css` — tema del plano de control del Edge Agent UI (paleta
    slate/blue).
- API de consumo:
  - `Assets embed.FS` — el FS crudo, con las rutas prefijadas por `css/`.
  - `FS() fs.FS` — el mismo FS sub-enrutado en `css`, para `http.FS`.
  - `GetCSS(name string) ([]byte, error)` — contenido de un CSS por nombre.
- Solo stdlib (`embed`, `io/fs`); sin dependencias de otros modulos de
  wapp-shared.
