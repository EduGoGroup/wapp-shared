package iam

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

// TestIdentityLoginViajaConElSystemConfigurado: el login lleva la clave namespaced del catálogo de
// identity. Es lo que el System Gate evalúa; sin ella no hay autorización de aplicación que valga.
func TestIdentityLoginViajaConElSystemConfigurado(t *testing.T) {
	t.Parallel()
	srv, captured := newStub(t, http.StatusOK, okIdentityBody("idt-abc", "rt-abc"))

	c := newTestClient(t, "wapp.bff", srv.URL, srv.URL)
	tokens, err := c.IdentityLogin(context.Background(), "a@b.com", "secret")
	if err != nil {
		t.Fatalf("el login contra identity no debía fallar: %v", err)
	}

	path, body, _ := captured.last()
	if path != "/api/v1/auth/login" {
		t.Errorf("ruta = %q, want /api/v1/auth/login", path)
	}
	if !strings.Contains(body, `"system":"wapp.bff"`) {
		t.Errorf("el cuerpo debía declarar el system de la aplicación, got %q", body)
	}
	if tokens.IdentityToken != "idt-abc" || tokens.RefreshToken != "rt-abc" {
		t.Errorf("el par de tokens no se leyó de la respuesta, got %+v", tokens)
	}
	if tokens.ExpiresIn != 900 {
		t.Errorf("expires_in = %d, want 900", tokens.ExpiresIn)
	}
}

// TestElMismoClienteSirveCualquierSystemSinRamas es el requisito que motivó extraer este módulo: el
// `system` dejó de ser una constante del binario (la consola lo tenía clavado a "wapp.platform") y
// pasó a ser un campo del cliente.
//
// La tabla incluye a propósito un valor que NO existe en el ecosistema: si alguien introdujera una
// lista blanca o un `if system == …`, el tercer caso lo delataría. Lo que se afirma es que lo que
// viaja por el cable es EXACTAMENTE lo configurado, para cualquier valor y por el mismo camino.
func TestElMismoClienteSirveCualquierSystemSinRamas(t *testing.T) {
	t.Parallel()
	for _, system := range []string{"wapp.bff", "wapp.platform", "otro.sistema.inventado"} {
		t.Run(system, func(t *testing.T) {
			t.Parallel()
			srv, captured := newStub(t, http.StatusOK, okIdentityBody("idt", "rt"))

			c := newTestClient(t, system, srv.URL, srv.URL)
			if _, err := c.IdentityLogin(context.Background(), "a@b.com", "secret"); err != nil {
				t.Fatalf("el login con system %q no debía fallar: %v", system, err)
			}
			if c.System() != system {
				t.Errorf("System() = %q, want %q", c.System(), system)
			}

			_, body, _ := captured.last()
			if !strings.Contains(body, `"system":"`+system+`"`) {
				t.Errorf("el cuerpo debía declarar el system %q, got %q", system, body)
			}
		})
	}
}

// TestIdentityLogin403NoEsCredencialInvalida: el System Gate niega con la contraseña CORRECTA. Es un
// caso distinto del 401 y no se colapsa con él —la consola de operadores sí los colapsaba, y el
// operador leía «credenciales inválidas» teniendo la contraseña buena.
func TestIdentityLogin403NoEsCredencialInvalida(t *testing.T) {
	t.Parallel()
	srv, _ := newStub(t, http.StatusForbidden, `{"error":"system_access_denied"}`)

	_, err := newTestClient(t, "wapp.bff", srv.URL, srv.URL).IdentityLogin(context.Background(), "a@b.com", "secret")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("un 403 del gate debía ser ErrForbidden, got %v", err)
	}
	if errors.Is(err, ErrUnauthorized) {
		t.Error("el 403 del gate NO debe confundirse con credenciales inválidas")
	}
	if got := StatusCodeOf(err); got != http.StatusForbidden {
		t.Errorf("el error debía preservar el 403 del emisor, got %d", got)
	}
}

// TestIdentityLogin401EsCredencialInvalidaYConservaElStatus: el 401 tiene que responder a las DOS
// preguntas a la vez. En la fuente que se extrajo el sentinela iba por fuera (fmt.Errorf) y
// StatusCodeOf devolvía 0 justo en el status que más se diagnostica; aquí el sentinela viaja dentro
// del APIError.
func TestIdentityLogin401EsCredencialInvalidaYConservaElStatus(t *testing.T) {
	t.Parallel()
	srv, _ := newStub(t, http.StatusUnauthorized, `{"error":"invalid_credentials"}`)

	_, err := newTestClient(t, "wapp.bff", srv.URL, srv.URL).IdentityLogin(context.Background(), "a@b.com", "mala")
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("un 401 debía ser ErrUnauthorized, got %v", err)
	}
	if errors.Is(err, ErrForbidden) {
		t.Error("el 401 NO es el gate de aplicación")
	}
	if got := StatusCodeOf(err); got != http.StatusUnauthorized {
		t.Errorf("StatusCodeOf del 401 = %d, want 401", got)
	}
}

// TestIdentityLogin5xxNoEsUnFalloDeCredencial: una avería del emisor no debe leerse como «tu
// contraseña no vale», que llevaría al llamante a borrar la sesión de un usuario legítimo.
func TestIdentityLogin5xxNoEsUnFalloDeCredencial(t *testing.T) {
	t.Parallel()
	srv, _ := newStub(t, http.StatusBadGateway, `{"error":"upstream"}`)

	_, err := newTestClient(t, "wapp.bff", srv.URL, srv.URL).IdentityLogin(context.Background(), "a@b.com", "secret")
	if err == nil {
		t.Fatal("un 502 debía devolver error")
	}
	if errors.Is(err, ErrUnauthorized) || errors.Is(err, ErrForbidden) {
		t.Error("una avería del emisor no es un fallo de credencial")
	}
	if got := StatusCodeOf(err); got != http.StatusBadGateway {
		t.Errorf("StatusCodeOf = %d, want 502", got)
	}
}

// TestIdentityExigeElParCompleto: una respuesta sin refresh dejaría una sesión sin forma de
// renovarse y el fallo aparecería quince minutos después, lejos de su causa. Se rechaza al llegar.
//
// La fuente de la consola no validaba nada: un 200 con `{}` devolvía tokens vacíos y el canje
// posterior fallaba con un 401 que parecía culpa del usuario.
func TestIdentityExigeElParCompleto(t *testing.T) {
	t.Parallel()
	casos := []struct {
		nombre string
		cuerpo string
	}{
		{"le falta el refresh", `{"status":"ok","identity_token":"idt-abc","expires_in":900}`},
		{"le falta el identity", `{"status":"ok","refresh_token":"rt-abc","expires_in":900}`},
		{"cuerpo vacío", `{}`},
	}
	for _, caso := range casos {
		name := caso.nombre
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			srv, _ := newStub(t, http.StatusOK, caso.cuerpo)
			c := newTestClient(t, "wapp.bff", srv.URL, srv.URL)
			if _, err := c.IdentityLogin(context.Background(), "a@b.com", "x"); err == nil {
				t.Fatalf("un 200 %s debía rechazarse", name)
			}
			if _, err := c.IdentityRefresh(context.Background(), "rt-1"); err == nil {
				t.Fatalf("un refresh con 200 %s debía rechazarse", name)
			}
		})
	}
}

// TestIdentityRefreshNoDeclaraSystem: la aplicación sale de la fila de la sesión en identity, jamás
// del cliente. Si el refresh pudiera declarar system se canjearía el refresh de una aplicación por
// el token de otra y el System Gate quedaría sorteado.
func TestIdentityRefreshNoDeclaraSystem(t *testing.T) {
	t.Parallel()
	srv, captured := newStub(t, http.StatusOK, okIdentityBody("idt-2", "rt-2"))

	c := newTestClient(t, "wapp.bff", srv.URL, srv.URL)
	if _, err := c.IdentityRefresh(context.Background(), "rt-1"); err != nil {
		t.Fatalf("el refresh contra identity no debía fallar: %v", err)
	}

	path, body, _ := captured.last()
	if path != "/api/v1/auth/refresh" {
		t.Errorf("ruta = %q, want /api/v1/auth/refresh", path)
	}
	if strings.Contains(body, "system") {
		t.Errorf("el refresh NO debe declarar system, got %q", body)
	}
	if !strings.Contains(body, `"refresh_token":"rt-1"`) {
		t.Errorf("el refresh debía presentar el token vigente, got %q", body)
	}
}

// TestLogoutPresentaSoloElRefreshSinBearer: identity resuelve al usuario server-side a partir del
// refresh. No lleva Authorization —el Context Token que el llamante custodia lo emitió wApp, no
// identity— y cierra solo la sesión de esta aplicación.
func TestLogoutPresentaSoloElRefreshSinBearer(t *testing.T) {
	t.Parallel()
	srv, captured := newStub(t, http.StatusNoContent, "")

	if err := newTestClient(t, "wapp.bff", srv.URL, srv.URL).Logout(context.Background(), "rt-bff"); err != nil {
		t.Fatalf("el logout contra identity no debía fallar: %v", err)
	}

	path, body, bearer := captured.last()
	if path != "/api/v1/auth/logout" {
		t.Errorf("ruta = %q, want /api/v1/auth/logout", path)
	}
	if bearer != "" {
		t.Errorf("el logout de identity no debe llevar Authorization, got %q", bearer)
	}
	if !strings.Contains(body, `"refresh_token":"rt-bff"`) {
		t.Errorf("el logout debía presentar el refresh de la sesión, got %q", body)
	}
}

// TestLogoutAceptaCualquier2xx: identity responde 204, pero el contrato es «2xx». La fuente de la
// consola exigía 200 exacto en login/refresh/canje, así que un 201 o un 204 habrían viajado como
// error.
func TestLogoutAceptaCualquier2xx(t *testing.T) {
	t.Parallel()
	for _, status := range []int{http.StatusOK, http.StatusNoContent, http.StatusAccepted} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			t.Parallel()
			srv, _ := newStub(t, status, "")
			if err := newTestClient(t, "wapp.bff", srv.URL, srv.URL).Logout(context.Background(), "rt-1"); err != nil {
				t.Fatalf("el logout con %d = %v, want nil", status, err)
			}
		})
	}
}

// TestLogoutPropagaElErrorDelUpstream fija el hallazgo de la consola (CODE-REVIEW-2026-08-15 #3):
// antes el Logout ignoraba el status y siempre devolvía nil, así que un 500 quedaba invisible y la
// consola decía «sesión cerrada» mientras el refresh token seguía vivo en identity.
func TestLogoutPropagaElErrorDelUpstream(t *testing.T) {
	t.Parallel()
	srv, _ := newStub(t, http.StatusInternalServerError, "")

	err := newTestClient(t, "wapp.bff", srv.URL, srv.URL).Logout(context.Background(), "rt-1")
	if err == nil {
		t.Fatal("Logout con identity devolviendo 500 = nil, want error")
	}
	if got := StatusCodeOf(err); got != http.StatusInternalServerError {
		t.Errorf("StatusCodeOf = %d, want 500", got)
	}
}
