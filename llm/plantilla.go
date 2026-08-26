package llm

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ============================================================================
// PLANTILLAS DE PROMPT — la costura por la que se ajusta el texto sin compilar
// ============================================================================
//
// 🧭 SI VIENES A CAMBIAR EL TEXTO DE UN PROMPT, EMPIEZA POR AQUÍ.
//
// Cada etapa de análisis compone su prompt así, y el ORDEN no es negociable:
//
//	promptHeader + Instruccion + jsonOnlyRules + Esquema + <los datos>
//
// De esas cinco piezas solo DOS se ajustan —`Instruccion` y `Esquema`—, y son
// justo las dos que este fichero saca a un tipo con nombre. Las otras tres no:
// `promptHeader` y `jsonOnlyRules` son contrato común de todas las etapas, y los
// datos los serializa el código.
//
// 🔴 POR QUÉ EL ORDEN ES INTOCABLE: lo estable va delante porque el proveedor
// cachea el PREFIJO del prompt (I6, ADR-0046). Mover una pieza variable hacia
// arriba invalida la caché en cada llamada y multiplica el prefill — que es el
// coste dominante de estas etapas. Por eso la composición vive en los
// `Build*PromptCon` de este paquete y NO en quien inyecta la plantilla.
//
// 🔴 ESTE PAQUETE NO LEE FICHEROS, y es deliberado: una librería compartida que
// abre rutas se vuelve imposible de testear y arrastra el sistema de ficheros a
// todos sus consumidores. Aquí solo entra TEXTO ya leído. Quien lo lee del disco
// es el cloud (`internal/prompts`), que es el único que tiene un directorio y un
// ciclo de arranque donde fallar ruidosamente.
//
// QUÉ ETAPAS SE AJUSTAN POR AQUÍ, y por qué P1 no está:
//
//   - P2, P3, P4, P5 → sí. Su texto es literal puro y hoy no hay ninguna otra
//     forma de tocarlo que no sea una release del módulo.
//   - P1 (clasificar la petición) → NO, y no es un olvido: su prompt lo gobierna
//     el CATÁLOGO DE INTENCIONES del tenant, que ya se edita por API
//     (`PUT /api/v1/intents`) y viaja al Edge por ConfigUpdate. P1 ya tenía su vía
//     de ajuste sin release; las otras cuatro no tenían ninguna. Meter P1 aquí
//     sería darle una SEGUNDA fuente de verdad al mismo prompt.

// Etapa nombra una etapa del pipeline cuyo prompt se puede ajustar.
//
// El valor es el identificador CORTO y en minúsculas («p2», «p4»), y es contrato
// hacia fuera: es lo que el cloud usa para emparejar un fichero con su etapa. No
// lo cambies sin cambiar la convención de nombres de los ficheros.
type Etapa string

// Las cuatro etapas ajustables. Ver el bloque de arriba para por qué P1 no está.
const (
	// EtapaP2 saca las ideas principales del mensaje del cliente.
	EtapaP2 Etapa = "p2"
	// EtapaP3 especifica UN ítem (producto, variante, añadidos).
	EtapaP3 Etapa = "p3"
	// EtapaP4 normaliza cantidades, paquetes, rangos y la fecha de entrega.
	EtapaP4 Etapa = "p4"
	// EtapaP5 redacta el mensaje de cotización que se le manda al cliente.
	EtapaP5 Etapa = "p5"
)

// EtapasAjustables es el orden canónico en que se enumeran las etapas. Va como
// slice y no como map para que los listados, los logs y los ficheros de ejemplo
// salgan SIEMPRE en el mismo orden: un diff de configuración que baila según el
// recorrido de un map es un diff que nadie revisa.
var EtapasAjustables = []Etapa{EtapaP2, EtapaP3, EtapaP4, EtapaP5}

// Plantilla es el texto ajustable de una etapa: las dos piezas del prompt que no
// son ni contrato común ni datos.
type Plantilla struct {
	// Instruccion es lo que se le manda hacer al modelo. Va DESPUÉS de la
	// cabecera común y ANTES de las reglas de salida.
	Instruccion string
	// Esquema es la forma de la respuesta, con su ejemplo. Va al final del bloque
	// estable, justo antes de los datos.
	//
	// 🔴 EN EL ESQUEMA NO PUEDE HABER UN VALOR QUE SU PROPIO VALIDADOR RECHACE.
	// El modelo COPIA el ejemplo, así que un `"package_size": 0` impreso aquí es
	// un `"package_size": 0` en la respuesta — y `validarPaquete` lo rechaza. Pasó:
	// P4 fue 0 de 14 en su primer día en campo por eso. ValidarPlantilla existe
	// para que esa clase de error no vuelva a salir de aquí sin avisar.
	Esquema string
}

// ErrPlantillaInvalida es el centinela de una plantilla que no puede servirse.
// Se inspecciona con errors.Is.
var ErrPlantillaInvalida = errors.New("llm: plantilla de prompt inválida")

// PlantillaPorDefecto devuelve la plantilla compilada de una etapa, y si la etapa
// existe. Es la que se usa cuando nadie inyecta otra, y la que hay que volcar a
// disco para que un operador arranque desde el texto REAL y no desde una copia
// que ya se quedó vieja.
func PlantillaPorDefecto(e Etapa) (Plantilla, bool) {
	switch e {
	case EtapaP2:
		return plantillaP2(), true
	case EtapaP3:
		return plantillaP3(), true
	case EtapaP4:
		return plantillaP4(), true
	case EtapaP5:
		return plantillaP5(), true
	default:
		return Plantilla{}, false
	}
}

// textoDeRellenoValidacion y fechaDeRellenoValidacion son los valores con los que
// ValidarPlantilla rellena los DOS huecos reconocibles del esquema. No son
// arbitrarios: tienen que pasar los validadores de contenido (no vacíos, no
// placeholder), porque lo que se está midiendo es el RESTO del esquema.
const (
	textoDeRellenoValidacion = "un texto cualquiera"
	fechaDeRellenoValidacion = "2026-07-13"
	// formatoDeFechaEsquema es el hueco de fecha que imprimen las plantillas. Es
	// un relleno RECONOCIBLE, igual que PlaceholderEsquema: si el modelo lo ecoa,
	// el validador lo caza por su nombre en vez de tragárselo.
	formatoDeFechaEsquema = "AAAA-MM-DD"
)

// ValidarPlantilla exige de una plantilla el invariante que costó dos releases
// aprender: EL EJEMPLO QUE LA PLANTILLA LE ENSEÑA AL MODELO TIENE QUE SER UNA
// RESPUESTA QUE EL VALIDADOR DE ESA MISMA ETAPA ACEPTE.
//
// Comprueba dos cosas, y las dos importan:
//
//  1. El esquema, con sus huecos RECONOCIBLES rellenos, lo ACEPTA el `Parse*` de
//     la etapa. Si no, la plantilla le está pidiendo al modelo que responda algo
//     que el sistema va a rechazar — y el modelo obedece: copia el ejemplo.
//  2. El esquema CRUDO lo sigue RECHAZANDO. Esa es la red que caza al modelo que
//     ecoa el prompt entero en vez de responder. Una plantilla que pasa cruda ha
//     perdido esa red.
//
// 🔴 LOS NÚMEROS NO SE RELLENAN, y ahí está el punto. Un `"..."` es un hueco que
// se puede detectar si el modelo lo copia; un `0` es indistinguible de un valor
// real. Por eso todo número impreso en el esquema tiene que ser válido TAL CUAL,
// y esta función lo mide sin tocarlo.
func ValidarPlantilla(e Etapa, p Plantilla) error {
	parse, ok := validadorDeEtapa(e)
	if !ok {
		return fmt.Errorf("%w: etapa %q desconocida", ErrPlantillaInvalida, e)
	}
	if strings.TrimSpace(p.Instruccion) == "" {
		return fmt.Errorf("%w: %s no tiene instrucción", ErrPlantillaInvalida, e)
	}

	crudo, err := ExtractJSON(p.Esquema)
	if err != nil {
		return fmt.Errorf("%w: el esquema de %s no es ni siquiera JSON extraíble: %w",
			ErrPlantillaInvalida, e, err)
	}

	relleno := strings.ReplaceAll(string(crudo), PlaceholderEsquema, textoDeRellenoValidacion)
	relleno = strings.ReplaceAll(relleno, formatoDeFechaEsquema, fechaDeRellenoValidacion)

	if err := parse(json.RawMessage(relleno)); err != nil {
		return fmt.Errorf("%w: el ejemplo que %s le enseña al modelo lo rechaza su propio "+
			"validador, así que el modelo que lo copie fallará: %w", ErrPlantillaInvalida, e, err)
	}

	if err := parse(crudo); err == nil {
		return fmt.Errorf("%w: el esquema CRUDO de %s pasa el validador, así que un modelo que "+
			"ecoe el prompt en vez de responder no se detectaría", ErrPlantillaInvalida, e)
	}
	return nil
}

// validadorDeEtapa empareja cada etapa con el Parse* que lee su salida. Es la
// tabla que hace de ValidarPlantilla una REGLA sobre las cuatro etapas y no
// cuatro comprobaciones sueltas: una etapa nueva sin entrada aquí falla por
// «desconocida» en vez de quedarse sin validar en silencio.
func validadorDeEtapa(e Etapa) (func(json.RawMessage) error, bool) {
	switch e {
	case EtapaP2:
		return func(raw json.RawMessage) error { _, err := ParseMainIdeas(raw); return err }, true
	case EtapaP3:
		return func(raw json.RawMessage) error { _, err := ParseItemSpecs(raw); return err }, true
	case EtapaP4:
		return func(raw json.RawMessage) error { _, err := ParseQuantities(raw); return err }, true
	case EtapaP5:
		return func(raw json.RawMessage) error { _, err := ParseQuoteText(raw); return err }, true
	default:
		return nil, false
	}
}
