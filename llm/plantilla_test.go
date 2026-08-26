package llm_test

import (
	"strings"
	"testing"

	"github.com/EduGoGroup/wapp-shared/llm"
	"github.com/stretchr/testify/require"
)

// TestValidarPlantilla_LasCuatroPorDefectoPasan es la red de arranque: si alguien
// edita el texto compilado de una etapa y le mete un ejemplo que su propio
// validador rechaza, se entera AQUÍ y no catorce jobs muertos después.
//
// Es un test de REGLA sobre las cuatro etapas ajustables, no cuatro tests de
// conducta: recorre llm.EtapasAjustables, así que una etapa nueva queda cubierta
// el día que se añade a esa lista, sin tocar este fichero.
func TestValidarPlantilla_LasCuatroPorDefectoPasan(t *testing.T) {
	require.NotEmpty(t, llm.EtapasAjustables)
	for _, e := range llm.EtapasAjustables {
		t.Run(string(e), func(t *testing.T) {
			p, ok := llm.PlantillaPorDefecto(e)
			require.True(t, ok, "la etapa está en EtapasAjustables pero no tiene plantilla por defecto")
			require.NoError(t, llm.ValidarPlantilla(e, p))
		})
	}
}

// TestValidarPlantilla_CazaElEjemploQueSuValidadorRechaza reproduce EL BUG que
// costó la release v0.4.1: el esquema de P4 imprimía `"package_size": 0` con
// `"unit_kind": "package"`, que es exactamente lo que validarPaquete rechaza. El
// modelo no desobedecía —copiaba el ejemplo— y la etapa fue 0 de 14 en campo.
//
// Con ValidarPlantilla, esa plantilla ya no puede servirse: falla al cargarse.
// Esta es la diferencia entre el bug de entonces y hoy — entonces se descubría en
// producción con jobs muertos; ahora es un error de arranque.
func TestValidarPlantilla_CazaElEjemploQueSuValidadorRechaza(t *testing.T) {
	buena, ok := llm.PlantillaPorDefecto(llm.EtapaP4)
	require.True(t, ok)
	require.NoError(t, llm.ValidarPlantilla(llm.EtapaP4, buena), "el control tiene que estar sano")

	mala := buena
	mala.Esquema = strings.Replace(buena.Esquema, `"package_size": 30`, `"package_size": 0`, 1)
	require.NotEqual(t, buena.Esquema, mala.Esquema,
		"el esquema de P4 ya no imprime `\"package_size\": 30`: actualiza este test antes de creerte el resto")

	err := llm.ValidarPlantilla(llm.EtapaP4, mala)
	require.ErrorIs(t, err, llm.ErrPlantillaInvalida)
	require.Contains(t, err.Error(), "lo rechaza su propio validador",
		"el error tiene que decir POR QUÉ, que es lo que le faltó al de entonces")
}

// TestValidarPlantilla_ExigeQueElEsquemaCRUDOSigaFallando protege la SEGUNDA
// mitad del invariante, que es la fácil de perder al pulir un prompt: los huecos
// reconocibles (`...`, `AAAA-MM-DD`) son los que permiten cazar a un modelo que
// ecoa el prompt entero en vez de responder. Una plantilla que rellena esos
// huecos con valores plausibles pasaría la primera comprobación y perdería la red.
func TestValidarPlantilla_ExigeQueElEsquemaCRUDOSigaFallando(t *testing.T) {
	buena, ok := llm.PlantillaPorDefecto(llm.EtapaP5)
	require.True(t, ok)

	sinHuecos := buena
	sinHuecos.Esquema = strings.ReplaceAll(buena.Esquema, llm.PlaceholderEsquema, "ya está redactado")
	require.NotEqual(t, buena.Esquema, sinHuecos.Esquema)

	err := llm.ValidarPlantilla(llm.EtapaP5, sinHuecos)
	require.ErrorIs(t, err, llm.ErrPlantillaInvalida)
	require.Contains(t, err.Error(), "ecoe el prompt")
}

// TestValidarPlantilla_RechazaLoQueNoPuedeServir cubre las dos negativas baratas:
// una etapa que no existe y una instrucción vacía. Las dos son configuraciones
// que un fichero mal puesto produce con facilidad, y las dos tienen que fallar
// por su nombre en el arranque en vez de servir un prompt mutilado.
func TestValidarPlantilla_RechazaLoQueNoPuedeServir(t *testing.T) {
	buena, _ := llm.PlantillaPorDefecto(llm.EtapaP2)

	require.ErrorIs(t, llm.ValidarPlantilla(llm.Etapa("p9"), buena), llm.ErrPlantillaInvalida)

	sinInstruccion := buena
	sinInstruccion.Instruccion = "   \n\t "
	require.ErrorIs(t, llm.ValidarPlantilla(llm.EtapaP2, sinInstruccion), llm.ErrPlantillaInvalida)
}

// TestPlantillaInyectada_CambiaElPromptYNoElORDEN fija la costura entera: el
// texto que se inyecta SALE en el prompt, y las piezas que no se inyectan siguen
// donde estaban. Lo segundo es lo que de verdad se puede romper sin darse cuenta
// — el orden es lo que mantiene cacheable el prefijo (I6, ADR-0046), y una
// refactorización que mueva `jsonOnlyRules` detrás del esquema no rompería ningún
// test de contenido.
func TestPlantillaInyectada_CambiaElPromptYNoElORDEN(t *testing.T) {
	p, ok := llm.PlantillaPorDefecto(llm.EtapaP2)
	require.True(t, ok)
	p.Instruccion = "\n\nINSTRUCCION INYECTADA DE PRUEBA.\n\n"

	prompt := llm.BuildExtractMainIdeasPromptCon(p, llm.ExtractMainIdeasInput{SourceText: "hola qué tal"})

	require.Contains(t, prompt, "INSTRUCCION INYECTADA DE PRUEBA")
	require.Contains(t, prompt, "hola qué tal")

	iInstr := strings.Index(prompt, "INSTRUCCION INYECTADA DE PRUEBA")
	iReglas := strings.Index(prompt, "Reglas de salida, sin excepciones:")
	iEsquema := strings.Index(prompt, "Esquema de la respuesta:")
	iDatos := strings.Index(prompt, "hola qué tal")
	require.True(t, iInstr < iReglas && iReglas < iEsquema && iEsquema < iDatos,
		"el orden tiene que ser instrucción → reglas → esquema → datos; salió %d/%d/%d/%d",
		iInstr, iReglas, iEsquema, iDatos)
}
