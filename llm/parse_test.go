package llm_test

import (
	"encoding/json"
	"testing"

	"github.com/EduGoGroup/wapp-shared/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// artefactoP2 es el artefacto de ideas del caso real que gobierna el Plan 044.
const artefactoP2 = `{
  "version": 1,
  "wants": [
    {"idea": "torta con decoración infantil, chocolate húmedo, 10 o 12 porciones",
     "evidence": "una torta sería con decoración infantil"},
    {"idea": "paquete de tequeños congelados de 30",
     "evidence": "un paquete de tequeños congelados de 30"}
  ],
  "delivery_hint": {"text": "el miércoles de la semana que viene",
                    "evidence": "para el miércoles de la semana que viene"}
}`

func TestParseMainIdeas_Completo(t *testing.T) {
	out, err := llm.ParseMainIdeas(json.RawMessage(artefactoP2))
	require.NoError(t, err)
	require.Len(t, out.Wants, 2)
	assert.Equal(t, "un paquete de tequeños congelados de 30", out.Wants[1].Evidence)
	require.NotNil(t, out.DeliveryHint)
	assert.Equal(t, "el miércoles de la semana que viene", out.DeliveryHint.Text)
}

func TestParseMainIdeas_SinPistaDeEntrega(t *testing.T) {
	out, err := llm.ParseMainIdeas(json.RawMessage(`{"version": 1, "wants": []}`))
	require.NoError(t, err)
	assert.Nil(t, out.DeliveryHint)
}

func TestParseItemSpecs_SeparaAnadidosDePersonalizaciones(t *testing.T) {
	out, err := llm.ParseItemSpecs(json.RawMessage(`{
      "version": 1,
      "items": [{"product": "torta", "variant": "10 o 12 porciones",
                 "addon_candidates": ["decoración infantil"],
                 "customizations": ["sin lactosa"],
                 "evidence": "de 10 o 12 porciones"}]
    }`))
	require.NoError(t, err)
	require.Len(t, out.Items, 1)
	assert.Equal(t, []string{"decoración infantil"}, out.Items[0].AddonCandidates)
	assert.Equal(t, []string{"sin lactosa"}, out.Items[0].Customizations)
	// El rango sigue siendo texto en P3: quien lo parte es P4.
	assert.Equal(t, "10 o 12 porciones", out.Items[0].Variant)
}

func TestParseQuantities_PaqueteYRango(t *testing.T) {
	out, err := llm.ParseQuantities(json.RawMessage(`{
      "version": 1,
      "delivery_date": "2026-07-22",
      "delivery_date_basis": "message_ts=2026-07-13",
      "items": [
        {"product": "torta", "qty": 1,
         "range": {"min": 10, "max": 12, "unit": "porciones"},
         "evidence": "de 10 o 12 porciones"},
        {"product": "tequeños congelados", "qty": 1,
         "unit_kind": "package", "package_size": 30,
         "evidence": "un paquete de tequeños congelados de 30"}
      ]
    }`))
	require.NoError(t, err)
	require.Len(t, out.Items, 2)
	require.NotNil(t, out.Items[0].Range)
	assert.Equal(t, 12, out.Items[0].Range.Max)
	// El paquete de 30 es UNA unidad de paquete, nunca 30 unidades.
	assert.Equal(t, 1, out.Items[1].Qty)
	assert.Equal(t, 30, out.Items[1].PackageSize)
	assert.Equal(t, "2026-07-22", out.DeliveryDate)
}

func TestParseQuoteText_Completo(t *testing.T) {
	out, err := llm.ParseQuoteText(json.RawMessage(`{"version": 1, "text": "Hola, te paso el detalle"}`))
	require.NoError(t, err)
	assert.Equal(t, "Hola, te paso el detalle", out.Text)
}

func TestParseClassification_Completo(t *testing.T) {
	out, err := llm.ParseClassification(json.RawMessage(
		`{"version": 1, "intent": "consulta_estado", "confidence": 0.82,
		  "params": {"numero_pedido": "42"}, "evidence": "cómo va el pedido 42"}`),
		entradaDeClasificacion())
	require.NoError(t, err)
	assert.Equal(t, "consulta_estado", out.Intent)
	assert.InDelta(t, 0.82, out.Confidence, 1e-9)
	assert.Equal(t, map[string]string{"numero_pedido": "42"}, out.Params)
	// El parser NO aplica el umbral ni sanea los params contra el texto: eso es
	// del caller, que es quien tiene el texto original y la config del tenant.
	assert.Equal(t, "cómo va el pedido 42", out.Evidence)
}

func TestParseClassification_IntentFueraDelCatalogoEsErrorDeCalidad(t *testing.T) {
	// Mutación que lo pone rojo: en ParseClassification, borrar la llamada a
	// validarEnum. Los cuatro casos vuelven a aceptarse en silencio.
	fuera := []string{
		`{"version": 1, "intent": "intake_requests", "confidence": 0.9, "evidence": "quiero una torta"}`,
		`{"version": 1, "intent": "Intake_Request", "confidence": 0.9, "evidence": "quiero una torta"}`,
		`{"version": 1, "intent": "", "confidence": 0.9, "evidence": "quiero una torta"}`,
		`{"version": 1, "confidence": 0.9, "evidence": "quiero una torta"}`,
	}
	for _, raw := range fuera {
		out, err := llm.ParseClassification(json.RawMessage(raw), entradaDeClasificacion())
		requiereErrorDeCalidad(t, err)
		assert.Nil(t, out)
	}
}

func TestParseClassification_SinCatalogoDeclaradoNoPasaNada(t *testing.T) {
	// Un caller que no declara su catálogo es un cableado roto y tiene que fallar
	// ruidoso: si aquí se dejara pasar «lo que sea» porque no hay con qué
	// comparar, el eco del esquema entraría sin error por esa puerta.
	// Mutación que lo pone rojo: en validarEnum, devolver nil cuando
	// len(permitidos) == 0.
	out, err := llm.ParseClassification(json.RawMessage(
		`{"version": 1, "intent": "intake_request", "confidence": 0.9, "evidence": "quiero una torta"}`),
		llm.ClassifyRequestInput{Text: textoAmbar})
	requiereErrorDeCalidad(t, err)
	assert.Nil(t, out)
}

func TestParseClassification_ConfianzaFueraDeRangoEsErrorDeCalidad(t *testing.T) {
	// Está MEDIDO en campo (wapp-edge-intent): sin acotar el rango el modelo
	// devuelve «100 de confianza» y entonces NADA cae nunca por debajo del umbral
	// del tenant — pasa todo, incluido lo que el modelo no entendió. Allí acota la
	// gramática de Ollama; aquí, con un proveedor que responde texto libre, este
	// parser es el único sitio donde se puede acotar.
	// Mutación que lo pone rojo: en ParseClassification, borrar la llamada a
	// validarConfianza.
	for _, raw := range []string{
		`{"version": 1, "intent": "intake_request", "confidence": 100, "evidence": "quiero una torta"}`,
		`{"version": 1, "intent": "intake_request", "confidence": 1.0001, "evidence": "quiero una torta"}`,
		`{"version": 1, "intent": "intake_request", "confidence": -0.5, "evidence": "quiero una torta"}`,
	} {
		out, err := llm.ParseClassification(json.RawMessage(raw), entradaDeClasificacion())
		requiereErrorDeCalidad(t, err)
		assert.Nil(t, out)
	}

	// Los extremos SÍ son válidos: 0 es «no tengo ni idea», y quien decide qué
	// hacer con esa cifra es el umbral del caller, no este parser.
	for _, raw := range []string{
		`{"version": 1, "intent": "intake_request", "confidence": 0, "evidence": "quiero una torta"}`,
		`{"version": 1, "intent": "intake_request", "confidence": 1, "evidence": "quiero una torta"}`,
	} {
		_, err := llm.ParseClassification(json.RawMessage(raw), entradaDeClasificacion())
		require.NoError(t, err)
	}
}

func TestParseClassification_LoDesconocidoNoNecesitaEvidencia(t *testing.T) {
	// La etiqueta de escape se acepta SIN estar en el catálogo (en el contrato de
	// wApp está prohibido declararla) y sin evidencia: exigirle una frase literal
	// a quien acaba de decir que no entendió solo consigue que el caller pague un
	// reintento para volver a oír lo mismo.
	// Mutación que lo pone rojo: en ParseClassification, usar validarObligatorio
	// también para la rama de UnknownLabel.
	out, err := llm.ParseClassification(json.RawMessage(
		`{"version": 1, "intent": "desconocido", "confidence": 0.1}`),
		entradaDeClasificacion())
	require.NoError(t, err)
	assert.Equal(t, "desconocido", out.Intent)

	// Una intención REAL sin evidencia sigue siendo un artefacto hueco.
	sinEvidencia, err := llm.ParseClassification(json.RawMessage(
		`{"version": 1, "intent": "intake_request", "confidence": 0.9}`),
		entradaDeClasificacion())
	requiereErrorDeCalidad(t, err)
	assert.Nil(t, sinEvidencia)

	// Y ni siquiera lo desconocido puede traer el relleno del esquema: omitir la
	// evidencia es una respuesta, copiar el esquema no lo es.
	conRelleno, err := llm.ParseClassification(json.RawMessage(
		`{"version": 1, "intent": "desconocido", "confidence": 0.1, "evidence": "..."}`),
		entradaDeClasificacion())
	requiereErrorDeCalidad(t, err)
	assert.Nil(t, conRelleno)
}

func TestParseClassification_ParamVacioPasaYElRellenoNo(t *testing.T) {
	// Un param VACÍO significa «el cliente no lo dijo», y quien decide si eso
	// sirve es el caller, que además lo contrasta contra el texto original. El
	// relleno del esquema no significa nada: es el modelo copiando el prompt.
	// Mutación que lo pone rojo: en validarParams, cambiar validarOpcional por
	// validarObligatorio (el primer caso se vuelve rojo) o borrar la llamada (el
	// segundo deja de fallar).
	_, err := llm.ParseClassification(json.RawMessage(
		`{"version": 1, "intent": "consulta_estado", "confidence": 0.7,
		  "params": {"numero_pedido": ""}, "evidence": "cómo va mi pedido"}`),
		entradaDeClasificacion())
	require.NoError(t, err)

	out, err := llm.ParseClassification(json.RawMessage(
		`{"version": 1, "intent": "consulta_estado", "confidence": 0.7,
		  "params": {"numero_pedido": "..."}, "evidence": "cómo va mi pedido"}`),
		entradaDeClasificacion())
	requiereErrorDeCalidad(t, err)
	assert.Nil(t, out)
}

// ecoDelEsquemaDeClassify saca del prompt REAL el objeto que el modelo puede
// repetir: se lo pasa a ExtractJSON tal cual y lo que sale es el esquema.
//
// El require.NoError de aquí no es decorado, es la prueba del agujero: ExtractJSON
// mira el prompt entero y devuelve su esquema como si fuera una respuesta. Ninguna
// heurística de extracción puede evitarlo —el esquema es JSON válido—, así que
// quien tiene que rechazarlo es ParseClassification.
func ecoDelEsquemaDeClassify(t *testing.T) json.RawMessage {
	t.Helper()
	eco, err := llm.ExtractJSON(llm.BuildClassifyRequestPrompt(entradaDeClasificacion()))
	require.NoError(t, err, "el prompt imprime el esquema y ExtractJSON lo saca: ése es el eco que hay que cazar aguas abajo")
	assert.Contains(t, string(eco), llm.PlaceholderEsquema)
	return eco
}

func TestParseClassification_EcoDelEsquemaEsErrorDeCalidad(t *testing.T) {
	// Es el peor modo de fallo del pipeline: un artefacto falso aceptado en
	// silencio, con Intent valiendo el relleno del esquema, contaminando todo lo
	// que venga detrás sin que nadie vea un error.
	//
	// Este test NO pinea ninguna guarda en concreto, y decirlo importa: el esquema
	// de P1 imprime el relleno en TRES campos —intent, el valor del param y
	// evidence—, así que lo cazan tres guardas independientes y quitar cualquiera
	// de ellas lo deja igual de verde (medido: quitando validarEnum y validarParams
	// a la vez, sigue pasando por validarObligatorio de evidence). Lo que este test
	// afirma es el resultado —el eco del prompt REAL no entra—, no el mecanismo.
	// Quien pinea el enum es TestParseClassification_IntentFueraDelCatalogoEsErrorDeCalidad,
	// cuyos casos traen evidencia y confianza buenas y solo fallan por el enum.
	out, err := llm.ParseClassification(ecoDelEsquemaDeClassify(t), entradaDeClasificacion())
	requiereErrorDeCalidad(t, err)
	assert.Nil(t, out)
}

// casoEco empareja un eco del esquema de prompt.go con el Parse* que lo recibe.
type casoEco struct {
	nombre string
	parse  func(json.RawMessage) error
	raw    string
}

// correrCasosEco exige que cada eco salga por ErrLLMQuality. Existe para que la
// tabla crezca sin que crezca ninguna función.
func correrCasosEco(t *testing.T, casos []casoEco) {
	t.Helper()
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			requiereErrorDeCalidad(t, c.parse(json.RawMessage(c.raw)))
		})
	}
}

// Adaptadores de un Parse* a la firma de casoEco: solo interesa el error.
func errDeMainIdeas(raw json.RawMessage) error  { _, err := llm.ParseMainIdeas(raw); return err }
func errDeItemSpecs(raw json.RawMessage) error  { _, err := llm.ParseItemSpecs(raw); return err }
func errDeQuantities(raw json.RawMessage) error { _, err := llm.ParseQuantities(raw); return err }
func errDeQuoteText(raw json.RawMessage) error  { _, err := llm.ParseQuoteText(raw); return err }

// P1 necesita su entrada para conocer el catálogo, así que su adaptador la cierra
// dentro en vez de recibirla.
func errDeClassification(raw json.RawMessage) error {
	_, err := llm.ParseClassification(raw, entradaDeClasificacion())
	return err
}

func TestParse_EcoDelEsquemaEsErrorDeCalidad(t *testing.T) {
	// Los raw de esta tabla son los esquemas que imprime prompt.go, copiados sin
	// tocar: son JSON válido y hasta llevan la version correcta, así que la única
	// forma de cazarlos es mirar el CONTENIDO de los campos obligatorios.
	// Mutación que lo pone rojo: en validarObligatorio, quitar la rama del
	// PlaceholderEsquema (dejar solo el rechazo del campo vacío).
	correrCasosEco(t, []casoEco{
		{
			nombre: "P2 · ideas principales",
			parse:  errDeMainIdeas,
			raw: `{"version": 1,
			       "wants": [{"idea": "...", "evidence": "..."}],
			       "delivery_hint": {"text": "...", "evidence": "..."}}`,
		},
		{
			nombre: "P2 · solo la pista de entrega hace eco",
			parse:  errDeMainIdeas,
			raw: `{"version": 1,
			       "wants": [{"idea": "una torta", "evidence": "quiero una torta"}],
			       "delivery_hint": {"text": "...", "evidence": "..."}}`,
		},
		{
			nombre: "P3 · especificaciones por ítem",
			parse:  errDeItemSpecs,
			raw: `{"version": 1,
			       "items": [{"product": "...", "variant": "...", "addon_candidates": ["..."],
			                  "customizations": ["..."], "notes": "...", "evidence": "..."}]}`,
		},
		{
			nombre: "P3 · solo la lista de añadidos hace eco",
			parse:  errDeItemSpecs,
			raw: `{"version": 1,
			       "items": [{"product": "torta", "addon_candidates": ["..."],
			                  "evidence": "quiero una torta"}]}`,
		},
		{
			nombre: "P4 · normalización",
			parse:  errDeQuantities,
			raw: `{"version": 1,
			       "delivery_date": "2026-07-22",
			       "delivery_date_basis": "message_ts=2026-07-13",
			       "items": [{"product": "...", "qty": 1,
			                  "addon_candidates": ["..."], "customizations": ["..."],
			                  "notes": "...", "evidence": "..."}]}`,
		},
		{
			nombre: "cotización",
			parse:  errDeQuoteText,
			raw:    `{"version": 1, "text": "..."}`,
		},
	})
}

func TestParse_CampoObligatorioVacioEsErrorDeCalidad(t *testing.T) {
	// El eco no es la única forma de artefacto hueco: un modelo que se queda sin
	// nada que decir devuelve la clave con la cadena vacía, y una evidencia vacía
	// no es rastreable a ninguna frase del original (design.md §7.3).
	// Mutación que lo pone rojo: en validarObligatorio, quitar la rama del campo
	// vacío (dejar solo el rechazo del PlaceholderEsquema).
	correrCasosEco(t, []casoEco{
		{
			// El eco del esquema de P1 lo caza el enum de intent, no esta rama;
			// la evidencia vacía con una intención REAL es el caso que solo
			// depende de validarObligatorio.
			nombre: "P1 · evidencia vacía con una intención del catálogo",
			parse:  errDeClassification,
			raw:    `{"version": 1, "intent": "intake_request", "confidence": 0.9, "evidence": ""}`,
		},
		{
			nombre: "P2 · idea vacía",
			parse:  errDeMainIdeas,
			raw:    `{"version": 1, "wants": [{"idea": "", "evidence": "quiero una torta"}]}`,
		},
		{
			nombre: "P3 · evidencia solo con espacios",
			parse:  errDeItemSpecs,
			raw:    `{"version": 1, "items": [{"product": "torta", "evidence": "   "}]}`,
		},
		{
			nombre: "cotización sin texto",
			parse:  errDeQuoteText,
			raw:    `{"version": 1, "text": ""}`,
		},
	})
}

// TestParseQuantities_ReglasDeCantidadDelDesign cubre la parte del artefacto de
// P4 que NO es texto y por tanto no la caza el PlaceholderEsquema: la plantilla
// de fecha —que el prompt SÍ imprime, porque «AAAA-MM-DD» es reconocible— y el
// paquete y el rango en cero, que el prompt YA NO imprime: los imprimía, y por
// eso la etapa fue 0 de 14 en campo. Los casos se quedan porque la regla
// semántica sigue viva: lo que dejó de existir es la fuente del eco.
//
// Las reglas las fija design.md §7.3 del Plan 044: la cantidad omitida vale 1,
// «un paquete de 30» es un paquete de 30 unidades, y los rangos no se colapsan.
func TestParseQuantities_ReglasDeCantidadDelDesign(t *testing.T) {
	// Mutaciones que lo ponen rojo, una por caso: hacer que validarFechaEntrega,
	// validarPaquete o validarRango devuelvan nil de entrada, y quitar de
	// validarNormalizedItem la comprobación de it.Qty < 1.
	correrCasosEco(t, []casoEco{
		{
			nombre: "la plantilla AAAA-MM-DD no es una fecha",
			parse:  errDeQuantities,
			raw: `{"version": 1, "delivery_date": "AAAA-MM-DD",
			       "items": [{"product": "torta", "qty": 1, "evidence": "quiero una torta"}]}`,
		},
		{
			nombre: "paquete de cero unidades",
			parse:  errDeQuantities,
			raw: `{"version": 1,
			       "items": [{"product": "tequeños", "qty": 1, "unit_kind": "package",
			                  "package_size": 0, "evidence": "un paquete de tequeños"}]}`,
		},
		{
			nombre: "rango en cero",
			parse:  errDeQuantities,
			raw: `{"version": 1,
			       "items": [{"product": "torta", "qty": 1,
			                  "range": {"min": 0, "max": 0, "unit": "porciones"},
			                  "evidence": "de 10 o 12 porciones"}]}`,
		},
		{
			nombre: "cantidad en cero",
			parse:  errDeQuantities,
			raw: `{"version": 1,
			       "items": [{"product": "torta", "qty": 0, "evidence": "quiero una torta"}]}`,
		},
		{
			nombre: "unit_kind fuera del enum",
			parse:  errDeQuantities,
			raw: `{"version": 1,
			       "items": [{"product": "torta", "qty": 1, "unit_kind": "caja",
			                  "package_size": 6, "evidence": "una caja de seis"}]}`,
		},
	})
}

func TestParse_VersionDesconocidaEsErrorDeCalidad(t *testing.T) {
	_, err := llm.ParseMainIdeas(json.RawMessage(`{"version": 99, "wants": []}`))
	requiereErrorDeCalidad(t, err)

	// La versión ausente cuenta como desconocida: el artefacto va versionado
	// desde el día uno y un cero es «no me lo dijo».
	_, err = llm.ParseMainIdeas(json.RawMessage(`{"wants": []}`))
	requiereErrorDeCalidad(t, err)
}

func TestParse_ArtefactoIlegibleEsErrorDeCalidad(t *testing.T) {
	ilegibles := []json.RawMessage{
		nil,
		json.RawMessage("   "),
		json.RawMessage(`{"version": "uno"}`),
		json.RawMessage(`no soy json`),
	}
	for _, raw := range ilegibles {
		_, err := llm.ParseMainIdeas(raw)
		requiereErrorDeCalidad(t, err)
	}
}

func TestParse_ToleraCamposFuturos(t *testing.T) {
	// Mismo criterio que el módulo intents: un proveedor que añade una clave no
	// rompe al lector.
	out, err := llm.ParseMainIdeas(json.RawMessage(
		`{"version": 1, "wants": [], "campo_que_no_existia": true}`))
	require.NoError(t, err)
	assert.Empty(t, out.Wants)
}
