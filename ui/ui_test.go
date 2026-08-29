package ui_test

import (
	"io/fs"
	"regexp"
	"strings"
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

// TestGetCSS_ComponentesPublicados fija el contrato que los consumidores no pueden
// compilar: una clase CSS que desaparece no rompe ningún build ni ningún test del
// consumidor, solo sirve la página sin estilo. Estas son las que hoy tienen consumidor
// externo declarado: el bloque de «secreto de un solo uso» de las consolas, y el chip de
// estado con sus cuatro variantes y los modificadores de botón que `wapp-client-console`
// consume desde que dejó de tener copia local (listado de invitaciones, Plan 047).
func TestGetCSS_ComponentesPublicados(t *testing.T) {
	content, err := ui.GetCSS("wapp-components.css")
	require.NoError(t, err)

	for _, class := range []string{
		".wapp-secret-box",
		".wapp-secret-expiry",
		".wapp-snackbar--roomy",
		".wapp-chip",
		".wapp-chip--success",
		".wapp-chip--danger",
		".wapp-chip--info",
		".wapp-chip--neutral",
		".wapp-btn--auto",
		".wapp-btn--compact",
		".wapp-btn--danger",
	} {
		assert.Contains(t, string(content), class,
			"wapp-components.css debe publicar %s: lo consumen las consolas para mostrar un secreto de un solo uso", class)
	}
}

// TestGetCSS_ParSemanticoDeExitoCompleto fija que el cuarteto de exito exista entero.
// Nacio de un fallo real: sin `--wapp-color-on-success-container`, `.wapp-snackbar--success`
// pintaba su texto con `--wapp-color-on-success` (#FFFFFF, el color para ir sobre el verde
// FUERTE) encima de `--wapp-color-success-container` (#DCFCE7) ⇒ 1,10:1, texto invisible.
// El gemelo de error ya estaba completo; el de exito no, y por eso la regla echo mano del
// unico `on-*` que habia.
func TestGetCSS_ParSemanticoDeExitoCompleto(t *testing.T) {
	content, err := ui.GetCSS("wapp-tokens.css")
	require.NoError(t, err)

	for _, token := range []string{
		"--wapp-color-success:",
		"--wapp-color-on-success:",
		"--wapp-color-success-container:",
		"--wapp-color-on-success-container:",
	} {
		assert.Contains(t, string(content), token,
			"wapp-tokens.css debe definir %s: un container sin su `on-container` obliga a los componentes a reusar el `on-` del color fuerte, que no contrasta", token)
	}
}

// TestGetCSS_SnackbarSuccessLegible comprueba la regla concreta, no solo el token: que el
// snackbar de exito pinte su texto con el `on-container` y no vuelva al blanco.
func TestGetCSS_SnackbarSuccessLegible(t *testing.T) {
	content, err := ui.GetCSS("wapp-components.css")
	require.NoError(t, err)

	bloque := bloqueCSS(t, string(content), ".wapp-snackbar--success")

	assert.Contains(t, bloque, "color: var(--wapp-color-on-success-container);",
		".wapp-snackbar--success debe usar el `on-container` (#166534 sobre #DCFCE7 = 6,49:1, AA)")
	assert.NotContains(t, bloque, "var(--wapp-color-on-success)",
		".wapp-snackbar--success no puede usar `--wapp-color-on-success` (#FFFFFF sobre #DCFCE7 = 1,10:1, texto invisible)")
}

// TestGetCSS_ParesDeColorPorComponente fija, regla a regla, que cada componente use el par de
// color COMPLETO de su familia —el `*-container` de fondo con su `on-*-container` de texto, o el
// color FUERTE con su `on-*`— y que no los cruce. Cruzarlos no rompe ninguna compilación ni ningún
// test de conducta: solo sirve la página con el texto ilegible, y así llegó a campo el `.chip--ok`
// de `wapp-client-console`, que pintaba `--wapp-color-on-success` (#FFFFFF) sobre
// `--wapp-color-success-container` (#DCFCE7) ⇒ 1,10:1. El `prohibido` de cada fila es justo el
// token del OTRO par de la misma familia: es el que se cuela cuando alguien tira del `on-*` que le
// suena. Ninguno de los dos pares prohibidos aparece por accidente, porque el paréntesis de cierre
// hace que `var(--wapp-color-on-success)` NO sea subcadena de `var(--wapp-color-on-success-container)`.
func TestGetCSS_ParesDeColorPorComponente(t *testing.T) {
	content, err := ui.GetCSS("wapp-components.css")
	require.NoError(t, err)

	casos := []struct {
		selector  string
		fondo     string
		texto     string
		prohibido string
		ratio     string
	}{
		{".wapp-chip--success", "--wapp-color-success-container", "--wapp-color-on-success-container", "--wapp-color-on-success", "6,49:1 en los dos temas"},
		{".wapp-chip--danger", "--wapp-color-error-container", "--wapp-color-on-error-container", "--wapp-color-on-error", "13,26:1 en los dos temas"},
		{".wapp-chip--info", "--wapp-color-brand-container", "--wapp-color-brand-on-container", "--wapp-color-brand-on-primary", "13,30:1 en los dos temas"},
		{".wapp-chip--neutral", "--wapp-color-surface-variant", "--wapp-color-on-surface-variant", "--wapp-color-on-surface", "7,22:1 en claro y 5,48:1 en oscuro"},
		{".wapp-btn--danger", "--wapp-color-error", "--wapp-color-on-error", "--wapp-color-on-error-container", "6,46:1 en los dos temas"},
	}

	for _, c := range casos {
		t.Run(c.selector, func(t *testing.T) {
			bloque := bloqueCSS(t, string(content), c.selector)

			assert.Contains(t, bloque, "background-color: var("+c.fondo+");",
				"%s debe tomar su fondo de %s", c.selector, c.fondo)
			// El salto de línea y la sangría son parte de la aguja a propósito: sin ellos la
			// declaración `background-color: …` satisfaría también la del texto, porque termina
			// en `color:`.
			assert.Contains(t, bloque, "\n  color: var("+c.texto+");",
				"%s debe pintar su texto con %s (%s)", c.selector, c.texto, c.ratio)
			assert.NotContains(t, bloque, "var("+c.prohibido+")",
				"%s no puede usar %s: es el `on-` del otro par de su familia y sobre este fondo no contrasta", c.selector, c.prohibido)
		})
	}
}

// bloqueCSS devuelve el cuerpo de la primera regla con ese selector. Sirve para assertar
// sobre una regla concreta en vez de sobre el fichero entero, donde un `Contains` se cumple
// con una aparicion en cualquier otra parte.
func bloqueCSS(t *testing.T, contenido, selector string) string {
	t.Helper()
	inicio := strings.Index(contenido, selector+" {")
	require.NotEqual(t, -1, inicio, "el CSS debe declarar %s", selector)
	resto := contenido[inicio:]
	fin := strings.Index(resto, "}")
	require.NotEqual(t, -1, fin, "el bloque %s debe cerrar", selector)
	return resto[:fin]
}

// TestGetCSS_SecretBoxDeTonoFijo fija que `.wapp-secret-box` no dependa de ningun token
// que el modo oscuro redefina. El componente se sirve dentro de un `.wapp-snackbar--success`
// de tono fijo (ningun container se redefine en oscuro) y su texto tambien es fijo: basta con
// que el fondo siga al tema para romper el par. Paso de verdad: al promover el componente
// desde `wapp-platform-console` se tokenizo su `#FFFFFF` literal como
// `var(--wapp-color-surface-card)`, y en oscuro el fondo se iba a `#161D1B` dejando el texto
// `#166534` en 2,40:1. Con el blanco fijo son 7,13:1 en los dos temas.
func TestGetCSS_SecretBoxDeTonoFijo(t *testing.T) {
	tokens, err := ui.GetCSS("wapp-tokens.css")
	require.NoError(t, err)

	// Los tokens que el bloque `@media (prefers-color-scheme: dark)` redefine: son los
	// unicos que cambian de valor segun el tema, y por eso los unicos prohibidos aqui.
	mediaOscuro := regexp.MustCompile(`(?s)@media \(prefers-color-scheme: dark\).*`).FindString(string(tokens))
	require.NotEmpty(t, mediaOscuro, "wapp-tokens.css debe seguir teniendo su bloque de modo oscuro")
	sensiblesAlTema := regexp.MustCompile(`--wapp-[a-z-]+(?:\s*):`).FindAllString(mediaOscuro, -1)
	for i, s := range sensiblesAlTema {
		sensiblesAlTema[i] = strings.TrimSuffix(strings.TrimSpace(s), ":")
	}
	// Sin esta cota el test seria vacuo: una lista vacia no prohibe nada.
	require.Contains(t, sensiblesAlTema, "--wapp-color-surface-card",
		"el token que causo la regresion debe seguir estando entre los sensibles al tema")

	componentes, err := ui.GetCSS("wapp-components.css")
	require.NoError(t, err)
	bloque := bloqueCSS(t, string(componentes), ".wapp-secret-box")

	assert.Contains(t, bloque, "background-color: #FFFFFF;",
		".wapp-secret-box debe llevar su fondo blanco como literal: es de tono fijo a proposito")
	for _, token := range sensiblesAlTema {
		assert.NotContains(t, bloque, "var("+token+")",
			".wapp-secret-box no puede usar %s: se redefine en modo oscuro y su texto no, asi que el par se rompe", token)
	}
}
