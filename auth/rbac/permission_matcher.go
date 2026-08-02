package rbac

import "strings"

// Grants es el wire format de permisos: una lista de patterns allow y una de patterns deny.
type Grants struct {
	Allow []string `json:"allow"`
	Deny  []string `json:"deny"`
}

// PermissionMatches reporta si `pattern` cubre el permiso `request`, siguiendo la gramática glob de wApp.
func PermissionMatches(pattern, request string) bool {
	if pattern == "*" {
		return true
	}
	if pattern == request {
		return true
	}
	if strings.HasSuffix(pattern, ".*") {
		prefix := pattern[:len(pattern)-2]
		return request == prefix || strings.HasPrefix(request, prefix+".")
	}
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[1:]
		return len(request) > len(suffix) && strings.HasSuffix(request, suffix)
	}
	if i := strings.Index(pattern, ".*."); i > 0 {
		head := pattern[:i+1]
		tail := pattern[i+2:]
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

// EvaluateGrants decide si `request` está permitido por `g` aplicando precedencia deny-sobre-allow.
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
