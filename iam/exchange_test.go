package iam

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

// TestExchangeDevuelveElContextToken: el canje presenta el Identity Token y recibe el Context Token
// con su vencimiento. No hay refresh en la respuesta y no debe haberlo: el refresh es de identity y
// vive donde vive la sesión.
//
// El cuerpo del stub trae `token_type` y `context`, que este módulo NO consume: el tenant se lee
// SIEMPRE de los claims del Context Token, no del sobre que lo trae.
func TestExchangeDevuelveElContextToken(t *testing.T) {
	t.Parallel()
	srv, captured := newStub(t, http.StatusOK, okExchangeBody("ctx-abc", "2026-08-02T18:00:00Z"))

	res, err := newTestClient(t, "wapp.bff", srv.URL, srv.URL).Exchange(context.Background(), "idt-abc")
	if err != nil {
		t.Fatalf("el canje no debía fallar: %v", err)
	}

	path, body, _ := captured.last()
	if path != "/api/v1/auth/exchange" {
		t.Errorf("ruta = %q, want /api/v1/auth/exchange", path)
	}
	if !strings.Contains(body, `"identity_token":"idt-abc"`) {
		t.Errorf("el canje debía presentar el Identity Token, got %q", body)
	}
	if res.ContextToken != "ctx-abc" || res.ExpiresAt != "2026-08-02T18:00:00Z" {
		t.Errorf("no se leyó la respuesta del canje, got %+v", res)
	}
}

// TestExchangeVaALaPlataformaNoAIdentity: son dos destinos distintos y el canje es de la plataforma.
// Con las dos URLs separadas, mandarlo a identity sería un 404 en campo y aquí un stub sin llamadas.
func TestExchangeVaALaPlataformaNoAIdentity(t *testing.T) {
	t.Parallel()
	identitySrv, identityStub := newStub(t, http.StatusOK, "{}")
	platformSrv, platformStub := newStub(t, http.StatusOK, okExchangeBody("ctx-abc", "2026-08-02T18:00:00Z"))

	c := newTestClient(t, "wapp.bff", identitySrv.URL, platformSrv.URL)
	if _, err := c.Exchange(context.Background(), "idt-abc"); err != nil {
		t.Fatalf("el canje no debía fallar: %v", err)
	}
	if identityStub.count() != 0 {
		t.Errorf("el canje no debe tocar identity, recibió %d peticiones", identityStub.count())
	}
	if platformStub.count() != 1 {
		t.Errorf("la plataforma debía recibir 1 petición, recibió %d", platformStub.count())
	}
}

// TestExchange503EsModoDualApagado: la plataforma sin verificador de Identity Tokens no puede canjear
// nada. Tiene error propio porque no es una avería: es un despliegue a medias y se arregla
// configurando, no reintentando.
func TestExchange503EsModoDualApagado(t *testing.T) {
	t.Parallel()
	srv, _ := newStub(t, http.StatusServiceUnavailable, `{"error":"dual_mode_off"}`)

	_, err := newTestClient(t, "wapp.bff", srv.URL, srv.URL).Exchange(context.Background(), "idt-abc")
	if !errors.Is(err, ErrDualModeOff) {
		t.Fatalf("un 503 del canje debía ser ErrDualModeOff, got %v", err)
	}
}

// TestExchange401EsUsuarioNoMigrado: el `sub` del Identity Token no corresponde a ningún usuario de
// wApp. Es un error explícito del contrato —no se crea un usuario al vuelo— y viaja como no
// autorizado, que es lo que hace que el llamante limpie la sesión en vez de degradar.
func TestExchange401EsUsuarioNoMigrado(t *testing.T) {
	t.Parallel()
	srv, _ := newStub(t, http.StatusUnauthorized, `{"error":"user_not_migrated"}`)

	_, err := newTestClient(t, "wapp.bff", srv.URL, srv.URL).Exchange(context.Background(), "idt-abc")
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("un 401 del canje debía ser ErrUnauthorized, got %v", err)
	}
}

// TestExchange409NoEsUnFalloDeCredencial: el usuario resultó tener más de un tenant, un estado que el
// canje rechaza en vez de elegir uno en silencio.
//
// Que NO viaje como ErrUnauthorized es lo que hay que sostener: de ese error depende que el llamante
// distinga «tu sesión ya no vale» —donde limpiar la cookie es lo correcto— de «tu sesión vale pero
// wApp no sabe en qué tenant ponerte», donde echar al usuario no arregla nada y borra la pista.
func TestExchange409NoEsUnFalloDeCredencial(t *testing.T) {
	t.Parallel()
	srv, _ := newStub(t, http.StatusConflict, `{"error":"multiple_tenants"}`)

	_, err := newTestClient(t, "wapp.bff", srv.URL, srv.URL).Exchange(context.Background(), "idt-abc")
	if err == nil {
		t.Fatal("un 409 del canje debía devolver error")
	}
	if errors.Is(err, ErrUnauthorized) {
		t.Error("el 409 multi-tenant NO es un fallo de credencial")
	}
	if errors.Is(err, ErrDualModeOff) {
		t.Error("el 409 tampoco es el modo dual apagado")
	}
	if got := StatusCodeOf(err); got != http.StatusConflict {
		t.Errorf("el error debía preservar el 409 de la plataforma, got %d", got)
	}
}

// TestExchangeSinContextTokenEsError: un 200 vacío dejaría al llamante custodiando una cookie sin
// token y el fallo aparecería en la siguiente llamada de negocio, no en el login.
func TestExchangeSinContextTokenEsError(t *testing.T) {
	t.Parallel()
	srv, _ := newStub(t, http.StatusOK, `{"expires_at":"2026-08-02T18:00:00Z"}`)

	if _, err := newTestClient(t, "wapp.bff", srv.URL, srv.URL).Exchange(context.Background(), "idt-abc"); err == nil {
		t.Fatal("un canje sin context_token debía rechazarse")
	}
}
