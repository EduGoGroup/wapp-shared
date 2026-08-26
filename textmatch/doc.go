// Package textmatch es el motor determinista de comparación de textos de wApp:
// decide si un texto del cliente corresponde a un ítem conocido (un producto del
// catálogo, una variante, un tag) sin inventar nada y sin depender de un LLM.
//
// Es una copia-adaptación de `edugo-shared/textmatch` (ADR-0004: se copia y se
// adapta, NO se importa), renombrada al namespace wApp. La adaptación quitó su
// única dependencia externa (`golang.org/x/text`): aquí el plegado de diacríticos
// es propio y de stdlib, ver normalize.go.
//
// Se organiza en dos niveles ortogonales:
//
//   - Nivel 1 (Comparator/Cascade): ¿este esperado ≈ este candidato? Una cascada
//     de estrategias baratas→caras: Exact → Fuzzy → zona gris. Positivo corta;
//     incierto/negativo escala; un error se propaga.
//   - Nivel 2 (SetMatcher): matchea un CONJUNTO de candidatos contra un CONJUNTO
//     de esperados, con una política de completitud (Strict/Lenient) que es
//     decisión de negocio, ortogonal a cómo se compara un par.
//
// El escalón caro —el LLM— NO vive aquí y este módulo NO importa
// `wapp-shared/llm` (DIP, Plan 044 · T3.1): se define como la interfaz GrayZone y
// se INYECTA. Sin inyectar nada, el motor sigue funcionando y es determinista
// puro. En el SetMatcher la zona gris se consulta FUERA del bucle: como mucho una
// vez por esperado que quedó sin cubrir, nunca por celda.
//
// El paquete no decide negocio: qué significa un no-match (ítem `unmatched`, sin
// precio) lo interpreta el caller. Todo es puro y sin estado salvo la
// construcción explícita de Cascade/SetMatcher.
package textmatch
