package iam

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// fakeJWT arma un JWT sin firmar (la firma es un relleno) con los claims que se le pasen. Sirve
// porque contextOf NO verifica la firma a propósito: solo lee el payload para la traza.
func fakeJWT(t *testing.T, claims map[string]any) string {
	t.Helper()
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("no se pudo serializar el payload del JWT de prueba: %v", err)
	}
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"ES256","typ":"JWT"}`))
	return header + "." + base64.RawURLEncoding.EncodeToString(payload) + ".firma-de-mentira"
}

// TestLoginEncadenaIdentityYCanje: los dos saltos server-to-server en uno. El Identity Token que sale
// de identity es exactamente el que se le presenta a la plataforma, y el refresh que vuelve al
// llamante es el de identity, no uno del canje (el canje no emite refresh).
func TestLoginEncadenaIdentityYCanje(t *testing.T) {
	t.Parallel()
	identitySrv, identityStub := newStub(t, http.StatusOK, okIdentityBody("idt-abc", "rt-abc"))
	ctxToken := fakeJWT(t, map[string]any{"tenant_id": "t-1", "user_id": "u-1", "roles": []string{"admin"}})
	platformSrv, platformStub := newStub(t, http.StatusOK, okExchangeBody(ctxToken, "2026-08-02T18:00:00Z"))

	res, err := newTestClient(t, "wapp.platform", identitySrv.URL, platformSrv.URL).
		Login(context.Background(), "a@b.com", "secret")
	if err != nil {
		t.Fatalf("el login delegado no debía fallar: %v", err)
	}

	if _, body, _ := identityStub.last(); !strings.Contains(body, `"system":"wapp.platform"`) {
		t.Errorf("identity debía recibir el system configurado, got %q", body)
	}
	if _, body, _ := platformStub.last(); !strings.Contains(body, `"identity_token":"idt-abc"`) {
		t.Errorf("la plataforma debía recibir el Identity Token recién emitido, got %q", body)
	}
	if res.AccessToken != ctxToken {
		t.Errorf("AccessToken debía ser el Context Token del canje")
	}
	if res.RefreshToken != "rt-abc" {
		t.Errorf("RefreshToken = %q, want el de identity (rt-abc)", res.RefreshToken)
	}
	if res.TokenType != "Bearer" {
		t.Errorf("TokenType = %q, want Bearer", res.TokenType)
	}
	if res.ExpiresAt != "2026-08-02T18:00:00Z" {
		t.Errorf("ExpiresAt = %q, want el que acotó la plataforma", res.ExpiresAt)
	}
	if res.Context.TenantID != "t-1" || res.Context.UserID != "u-1" || len(res.Context.Roles) != 1 {
		t.Errorf("el contexto debía salir de los claims del Context Token, got %+v", res.Context)
	}
}

// TestLoginNoDevuelveNuncaElIdentityToken: el Identity Token muere dentro del módulo. Si llegara al
// llamante acabaría en la cookie, y un Identity Token no tiene claims de negocio —no puede
// tenerlos—, así que el tenant desaparecería sin más aviso.
func TestLoginNoDevuelveNuncaElIdentityToken(t *testing.T) {
	t.Parallel()
	identitySrv, _ := newStub(t, http.StatusOK, okIdentityBody("idt-secreto", "rt-abc"))
	platformSrv, _ := newStub(t, http.StatusOK, okExchangeBody("ctx-abc", "2026-08-02T18:00:00Z"))

	res, err := newTestClient(t, "wapp.bff", identitySrv.URL, platformSrv.URL).
		Login(context.Background(), "a@b.com", "secret")
	if err != nil {
		t.Fatalf("el login delegado no debía fallar: %v", err)
	}
	for campo, valor := range map[string]string{
		"AccessToken":  res.AccessToken,
		"RefreshToken": res.RefreshToken,
		"ExpiresAt":    res.ExpiresAt,
	} {
		if strings.Contains(valor, "idt-secreto") {
			t.Errorf("%s devolvió el Identity Token al llamante: %q", campo, valor)
		}
	}
}

// TestLoginNoCanjeaSiIdentityFalla: sin Identity Token no hay nada que canjear. Llamar igualmente
// gastaría una petición y traduciría el fallo al vocabulario del canje, que es el equivocado.
func TestLoginNoCanjeaSiIdentityFalla(t *testing.T) {
	t.Parallel()
	identitySrv, _ := newStub(t, http.StatusUnauthorized, `{"error":"invalid_credentials"}`)
	platformSrv, platformStub := newStub(t, http.StatusOK, okExchangeBody("ctx-abc", "x"))

	_, err := newTestClient(t, "wapp.bff", identitySrv.URL, platformSrv.URL).
		Login(context.Background(), "a@b.com", "mala")
	if err == nil {
		t.Fatal("un login con credenciales malas debía fallar")
	}
	if platformStub.count() != 0 {
		t.Errorf("no debía llamarse al canje, recibió %d peticiones", platformStub.count())
	}
}

// TestRefreshRotaYVuelveACanjear: el refresh presentado es el vigente y el que vuelve es el rotado.
// Devolver el viejo dejaría al llamante renovando con un token que identity ya invalidó.
func TestRefreshRotaYVuelveACanjear(t *testing.T) {
	t.Parallel()
	identitySrv, identityStub := newStub(t, http.StatusOK, okIdentityBody("idt-2", "rt-nuevo"))
	platformSrv, _ := newStub(t, http.StatusOK, okExchangeBody("ctx-2", "2026-08-02T19:00:00Z"))

	res, err := newTestClient(t, "wapp.bff", identitySrv.URL, platformSrv.URL).
		Refresh(context.Background(), "rt-viejo")
	if err != nil {
		t.Fatalf("el refresh delegado no debía fallar: %v", err)
	}
	if _, body, _ := identityStub.last(); !strings.Contains(body, `"refresh_token":"rt-viejo"`) {
		t.Errorf("identity debía recibir el refresh vigente, got %q", body)
	}
	if res.RefreshToken != "rt-nuevo" {
		t.Errorf("RefreshToken = %q, want el rotado (rt-nuevo)", res.RefreshToken)
	}
}

// TestContextTokenIlegibleNoTumbaElLogin: la sesión es válida —la emitió la plataforma— aunque este
// módulo no sepa leerle los claims. Un contexto vacío degrada la traza; un error tumbaría el login.
func TestContextTokenIlegibleNoTumbaElLogin(t *testing.T) {
	t.Parallel()
	for name, token := range map[string]string{
		"no es un JWT":       "esto-no-es-un-jwt",
		"payload no base64":  "aaa.@@@@.bbb",
		"payload no es JSON": "aaa." + base64.RawURLEncoding.EncodeToString([]byte("no soy json")) + ".bbb",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			identitySrv, _ := newStub(t, http.StatusOK, okIdentityBody("idt", "rt"))
			platformSrv, _ := newStub(t, http.StatusOK, okExchangeBody(token, "2026-08-02T18:00:00Z"))

			res, err := newTestClient(t, "wapp.bff", identitySrv.URL, platformSrv.URL).
				Login(context.Background(), "a@b.com", "secret")
			if err != nil {
				t.Fatalf("un Context Token ilegible no debía tumbar el login: %v", err)
			}
			if res.AccessToken != token {
				t.Errorf("el token debía devolverse tal cual llegó")
			}
			if res.Context.TenantID != "" || res.Context.UserID != "" || len(res.Context.Roles) != 0 {
				t.Errorf("un token ilegible debía dar contexto vacío, got %+v", res.Context)
			}
		})
	}
}

// TestContextOfAceptaElPayloadConRelleno: el payload de un JWT es base64url sin relleno, pero hay
// emisores que lo mandan con él. Aceptar los dos evita un contexto vacío que nadie sabría explicar.
func TestContextOfAceptaElPayloadConRelleno(t *testing.T) {
	t.Parallel()
	payload := base64.URLEncoding.EncodeToString([]byte(`{"tenant_id":"t-9","user_id":"u-9"}`))
	if !strings.HasSuffix(payload, "=") {
		t.Fatalf("el fixture debía llevar relleno para probar lo que dice, got %q", payload)
	}
	got := contextOf("cabecera." + payload + ".firma")
	if got.TenantID != "t-9" || got.UserID != "u-9" {
		t.Errorf("contextOf con relleno = %+v, want tenant t-9 / user u-9", got)
	}
}
