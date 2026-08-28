package web

import (
	"net/http"
	"testing"
)

// TestCORS_ElWildcardSeDescarta: un "*" que se cuele por configuración no puede
// convertirse en una allowlist que lo permita todo, ni por la vía del parser ni
// por la de la política.
func TestCORS_ElWildcardSeDescarta(t *testing.T) {
	t.Parallel()

	if got := ParseOrigins("*"); len(got) != 0 {
		t.Errorf(`ParseOrigins("*") = %v, want vacío`, got)
	}
	if got := ParseOrigins(" https://a.example , , * ,https://b.example"); len(got) != 2 {
		t.Errorf("ParseOrigins descartó mal: %v", got)
	}

	p := NewCORSPolicy(CORSOptions{AllowedOrigins: []string{"*", "", "  "}})
	if p.Allowed("*") {
		t.Error(`la política no puede admitir "*"`)
	}
	h := http.Header{}
	if p.Apply(h, "https://cualquiera.example") {
		t.Error("con allowlist vacía no debe aplicarse CORS a nadie")
	}
	if got := h.Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("no debe emitirse ACAO, got %q", got)
	}
}

func TestCORS_EcoExactoYCabeceras(t *testing.T) {
	t.Parallel()

	p := NewCORSPolicy(CORSOptions{AllowedOrigins: []string{"https://ok.example"}})

	h := http.Header{}
	if !p.Apply(h, "https://ok.example") {
		t.Fatal("el origen de la allowlist debía aplicarse")
	}
	if got := h.Get("Access-Control-Allow-Origin"); got != "https://ok.example" {
		t.Errorf("ACAO = %q, want el eco exacto", got)
	}
	if got := h.Get("Access-Control-Max-Age"); got != DefaultCORSMaxAge {
		t.Errorf("falta el Max-Age del preflight, got %q", got)
	}
	if got := h.Get("Access-Control-Allow-Headers"); got != DefaultCORSHeaders {
		t.Errorf("Allow-Headers = %q, want %q", got, DefaultCORSHeaders)
	}
	if got := h.Get("Vary"); got != "Origin" {
		t.Errorf("Vary = %q, want Origin", got)
	}

	otro := http.Header{}
	if p.Apply(otro, "https://evil.example") {
		t.Error("un origen fuera de la allowlist no debe aplicarse")
	}
	if got := otro.Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("jamás se hace eco de un origen no listado, got %q", got)
	}
	// El Vary va igual: la respuesta depende del Origin también cuando se niega.
	if got := otro.Get("Vary"); got != "Origin" {
		t.Errorf("Vary = %q, want Origin también en el caso negado", got)
	}
}

func TestAppendVary_NoPisaLoQueYaHabia(t *testing.T) {
	t.Parallel()

	h := http.Header{}
	h.Set("Vary", "Accept-Encoding")
	AppendVary(h, "Origin")
	if got := h.Get("Vary"); got != "Accept-Encoding, Origin" {
		t.Fatalf("Vary = %q", got)
	}
	AppendVary(h, "Origin") // idempotente
	if got := h.Get("Vary"); got != "Accept-Encoding, Origin" {
		t.Fatalf("Vary duplicó el valor: %q", got)
	}
}
