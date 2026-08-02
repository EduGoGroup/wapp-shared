package rbac

// ResolveRoleChain devuelve la cadena de herencia de un rol.
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

// MergeGrantChain funde una cadena de Grants en un único Grants plano.
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
