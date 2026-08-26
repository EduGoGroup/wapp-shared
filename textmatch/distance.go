package textmatch

// EditDistance calcula la distancia de edición entre dos strings operando por
// RUNAS (para tratar bien el español acentuado y multibyte). Usa la variante
// Damerau-Levenshtein restringida (optimal string alignment, OSA): además de
// inserción, borrado y sustitución, cuenta la TRANSPOSICIÓN de dos runas
// adyacentes como una sola edición.
//
// La transposición es deliberada y es la que sostiene el caso canónico
// «whastapp»≈«whatsapp»: es un intercambio adyacente s↔t. Con Levenshtein puro esa
// distancia es 2 (similitud 0,75 < 0,85) y el fuzzy NO rescataría el typo; con
// transposición la distancia es 1 (similitud 0,875) y se rescata SIN bajar el
// umbral. Las inserciones/borrados/sustituciones simples («instalgram»≈«instagram»)
// no cambian de valor.
//
// Espacio O(min) usando tres filas (la transposición necesita la fila i-2).
func EditDistance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	la, lb := len(ra), len(rb)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	prevPrev := make([]int, lb+1) // fila i-2 (para la transposición)
	prev := make([]int, lb+1)     // fila i-1
	curr := make([]int, lb+1)     // fila i
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			d := min(curr[j-1]+1, prev[j]+1, prev[j-1]+cost)
			// Transposición de dos runas adyacentes (a[i-1]a[i-2] == b[j-2]b[j-1]).
			if i > 1 && j > 1 && ra[i-1] == rb[j-2] && ra[i-2] == rb[j-1] {
				if t := prevPrev[j-2] + 1; t < d {
					d = t
				}
			}
			curr[j] = d
		}
		prevPrev, prev, curr = prev, curr, prevPrev
	}
	return prev[lb]
}
