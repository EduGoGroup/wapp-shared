# Changelog — llm

El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.0.0/)
y [Versionado Semantico](https://semver.org/lang/es/).

## [Unreleased]

## [0.4.1] - 2026-08-26

### Fixed

- 🔴 **La plantilla de P4 imprimía lo que su propio validador rechaza** (Plan 044).
  `BuildNormalizeQuantitiesPrompt` enseñaba en el esquema de salida
  `"unit_kind": "package", "package_size": 0` y `"range": {"min": 0, "max": 0, "unit": "..."}`,
  y eso es exactamente lo que rechazan `validarPaquete` (`package_size < 1` con
  `unit_kind "package"`) y `validarRango` (`unit` de relleno y `min < 1`). **El modelo no
  desobedecía: copiaba el ejemplo que se le daba.** La etapa fue **0 de 14** en su primer
  día en campo. Ahora la plantilla lleva los MISMOS valores del ejemplo de sus reglas
  (`package_size 30`, `{"min": 10, "max": 12, "unit": "porciones"}`), válidos para el
  validador, más la coletilla de que esos números son un ejemplo de la forma.
  - **La regla que ordena esto**: un `"..."` puede quedarse en la plantilla porque es un
    relleno **RECONOCIBLE** —`PlaceholderEsquema` lo caza si el modelo lo ecoa—, pero un
    `0` es indistinguible de un valor real: no se puede detectar, solo rechazar. **En la
    plantilla no puede haber ningún valor numérico que su propio validador rechace.**
  - Las claves opcionales **no** se sacan del esquema: `jsonOnlyRules` le ordena al modelo
    «usa exactamente las claves del esquema, sin añadir ni quitar ninguna», así que una
    clave ausente de la forma es una clave que el modelo no tiene permiso de emitir.

### Added

- **La regla gemela que faltaba en P4**: existía «si el cliente no dijo cuántos, `qty`
  vale 1» y no existía su equivalente para el tamaño del paquete. Ahora el prompt dice
  «*Si es un paquete pero el cliente no dijo cuántas unidades trae, omite `unit_kind` y
  `package_size`: no pongas 0 ni te inventes el tamaño*» — que es la **única** salida que
  el contrato ofrece: `validarPaquete` solo sale por arriba si `unit_kind` viene vacío.
- `TestPlantillaDelPrompt_PasaSuPropioValidador`: test de **regla** sobre las **cinco**
  etapas que extrae la plantilla del prompt con el `ExtractJSON` de producción, rellena
  sus huecos reconocibles y exige que el `Parse*` de esa misma etapa la **acepte** (y que
  cruda la siga **rechazando**). Las cinco pasan.

## [0.4.0] - 2026-08-25

### Changed

- 🔴 **BREAKING · P1 solo DETECTA: se retira `params` del prompt de clasificación**
  (Plan 044, Ola 1.8, T1.8-3; decisión **D3**, **Enmienda 1 del ADR-0045**).
  `BuildClassifyRequestPrompt` deja de pedir `params`: desaparecen de la línea del
  catálogo, de la regla que los explicaba, del esquema de salida y del few-shot. El
  esquema que P1 pide pasa a ser
  `{"version": N, "intent": "...", "confidence": 0.0, "evidence": "..."}`.
  - **Por qué**: la descomposición en ítems es trabajo de **P2–P4**, y pedírsela también a
    P1 le hacía **perder ítems** (D-044.20). Y desde la **Ola 1.6** (pull, ADR-0045)
    **nadie consumía** lo que P1 extraía: `sig.Intent` sale siempre `nil`, el pool los
    descarta y en UAT hay **cero** reglas `kind='llm'`. Era un campo que costaba tokens en
    cada mensaje y no alimentaba a nadie.
  - **Medido** contra el VPS, catálogo real de UAT y parámetros de producción: el acierto
    de intent **no baja** (**22/24** antes y después), el `prompt_eval_count` cae de
    **2.221 a 1.977 tokens (−11,0 %)**, el prompt de **7.749 a 6.849 B (−11,6 %)**, y las
    respuestas que traían `params` pasan de **9 de 24 a 0 de 24** — la prueba de que la
    retirada surtió efecto **en el modelo** y no solo en el texto del prompt.
  - ⚠️ **El lote del eval es de calidad C** (24 frases redactadas por Claude, no mensajes
    de clientes): sirve como **detector de regresión**, **no** como medida de acierto
    absoluto. Va como fixture reutilizable en `llm/testdata/eval-intents-c.json`, con sus
    condiciones de uso **dentro del propio fichero**.

### Unchanged (a propósito)

- **`ParseClassification` NO cambia**: sigue aceptando una respuesta **con** `params` —un
  Edge sin actualizar los seguirá mandando— y otra **sin**, y produce el mismo
  `ClassifiedIntent`. Eso es lo que permite desplegar **sin coordinar versiones** entre
  Cloud y Edge, y por eso el *breaking* es del **contrato del prompt**, no de la API de Go.

## [0.3.0] - 2026-08-24

### Changed

- **Los prompts se reordenan para que el prefijo cacheable sea el prompt casi entero**
  (Plan 044, Ola 1.7, T1.7-1; invariante **I6** del ADR-0046). Ollama reutiliza el
  KV-cache del **prefijo** del prompt: lo que va DETRÁS del primer byte que cambia entre
  dos llamadas se re-prefilla entero, a ~21,6 ms/token. Un prompt con un dato variable a
  mitad de camino paga ese prefill en CADA llamada. Cambia el texto que se le manda al
  modelo, no la API: ninguna firma se mueve.
  - **P4 (`BuildNormalizeQuantitiesPrompt`) era el caso malo**: la fecha de referencia
    (D-044.9) se interpolaba en la regla —byte 558— y otra vez en el esquema —byte 1304—,
    así que de 1967 B solo **558 B eran prefijo estable (28,4 %)** y los ~1.400 B de
    literal que quedaban detrás se re-prefillaban cada día. Ahora el cuerpo lleva el
    literal **`message_ts=AAAA-MM-DD`**, que es un FORMATO y no un valor, y **la fecha
    real se emite al FINAL**, pegada a los ítems y al texto que se normalizan. El prefijo
    estable pasa a **96,6 %**. La semántica se conserva: la instrucción dice
    explícitamente que la fecha de referencia va al final y sigue siendo «la del mensaje,
    no la de hoy», y el esquema dice que `AAAA-MM-DD` es la forma con la que hay que
    copiarla en `delivery_date_basis`. El prompt crece de 1967 a 2331 B —la explicación
    de dónde está la fecha cuesta bytes—, pero esos bytes son estables y se prefillan una
    vez, no una por llamada.
  - **P3 (`BuildExtractItemSpecsPrompt`): el `SourceText` va ahora ANTES que la `Idea`.**
    P3 se llama **una vez por ítem** con el MISMO hilo, así que el hilo es lo estable
    entre las N llamadas y la `Idea` es lo único que cambia. Con el orden anterior cada
    ítem re-prefillaba el hilo entero; ahora N−1 de las N llamadas entran calientes hasta
    la `Idea`. Prefijo estable **87,6 % → 97,9 %**.
  - **P1, P2 y P5 no se tocan**: ya cumplían (96,8 %, 95,0 % y 97,4 %).

### Added

- **Test de REGLA `TestPrompts_LoEstableVaDelante`** (`prompt_prefijo_test.go`): uno solo
  que recorre los **cinco** constructores en vez de cinco tests de conducta, porque el
  invariante es el mismo en los cinco sitios. Para cada etapa construye dos prompts con
  entradas que difieren **solo en lo que tiene derecho a variar** —dos mensajes del mismo
  tenant en P1/P2, **dos ítems del mismo pedido** en P3, dos fechas en P4, dos borradores
  en P5— y exige que compartan un prefijo común **byte a byte ≥ 90 %** del más corto.
  El umbral no se toca: si un constructor no llega, el arreglo es mover el dato variable
  al final del prompt, no aflojar el detector.
  - **Mide DOS cosas y el fallo dice cuál se rompió**, porque un test que falla por una
    razón que no es la que dice se paga caro. (1) **El orden**, guarda dura e
    **independiente del tamaño de la entrada**: detrás del primer byte que cambia no puede
    quedar ni un literal del andamiaje del prompt —un literal no depende de la entrada, así
    que si aparece ahí es que está colocado después de un dato variable—; el mensaje empieza
    por `SE ROMPIÓ EL ORDEN, no el tamaño` y **nombra el literal** que se quedó atrás.
    (2) **El ratio** del 90 %; su mensaje empieza por `EL ORDEN ESTÁ BIEN` y manda mirar el
    tamaño de las entradas, no reordenar el prompt.
  - ⚠️ **Limitación conocida, documentada en el propio fichero**: el ratio es prefijo/total,
    así que depende también del **tamaño de las entradas del test**. P2 cumple el 90 %
    mientras el mensaje más corto del par no pase de **~149 B**; con un hilo real largo el
    ratio baja **sin que haya defecto de código**. Para eso está la guarda del orden.
  - Verificado con **tres mutaciones ejecutadas**: devolver la fecha de P4 al cuerpo y poner
    la `Idea` delante en P3 fallan **por la guarda del orden**, nombrando el literal exacto;
    y dos hilos largos en las entradas, con `prompt.go` intacto, fallan **solo por el
    ratio**, con el mensaje de tamaño. Más control A/B contra el código de `llm/v0.2.0`,
    donde sale **rojo** en P3 y P4 y verde en las otras tres.


## [0.2.0] - 2026-08-24

### Added

- `ClassifyRequest` **vuelve al puerto** — la etapa P1 del pipeline (Plan 044 Ola 1.6,
  T1.6-3; D-044.29, ADR-0045). Es un cambio **aditivo** sobre 0.1.0 para quien llama, y
  **rompedor para quien implementa** `LLMProvider`: la interfaz pasa de cuatro metodos a
  cinco.
  - **Por que vuelve.** D-044.23 (2026-08-22) lo saco de 0.1.0 el dia antes de publicar
    porque no tenia **ni un consumidor** en codigo: la clasificacion P1 la hacia el Edge
    (`wapp-edge-intent`) y ninguna ola del plan lo llamaba, asi que publicarlo era
    estrenar superficie sin dueno. D-044.29 le da el dueno: el Cloud **pide** P1 por la
    via configurada del tenant dentro de su ventana de agregacion (pull), y un resultado
    por encima del umbral adelanta el cierre de la ventana (REQ-09, REQ-35, T1.6-4). La
    condicion que D-044.23 puso para su regreso —«vuelve el dia que P1 tenga dueno en la
    nube»— se cumplio.
  - **No vuelve igual que se fue**, porque el proposito es otro. Entonces clasificaba un
    texto contra un conjunto de categorias sueltas y devolvia `{category, evidence}` SIN
    confianza. Ahora clasifica contra el **catalogo de intenciones del tenant** y devuelve
    `Classification{version, intent, confidence, params, evidence}`: la misma forma que ya
    produce el clasificador que corre en campo, que es lo que permite que la via local y
    la via API respondan lo mismo.
  - La **confianza** entra porque el umbral por tenant es el mecanismo con el que el
    ecosistema decide entre actuar y no actuar. No contradice la leccion medida de
    design.md §3.2 («score continuo no funciona → 3 categorias»): esa leccion es sobre
    pedirle al modelo que PUNTUE una comparacion, y aqui la respuesta sigue siendo una
    etiqueta de un conjunto cerrado. `ParseClassification` **acota `confidence` a
    `[0,1]`** y rechaza lo demas por `ErrLLMQuality`: esta medido en campo que sin ese
    rango el modelo devuelve «100 de confianza» y entonces ninguna respuesta cae jamas por
    debajo de ningun umbral.
  - **El umbral y el saneo de `params` NO viven aqui**: los aplica el caller, que es quien
    tiene el texto original y la config del tenant. Este paquete solo rechaza el relleno
    del esquema en los valores de `params`.
  - `ClassifyRequestInput{Text, Catalog, UnknownLabel, Vocabulary}` con
    `IntentSpec{Name, Description, Params, Examples}` e `IntentExample{Message, Params}`.
    El catalogo viaja **aplanado**, no como `*intents.Config`: el paquete **sigue sin
    depender de ningun otro modulo de wapp-shared** (solo stdlib), que es un invariante
    escrito en `doc.go`. Quien aplana es el caller, que ya tiene la config en la mano.
  - `ParseClassification(raw, in)` recibe la **misma entrada** con la que se armo el
    prompt, y no una lista de nombres suelta: el conjunto aceptable es exactamente el que
    el prompt le ofrecio al modelo, y pasarlo dos veces por separado es la forma de que un
    dia dejen de coincidir. Un catalogo vacio rechaza **todo** artefacto, a proposito.
  - `BuildClassifyRequestPrompt` sigue la forma del clasificador de campo, **ejemplos
    incluidos**: esta medido que en un modelo de 1–2B el few-shot pesa mas que las
    instrucciones, y la via local ejecuta uno de ese tamano. `UnknownLabel` y
    `Vocabulary` son opcionales y su seccion **desaparece** del prompt cuando no vienen.
  - `evidence` es obligatoria **salvo** cuando el intent es la etiqueta de lo desconocido:
    exigirle una frase literal a quien acaba de decir que no entendio solo consigue que el
    caller pague un reintento para volver a oir lo mismo.
  - `api`: `anthropic` implementa el metodo nuevo con el prompt compartido; `gemini` lo
    anade como **stub**, fallando con `ErrNotImplemented` igual que los otros cuatro.

### Fixed

- `README.md` citaba como ejemplo de eco un esquema con `category`, que **ningun prompt
  imprime** desde D-044.23. Corregido al esquema real.
- `doc.go` afirmaba que la via local «no se cablea en el Plan 044» (D-044.4). Lo **deroga
  D-044.29**: la via existe y su adaptador vive en `cloud-platform`, no en este modulo.

## [0.1.0] - 2026-08-22

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
