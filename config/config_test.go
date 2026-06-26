package config_test

import (
	"os"
	"path/filepath"
	"testing"

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
