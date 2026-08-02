# Changelog — ui

El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.0.0/)
y [Versionado Semantico](https://semver.org/lang/es/).

## [Unreleased]

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
