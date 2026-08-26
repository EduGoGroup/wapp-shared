# textmatch

Motor **determinista** de comparacion de textos de wApp: decide si un texto del
cliente corresponde a un item conocido (un producto del catalogo, una variante, un
tag) **sin inventar nada** y **sin depender de un LLM**. Solo stdlib (testify solo
en los tests); no depende de otros modulos de wapp-shared.

Es una **copia-adaptacion** de `edugo-shared/textmatch` (ADR-0004: se copia y se
adapta, NO se importa), renombrada al namespace wApp. La adaptacion quito su unica
dependencia externa (`golang.org/x/text`): aqui el plegado de diacriticos es propio
y de stdlib.

## Instalacion

```bash
go get github.com/EduGoGroup/wapp-shared/textmatch
```

## Los dos niveles

| Nivel | Pieza | Pregunta que responde |
| --- | --- | --- |
| 1 | `Cascade` (`Comparator`) | Este esperado, ¿es este candidato? Exact -> Fuzzy -> zona gris |
| 2 | `SetMatcher` | Un CONJUNTO de candidatos contra un CONJUNTO de esperados, con politica de completitud |

## Uso

```go
cascada := textmatch.NewCascade(textmatch.Exact{}, textmatch.NewFuzzy(0)) // 0 => 0,85
r, err := cascada.Compare(ctx, "whatsapp", "whastapp")
// r.Outcome == textmatch.OutcomeMatch, r.Confidence == 0.875, r.Strategy == "fuzzy"

m := textmatch.NewSetMatcher(cascada, textmatch.PolicyLenient)
rep, err := m.MatchAnswer(ctx, []string{"tequeños", "torta"}, "quiero tequenos y una torta")
// rep.Covered == [true true]
```

## La zona gris (el LLM) se INYECTA

El escalon caro **no vive aqui** y este modulo **no importa** `wapp-shared/llm`
(DIP). Se declara como la interfaz `GrayZone` y lo inyecta el caller:

```go
cascada = cascada.WithGrayZone(miAdaptadorLLM)          // nivel 1 (un par)
m = m.WithGrayZone(miAdaptadorLLM)                      // nivel 2 (una vez por esperado sin cubrir)
```

Reglas que el modulo garantiza (y que sus tests miden con un contador):

- **Sin nada inyectado la cascada funciona igual y es determinista pura**: no hay
  tercer escalon, no panica y no falla.
- **El escalon caro nunca entra en el bucle del `SetMatcher`**: se consulta despues
  del barrido determinista, **como mucho una vez por esperado sin cubrir**, y se le
  ofrecen de golpe todos los candidatos que siguen libres.
- Un `Index` fuera de rango se lee como "ninguno corresponde"; un error se propaga.

## Normalizacion: la ñ se PRESERVA

`Normalize` baja a minusculas, quita tildes y dieresis, colapsa espacios... y deja
la **ñ** intacta: es una letra, no una "n" con tilde. El invariante lo custodia un
test sobre la propia tabla de plegado, no un comentario.

Consecuencia medida (aritmetica del fuzzy, `sim = 1 - dist/maxRunas`):

| Par | Runas | Distancia | Similitud | ¿Casa a 0,85? |
| --- | --- | --- | --- | --- |
| `whatsapp` / `whastapp` | 8 | 1 (transposicion) | 0,875 | si |
| `tequeños` / `tequenos` | 8 | 1 | 0,875 | si |
| `ñoquis` / `noquis` | 6 | 1 | 0,833 | **no** |
| `año` / `ano` | 3 | 1 | 0,667 | no |

Es decir: en palabras cortas la ñ ausente **no** la rescata el umbral por defecto.
Si el caller quiere rescatarla debe bajar su umbral explicitamente
(`NewFuzzy(0.80)`), decidiendo a la vez que acepta mas falsos positivos.

## Distancia

`EditDistance` es Damerau-Levenshtein restringida (OSA) **por runas**: cuenta la
transposicion de dos runas adyacentes como UNA edicion. Es lo que deja
`whastapp` a 0,875 y no a 0,75, y por eso el umbral 0,85 puede seguir siendo
conservador.
