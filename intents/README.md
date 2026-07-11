# intents

Contrato canonico de configuracion de **intenciones por tenant** del ecosistema wApp
y su validacion estructural. Define los tipos que describen las intenciones que un
clasificador debe reconocer y valida ese contrato antes de aceptarlo. Solo stdlib; no
depende de otros modulos de wapp-shared.

Lo produce el operador (via el API de `cloud-platform`) y lo consumen dos lados:
`cloud-platform` lo valida al recibir el PUT de configuracion, y `wapp-edge-intent`
lo recibe ya validado para alimentar al clasificador. Este modulo define y valida el
contrato; **no** clasifica.

## Instalacion

```bash
go get github.com/EduGoGroup/wapp-shared/intents
```

## Uso

```go
package main

import (
	"errors"
	"fmt"

	"github.com/EduGoGroup/wapp-shared/intents"
)

func main() {
	raw := []byte(`{
      "version": "1",
      "intents": [
        { "name": "saludar", "descripcion": "El usuario saluda",
          "ejemplos": [ { "mensaje": "hola" } ] }
      ]
    }`)

	cfg, err := intents.ParseAndValidate(raw)
	if err != nil {
		if errors.Is(err, intents.ErrInvalidConfig) {
			// contrato invalido: el error identifica el intent/campo ofensor
		}
		return
	}
	fmt.Println(cfg.UmbralConfianza) // 0.6 (normalizado por defecto)
}
```

## Contrato JSON

Las claves mezclan español e inglés de forma **deliberada** (contrato canonico
heredado del prototipo validado); no se renombran.

```json
{
  "version": "1",
  "umbral_confianza": 0.8,
  "vocabulario": ["pizza"],
  "intents": [
    {
      "name": "pedir_comida",
      "descripcion": "El usuario pide un plato",
      "params": ["plato", "cantidad"],
      "ejemplos": [
        { "mensaje": "quiero 2 pizzas", "params": { "plato": "pizza", "cantidad": "2" } }
      ]
    }
  ]
}
```

## Reglas de validacion

- Tamaño `≤ MaxConfigBytes` (256 KiB). JSON **tolerante a campos futuros** (no se
  usa `DisallowUnknownFields`).
- `version` no vacia.
- `umbral_confianza`: ausente o `0` ⇒ `DefaultThreshold` (0.6); si presente, en el
  rango `(0, 1]`.
- `≥ 1` intent. `name` cumple `^[a-z][a-z0-9_]{1,63}$`, unico, y `≠ desconocido`
  (`ReservedUnknown`).
- `descripcion` no vacia. `≥ 1` ejemplo por intent con `mensaje` no vacio.
- `params` declarados: mismo patron que `name`, sin duplicados. Las claves de
  `Ejemplo.Params` deben ser un subconjunto de los `params` del intent.

## API

- `ParseAndValidate([]byte) (*Config, error)` — decodifica y valida; devuelve el
  `Config` normalizado o un error que envuelve `ErrInvalidConfig`.
- `const DefaultThreshold = 0.6`
- `const MaxConfigBytes = 256 * 1024`
- `const ReservedUnknown = "desconocido"`
- `var ErrInvalidConfig` — error centinela de todos los rechazos.
- `type Config` / `type Intent` / `type Ejemplo`.

## Navegacion

- [Changelog](CHANGELOG.md)

## Comandos disponibles

```bash
make build     # Compilar
make test      # Tests
make check     # Lint y validacion
```
