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

// ClassifyRequestInput es la entrada de ClassifyRequest: un texto y el conjunto
// CERRADO de categorías entre las que el modelo debe elegir una.
//
// Las categorías se pasan por parámetro y no se declaran aquí porque quien las
// conoce es el caller. Y son categorías, no un score continuo: la lección está
// medida (design.md §3.2 del Plan 044 — «score continuo no funciona, tres
// categorías sí»), por eso el artefacto de esta tarea no lleva confianza.
type ClassifyRequestInput struct {
	// Text es el texto a clasificar, tal cual lo escribió el cliente.
	Text string
	// Categories son las etiquetas permitidas. El modelo debe devolver una de
	// ellas, literal; el caller rechaza cualquier otra cosa.
	Categories []string
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
	// ClassifyRequest elige una de las categorías dadas para un texto.
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
