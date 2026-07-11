// Package intents define el contrato canónico de configuración de intenciones
// por tenant del ecosistema wApp: los tipos que describen las intenciones que un
// clasificador debe reconocer y la validación estructural de ese contrato.
//
// El contrato lo produce el operador (vía el API de cloud-platform) y lo consumen
// dos lados: cloud-platform lo valida al recibir el PUT de configuración, y el
// Edge de intención (wapp-edge-intent) lo recibe ya validado para alimentar al
// clasificador. Este paquete solo define y valida el contrato; no clasifica.
//
//	cfg, err := intents.ParseAndValidate(raw)
//	if err != nil {
//	    // errors.Is(err, intents.ErrInvalidConfig) == true
//	}
//	_ = cfg.UmbralConfianza // ya normalizado a DefaultThreshold si venía ausente
//
// Las claves JSON del contrato mezclan español e inglés de forma deliberada
// (heredado del prototipo validado); no se renombran. El paquete no depende de
// drivers ni de otros módulos de wapp-shared: solo stdlib.
package intents
