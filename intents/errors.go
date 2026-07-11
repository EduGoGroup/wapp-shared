package intents

import "errors"

// ErrInvalidConfig indica que la configuración de intenciones es inválida:
// JSON corrupto, tamaño excedido, o cualquier regla estructural incumplida.
// ParseAndValidate lo envuelve con contexto (intent/campo ofensor) mediante
// fmt.Errorf("%w: ...").
var ErrInvalidConfig = errors.New("intents: invalid config")
