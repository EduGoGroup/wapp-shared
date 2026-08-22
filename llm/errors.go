package llm

import "errors"

// ErrLLMQuality indica que el proveedor respondió, pero su salida no es
// interpretable: no trae un objeto JSON completo, el JSON está truncado, o el
// artefacto no cumple la forma versionada que este paquete sabe leer.
//
// Es deliberadamente DISTINTO de un error de infraestructura (timeout, 5xx,
// credencial inválida), porque el tratamiento es distinto: la calidad se
// reintenta UNA vez con TemperatureRetry y, si persiste, se aísla la unidad
// envenenada y el resto del trabajo continúa; la infraestructura es transitoria
// y la reintenta el job entero más tarde. Los providers NO reintentan por su
// cuenta: el retry vive en el caller (REQ-02, REQ-03).
//
// Lo custodian TestExtractJSON_TruncadoEsErrorDeCalidad (paquete llm) y
// TestAnthropic_SalidaNoJSONEsErrorDeCalidad (paquete llm/api).
var ErrLLMQuality = errors.New("llm: model output failed quality check")
