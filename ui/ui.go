package ui

import (
	"embed"
	"io/fs"
)

// Assets contiene los archivos CSS del sistema de diseño wApp embebidos nativamente en Go.
//
//go:embed css/*.css
var Assets embed.FS

// FS devuelve el sistema de archivos de los estilos compartidos sub-enrutado en css.
func FS() fs.FS {
	sub, err := fs.Sub(Assets, "css")
	if err != nil {
		panic(err)
	}
	return sub
}

// GetCSS devuelve el contenido de un archivo CSS por nombre (e.g. "wapp-tokens.css").
func GetCSS(name string) ([]byte, error) {
	return Assets.ReadFile("css/" + name)
}
