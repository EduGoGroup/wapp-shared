package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/EduGoGroup/wapp-shared/llm"
	"github.com/EduGoGroup/wapp-shared/llm/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// peticionCapturada guarda lo que el fake recibió, para afirmarlo desde el
// cuerpo del test y no desde la goroutine del servidor.
type peticionCapturada struct {
	ruta        string
	apiKey      string
	version     string
	contentType string
	cuerpo      map[string]any
}

// servidorFake levanta un servidor HTTP que responde lo que se le indique y
// captura la petición recibida.
func servidorFake(t *testing.T, estado int, cuerpo string) (*httptest.Server, *peticionCapturada) {
	t.Helper()
	capturada := &peticionCapturada{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturada.ruta = r.URL.Path
		capturada.apiKey = r.Header.Get("x-api-key")
		capturada.version = r.Header.Get("anthropic-version")
		capturada.contentType = r.Header.Get("content-type")
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&capturada.cuerpo))

		w.WriteHeader(estado)
		_, werr := w.Write([]byte(cuerpo))
		assert.NoError(t, werr)
	}))
	t.Cleanup(srv.Close)
	return srv, capturada
}

// respuestaAnthropic arma un cuerpo con la forma de la Messages API.
func respuestaAnthropic(t *testing.T, bloques ...map[string]string) string {
	t.Helper()
	body, err := json.Marshal(map[string]any{"content": bloques})
	require.NoError(t, err)
	return string(body)
}

// bloqueTexto es un bloque de contenido de tipo texto.
func bloqueTexto(texto string) map[string]string {
	return map[string]string{"type": "text", "text": texto}
}

// providerAnthropic construye el provider apuntando al servidor fake.
func providerAnthropic(t *testing.T, baseURL string) llm.LLMProvider {
	t.Helper()
	// #nosec G101 -- «clave-de-prueba» es un literal de test contra un httptest
	// local: no es una credencial real y nunca sale del proceso de test.
	p, err := api.New(api.Config{
		Provider: api.ProviderAnthropic,
		APIKey:   "clave-de-prueba",
		Model:    "modelo-de-prueba",
		BaseURL:  baseURL,
		Timeout:  5 * time.Second,
	})
	require.NoError(t, err)
	require.NotNil(t, p)
	return p
}

// entradaIdeas es la entrada que usan los tests de camino común.
func entradaIdeas() llm.ExtractMainIdeasInput {
	return llm.ExtractMainIdeasInput{SourceText: "quiero un paquete de tequeños de 30"}
}

func TestAnthropic_HappyPath(t *testing.T) {
	cuerpo := respuestaAnthropic(t, bloqueTexto(`{"version": 1, "wants": [{"idea":"tequeños","evidence":"tequeños de 30"}]}`))
	srv, _ := servidorFake(t, http.StatusOK, cuerpo)

	out, err := providerAnthropic(t, srv.URL).ExtractMainIdeas(context.Background(), entradaIdeas(), llm.Options{})
	require.NoError(t, err)

	ideas, err := llm.ParseMainIdeas(out)
	require.NoError(t, err)
	require.Len(t, ideas.Wants, 1)
	assert.Equal(t, "tequeños", ideas.Wants[0].Idea)
}

func TestAnthropic_PeticionBienFormada(t *testing.T) {
	srv, capturada := servidorFake(t, http.StatusOK, respuestaAnthropic(t, bloqueTexto(`{"version":1,"wants":[]}`)))

	_, err := providerAnthropic(t, srv.URL).ExtractMainIdeas(context.Background(), entradaIdeas(), llm.Options{})
	require.NoError(t, err)

	assert.Equal(t, "/v1/messages", capturada.ruta)
	assert.Equal(t, "clave-de-prueba", capturada.apiKey)
	assert.Equal(t, "2023-06-01", capturada.version)
	assert.Equal(t, "application/json", capturada.contentType)
	assert.Equal(t, "modelo-de-prueba", capturada.cuerpo["model"])
	assert.InDelta(t, float64(api.DefaultMaxTokens), capturada.cuerpo["max_tokens"], 0.5)
	assert.InDelta(t, llm.TemperatureGreedy, capturada.cuerpo["temperature"], 1e-9)
}

func TestAnthropic_TemperaturaDelReintentoViaja(t *testing.T) {
	srv, capturada := servidorFake(t, http.StatusOK, respuestaAnthropic(t, bloqueTexto(`{"version":1,"wants":[]}`)))

	_, err := providerAnthropic(t, srv.URL).ExtractMainIdeas(
		context.Background(), entradaIdeas(), llm.Options{Temperature: llm.TemperatureRetry})
	require.NoError(t, err)
	assert.InDelta(t, llm.TemperatureRetry, capturada.cuerpo["temperature"], 1e-9)
}

func TestAnthropic_ConcatenaBloquesDeTexto(t *testing.T) {
	// El JSON puede quedar partido entre dos bloques: se pegan sin separador.
	cuerpo := respuestaAnthropic(t,
		bloqueTexto(`{"version": 1, "wa`),
		map[string]string{"type": "thinking", "text": "esto no debe entrar"},
		bloqueTexto(`nts": []}`),
	)
	srv, _ := servidorFake(t, http.StatusOK, cuerpo)

	out, err := providerAnthropic(t, srv.URL).ExtractMainIdeas(context.Background(), entradaIdeas(), llm.Options{})
	require.NoError(t, err)
	assert.JSONEq(t, `{"version": 1, "wants": []}`, string(out))
}

func TestAnthropic_ErrorDeInfraNoEsErrorDeCalidad(t *testing.T) {
	// La distinción es el corazón de la tarea: «la API está caída» no se trata
	// como «el modelo devolvió basura».
	for _, estado := range []int{http.StatusInternalServerError, http.StatusBadGateway, http.StatusTooManyRequests} {
		srv, _ := servidorFake(t, estado, `{"error":{"message":"upstream caído"}}`)

		out, err := providerAnthropic(t, srv.URL).ExtractMainIdeas(context.Background(), entradaIdeas(), llm.Options{})
		require.Error(t, err)
		assert.Nil(t, out)
		assert.ErrorIs(t, err, api.ErrUpstream)
		assert.NotErrorIs(t, err, llm.ErrLLMQuality)
	}
}

func TestAnthropic_RespuestaQueNoEsDeLaMessagesAPIEsInfra(t *testing.T) {
	srv, _ := servidorFake(t, http.StatusOK, "<html>proxy interpuesto</html>")

	_, err := providerAnthropic(t, srv.URL).ExtractMainIdeas(context.Background(), entradaIdeas(), llm.Options{})
	require.Error(t, err)
	assert.ErrorIs(t, err, api.ErrUpstream)
	assert.NotErrorIs(t, err, llm.ErrLLMQuality)
}

func TestAnthropic_SalidaNoJSONEsErrorDeCalidad(t *testing.T) {
	// El proveedor respondió 200 y con su forma: lo que no sirve es lo que dijo
	// el modelo.
	srv, _ := servidorFake(t, http.StatusOK,
		respuestaAnthropic(t, bloqueTexto("Lo siento, no puedo ayudarte con eso.")))

	out, err := providerAnthropic(t, srv.URL).ExtractMainIdeas(context.Background(), entradaIdeas(), llm.Options{})
	require.Error(t, err)
	assert.Nil(t, out)
	assert.ErrorIs(t, err, llm.ErrLLMQuality)
	assert.NotErrorIs(t, err, api.ErrUpstream)
}

func TestAnthropic_SalidaTruncadaEsErrorDeCalidad(t *testing.T) {
	srv, _ := servidorFake(t, http.StatusOK,
		respuestaAnthropic(t, bloqueTexto(`{"version": 1, "wants": [{"idea": "tequ`)))

	_, err := providerAnthropic(t, srv.URL).ExtractMainIdeas(context.Background(), entradaIdeas(), llm.Options{})
	require.Error(t, err)
	assert.ErrorIs(t, err, llm.ErrLLMQuality)
}

func TestAnthropic_ContextoCanceladoEsInfra(t *testing.T) {
	srv, _ := servidorFake(t, http.StatusOK, respuestaAnthropic(t, bloqueTexto(`{"version":1,"wants":[]}`)))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := providerAnthropic(t, srv.URL).ExtractMainIdeas(ctx, entradaIdeas(), llm.Options{})
	require.Error(t, err)
	assert.ErrorIs(t, err, api.ErrUpstream)
	assert.NotErrorIs(t, err, llm.ErrLLMQuality)
}
