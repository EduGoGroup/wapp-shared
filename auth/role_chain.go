package auth

// ResolveRoleChain devuelve la cadena de herencia de un rol: el propio rol
// (rootID) seguido de sus ancestros a lo largo de parent_role_id, de más
// cercano a más lejano. Es agnóstica al almacenamiento: `parentOf` resuelve el
// padre de un rol y reporta ok=false (o parent="") cuando el rol es canónico
// (sin padre).
//
// Soporta profundidad N. Incluye guard de ciclos: si un rol ya fue visitado,
// corta la cadena sin error (defensa ante datos corruptos; el schema no impide
// un ciclo A→B→A).
func ResolveRoleChain(rootID string, parentOf func(id string) (parent string, ok bool, err error)) ([]string, error) {
	chain := make([]string, 0, 4)
	visited := make(map[string]struct{}, 4)
	current := rootID
	for current != "" {
		if _, seen := visited[current]; seen {
			break
		}
		visited[current] = struct{}{}
		chain = append(chain, current)

		parent, ok, err := parentOf(current)
		if err != nil {
			return nil, err
		}
		if !ok || parent == "" {
			break
		}
		current = parent
	}
	return chain, nil
}

// MergeGrantChain funde una cadena de Grants (uno por nivel de la jerarquía de
// roles) en un único Grants plano: une todos los allow y todos los deny y
// deduplica preservando la primera aparición. El resultado es invariante al
// orden.
//
// El aplanado es deliberado: la precedencia deny-sobre-allow NO se aplica aquí
// (no se descartan allows que algún deny cubra); eso es responsabilidad del
// matcher (EvaluateGrants), que evalúa por request. Así un deny de cualquier
// nivel gana sobre un allow de otro nivel al evaluar.
func MergeGrantChain(chain []Grants) Grants {
	out := Grants{Allow: []string{}, Deny: []string{}}
	seenAllow := make(map[string]struct{})
	seenDeny := make(map[string]struct{})
	for _, g := range chain {
		for _, p := range g.Allow {
			if _, ok := seenAllow[p]; ok {
				continue
			}
			seenAllow[p] = struct{}{}
			out.Allow = append(out.Allow, p)
		}
		for _, p := range g.Deny {
			if _, ok := seenDeny[p]; ok {
				continue
			}
			seenDeny[p] = struct{}{}
			out.Deny = append(out.Deny, p)
		}
	}
	return out
}
