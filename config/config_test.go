package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/EduGoGroup/wapp-shared/config"
)

func TestUnmarshalFromYAMLFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := "host: localhost\npuerto: 8080\ndebug: true\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("no se pudo escribir el YAML temporal: %v", err)
	}

	loader := config.New(config.WithFile(path))

	var cfg struct {
		Host   string `yaml:"host"`
		Puerto int    `yaml:"puerto"`
		Debug  bool   `yaml:"debug"`
	}
	if err := loader.Unmarshal(&cfg); err != nil {
		t.Fatalf("Unmarshal devolvio error: %v", err)
	}

	if cfg.Host != "localhost" {
		t.Errorf("Host = %q, esperaba 'localhost'", cfg.Host)
	}
	if cfg.Puerto != 8080 {
		t.Errorf("Puerto = %d, esperaba 8080", cfg.Puerto)
	}
	if !cfg.Debug {
		t.Errorf("Debug = %v, esperaba true", cfg.Debug)
	}
}

func TestUnmarshalMissingFileDoesNotFail(t *testing.T) {
	loader := config.New(config.WithFile(filepath.Join(t.TempDir(), "noexiste.yaml")))

	var cfg struct {
		Host string `yaml:"host"`
	}
	if err := loader.Unmarshal(&cfg); err != nil {
		t.Fatalf("Unmarshal con archivo inexistente no deberia fallar: %v", err)
	}
	if cfg.Host != "" {
		t.Errorf("Host = %q, esperaba vacio", cfg.Host)
	}
}

func TestUnmarshalNoFileConfigured(t *testing.T) {
	loader := config.New()

	var cfg struct {
		Host string `yaml:"host"`
	}
	if err := loader.Unmarshal(&cfg); err != nil {
		t.Fatalf("Unmarshal sin archivo no deberia fallar: %v", err)
	}
}

func TestGettersWithPrefix(t *testing.T) {
	t.Setenv("WAPP_HOST", "db.internal")
	t.Setenv("WAPP_PORT", "5432")
	t.Setenv("WAPP_TLS", "true")

	loader := config.New(config.WithEnvPrefix("WAPP_"))

	if got := loader.GetString("HOST", "localhost"); got != "db.internal" {
		t.Errorf("GetString(HOST) = %q, esperaba 'db.internal'", got)
	}
	if got := loader.GetInt("PORT", 1234); got != 5432 {
		t.Errorf("GetInt(PORT) = %d, esperaba 5432", got)
	}
	if got := loader.GetBool("TLS", false); !got {
		t.Errorf("GetBool(TLS) = %v, esperaba true", got)
	}
}

func TestGettersDefaults(t *testing.T) {
	loader := config.New(config.WithEnvPrefix("WAPP_"))

	if got := loader.GetString("AUSENTE", "def"); got != "def" {
		t.Errorf("GetString default = %q, esperaba 'def'", got)
	}
	if got := loader.GetInt("AUSENTE", 99); got != 99 {
		t.Errorf("GetInt default = %d, esperaba 99", got)
	}
	if got := loader.GetBool("AUSENTE", true); !got {
		t.Errorf("GetBool default = %v, esperaba true", got)
	}
}

func TestGettersInvalidValueFallsBackToDefault(t *testing.T) {
	t.Setenv("PORT", "no-numero")
	t.Setenv("FLAG", "quizas")

	loader := config.New() // sin prefijo

	if got := loader.GetInt("PORT", 7); got != 7 {
		t.Errorf("GetInt con valor invalido = %d, esperaba default 7", got)
	}
	if got := loader.GetBool("FLAG", false); got {
		t.Errorf("GetBool con valor invalido = %v, esperaba default false", got)
	}
}

func TestGettersWithoutPrefix(t *testing.T) {
	t.Setenv("PLAIN_KEY", "valor")

	loader := config.New()

	if got := loader.GetString("PLAIN_KEY", "def"); got != "valor" {
		t.Errorf("GetString sin prefijo = %q, esperaba 'valor'", got)
	}
}

func TestGetIntE_InvalidReturnsError(t *testing.T) {
	t.Setenv("WAPP_PORT", "81OO") // O en vez de 0: valor presente pero inválido

	loader := config.New(config.WithEnvPrefix("WAPP_"))

	got, err := loader.GetIntE("PORT", 7)
	if !errors.Is(err, config.ErrInvalid) {
		t.Errorf("GetIntE con valor inválido, err = %v, esperaba ErrInvalid", err)
	}
	if got != 7 {
		t.Errorf("GetIntE con valor inválido devuelve %d, esperaba default 7", got)
	}
}

func TestGetIntE_AbsentNoError(t *testing.T) {
	loader := config.New(config.WithEnvPrefix("WAPP_"))

	got, err := loader.GetIntE("AUSENTE", 42)
	if err != nil {
		t.Errorf("GetIntE ausente devolvió error inesperado: %v", err)
	}
	if got != 42 {
		t.Errorf("GetIntE ausente = %d, esperaba default 42", got)
	}
}

func TestGetBoolE_InvalidReturnsError(t *testing.T) {
	t.Setenv("FLAG", "quizas")

	loader := config.New()

	got, err := loader.GetBoolE("FLAG", true)
	if !errors.Is(err, config.ErrInvalid) {
		t.Errorf("GetBoolE con valor inválido, err = %v, esperaba ErrInvalid", err)
	}
	if !got {
		t.Errorf("GetBoolE con valor inválido = %v, esperaba default true", got)
	}
}

func TestGetDuration(t *testing.T) {
	t.Setenv("WAPP_TIMEOUT", "1500ms")

	loader := config.New(config.WithEnvPrefix("WAPP_"))

	if got := loader.GetDuration("TIMEOUT", time.Second); got != 1500*time.Millisecond {
		t.Errorf("GetDuration = %v, esperaba 1.5s", got)
	}
	if got := loader.GetDuration("AUSENTE", 3*time.Second); got != 3*time.Second {
		t.Errorf("GetDuration ausente = %v, esperaba default 3s", got)
	}
}

func TestGetDurationE_InvalidReturnsError(t *testing.T) {
	t.Setenv("WAPP_TIMEOUT", "no-dura")

	loader := config.New(config.WithEnvPrefix("WAPP_"))

	got, err := loader.GetDurationE("TIMEOUT", time.Second)
	if !errors.Is(err, config.ErrInvalid) {
		t.Errorf("GetDurationE con valor inválido, err = %v, esperaba ErrInvalid", err)
	}
	if got != time.Second {
		t.Errorf("GetDurationE con valor inválido = %v, esperaba default 1s", got)
	}
}

func TestRequire_MissingReturnsErrMissing(t *testing.T) {
	loader := config.New(config.WithEnvPrefix("WAPP_"))

	if _, err := loader.RequireString("AUSENTE"); !errors.Is(err, config.ErrMissing) {
		t.Errorf("RequireString ausente, err = %v, esperaba ErrMissing", err)
	}
	if _, err := loader.RequireInt("AUSENTE"); !errors.Is(err, config.ErrMissing) {
		t.Errorf("RequireInt ausente, err = %v, esperaba ErrMissing", err)
	}
	if _, err := loader.RequireBool("AUSENTE"); !errors.Is(err, config.ErrMissing) {
		t.Errorf("RequireBool ausente, err = %v, esperaba ErrMissing", err)
	}
	if _, err := loader.RequireDuration("AUSENTE"); !errors.Is(err, config.ErrMissing) {
		t.Errorf("RequireDuration ausente, err = %v, esperaba ErrMissing", err)
	}
}

func TestRequire_PresentValid(t *testing.T) {
	t.Setenv("WAPP_HOST", "db.internal")
	t.Setenv("WAPP_PORT", "5432")
	t.Setenv("WAPP_TLS", "true")
	t.Setenv("WAPP_TIMEOUT", "30s")

	loader := config.New(config.WithEnvPrefix("WAPP_"))

	if got, err := loader.RequireString("HOST"); err != nil || got != "db.internal" {
		t.Errorf("RequireString = (%q, %v), esperaba ('db.internal', nil)", got, err)
	}
	if got, err := loader.RequireInt("PORT"); err != nil || got != 5432 {
		t.Errorf("RequireInt = (%d, %v), esperaba (5432, nil)", got, err)
	}
	if got, err := loader.RequireBool("TLS"); err != nil || !got {
		t.Errorf("RequireBool = (%v, %v), esperaba (true, nil)", got, err)
	}
	if got, err := loader.RequireDuration("TIMEOUT"); err != nil || got != 30*time.Second {
		t.Errorf("RequireDuration = (%v, %v), esperaba (30s, nil)", got, err)
	}
}

func TestRequireInt_PresentInvalid(t *testing.T) {
	t.Setenv("WAPP_PORT", "no-numero")

	loader := config.New(config.WithEnvPrefix("WAPP_"))

	if _, err := loader.RequireInt("PORT"); !errors.Is(err, config.ErrInvalid) {
		t.Errorf("RequireInt inválido, err = %v, esperaba ErrInvalid", err)
	}
}

func TestMapEnvProvider(t *testing.T) {
	envMap := config.MapEnvProvider{
		"WAPP_HOST": "127.0.0.1",
		"WAPP_PORT": "9090",
	}

	loader := config.New(
		config.WithEnvPrefix("WAPP_"),
		config.WithEnvProvider(envMap),
	)

	if got := loader.GetString("HOST", "localhost"); got != "127.0.0.1" {
		t.Errorf("GetString(HOST) = %q, esperaba '127.0.0.1'", got)
	}
	if got := loader.GetInt("PORT", 8080); got != 9090 {
		t.Errorf("GetInt(PORT) = %d, esperaba 9090", got)
	}
}

