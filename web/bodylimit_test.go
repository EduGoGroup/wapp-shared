package web

import (
	"net/http"
	"testing"
)

// TestBodyLimit_SoloRutasDeclaradasYMetodosQueMutan: el tope se aplica por lista
// a propósito, para no cambiarle el comportamiento a una pantalla ajena.
func TestBodyLimit_SoloRutasDeclaradasYMetodosQueMutan(t *testing.T) {
	t.Parallel()

	b := NewBodyLimit(1024, "/catalog/import")

	if !b.Guards(http.MethodPost, "/catalog/import") {
		t.Error("la ruta declarada con POST sí se acota")
	}
	if b.Guards(http.MethodGet, "/catalog/import") {
		t.Error("un GET no sube nada: no se acota")
	}
	if b.Guards(http.MethodPost, "/flows") {
		t.Error("una ruta NO declarada no se acota (publica definiciones grandes a propósito)")
	}
	if got := b.Limit(); got != 1024 {
		t.Errorf("Limit = %d", got)
	}
}
