# Changelog — ui

El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.0.0/)
y [Versionado Semantico](https://semver.org/lang/es/).

## [Unreleased]

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
