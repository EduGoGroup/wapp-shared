package ui_test

import (
	"io/fs"
	"testing"

	"github.com/EduGoGroup/wapp-shared/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCSS_Success(t *testing.T) {
	cssFiles := []string{
		"wapp-tokens.css",
		"wapp-components.css",
		"theme-bff.css",
		"theme-edge.css",
		"theme-platform.css",
	}

	for _, fileName := range cssFiles {
		t.Run(fileName, func(t *testing.T) {
			content, err := ui.GetCSS(fileName)
			require.NoError(t, err, "GetCSS debe leer %s sin errores", fileName)
			assert.NotEmpty(t, content, "El contenido de %s no debe estar vacío", fileName)
		})
	}
}

func TestFS_Subdirectory(t *testing.T) {
	subFS := ui.FS()
	require.NotNil(t, subFS, "FS() debe retornar un fs.FS no nulo")

	entries, err := fs.ReadDir(subFS, ".")
	require.NoError(t, err, "fs.ReadDir en FS() debe ser exitoso")
	assert.GreaterOrEqual(t, len(entries), 5, "Debe haber al menos 5 archivos CSS en el FS sub-enrutado")
}
