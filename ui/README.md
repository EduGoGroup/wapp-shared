# ui

Sistema de diseño visual del ecosistema wApp: los **design tokens** CSS y los
componentes estándar, empaquetados en un módulo Go que los sirve desde un
`embed.FS`. Fuente única de verdad de la estética de todas las interfaces web
del ecosistema (Consola Cloud BFF y Edge Agent UI).

No tiene lógica: es un contenedor de assets con dos auxiliares de lectura. Solo
depende de la stdlib (`embed`, `io/fs`).

## Instalacion

```bash
go get github.com/EduGoGroup/wapp-shared/ui
```

## Uso

```go
package main

import (
	"net/http"

	"github.com/EduGoGroup/wapp-shared/ui"
)

func main() {
	// Servir los CSS como estaticos bajo /assets/css/.
	http.Handle("/assets/css/", http.StripPrefix("/assets/css/", http.FileServer(http.FS(ui.FS()))))

	// O leer un archivo concreto (p. ej. para inlinearlo en una plantilla).
	tokens, err := ui.GetCSS("wapp-tokens.css")
	if err != nil {
		panic(err)
	}
	_ = tokens
}
```

## API

- `Assets embed.FS` — el FS crudo; las rutas incluyen el prefijo `css/`.
- `FS() fs.FS` — el mismo FS sub-enrutado en `css`, listo para `http.FS`. Las
  rutas van sin prefijo (`wapp-tokens.css`).
- `GetCSS(name string) ([]byte, error)` — contenido de un CSS por nombre, sin
  prefijo (`ui.GetCSS("theme-bff.css")`).

## Archivos CSS

| Archivo | Que aporta |
| --- | --- |
| `wapp-tokens.css` | Identidad visual base en `:root`: variables `--wapp-*` de marca, color semantico, superficie, bordes, radios y sombras (con su variante oscura). |
| `wapp-components.css` | Componentes estandar sobre esos tokens: `.wapp-card`, `.wapp-btn`, `.wapp-field`, `.wapp-snackbar`, el bloque de login y sus auxiliares. |
| `theme-bff.css` | Tema de la Consola Cloud BFF (paleta teal): sobreescribe los tokens de acento. |
| `theme-edge.css` | Tema del plano de control del Edge Agent (paleta slate/blue): sobreescribe la familia de marca completa. |

El orden de carga importa: `wapp-tokens.css` → `wapp-components.css` → el tema
de la aplicacion, que es el ultimo porque redefine tokens.

El detalle de cada variable, la tabla completa de valores claro/oscuro y las
reglas de consumo estan en el manual del ecosistema, `docs/manuales/ui-design-tokens.md`
del meta-repo wApp (fuera de este repositorio).

## Navegacion

- [Changelog](CHANGELOG.md)

## Comandos disponibles

```bash
make build     # Compilar
make test      # Tests
make check     # Lint y validacion
```
