# llm

Puerto unico con el que el ecosistema wApp habla con un modelo de lenguaje, mas los
prompts y los parsers que comparten todas las vias. Copia-adaptacion de la **forma**
de `edugo-worker` (ADR-0004: se copia la forma, jamas se importa el repo). Solo
stdlib en produccion.

Lo consume `cloud-platform`: el pipeline de solicitudes interpretadas del Plan 044
corre **en el cloud**, con las credenciales del tenant, que nunca viajan al Edge
(ADR-0030).

> **Alcance (D-044.21): puerto simple por tarea, NO enrutador.** Este modulo expone la
> interfaz, los prompts, los parsers y el centinela — y ninguna decision de «a que via
> va esta tarea». No hay registro de proveedores, ni factory tarea→proveedor, ni tabla
> de rutas. Quien arranca elige UNA implementacion; a quien llama le da igual cual es.

## Instalacion

```bash
go get github.com/EduGoGroup/wapp-shared/llm
```

## Uso

```go
package main

import (
	"context"
	"errors"
	"log"

	"github.com/EduGoGroup/wapp-shared/llm"
	"github.com/EduGoGroup/wapp-shared/llm/api"
)

func main() {
	// La configuracion se INYECTA entera: el provider nunca lee el entorno.
	provider, err := api.New(api.Config{
		Provider: api.ProviderAnthropic,
		APIKey:   claveDescifradaDelTenant(),
		Model:    "claude-...",
	})
	if err != nil {
		log.Fatal(err) // proveedor desconocido o config incompleta: falla al arrancar
	}

	out, err := provider.ExtractMainIdeas(context.Background(),
		llm.ExtractMainIdeasInput{SourceText: hiloAgregado}, llm.Options{})
	switch {
	case errors.Is(err, llm.ErrLLMQuality):
		// El modelo respondio basura: UN reintento con llm.TemperatureRetry y,
		// si insiste, se aisla la unidad y el resto del trabajo sigue.
	case err != nil:
		// Infraestructura: se reintenta mas tarde, con el mismo prompt.
	}

	ideas, err := llm.ParseMainIdeas(out) // la validacion de negocio es del caller
	_ = ideas
}
```

## El puerto

`LLMProvider` tiene un metodo por tarea del pipeline, y todos devuelven
`json.RawMessage` — un modelo que alucine el formato se rechaza igual que un JSON malo
de cualquier otra fuente:

| Metodo | Etapa | Artefacto |
|---|---|---|
| `ClassifyRequest` | — | `Classification` |
| `ExtractMainIdeas` | P2 | `MainIdeas` |
| `ExtractItemSpecs` | P3 (una llamada POR item) | `ItemSpecs` |
| `NormalizeQuantities` | P4 | `Quantities` |
| `GenerateQuoteText` | cotizacion | `QuoteText` |

El ciclo es siempre: `Build<Tarea>Prompt` → la implementacion llama al modelo →
`ExtractJSON` → `Parse<Artefacto>` → validacion de negocio en Go, en el caller.

## Los dos errores, que no son el mismo

- **`llm.ErrLLMQuality`** — el proveedor respondio, pero lo que dijo no sirve: no hay
  objeto JSON, esta truncado, o la version del artefacto es otra. Se reintenta **una
  vez** con `TemperatureRetry` (0.3) y, si persiste, se **aisla** la unidad.
- **`api.ErrUpstream`** — el proveedor no respondio, tardo de mas o devolvio un codigo
  de error. Se reintenta **mas tarde**, con el mismo prompt.

Confundirlos hace que un incidente del proveedor se lea como un problema de calidad del
modelo. **Los providers NO reintentan por su cuenta**: el retry vive en el caller.

## Rescatar la salida son DOS capas

### Capa 1 — `ExtractJSON`

Aisla el objeto JSON de la salida cruda, en este orden:

1. **Si hay valla de codigo Markdown con algo estructural dentro, manda la valla** y se
   ignora todo lo de fuera.
2. Si el texto empieza por `[`, es un array de tope y se toma su **primer objeto**. Un
   array de tope que **no cierra** es salida truncada: `ErrLLMQuality`, nunca su primer
   objeto.
3. Si no, prueba candidatos desde **cada** `{` que abra un objeto de verdad (el siguiente
   byte con contenido es `"` o `}`) y gana el primero que balancee y sea JSON valido. Asi
   una prosa previa con una llave sin cerrar o con las comillas descuadradas deja de tapar
   una respuesta buena. Un candidato que abre y **nunca cierra** para el barrido: es la
   firma del JSON truncado y da `ErrLLMQuality`.
4. Al ganador se le quita **una sola vez** la envoltura espuria de los modelos chicos
   (`{"bytes":{...}}`; tambien `result`, `data`, `response`, `output`, `json`) cuando el
   objeto de arriba tiene **exactamente una** clave y lo que envuelve es otro objeto.

### Capa 2 — los `Parse<Artefacto>`

La extraccion sola **no basta**, y esto no es una precaucion teorica: los prompts imprimen
el esquema de la respuesta (`{"version": 1, "category": "...", "evidence": "..."}`), asi
que un modelo que lo repita produce **JSON valido**, con la `version` correcta, que sale
de `ExtractJSON` como si fuera una respuesta. Ninguna heuristica de extraccion puede
distinguirlo.

Por eso los `Parse*` rechazan con `ErrLLMQuality`:

- un campo obligatorio **vacio** o con el relleno `llm.PlaceholderEsquema` (`"..."`);
- una `category` que **no este en el conjunto cerrado** que el caller declaro — por eso
  `ParseClassification(raw, categorias)` recibe ese conjunto: el enum **no lo fija este
  paquete**, lo conoce el caller (design.md §3.2 del Plan 044 solo deja escrito que el
  score continuo no funciona y las categorias si);
- un `delivery_date` que sea la plantilla `AAAA-MM-DD` y no una fecha;
- un `unit_kind` fuera de `llm.UnitKindPackage`, un paquete de cero unidades, una `qty`
  por debajo de 1 o un rango que no es un rango.

## La implementacion `api`

`api.Config{Provider, APIKey, Model, BaseURL, Timeout, MaxTokens}`, inyectada entera.

- **`anthropic`** — completo. `POST {BaseURL}/v1/messages`, headers `x-api-key` y
  `anthropic-version: 2023-06-01`, `Timeout` por defecto 60 s, `MaxTokens` por defecto
  4096, concatena los bloques de contenido de tipo texto.
- **`gemini`** — stub: se **construye** (un tenant puede tenerlo configurado) y cada
  llamada falla con `ErrNotImplemented`, nombrando la tarea.
- **cualquier otro**, `local` incluido — falla al **construir**, con
  `ErrUnsupportedProvider`. La via local no esta cableada en el Plan 044, y su rechazo
  con mensaje propio vive en la capa de configuracion del tenant, no aqui.

`Model` y `APIKey` **no tienen valor por defecto**: inventarlos convierte un error de
configuracion en una factura.

## La via local, anotada antes de existir

Los trucos ya medidos quedan escritos para que la implementacion local los herede sin
volver a pagarlos: `POST /api/generate` con `stream:false`, `format:"json"`,
**`think:false` SIEMPRE** (qwen3 con `format:json` y thinking activo emite un objeto
vacio) y `options.temperature 0` (sin fijarla Ollama usa 0.8 y el veredicto parpadea
entre corridas). Ver `docs/plans/044-carrito-llm-2-presupuestos/design.md` §3.1.

## Navegacion

- [Changelog](CHANGELOG.md)

## Comandos disponibles

```bash
make build     # Compilar
make test      # Tests
make check     # Lint y validacion
```
