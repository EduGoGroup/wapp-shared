package web

import "testing"

// TestFlashCatalog_NuncaRefleja es la invariante que sube al módulo:
// un código desconocido —por ejemplo, texto arbitrario inyectado en la query
// string por un tercero— jamás se pinta tal cual.
func TestFlashCatalog_NuncaRefleja(t *testing.T) {
	t.Parallel()

	c := NewFlashCatalog("", map[string]string{"revoke_failed": "No se pudo cortar el servicio."})

	crudo := "algo&arbitrario#inyectado<script>"
	if got := c.Message(crudo); got != DefaultFlashFallback {
		t.Fatalf("Message(%q) = %q, want el mensaje genérico", crudo, got)
	}
	if got := c.Message("revoke_failed"); got != "No se pudo cortar el servicio." {
		t.Errorf("un código conocido debe traducirse, got %q", got)
	}
	if got := c.Message(""); got != "" {
		t.Errorf(`Message("") = %q, want "" (no hay flash que pintar)`, got)
	}
	if !c.Known("revoke_failed") || c.Known(crudo) {
		t.Error("Known no distingue lo que está en el catálogo de lo que no")
	}
}

// TestFlashCatalog_ElMapaSeCopia: el catálogo no puede cambiar bajo los pies de
// quien lo consulta.
func TestFlashCatalog_ElMapaSeCopia(t *testing.T) {
	t.Parallel()

	origen := map[string]string{"ok": "Listo."}
	c := NewFlashCatalog("Vaya.", origen)
	origen["ok"] = "PISADO"
	delete(origen, "ok")

	if got := c.Message("ok"); got != "Listo." {
		t.Fatalf("Message = %q: el catálogo debe copiar el mapa", got)
	}
	if got := c.Message("desconocido"); got != "Vaya." {
		t.Errorf("el fallback debe ser configurable, got %q", got)
	}
}
