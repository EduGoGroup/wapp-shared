package llm

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// promptHeader es la cabecera común de los prompts de análisis del pipeline.
//
// Está en español a propósito: el material que analiza son conversaciones de
// WhatsApp en español, y mezclar idiomas entre la instrucción y el dato empeora
// la extracción de la evidencia literal, que es justo lo único que este pipeline
// le pide al modelo.
const promptHeader = `Analizas mensajes que un cliente le escribió por WhatsApp a un negocio pequeño.
Tu trabajo es entender lo que el cliente pidió. No respondes al cliente, no
saludas, no propones precios y no inventas nada.`

// jsonOnlyRules son las reglas de formato que TODOS los prompts repiten.
//
// La salida se pasa después por ExtractJSON, que tolera adornos, pero pedirlo
// explícito reduce las salidas degeneradas: cuanto menos tenga que rescatar el
// parser, menos reintentos por calidad se pagan.
const jsonOnlyRules = `Reglas de salida, sin excepciones:
- Responde ÚNICAMENTE con un objeto JSON válido.
- No escribas absolutamente nada antes ni después del JSON.
- No uses vallas de código Markdown, ni comentarios, ni explicaciones.
- No envuelvas la respuesta en otro objeto.
- Usa exactamente las claves del esquema, sin añadir ni quitar ninguna.
- Todo campo de evidencia debe ser una FRASE COPIADA LITERALMENTE del texto del
  cliente, no un resumen tuyo. Si no puedes copiar una frase, omite el dato.`

// weekdaysES nombra los días para que la fecha de referencia del prompt se lea
// como la lee el cliente («el miércoles de la semana que viene»).
var weekdaysES = [...]string{
	"domingo", "lunes", "martes", "miércoles", "jueves", "viernes", "sábado",
}

// BuildClassifyRequestPrompt arma el prompt de ClassifyRequest.
func BuildClassifyRequestPrompt(in ClassifyRequestInput) string {
	categorias := make([]string, 0, len(in.Categories))
	for _, c := range in.Categories {
		categorias = append(categorias, "- "+c)
	}

	instruccion := `

Clasifica el texto del cliente en UNA de estas categorías, copiada literal:
` + strings.Join(categorias, "\n") + `

Si ninguna encaja, elige la que menos se aleje: no inventes categorías nuevas.
No devuelvas un número de confianza; la categoría es la respuesta.

`
	esquema := fmt.Sprintf(`

Esquema de la respuesta:
{"version": %d, "category": "...", "evidence": "..."}
`, ArtifactVersion)

	return promptHeader + instruccion + jsonOnlyRules + esquema +
		"\nTexto del cliente:\n" + in.Text
}

// BuildExtractMainIdeasPrompt arma el prompt de la etapa P2.
func BuildExtractMainIdeasPrompt(in ExtractMainIdeasInput) string {
	instruccion := `

Saca las ideas principales de lo que el cliente quiere. Una entrada por cosa
distinta que pide: si menciona dos tortas diferentes, son dos entradas. Escribe
cada idea con las palabras del propio cliente, sin normalizar cantidades ni
resolver fechas: eso se hace después.

Si el cliente dice cuándo lo necesita, ponlo en delivery_hint tal como lo dijo
(«el miércoles de la semana que viene»), sin convertirlo a fecha.

`
	esquema := fmt.Sprintf(`

Esquema de la respuesta:
{"version": %d,
 "wants": [{"idea": "...", "evidence": "..."}],
 "delivery_hint": {"text": "...", "evidence": "..."}}

Si el cliente no dice cuándo, omite delivery_hint.
`, ArtifactVersion)

	return promptHeader + instruccion + jsonOnlyRules + esquema +
		"\nTexto del cliente:\n" + in.SourceText
}

// BuildExtractItemSpecsPrompt arma el prompt de la etapa P3.
//
// Se llama UNA vez por ítem, con contexto fresco: el prompt lleva el hilo entero
// como contexto pero le pide al modelo que se ocupe de una sola idea.
func BuildExtractItemSpecsPrompt(in ExtractItemSpecsInput) string {
	instruccion := `

Especifica UN SOLO ítem: el que se te indica más abajo. Ignora los demás, aunque
aparezcan en el texto.

Separa dos cosas que se parecen y no son lo mismo:
- addon_candidates: añadidos que PODRÍAN ser un artículo aparte del catálogo
  («decoración infantil», «extra de queso»).
- customizations: indicaciones que no son un artículo y nunca se cobran aparte
  («sin sal», «sin cebolla», «sin lactosa»).
Ante la duda, ponlo en addon_candidates: quien decide si existe en el catálogo es
el sistema, no tú.

Los rangos se copian TAL CUAL en variant («10 o 12 porciones»): no elijas un
número ni promedies.

`
	esquema := fmt.Sprintf(`

Esquema de la respuesta:
{"version": %d,
 "items": [{"product": "...", "variant": "...", "addon_candidates": ["..."],
            "customizations": ["..."], "notes": "...", "evidence": "..."}]}
`, ArtifactVersion)

	return promptHeader + instruccion + jsonOnlyRules + esquema +
		"\nÍtem que debes especificar:\n" + in.Idea +
		"\n\nTexto completo del cliente (contexto y fuente de la evidencia):\n" + in.SourceText
}

// BuildNormalizeQuantitiesPrompt arma el prompt de la etapa P4.
//
// La fecha de referencia viaja en el prompt porque las expresiones relativas se
// resuelven contra la fecha DEL MENSAJE y no contra hoy (D-044.9): un trabajo
// reanudado dos días después tiene que dar la misma fecha.
func BuildNormalizeQuantitiesPrompt(in NormalizeQuantitiesInput) string {
	ref := in.MessageTS.UTC()
	fecha := ref.Format(time.DateOnly)

	instruccion := fmt.Sprintf(`

Normaliza las cantidades de los ítems que se te dan.

Reglas:
- Si el cliente no dijo cuántos, qty vale 1.
- «Un paquete de 30» es qty 1 con unit_kind "package" y package_size 30. Nunca
  es qty 30.
- Los rangos se conservan como rango: {"min": 10, "max": 12, "unit": "porciones"}.
  No los colapses a un número.
- La fecha de referencia es el %s %s (la fecha del mensaje, no la de hoy).
  Resuelve contra ella las expresiones relativas y devuelve delivery_date en
  formato AAAA-MM-DD.

`, weekdaysES[ref.Weekday()], fecha)

	esquema := fmt.Sprintf(`

Esquema de la respuesta:
{"version": %d,
 "delivery_date": "AAAA-MM-DD",
 "delivery_date_basis": "message_ts=%s",
 "items": [{"product": "...", "qty": 1,
            "range": {"min": 0, "max": 0, "unit": "..."},
            "unit_kind": "package", "package_size": 0,
            "addon_candidates": ["..."], "customizations": ["..."],
            "notes": "...", "evidence": "..."}]}

Omite range, unit_kind y package_size cuando no apliquen. Omite delivery_date si
el cliente no dijo cuándo.
`, ArtifactVersion, fecha)

	return promptHeader + instruccion + jsonOnlyRules + esquema +
		"\nÍtems a normalizar:\n" + marshalForPrompt(in.Items) +
		"\n\nTexto completo del cliente (fuente de la evidencia):\n" + in.SourceText
}

// BuildGenerateQuoteTextPrompt arma el prompt del generador de cotización.
//
// No usa promptHeader: aquí el modelo no analiza lo que pidió el cliente, redacta
// lo que el negocio ya decidió responder.
func BuildGenerateQuoteTextPrompt(in GenerateQuoteTextInput) string {
	instruccion := `Redactas el mensaje con el que un negocio pequeño le pasa una cotización a su
cliente por WhatsApp. Escribes como escribe ese negocio, no como escribe una
empresa grande.

Reglas duras:
- Los importes y las cantidades se copian EXACTAMENTE del borrador. No sumes, no
  redondees, no conviertas moneda, no añadas ni quites líneas.
- No inventes plazos, condiciones ni descuentos que no estén en el borrador.
- Es un mensaje de WhatsApp: sin encabezados de carta y sin firma corporativa.

`
	esquema := fmt.Sprintf(`

Esquema de la respuesta:
{"version": %d, "text": "..."}
`, ArtifactVersion)

	ejemplos := ""
	if len(in.Examples) > 0 {
		bloques := make([]string, 0, len(in.Examples))
		for i, ex := range in.Examples {
			bloques = append(bloques, fmt.Sprintf("--- ejemplo %d ---\n%s", i+1, ex))
		}
		ejemplos = "\nAsí escribe este negocio (imita el tono, no el contenido):\n" +
			strings.Join(bloques, "\n") + "\n"
	}

	return instruccion + jsonOnlyRules + esquema + ejemplos +
		"\nBorrador a redactar:\n" + string(in.Quote) + "\n"
}

// marshalForPrompt serializa un valor para incrustarlo en un prompt.
//
// Los tipos que pasan por aquí son structs de este paquete, serializables por
// construcción. Si alguna vez fallara, el prompt se queda sin datos y la etapa
// termina fallando por calidad aguas abajo, que es el modo de fallo correcto:
// una etapa sin material no debe inventarse el resultado.
func marshalForPrompt(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(b)
}
