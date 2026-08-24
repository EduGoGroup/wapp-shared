package llm

import (
	"context"
	"encoding/json"
	"time"
)

// Temperaturas previstas por el pipeline. Son las dos únicas que existen: la de
// siempre y la del reintento por calidad. No hay barrido de temperaturas ni
// ajuste por tarea (REQ-03).
const (
	// TemperatureGreedy es la temperatura por defecto de toda llamada: 0, es
	// decir greedy determinista. Es el valor cero de Options.Temperature, así
	// que una Options vacía ya pide greedy.
	TemperatureGreedy = 0.0

	// TemperatureRetry es la temperatura del reintento ÚNICO que el caller hace
	// cuando una etapa falla con ErrLLMQuality. Sube lo justo para que el modelo
	// no repita palabra por palabra la salida degenerada.
	TemperatureRetry = 0.3
)

// Options son los ajustes de UNA llamada concreta al modelo. Todo lo que no
// cambia entre llamadas (proveedor, modelo, credencial, timeout) vive en la
// configuración de la implementación y no aquí.
type Options struct {
	// Temperature es la temperatura de muestreo. El valor cero es
	// TemperatureGreedy; el caller solo la mueve a TemperatureRetry en el
	// reintento por calidad.
	Temperature float64
}

// IntentExample es un mensaje de muestra de una intención, opcionalmente anotado
// con los valores de parámetro que ese mensaje contiene.
//
// Los ejemplos NO son decorado. Está medido en campo (wapp-edge-intent) que en un
// modelo de 1–2B el few-shot pesa MÁS que las instrucciones: sin él se cae tanto
// el mapeo de intenciones como la extracción de parámetros. Y la vía local del
// Plan 044 ejecuta justo un modelo de ese tamaño, así que un catálogo sin ejemplos
// produce un prompt legal y flojo.
type IntentExample struct {
	// Message es el mensaje de muestra, tal como lo escribiría el cliente.
	Message string
	// Params son los valores que ese mensaje contiene, por nombre de parámetro.
	// Puede venir vacío.
	Params map[string]string
}

// IntentSpec es UNA intención del catálogo contra el que se clasifica.
//
// Es una forma POBRE a propósito. El catálogo real de wApp vive en el módulo
// wapp-shared/intents (Config/Intent/Ejemplo) y este paquete NO lo importa: la
// regla de no depender de otro módulo de wapp-shared es un invariante escrito de
// este paquete (ver doc.go), y el puerto no tiene por qué conocer el contrato de
// negocio de un tenant. Quien aplana su config a estas cuatro cosas —lo único que
// el prompt necesita— es el caller.
type IntentSpec struct {
	// Name es el nombre de la intención, que el modelo copia literal en la
	// respuesta. El conjunto de nombres del catálogo es el enum cerrado que
	// ParseClassification exige.
	Name string
	// Description le dice al modelo cuándo aplica esta intención.
	Description string
	// Params son los nombres de parámetro que esta intención extrae. Puede venir
	// vacío, y lo normal hoy es que lo esté: el contrato publicado en campo
	// declara `params: []` (D-044.20).
	Params []string
	// Examples son los mensajes de muestra (few-shot) de esta intención.
	Examples []IntentExample
}

// ClassifyRequestInput es la entrada de la etapa P1: un texto y el catálogo
// CERRADO de intenciones del tenant, entre las que el modelo elige una.
//
// El catálogo viaja por parámetro y no lo declara este paquete porque quien lo
// conoce es el caller: lo configura cada tenant y cambia sin tocar código.
//
// A diferencia del resto del pipeline, el artefacto de esta etapa SÍ lleva un
// número de confianza, y no contradice la lección medida de design.md §3.2 del
// Plan 044 («score continuo no funciona → 3 categorías»): esa lección es sobre
// pedirle al modelo que PUNTÚE una comparación, no sobre esto. Aquí la respuesta
// sigue siendo una etiqueta de un conjunto cerrado, y la confianza es el mecanismo
// con el que el ecosistema decide entre actuar y no actuar —el umbral por tenant,
// que ya gobierna al clasificador que corre en campo—. Quien compara contra ese
// umbral es el caller; este paquete solo exige que el número esté acotado.
type ClassifyRequestInput struct {
	// Text es el texto a clasificar, tal cual lo escribió el cliente.
	Text string
	// Catalog son las intenciones permitidas. El modelo debe devolver el nombre
	// de una de ellas, literal; ParseClassification rechaza cualquier otro.
	Catalog []IntentSpec
	// UnknownLabel es la etiqueta de escape para el mensaje que no encaja en
	// ninguna intención. Si viene, el prompt se la ofrece al modelo y
	// ParseClassification la acepta aunque NO esté en Catalog —por eso no hace
	// falta declararla como una intención más—. Si va vacía, el modelo no tiene
	// salida y ha de elegir del catálogo con la confianza que corresponda.
	UnknownLabel string
	// Vocabulary son pistas de dominio del negocio (nombres de producto, jerga)
	// que anclan la extracción. Opcional.
	Vocabulary []string
}

// ExtractMainIdeasInput es la entrada de la etapa P2: el hilo agregado completo.
type ExtractMainIdeasInput struct {
	// SourceText es el hilo de la conversación ya agregado por la O1.
	SourceText string
}

// ExtractItemSpecsInput es la entrada de la etapa P3. Se hace UNA llamada POR
// ítem (contexto fresco por llamada), así que Idea trae una sola de las ideas
// que devolvió P2 y SourceText queda como contexto para anclar la evidencia.
type ExtractItemSpecsInput struct {
	// SourceText es el hilo completo; sirve de contexto y es el texto contra el
	// que el caller comprueba después que la evidencia es una subcadena real.
	SourceText string
	// Idea es la idea de P2 que esta llamada debe especificar. Una por llamada.
	Idea string
}

// NormalizeQuantitiesInput es la entrada de la etapa P4.
type NormalizeQuantitiesInput struct {
	// SourceText es el hilo completo, para que la evidencia siga siendo
	// rastreable a una frase del original.
	SourceText string
	// Items son los ítems que devolvió P3.
	Items []ItemSpec
	// MessageTS es la fecha del PRIMER mensaje del hilo agregado. Es la
	// referencia con la que se resuelven las expresiones relativas («el
	// miércoles de la semana que viene»). Ver D-044.9.
	MessageTS time.Time
}

// GenerateQuoteTextInput es la entrada del generador de cotización.
//
// El borrador viaja como JSON ya construido por el caller y no como una
// estructura de este paquete: su forma la fijan las olas 3 y 5 del Plan 044
// (design.md §7.4), no el puerto, y duplicarla aquí sería inventar un modelo de
// líneas paralelo al que ya existe.
type GenerateQuoteTextInput struct {
	// Quote es el borrador aprobado, en JSON, con sus líneas y sus precios.
	Quote json.RawMessage
	// Examples son cotizaciones anteriores del mismo tenant que sirven de
	// few-shot para imitar su voz. Puede venir vacío.
	Examples []string
}

// LLMProvider es el puerto único: un método por tarea del pipeline, cada uno
// devuelve el json.RawMessage que el caller valida.
//
// Contrato que toda implementación cumple:
//
//   - construye el prompt con el Build...Prompt de este paquete, nunca con uno
//     propio;
//   - devuelve el resultado ya pasado por ExtractJSON, de modo que quien llama
//     recibe un objeto JSON o un error;
//   - envuelve ErrLLMQuality cuando el modelo respondió algo no interpretable, y
//     un error propio de la implementación cuando el fallo es de transporte;
//   - NO reintenta: el reintento único por calidad lo decide el caller.
//
//nolint:revive // llm.LLMProvider tartamudea; el nombre lo fijan REQ-01 y el ADR-0030, no se renombra.
type LLMProvider interface {
	// ClassifyRequest es la etapa P1: elige UNA intención del catálogo del
	// tenant para el texto dado, con su confianza y los parámetros que el
	// cliente dijo. NO aplica el umbral ni sanea los parámetros: eso es del
	// caller, que es quien tiene el texto original y la config del tenant.
	ClassifyRequest(ctx context.Context, in ClassifyRequestInput, opts Options) (json.RawMessage, error)
	// ExtractMainIdeas es la etapa P2: saca las ideas principales del hilo.
	ExtractMainIdeas(ctx context.Context, in ExtractMainIdeasInput, opts Options) (json.RawMessage, error)
	// ExtractItemSpecs es la etapa P3: especifica UN ítem por llamada.
	ExtractItemSpecs(ctx context.Context, in ExtractItemSpecsInput, opts Options) (json.RawMessage, error)
	// NormalizeQuantities es la etapa P4: cantidades, paquetes, rangos y fecha.
	NormalizeQuantities(ctx context.Context, in NormalizeQuantitiesInput, opts Options) (json.RawMessage, error)
	// GenerateQuoteText redacta la cotización con la voz del negocio.
	GenerateQuoteText(ctx context.Context, in GenerateQuoteTextInput, opts Options) (json.RawMessage, error)
}
