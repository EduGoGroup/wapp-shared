// Package llm define el puerto único con el que el ecosistema wApp habla con un
// modelo de lenguaje, junto con los prompts y los parsers que ambas vías (la API
// externa hoy, la local mañana) comparten.
//
// La forma la fija el ADR-0030 y la copia-adaptación de edugo-worker (ADR-0004:
// se copia la FORMA, jamás se importa un repo de EduGo):
//
//   - un método por tarea del pipeline, y cada uno devuelve json.RawMessage que
//     valida el caller — un modelo que alucine el formato se rechaza igual que
//     un JSON malo de cualquier otra fuente;
//   - los prompts y los parsers viven AQUÍ, fuera de las implementaciones, para
//     que entre vías cambie el transporte y nunca el prompt;
//   - ErrLLMQuality separa «el modelo devolvió basura» de «el proveedor está
//     caído», porque el tratamiento es distinto: la calidad se reintenta UNA vez
//     con TemperatureRetry y luego se aísla la unidad; la infraestructura es
//     transitoria y la vuelve a intentar el job.
//
// El ciclo de una tarea es siempre el mismo:
//
//	in := llm.ExtractMainIdeasInput{SourceText: hilo}
//	out, err := provider.ExtractMainIdeas(ctx, in, llm.Options{})
//	ideas, err := llm.ParseMainIdeas(out)
//
// El prompt lo construye la implementación con BuildExtractMainIdeasPrompt, que
// también vive aquí: así el harness mide el modelo y no diferencias de prompt.
//
// # Rescatar la salida son DOS capas, y hacen falta las dos
//
// ExtractJSON aísla el JSON de entre los adornos del modelo, pero no puede
// juzgar lo que aísla: los prompts imprimen el esquema de la respuesta y un
// modelo que lo repita produce JSON perfectamente válido. Por eso la segunda
// capa son los Parse*, que rechazan por ErrLLMQuality los valores fuera del enum
// declarado y los campos obligatorios que vienen vacíos o con el relleno
// PlaceholderEsquema. Sin esa segunda capa, un eco del esquema entra al pipeline
// SIN ERROR, que es el peor modo de fallo posible; sin la primera, cualquier
// cháchara alrededor del JSON lo tira.
//
// # Alcance: puerto simple por tarea, no enrutador (D-044.21)
//
// Este paquete expone la interfaz, los prompts, los parsers y el centinela, y
// NINGUNA decisión de «a qué vía va esta tarea». No hay registro de proveedores,
// ni factory tarea→proveedor, ni tabla de rutas: quien arranca el proceso elige
// UNA implementación y a quien llama le da igual cuál es. El día que exista la
// vía local se añade otra implementación del mismo puerto.
//
// # Trucos de la vía local (D-044.4, acotada por D-044.29)
//
// D-044.4 decía que la implementación local no se cablearía en el Plan 044; la
// DEROGA D-044.29: la vía local existe, y su adaptador vive en cloud-platform —no
// aquí—, hablando el frame de inferencia de CloudLink contra el Ollama del Edge.
// Los trucos ya estaban medidos y se dejaron escritos para que los heredara sin
// volver a pagarlos:
// POST /api/generate con stream:false, format:"json", think:false SIEMPRE (qwen3
// con format json y thinking activo emite un objeto vacío) y
// options.temperature 0 (sin fijarla Ollama usa 0.8 y el veredicto parpadea
// entre corridas). Ver design.md §3.1 del Plan 044.
//
// # Solo stdlib, y es un invariante
//
// El paquete no depende de otros módulos de wapp-shared: solo stdlib. No es una
// casualidad que se pueda romper sin coste. La tentación concreta la trae P1: el
// catálogo de intenciones contra el que clasifica vive en wapp-shared/intents, y
// pasarlo como *intents.Config aquí ataría el puerto genérico del modelo al
// contrato de negocio de un tenant y su cadencia de releases. Por eso
// ClassifyRequestInput lo recibe aplanado (IntentSpec), y quien aplana es el
// caller, que ya tiene la config en la mano.
package llm
