package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"
)

// ArtifactVersion es la única versión de artefacto que este paquete sabe leer.
// Los artefactos van versionados desde el día uno («{"version":1,...}») para que
// un cambio de forma sea un rechazo explícito y no una lectura silenciosa a
// medias.
const ArtifactVersion = 1

// Classification es el artefacto de la etapa P1.
//
// Lleva la MISMA forma que ya produce el clasificador que corre en el Edge
// (wapp-edge-intent: intent + params + confidence), más la version del artefacto y
// la evidencia que este paquete le exige a toda etapa. Que las dos formas
// coincidan no es casualidad: es lo que permite que la vía local y la vía API
// respondan lo mismo y que el caller no sepa por cuál vino.
type Classification struct {
	// Version es la versión del artefacto.
	Version int `json:"version"`
	// Intent es el nombre de la intención elegida, copiado literal del catálogo
	// —o la etiqueta de lo desconocido, si el caller la declaró—.
	Intent string `json:"intent"`
	// Confidence es la confianza del modelo, acotada a [0,1]. Quien la compara
	// contra el umbral del tenant es el CALLER, nunca este paquete.
	Confidence float64 `json:"confidence"`
	// Params son los valores extraídos, por nombre de parámetro. Este paquete NO
	// los sanea contra el texto original —ese allowlist lo aplica el caller, que
	// es quien tiene el texto—: aquí solo se rechaza el relleno del esquema.
	Params map[string]string `json:"params,omitempty"`
	// Evidence es la frase del texto original que sostiene la intención. Es
	// obligatoria salvo cuando Intent es la etiqueta de lo desconocido: no haber
	// entendido nada no tiene ninguna frase que copiar.
	Evidence string `json:"evidence"`
}

// MainIdeas es el artefacto de la etapa P2 (design.md §7.1).
type MainIdeas struct {
	// Version es la versión del artefacto.
	Version int `json:"version"`
	// Wants son las cosas distintas que el cliente pide, una por entrada.
	Wants []Want `json:"wants"`
	// DeliveryHint es la pista de entrega, si el texto la trae.
	DeliveryHint *Hint `json:"delivery_hint,omitempty"`
}

// Want es una idea suelta de lo que el cliente quiere, con su evidencia.
type Want struct {
	// Idea es la idea en palabras del propio cliente, sin interpretar precios.
	Idea string `json:"idea"`
	// Evidence es la frase literal del texto original de la que sale la idea.
	Evidence string `json:"evidence"`
}

// Hint es una pista con su evidencia (por ahora, la de entrega).
type Hint struct {
	// Text es la pista tal como la expresó el cliente.
	Text string `json:"text"`
	// Evidence es la frase literal del texto original.
	Evidence string `json:"evidence"`
}

// ItemSpecs es el artefacto de la etapa P3 (design.md §7.2).
type ItemSpecs struct {
	// Version es la versión del artefacto.
	Version int `json:"version"`
	// Items son las especificaciones extraídas. P3 se llama una vez por ítem,
	// así que lo normal es que traiga uno.
	Items []ItemSpec `json:"items"`
}

// ItemSpec es la especificación de un ítem tal como P3 la propone.
//
// P3 PROPONE y el match DECIDE (D-044.14): un candidato a añadido que encuentra
// ítem en el catálogo se vuelve línea con precio, y el que no lo encuentra cae a
// personalización de la línea. Por eso los dos campos viajan separados y ninguno
// de los dos es todavía una línea.
type ItemSpec struct {
	// Product es el producto pedido, en las palabras del cliente.
	Product string `json:"product"`
	// Variant es la variante mencionada; los rangos se conservan TEXTUALES en
	// esta etapa («10 o 12 porciones»), quien los parte es P4.
	Variant string `json:"variant,omitempty"`
	// AddonCandidates son añadidos que PODRÍAN ser ítems del catálogo.
	AddonCandidates []string `json:"addon_candidates,omitempty"`
	// Customizations son indicaciones que no son ítems («sin sal»). Nunca se
	// vuelven línea ni entran en ningún total.
	Customizations []string `json:"customizations,omitempty"`
	// Notes es el detalle libre que acompaña al ítem.
	Notes string `json:"notes,omitempty"`
	// Evidence es la frase literal del texto original.
	Evidence string `json:"evidence"`
}

// Quantities es el artefacto de la etapa P4 (design.md §7.3).
type Quantities struct {
	// Version es la versión del artefacto.
	Version int `json:"version"`
	// DeliveryDate es la fecha absoluta de entrega, en formato ISO.
	//
	// El modelo la PROPONE a partir de la fecha de referencia que lleva el
	// prompt; la aritmética que manda es la de Go, determinista, en la etapa P4
	// del cloud (tarea T2.4 del Plan 044). Este campo se decodifica, no se da por
	// bueno.
	DeliveryDate string `json:"delivery_date,omitempty"`
	// DeliveryDateBasis deja escrito desde qué fecha se calculó.
	DeliveryDateBasis string `json:"delivery_date_basis,omitempty"`
	// Items son los ítems ya normalizados.
	Items []NormalizedItem `json:"items"`
}

// NormalizedItem es un ítem con cantidades ya normalizadas.
type NormalizedItem struct {
	// Product es el producto pedido.
	Product string `json:"product"`
	// Qty es la cantidad de unidades. Si el cliente no la dijo, vale 1.
	Qty int `json:"qty"`
	// Range es el rango pedido cuando lo hay («10 o 12 porciones»). Se conserva
	// como rango: NUNCA se colapsa a un número.
	Range *Range `json:"range,omitempty"`
	// UnitKind distingue la unidad suelta del paquete («package»).
	UnitKind string `json:"unit_kind,omitempty"`
	// PackageSize es cuántas unidades trae el paquete. «Un paquete de 30» es
	// Qty 1 con PackageSize 30, jamás Qty 30.
	PackageSize int `json:"package_size,omitempty"`
	// AddonCandidates viaja igual que en P3: propuestas, no líneas.
	AddonCandidates []string `json:"addon_candidates,omitempty"`
	// Customizations viaja igual que en P3.
	Customizations []string `json:"customizations,omitempty"`
	// Notes es el detalle libre que acompaña al ítem.
	Notes string `json:"notes,omitempty"`
	// Evidence es la frase literal del texto original.
	Evidence string `json:"evidence"`
}

// Range es un rango pedido por el cliente, con su unidad.
type Range struct {
	// Min es el extremo bajo del rango.
	Min int `json:"min"`
	// Max es el extremo alto del rango.
	Max int `json:"max"`
	// Unit es la unidad del rango («porciones»).
	Unit string `json:"unit"`
}

// QuoteText es el artefacto de GenerateQuoteText: el mensaje listo para enviar.
//
// Viaja en JSON como todo lo demás para que la salida se valide igual que la de
// cualquier otra tarea; quien comprueba que los precios del texto coinciden con
// los de las líneas es el caller, en Go.
type QuoteText struct {
	// Version es la versión del artefacto.
	Version int `json:"version"`
	// Text es la cotización redactada, lista para enviar tal cual.
	Text string `json:"text"`
}

// PlaceholderEsquema es el relleno que los prompts imprimen en el esquema de la
// respuesta («"evidence": "..."»).
//
// Está exportado porque es parte del contrato de rechazo: un artefacto que trae
// este valor en un campo obligatorio no es una respuesta, es el ESQUEMA repetido,
// y sale por ErrLLMQuality. Lo custodian los tests TestParse*_EcoDelEsquema*.
const PlaceholderEsquema = "..."

// UnitKindPackage es el único valor no vacío que admite NormalizedItem.UnitKind.
// El enum lo fija design.md §7.3 del Plan 044, que solo contempla el paquete
// frente a la unidad suelta (campo omitido).
const UnitKindPackage = "package"

// ParseClassification lee el artefacto de la etapa P1.
//
// Recibe la MISMA entrada con la que se armó el prompt, y no una lista de
// nombres suelta, por una razón concreta: el conjunto de valores aceptables es
// exactamente el que el prompt le ofreció al modelo, y pasarlo dos veces por
// separado es la forma de que un día dejen de coincidir. De ahí salen también las
// dos cosas que el parser necesita y que un []string no lleva: los nombres del
// catálogo y la etiqueta de lo desconocido.
//
// Un catálogo vacío (y sin etiqueta de escape) rechaza TODO artefacto, a
// propósito: significa que el caller no declaró contra qué clasificar, y eso tiene
// que fallar ruidoso. Dejar pasar «lo que sea porque no había con qué comparar»
// sería abrirle la puerta al eco del esquema, que es JSON perfectamente válido.
func ParseClassification(raw json.RawMessage, in ClassifyRequestInput) (*Classification, error) {
	var out Classification
	if err := decodeArtifact(raw, &out); err != nil {
		return nil, err
	}
	if err := checkVersion(out.Version); err != nil {
		return nil, err
	}
	if err := validarEnum("intent", out.Intent, intentsPermitidos(in)); err != nil {
		return nil, err
	}
	if err := validarConfianza(out.Confidence); err != nil {
		return nil, err
	}
	if err := validarParams(out.Params); err != nil {
		return nil, err
	}
	// La evidencia es obligatoria salvo en la respuesta «no entendí»: exigirle una
	// frase literal a quien acaba de decir que no encaja nada solo consigue que el
	// caller pague un reintento para volver a oír lo mismo.
	if in.UnknownLabel != "" && out.Intent == in.UnknownLabel {
		if err := validarOpcional("evidence", out.Evidence); err != nil {
			return nil, err
		}
		return &out, nil
	}
	if err := validarObligatorio("evidence", out.Evidence); err != nil {
		return nil, err
	}
	return &out, nil
}

// intentsPermitidos es el conjunto cerrado que puede salir de P1: los nombres del
// catálogo más la etiqueta de lo desconocido, que se acepta SIN estar declarada
// como intención (en el contrato de wApp está prohibido declararla).
func intentsPermitidos(in ClassifyRequestInput) []string {
	permitidos := make([]string, 0, len(in.Catalog)+1)
	for _, it := range in.Catalog {
		permitidos = append(permitidos, it.Name)
	}
	if in.UnknownLabel != "" {
		permitidos = append(permitidos, in.UnknownLabel)
	}
	return permitidos
}

// validarConfianza exige que la confianza esté dentro de [0,1].
//
// No es cosmética. El umbral del tenant solo significa algo si el número está
// acotado, y está MEDIDO en campo (wapp-edge-intent, classifier.go) que sin acotar
// el rango el modelo devuelve «100 de confianza» y entonces ninguna respuesta cae
// jamás por debajo de ningún umbral: pasa todo, incluido lo que el modelo no
// entendió. Allí quien acota es la gramática que Ollama fuerza; aquí, donde el
// proveedor responde texto libre y no hay gramática ninguna, el único sitio donde
// se puede acotar es éste.
func validarConfianza(c float64) error {
	if c < 0 || c > 1 {
		return fmt.Errorf("%w: confidence vale %v, fuera del rango [0,1]: contra un número sin acotar el umbral del tenant no significa nada",
			ErrLLMQuality, c)
	}
	return nil
}

// validarParams rechaza los parámetros que traen el relleno del esquema. Un valor
// VACÍO sí pasa: significa «el cliente no lo dijo», y quien decide si un parámetro
// vacío sirve es el caller, que además lo contrasta contra el texto original.
//
// Las claves se recorren ordenadas para que el error nombre siempre el mismo
// parámetro: el orden de un mapa en Go es aleatorio, y un mensaje que cambia entre
// corridas es un mensaje que nadie puede reproducir.
func validarParams(params map[string]string) error {
	for _, k := range slices.Sorted(maps.Keys(params)) {
		if err := validarOpcional(fmt.Sprintf("params[%q]", k), params[k]); err != nil {
			return err
		}
	}
	return nil
}

// ParseMainIdeas lee el artefacto de la etapa P2.
//
// Una lista de wants VACÍA es válida: «cero resultados válidos tampoco es fatal»
// (design.md §3.2). Lo que no es válido es un want con la idea o la evidencia en
// blanco o con el relleno del esquema.
func ParseMainIdeas(raw json.RawMessage) (*MainIdeas, error) {
	var out MainIdeas
	if err := decodeArtifact(raw, &out); err != nil {
		return nil, err
	}
	if err := checkVersion(out.Version); err != nil {
		return nil, err
	}
	if err := validarWants(out.Wants); err != nil {
		return nil, err
	}
	if err := validarHint(out.DeliveryHint); err != nil {
		return nil, err
	}
	return &out, nil
}

// ParseItemSpecs lee el artefacto de la etapa P3.
func ParseItemSpecs(raw json.RawMessage) (*ItemSpecs, error) {
	var out ItemSpecs
	if err := decodeArtifact(raw, &out); err != nil {
		return nil, err
	}
	if err := checkVersion(out.Version); err != nil {
		return nil, err
	}
	for i := range out.Items {
		if err := validarItemSpec(i, &out.Items[i]); err != nil {
			return nil, err
		}
	}
	return &out, nil
}

// ParseQuantities lee el artefacto de la etapa P4.
func ParseQuantities(raw json.RawMessage) (*Quantities, error) {
	var out Quantities
	if err := decodeArtifact(raw, &out); err != nil {
		return nil, err
	}
	if err := checkVersion(out.Version); err != nil {
		return nil, err
	}
	if err := validarFechaEntrega(out.DeliveryDate); err != nil {
		return nil, err
	}
	if err := validarOpcional("delivery_date_basis", out.DeliveryDateBasis); err != nil {
		return nil, err
	}
	for i := range out.Items {
		if err := validarNormalizedItem(i, &out.Items[i]); err != nil {
			return nil, err
		}
	}
	return &out, nil
}

// ParseQuoteText lee el artefacto del generador de cotización.
func ParseQuoteText(raw json.RawMessage) (*QuoteText, error) {
	var out QuoteText
	if err := decodeArtifact(raw, &out); err != nil {
		return nil, err
	}
	if err := checkVersion(out.Version); err != nil {
		return nil, err
	}
	if err := validarObligatorio("text", out.Text); err != nil {
		return nil, err
	}
	return &out, nil
}

// validarWants comprueba los campos obligatorios de cada idea de P2.
func validarWants(wants []Want) error {
	for i := range wants {
		if err := validarObligatorio(campoDe("wants", i, "idea"), wants[i].Idea); err != nil {
			return err
		}
		if err := validarObligatorio(campoDe("wants", i, "evidence"), wants[i].Evidence); err != nil {
			return err
		}
	}
	return nil
}

// validarHint comprueba la pista de entrega. Es opcional; si viene, sus dos
// campos son obligatorios.
func validarHint(h *Hint) error {
	if h == nil {
		return nil
	}
	if err := validarObligatorio("delivery_hint.text", h.Text); err != nil {
		return err
	}
	return validarObligatorio("delivery_hint.evidence", h.Evidence)
}

// validarItemSpec comprueba un ítem de P3.
//
// Obligatorios product y evidence (design.md §7.2); el resto son opcionales, pero
// si vienen no pueden traer el relleno del esquema: un «variant» que vale «...»
// es el esquema repetido, no una variante.
func validarItemSpec(i int, it *ItemSpec) error {
	if err := validarObligatorio(campoDe("items", i, "product"), it.Product); err != nil {
		return err
	}
	if err := validarObligatorio(campoDe("items", i, "evidence"), it.Evidence); err != nil {
		return err
	}
	if err := validarOpcional(campoDe("items", i, "variant"), it.Variant); err != nil {
		return err
	}
	if err := validarOpcional(campoDe("items", i, "notes"), it.Notes); err != nil {
		return err
	}
	if err := validarLista(campoDe("items", i, "addon_candidates"), it.AddonCandidates); err != nil {
		return err
	}
	return validarLista(campoDe("items", i, "customizations"), it.Customizations)
}

// validarNormalizedItem comprueba un ítem ya normalizado de P4.
//
// Además de los textos, comprueba las reglas de cantidad que fija design.md §7.3:
// la cantidad omitida vale 1 (nunca 0), y «un paquete de 30» es un paquete con
// package_size 30. Un package_size en cero con unit_kind «package» es el esquema
// repetido, no un paquete.
func validarNormalizedItem(i int, it *NormalizedItem) error {
	if err := validarObligatorio(campoDe("items", i, "product"), it.Product); err != nil {
		return err
	}
	if err := validarObligatorio(campoDe("items", i, "evidence"), it.Evidence); err != nil {
		return err
	}
	if err := validarOpcional(campoDe("items", i, "notes"), it.Notes); err != nil {
		return err
	}
	if it.Qty < 1 {
		return fmt.Errorf("%w: %s vale %d; la cantidad omitida es 1, nunca 0", ErrLLMQuality, campoDe("items", i, "qty"), it.Qty)
	}
	if err := validarPaquete(i, it); err != nil {
		return err
	}
	if err := validarRango(i, it.Range); err != nil {
		return err
	}
	if err := validarLista(campoDe("items", i, "addon_candidates"), it.AddonCandidates); err != nil {
		return err
	}
	return validarLista(campoDe("items", i, "customizations"), it.Customizations)
}

// validarPaquete comprueba el enum de unit_kind y el tamaño del paquete.
func validarPaquete(i int, it *NormalizedItem) error {
	if it.UnitKind == "" {
		return nil
	}
	if it.UnitKind != UnitKindPackage {
		return fmt.Errorf("%w: %s vale %q; el único valor previsto es %q",
			ErrLLMQuality, campoDe("items", i, "unit_kind"), it.UnitKind, UnitKindPackage)
	}
	if it.PackageSize < 1 {
		return fmt.Errorf("%w: %s vale %d con unit_kind %q; un paquete trae al menos una unidad",
			ErrLLMQuality, campoDe("items", i, "package_size"), it.PackageSize, UnitKindPackage)
	}
	return nil
}

// validarRango comprueba el rango pedido. Es opcional; si viene, su unidad es
// obligatoria y los extremos tienen que estar en orden.
func validarRango(i int, r *Range) error {
	if r == nil {
		return nil
	}
	if err := validarObligatorio(campoDe("items", i, "range.unit"), r.Unit); err != nil {
		return err
	}
	if r.Min < 1 || r.Max < r.Min {
		return fmt.Errorf("%w: %s trae el rango [%d, %d], que no es un rango",
			ErrLLMQuality, campoDe("items", i, "range"), r.Min, r.Max)
	}
	return nil
}

// validarFechaEntrega comprueba delivery_date. Es opcional —el cliente puede no
// haber dicho cuándo—, pero si viene tiene que ser una fecha AAAA-MM-DD de
// verdad: el esquema del prompt imprime ahí esa plantilla literal, y una
// plantilla no es una fecha.
func validarFechaEntrega(fecha string) error {
	if fecha == "" {
		return nil
	}
	if _, err := time.Parse(time.DateOnly, fecha); err != nil {
		return fmt.Errorf("%w: delivery_date vale %q y no es una fecha AAAA-MM-DD", ErrLLMQuality, fecha)
	}
	return nil
}

// validarEnum exige que el valor sea uno de los permitidos, copiado literal.
//
// Un conjunto de permitidos vacío rechaza cualquier valor a propósito: significa
// que el caller no declaró el enum, y adivinarlo aquí sería dejar pasar el eco
// del esquema con la excusa de que «no había con qué comparar».
func validarEnum(campo, valor string, permitidos []string) error {
	for _, p := range permitidos {
		if valor == p {
			return nil
		}
	}
	return fmt.Errorf("%w: %s vale %q, que no es ninguno de los %d valores permitidos",
		ErrLLMQuality, campo, valor, len(permitidos))
}

// validarObligatorio exige que el campo traiga texto de verdad: ni vacío, ni solo
// espacios, ni el relleno del esquema.
func validarObligatorio(campo, valor string) error {
	limpio := strings.TrimSpace(valor)
	if limpio == "" {
		return fmt.Errorf("%w: el campo obligatorio %s vino vacío", ErrLLMQuality, campo)
	}
	if limpio == PlaceholderEsquema {
		return fmt.Errorf("%w: el campo %s trae %q, el relleno del esquema del prompt: el modelo repitió el esquema en vez de responder",
			ErrLLMQuality, campo, PlaceholderEsquema)
	}
	return nil
}

// validarOpcional deja pasar el campo ausente, pero no el que trae el relleno del
// esquema: omitirlo es una respuesta, copiarlo no lo es.
func validarOpcional(campo, valor string) error {
	if strings.TrimSpace(valor) == "" {
		return nil
	}
	return validarObligatorio(campo, valor)
}

// validarLista aplica validarObligatorio a cada entrada de una lista de cadenas.
// La lista puede estar vacía; lo que no puede es traer entradas de relleno.
func validarLista(campo string, valores []string) error {
	for i, v := range valores {
		if err := validarObligatorio(fmt.Sprintf("%s[%d]", campo, i), v); err != nil {
			return err
		}
	}
	return nil
}

// campoDe nombra un campo dentro de una lista para que el error diga cuál falló.
func campoDe(lista string, i int, campo string) string {
	return fmt.Sprintf("%s[%d].%s", lista, i, campo)
}

// decodeArtifact decodifica un artefacto. Es tolerante a campos futuros (no usa
// DisallowUnknownFields), igual que el módulo intents: un proveedor que añada
// una clave no rompe al lector.
func decodeArtifact(raw json.RawMessage, dst any) error {
	if len(bytes.TrimSpace(raw)) == 0 {
		return fmt.Errorf("%w: artefacto vacío", ErrLLMQuality)
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return fmt.Errorf("%w: artefacto no decodificable: %w", ErrLLMQuality, err)
	}
	return nil
}

// checkVersion rechaza los artefactos de una versión que este paquete no sabe
// leer. Es un fallo de calidad, no de infraestructura: el proveedor respondió
// algo con la forma equivocada.
func checkVersion(got int) error {
	if got != ArtifactVersion {
		return fmt.Errorf("%w: versión de artefacto %d, se esperaba %d", ErrLLMQuality, got, ArtifactVersion)
	}
	return nil
}
