package web

import "testing"

// TestParseTrustedProxies_VacioEsNil fija la postura por defecto: sin proxies de
// confianza el framework resuelve la IP desde la conexión e IGNORA el
// X-Forwarded-For, que es lo que impide falsear la clave del rate-limit.
func TestParseTrustedProxies_VacioEsNil(t *testing.T) {
	t.Parallel()

	if got := ParseTrustedProxies(""); got != nil {
		t.Errorf(`ParseTrustedProxies("") = %v, want nil`, got)
	}
	if got := ParseTrustedProxies("  ,  , "); got != nil {
		t.Errorf("solo separadores debe dar nil, got %v", got)
	}
	got := ParseTrustedProxies(" 10.0.0.1 ,10.0.0.0/8 ")
	if len(got) != 2 || got[0] != "10.0.0.1" || got[1] != "10.0.0.0/8" {
		t.Errorf("ParseTrustedProxies = %v", got)
	}
}
