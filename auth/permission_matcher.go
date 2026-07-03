package auth

import "strings"

// PermissionMatches reporta si `pattern` cubre el permiso `request`, siguiendo
// la gramática glob de wApp (`recurso.accion`):
//
//	`*`               → cubre cualquier request
//	`pattern==request`→ coincidencia exacta
//	`prefix.*`        → cubre `prefix` y `prefix.<lo-que-sea>` (subárbol)
//	`*.suffix`        → cubre cualquier request cuyo último tramo sea `.suffix`
//	                    (ej. `*.read` matchea `contacts.read` y `flows.node.read`)
//	`prefix.*.suffix` → request que empieza con `prefix.`, tiene al menos un
//	                    segmento intermedio y termina con `.suffix`
func PermissionMatches(pattern, request string) bool {
	if pattern == "*" {
		return true
	}
	if pattern == request {
		return true
	}
	// prefix.*  (subárbol). Se evalúa antes que prefix.*.suffix para preservar
	// la semántica cuando el pattern termina literalmente con `.*`.
	if strings.HasSuffix(pattern, ".*") {
		prefix := pattern[:len(pattern)-2]
		return request == prefix || strings.HasPrefix(request, prefix+".")
	}
	// *.suffix  → cualquier request `<algo>.suffix`
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[1:] // conserva el punto, ej. ".read"
		return len(request) > len(suffix) && strings.HasSuffix(request, suffix)
	}
	// prefix.*.suffix  → request startsWith `prefix.` + algo + `.suffix`
	if i := strings.Index(pattern, ".*."); i > 0 {
		head := pattern[:i+1] // `prefix.`
		tail := pattern[i+2:] // `.suffix`
		if strings.Contains(head, "*") || strings.Contains(tail, "*") {
			return false
		}
		if !strings.HasPrefix(request, head) || !strings.HasSuffix(request, tail) {
			return false
		}
		if len(request) <= len(head)+len(tail) {
			return false
		}
		middle := request[len(head) : len(request)-len(tail)]
		return !strings.HasPrefix(middle, ".") && !strings.HasSuffix(middle, ".")
	}
	return false
}

// EvaluateGrants decide si `request` está permitido por `g` aplicando
// precedencia deny-sobre-allow con default DENY: si algún deny matchea → false;
// si no, true sólo si algún allow matchea; sin allow → false.
func EvaluateGrants(g Grants, request string) bool {
	for _, d := range g.Deny {
		if PermissionMatches(d, request) {
			return false
		}
	}
	for _, a := range g.Allow {
		if PermissionMatches(a, request) {
			return true
		}
	}
	return false
}
