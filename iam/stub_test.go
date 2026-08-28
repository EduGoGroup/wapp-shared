package iam

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// stub es un emisor fake que responde un status+cuerpo fijos y captura lo que le llegó.
//
// Se afirma sobre lo CAPTURADO —ruta, cuerpo y cabeceras— porque el contrato es lo que viaja por el
// cable, no lo que el código pretendía mandar. El mutex está porque el handler corre en otra
// goroutine y `make test-race` la mira.
type stub struct {
	mu     sync.Mutex
	path   string
	body   string
	bearer string
	calls  int
}

// newStub levanta el servidor y devuelve la captura. Se cierra solo al acabar el test.
func newStub(t *testing.T, status int, payload string) (*httptest.Server, *stub) {
	t.Helper()
	s := &stub{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, readErr := io.ReadAll(r.Body)
		if readErr != nil {
			t.Errorf("el stub no pudo leer el cuerpo de la peticion: %v", readErr)
		}
		s.mu.Lock()
		s.path = r.URL.Path
		s.body = string(raw)
		s.bearer = r.Header.Get("Authorization")
		s.calls++
		s.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if payload != "" {
			if _, writeErr := io.WriteString(w, payload); writeErr != nil {
				t.Errorf("el stub no pudo escribir la respuesta: %v", writeErr)
			}
		}
	}))
	t.Cleanup(srv.Close)
	return srv, s
}

// last devuelve la última petición capturada.
func (s *stub) last() (path, body, bearer string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.path, s.body, s.bearer
}

// count devuelve cuántas peticiones recibió el stub.
func (s *stub) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

// newTestClient construye un Client contra dos stubs y aborta si las opciones no valen: en un test
// un error de construcción es un fallo del propio test, no el caso bajo prueba.
func newTestClient(t *testing.T, system, identityURL, platformURL string) *Client {
	t.Helper()
	c, err := NewClient(Options{
		System:          system,
		IdentityBaseURL: identityURL,
		PlatformBaseURL: platformURL,
	})
	if err != nil {
		t.Fatalf("NewClient devolvió error inesperado: %v", err)
	}
	return c
}

// okIdentityBody es una respuesta de login/refresh de identity con el par completo.
func okIdentityBody(identityToken, refreshToken string) string {
	return `{"status":"ok","session_id":"s-1","system":"cualquiera",` +
		`"identity_token":"` + identityToken + `","refresh_token":"` + refreshToken + `","expires_in":900}`
}

// okExchangeBody es una respuesta del canje tal como la emite la plataforma, con los campos que este
// módulo NO consume incluidos: el contrato trae más de lo que se lee y eso no debe romper nada.
func okExchangeBody(contextToken, expiresAt string) string {
	return `{"context_token":"` + contextToken + `","token_type":"Bearer","expires_at":"` + expiresAt + `",` +
		`"context":{"tenant_id":"t-ignorado","user_id":"u-ignorado","roles":["ignorado"]}}`
}
