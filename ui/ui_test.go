package ui_test

import (
	"io/fs"
	"regexp"
	"sort"
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
// externo declarado: el bloque de «secreto de un solo uso» de las consolas, el chip de
// estado con sus cuatro variantes, los modificadores de botón que `wapp-client-console`
// consume desde que dejó de tener copia local (listado de invitaciones, Plan 047), y
// `.wapp-btn--outlined`/`.wapp-caption`, que consume `wapp-edge-agent` desde que su login
// dejó de forkear estos tokens a mano (deuda U-2 de `edge/wapp-edge-agent/documentations/deuda.md`).
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
		".wapp-btn--outlined",
		".wapp-caption",
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

// TestGetCSS_TokensDeTextoDefinidosEnLosDosTemas es el candado que faltaba, y no vigila un valor:
// vigila la SIMETRIA entre los dos temas. El defecto raiz de los tres fallos de contraste medidos
// en campo (Plan 047 · Ola A) es siempre el mismo, y ya nos mordio antes con
// `--wapp-color-on-success-container`: un token de color que existe en un tema y no en el otro.
// No rompe ninguna compilacion, no rompe ningun test de conducta y no se ve en modo claro, que es
// donde se desarrolla; solo aparece en la pantalla de quien tiene el sistema en oscuro.
//
// El test DERIVA las dos listas del propio CSS —lo definido antes del `@media` y lo redefinido
// dentro— en vez de fijar valores a mano, para que no caduque cuando la paleta se mueva. Sobre
// cada token que `wapp-components.css` usa como color de TEXTO exige una de dos cosas: o esta en
// los dos temas, o esta en la lista de excepciones de abajo, que es EXPLICITA y dice por que.
//
// La lista tambien se vigila a si misma: una excepcion que ya este definida en oscuro, o que ya no
// use nadie, hace fallar el test. Asi encoge sola en vez de envejecer.
func TestGetCSS_TokensDeTextoDefinidosEnLosDosTemas(t *testing.T) {
	// Tokens que se usan como color de TEXTO y viven SOLO en `:root` a proposito. El criterio para
	// entrar aqui es uno y solo uno: el token es MONOTEMA y el fondo sobre el que se pinta tambien,
	// asi que el par viaja entero y el numero de contraste es el mismo en los dos temas. Lo que NO
	// vale es un token fijo pintado sobre una superficie que si sigue al tema: eso es exactamente
	// lo que le pasaba a `.link-list__link` de `wapp-client-console` con `--wapp-color-brand-primary`
	// (2,64:1 sobre `surface-card` en oscuro), y por eso el enlace tiene ahora su propio
	// `--wapp-color-link`, que si esta en los dos bloques.
	monotemaJustificado := map[string]string{
		// `.wapp-brand-badge`: la marca como TEXTO sobre `--wapp-color-brand-container`. Los dos son
		// fijos ⇒ 5,04:1 en los dos temas. Es el mismo token que rompia el enlace: aqui es seguro
		// PORQUE la regla declara su propio fondo, alli no lo declaraba y flotaba sobre el tema.
		"--wapp-color-brand-primary": ".wapp-brand-badge lo pinta sobre --wapp-color-brand-container, tambien fijo (5,04:1 en los dos temas)",
		// `.wapp-btn--filled`: el blanco sobre el verde fuerte de la marca. Par fuerte completo.
		"--wapp-color-brand-on-primary": ".wapp-btn--filled lo pinta sobre --wapp-color-brand-primary, tambien fijo (6,50:1 en los dos temas)",
		// `.wapp-chip--info`: par de container completo de la familia marca.
		"--wapp-color-brand-on-container": ".wapp-chip--info lo pinta sobre --wapp-color-brand-container, tambien fijo (13,30:1 en los dos temas)",
		// `.wapp-field--alpha .wapp-field__label`: el fondo lo pone el contenedor `.wapp-field--alpha`
		// (`--wapp-color-secondary-container`), que tampoco se redefine en oscuro.
		"--wapp-color-on-secondary-container": ".wapp-field--alpha aporta el fondo --wapp-color-secondary-container, tambien fijo",
		// `.wapp-btn--danger`: par FUERTE de la familia error, deliberadamente distinto del par de
		// container que usa `.wapp-chip--danger` (ver el comentario de la regla).
		"--wapp-color-on-error": ".wapp-btn--danger lo pinta sobre --wapp-color-error, tambien fijo (6,46:1 en los dos temas)",
		// `.wapp-snackbar--error` y `.wapp-chip--danger`: par de container de la familia error.
		"--wapp-color-on-error-container": ".wapp-snackbar--error y .wapp-chip--danger lo pintan sobre --wapp-color-error-container, tambien fijo (13,26:1)",
		// `.wapp-snackbar--success`, `.wapp-chip--success` y `.wapp-secret-box`. Este es EL caso
		// fundacional: el cuarteto de exito es fijo de arriba abajo, y por eso `.wapp-secret-box`
		// lleva su fondo blanco como literal en vez de tokenizarlo (ver `TestGetCSS_SecretBoxDeTonoFijo`).
		"--wapp-color-on-success-container": ".wapp-snackbar--success, .wapp-chip--success y .wapp-secret-box lo pintan sobre fondos tambien fijos (6,49:1 y 7,13:1)",
	}

	tokensCSS, err := ui.GetCSS("wapp-tokens.css")
	require.NoError(t, err)
	componentesCSS, err := ui.GetCSS("wapp-components.css")
	require.NoError(t, err)

	claro, oscuro := temasDeTokens(t, string(tokensCSS))
	// Sin estas cotas el test seria vacuo: si el corte o la expresion dejaran de casar, las listas
	// saldrian vacias y no habria nada que comprobar.
	require.Contains(t, claro, "--wapp-color-on-surface", "el bloque `:root` debe seguir declarando tokens")
	require.Contains(t, oscuro, "--wapp-color-surface-card", "el bloque de modo oscuro debe seguir redefiniendo tokens")

	usados := tokensDeTexto(string(componentesCSS))
	require.Contains(t, usados, "--wapp-color-on-surface-variant", "la lectura de declaraciones `color:` debe seguir encontrando las reglas del fichero")
	require.Contains(t, usados, "--wapp-color-link", "wapp-components.css debe seguir teniendo la regla base de `a` con var(--wapp-color-link): sin ella un enlace sin clase vuelve al azul #0000EE del navegador (1,82:1 en oscuro)")

	for _, token := range usados {
		t.Run(token, func(t *testing.T) {
			require.Contains(t, claro, token,
				"%s se usa como color de texto en wapp-components.css pero no lo declara `:root`", token)

			if razon, esExcepcion := monotemaJustificado[token]; esExcepcion {
				assert.NotContains(t, oscuro, token,
					"%s ya se redefine en modo oscuro: la excepcion esta CADUCADA y hay que quitarla de monotemaJustificado (razon que decia: %s)", token, razon)
				return
			}

			assert.Contains(t, oscuro, token,
				"%s se usa como color de TEXTO y solo esta definido en `:root`: en modo oscuro se pinta con su valor de claro sobre una superficie oscura. O se redefine en `@media (prefers-color-scheme: dark)`, o se justifica en monotemaJustificado explicando sobre que fondo FIJO se pinta", token)
		})
	}

	// La lista no puede guardar tokens que ya no use nadie: una excepcion muerta es una excepcion
	// que manana alguien reutiliza sin releerla.
	for token, razon := range monotemaJustificado {
		assert.Contains(t, usados, token,
			"%s esta en monotemaJustificado pero ya no se usa como color de texto en wapp-components.css: quitalo (razon que decia: %s)", token, razon)
	}
}

// TestGetCSS_EncabezadoDentroDeSnackbarHereda vigila la regla que impide que un encabezado con
// clase propia pise el color que fija el snackbar. Nacio de un fallo medido en campo: el
// `<h2 class="section-title">` de `invitaciones.html` traia `--wapp-color-on-surface` y en oscuro
// eso ponia texto casi blanco sobre el verde claro FIJO del contenedor de exito ⇒ 1,18:1.
// No comprueba la forma exacta del selector —`:is(...)` o la lista separada por comas dan lo
// mismo— sino que los seis niveles queden cubiertos por alguna regla que herede del contenedor.
func TestGetCSS_EncabezadoDentroDeSnackbarHereda(t *testing.T) {
	content, err := ui.GetCSS("wapp-components.css")
	require.NoError(t, err)

	var selectoresQueHeredan []string
	for _, regla := range reglasCSS.FindAllStringSubmatch(sinComentarios(string(content)), -1) {
		selector, cuerpo := strings.TrimSpace(regla[1]), regla[2]
		if strings.Contains(selector, ".wapp-snackbar") && strings.Contains(cuerpo, "color: inherit") {
			selectoresQueHeredan = append(selectoresQueHeredan, selector)
		}
	}
	require.NotEmpty(t, selectoresQueHeredan,
		"wapp-components.css debe declarar una regla dentro de `.wapp-snackbar` con `color: inherit`: sin ella cualquier clase con `color` propio pisa el color del contenedor y el texto deja de contrastar")

	// El enlace va en la MISMA lista que los encabezados, y no por simetria: la regla base
	// `a { color: var(--wapp-color-link) }` le da un color que SIGUE AL TEMA, y el fondo del
	// snackbar es de tono fijo. Sin heredar, en oscuro daba 1,55:1 (medido en campo 2026-08-29).
	for _, nivel := range []string{"h1", "h2", "h3", "h4", "h5", "h6", "a"} {
		hueco := regexp.MustCompile(`\b` + nivel + `\b`)
		cubierto := false
		for _, selector := range selectoresQueHeredan {
			if hueco.MatchString(selector) {
				cubierto = true
				break
			}
		}
		assert.True(t, cubierto,
			"<%s> dentro de un `.wapp-snackbar` debe heredar el color del contenedor: hoy lo cubren %v", nivel, selectoresQueHeredan)
	}

	// Heredar el color deja al enlace EXACTAMENTE del color del texto que lo rodea. Si ademas no
	// se subraya, deja de distinguirse por otra cosa que no sea el color — que es justo lo que
	// prohibe WCAG 1.4.1. El subrayado es lo que hace legitimo el `inherit` de arriba, asi que
	// los dos se vigilan juntos: quitar uno sin el otro tiene que caer.
	subrayado := false
	for _, regla := range reglasCSS.FindAllStringSubmatch(sinComentarios(string(content)), -1) {
		selector, cuerpo := strings.TrimSpace(regla[1]), regla[2]
		if strings.Contains(selector, ".wapp-snackbar") && strings.Contains(selector, "a") &&
			strings.Contains(cuerpo, "text-decoration") && strings.Contains(cuerpo, "underline") {
			subrayado = true
			break
		}
	}
	assert.True(t, subrayado,
		"un `<a>` dentro de un `.wapp-snackbar` hereda el color del contenedor, asi que necesita `text-decoration: underline` para seguir siendo identificable sin depender del color (WCAG 1.4.1)")
}

// --- Utilidades de lectura del CSS ---

var (
	reComentarioCSS   = regexp.MustCompile(`(?s)/\*.*?\*/`)
	reDeclaracionToks = regexp.MustCompile(`(?m)^\s*(--wapp-[a-z0-9-]+)\s*:`)
	// El `[;{]` de delante es lo que separa una declaracion `color:` de texto de un
	// `background-color:` o un `border-color:`, que terminan igual pero no empiezan igual.
	reColorDeTexto = regexp.MustCompile(`(?m)(?:^|[;{])\s*color:\s*var\(\s*(--wapp-[a-z0-9-]+)\s*\)`)
	reglasCSS      = regexp.MustCompile(`(?s)([^{}]+)\{([^{}]*)\}`)
)

// sinComentarios quita los bloques `/* ... */` para que un token NOMBRADO en la prosa de un
// comentario no se cuente como definido ni como usado.
func sinComentarios(css string) string {
	return reComentarioCSS.ReplaceAllString(css, "")
}

// temasDeTokens parte wapp-tokens.css por su `@media (prefers-color-scheme: dark)` y devuelve los
// tokens que declara cada lado. Se deriva del fichero a proposito: una lista escrita a mano aqui
// se desincronizaria del CSS en el primer cambio de paleta y volveria a dejar pasar el defecto.
func temasDeTokens(t *testing.T, tokensCSS string) (claro, oscuro []string) {
	t.Helper()
	limpio := sinComentarios(tokensCSS)
	corte := strings.Index(limpio, "@media (prefers-color-scheme: dark)")
	require.NotEqual(t, -1, corte, "wapp-tokens.css debe seguir teniendo su bloque de modo oscuro")
	return declaracionesDeToken(limpio[:corte]), declaracionesDeToken(limpio[corte:])
}

func declaracionesDeToken(css string) []string {
	coincidencias := reDeclaracionToks.FindAllStringSubmatch(css, -1)
	tokens := make([]string, 0, len(coincidencias))
	for _, m := range coincidencias {
		tokens = append(tokens, m[1])
	}
	return tokens
}

// tokensDeTexto devuelve, sin repetidos y en orden estable, los tokens que aparecen como valor de
// una declaracion `color:` (el color del TEXTO) en el CSS que se le pase.
func tokensDeTexto(css string) []string {
	vistos := map[string]bool{}
	var tokens []string
	for _, m := range reColorDeTexto.FindAllStringSubmatch(sinComentarios(css), -1) {
		if !vistos[m[1]] {
			vistos[m[1]] = true
			tokens = append(tokens, m[1])
		}
	}
	sort.Strings(tokens)
	return tokens
}
