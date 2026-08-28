package web

import (
	"testing"
	"time"
)

func TestSesion_IdaYVuelta(t *testing.T) {
	t.Parallel()

	quiero := SessionData{AccessToken: "a.b.c", RefreshToken: "rt", ExpiresAt: "2026-08-28T10:00:00Z"}
	valor, err := EncodeSession(quiero)
	if err != nil {
		t.Fatalf("EncodeSession: %v", err)
	}
	got, err := DecodeSession(valor)
	if err != nil {
		t.Fatalf("DecodeSession: %v", err)
	}
	if got != quiero {
		t.Fatalf("ida y vuelta cambió la sesión: %+v", got)
	}
	if _, err := DecodeSession("esto no es base64 válido !!"); err == nil {
		t.Error("un valor corrupto debe devolver error, no una sesión vacía silenciosa")
	}
}

func TestSessionValid(t *testing.T) {
	t.Parallel()

	futuro := time.Now().Add(time.Hour)
	pasado := time.Now().Add(-time.Hour)

	if !SessionValid(&futuro) {
		t.Error("un exp en el futuro es válido")
	}
	if SessionValid(&pasado) {
		t.Error("un exp pasado no es válido")
	}
	if SessionValid(nil) {
		t.Error("sin exp la sesión NO es válida (fail-closed)")
	}
}

func TestRefreshDue(t *testing.T) {
	t.Parallel()

	lejos := time.Now().Add(time.Hour)
	cerca := time.Now().Add(30 * time.Second)

	if RefreshDue(&lejos, 0) {
		t.Error("con una hora por delante no toca refrescar")
	}
	if !RefreshDue(&cerca, 0) {
		t.Error("dentro del margen por defecto (2 min) sí toca refrescar")
	}
	if !RefreshDue(nil, 0) {
		t.Error("sin exp hay que refrescar siempre: no se sabe cuánto queda")
	}
	if RefreshDue(&cerca, time.Second) {
		t.Error("el margen es configurable: con 1 s no debía tocar")
	}
}
