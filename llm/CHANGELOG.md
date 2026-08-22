# Changelog — llm

El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.0.0/)
y [Versionado Semantico](https://semver.org/lang/es/).

## [Unreleased]

### Added

- Version inicial del modulo `llm`: el puerto unico con el que wApp habla con un
  modelo de lenguaje, sus prompts compartidos y la implementacion contra API
  externa (Plan 044 Ola 0, T0.1 y T0.2; ADR-0030, ADR-0004).
  - Puerto `LLMProvider` con un metodo por tarea del pipeline
    (`ExtractMainIdeas`, `ExtractItemSpecs`, `NormalizeQuantities`,
    `GenerateQuoteText`), todos devuelven
    `json.RawMessage` que valida el caller. `Options{Temperature}` por llamada,
    con `TemperatureGreedy` (0) y `TemperatureRetry` (0.3).
  - Prompts compartidos en espanol que exigen SOLO JSON valido:
    `BuildExtractMainIdeasPrompt`,
    `BuildExtractItemSpecsPrompt`, `BuildNormalizeQuantitiesPrompt`,
    `BuildGenerateQuoteTextPrompt`.
  - Robustez de salida, en DOS capas. Extraccion: `ExtractJSON` prefiere el
    contenido de la valla de codigo cuando la hay, y si no la hay prueba
    candidatos desde cada apertura de objeto del texto (balanceo de llaves
    respetando cadenas y escapes), mas desenvoltura unica de envolturas espurias
    (`{"bytes":{...}}` y las claves `result`, `data`, `response`, `output`,
    `json`). Truncado ⇒ `ErrLLMQuality`, tanto si el tope es un objeto como si
    es un array.
  - Parsers y artefactos versionados `{"version":1,...}`: `ParseMainIdeas`,
    `ParseItemSpecs`, `ParseQuantities`, `ParseQuoteText`.
    Segunda capa de la robustez: rechazan por `ErrLLMQuality` el ESQUEMA repetido
    —el valor `PlaceholderEsquema` (`"..."`) en un campo obligatorio, la
    plantilla `AAAA-MM-DD` como fecha, un `unit_kind` fuera de
    `UnitKindPackage`.

    D-044.23 (2026-08-22): el metodo `ClassifyRequest` y todo su andamiaje
    —`ClassifyRequestInput`, `BuildClassifyRequestPrompt`, `Classification`,
    `ParseClassification`— NO entran en esta version. La clasificacion P1 la
    hace el Edge (`wapp-edge-intent`) y ninguna ola del Plan 044 lo llama, asi
    que publicarlo seria estrenar superficie sin consumidor —lo que D-044.21
    condena— y quitarlo despues seria un breaking change. Vuelve el dia que P1
    tenga dueno en la nube; el codigo esta en la historia de git.
  - Error centinela `ErrLLMQuality`, separado de los errores de infraestructura:
    la calidad se reintenta una vez con otra temperatura, la infraestructura se
    reintenta mas tarde. Los providers NO reintentan por su cuenta.
  - Paquete `llm/api`: `Config{Provider, APIKey, Model, BaseURL, Timeout,
    MaxTokens}` 100 % inyectada (el provider nunca lee el entorno); `anthropic`
    completo sobre la Messages API (`POST {base}/v1/messages`, headers
    `x-api-key` y `anthropic-version: 2023-06-01`, timeout 60 s, `max_tokens`
    4096, concatena los bloques de texto); `gemini` como stub que se construye y
    falla con `ErrNotImplemented`; cualquier otro proveedor —`local` incluido—
    falla al construir con `ErrUnsupportedProvider`.
  - Centinela `ErrUpstream` para los fallos de infraestructura del proveedor.
  - Alcance fijado por D-044.21: puerto simple por tarea. No hay enrutador, ni
    registro de vias, ni configuracion tarea→proveedor.
  - Solo stdlib en produccion (testify solo en tests).
