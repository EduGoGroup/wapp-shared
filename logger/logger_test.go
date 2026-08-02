package logger_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/EduGoGroup/wapp-shared/logger"
)

// decodeLines parsea la salida JSON (una linea por registro) en mapas.
func decodeLines(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var records []map[string]any
	for _, line := range bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var rec map[string]any
		if err := json.Unmarshal(line, &rec); err != nil {
			t.Fatalf("no se pudo parsear linea JSON %q: %v", line, err)
		}
		records = append(records, rec)
	}
	return records
}

func TestInfoEmitsRecord(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New(
		logger.WithJSON(true),
		logger.WithWriter(&buf),
		logger.WithLevel(slog.LevelInfo),
	)

	log.Info("hola mundo", "clave", "valor")

	records := decodeLines(t, &buf)
	if len(records) != 1 {
		t.Fatalf("esperaba 1 registro, obtuve %d", len(records))
	}
	rec := records[0]
	if rec["msg"] != "hola mundo" {
		t.Errorf("msg = %v, esperaba 'hola mundo'", rec["msg"])
	}
	if rec["level"] != "INFO" {
		t.Errorf("level = %v, esperaba 'INFO'", rec["level"])
	}
	if rec["clave"] != "valor" {
		t.Errorf("clave = %v, esperaba 'valor'", rec["clave"])
	}
}

func TestWithAddsFields(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New(logger.WithJSON(true), logger.WithWriter(&buf))

	child := log.With("request_id", "abc-123")
	child.Info("procesando")

	records := decodeLines(t, &buf)
	if len(records) != 1 {
		t.Fatalf("esperaba 1 registro, obtuve %d", len(records))
	}
	if records[0]["request_id"] != "abc-123" {
		t.Errorf("request_id = %v, esperaba 'abc-123'", records[0]["request_id"])
	}
}

func TestWithDoesNotMutateParent(t *testing.T) {
	var buf bytes.Buffer
	parent := logger.New(logger.WithJSON(true), logger.WithWriter(&buf))

	_ = parent.With("scoped", "yes")
	parent.Info("sin scope")

	records := decodeLines(t, &buf)
	if len(records) != 1 {
		t.Fatalf("esperaba 1 registro, obtuve %d", len(records))
	}
	if _, ok := records[0]["scoped"]; ok {
		t.Errorf("el padre no deberia arrastrar campos del hijo: %v", records[0])
	}
}

func TestLevelFiltersBelowThreshold(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New(
		logger.WithJSON(true),
		logger.WithWriter(&buf),
		logger.WithLevel(slog.LevelWarn),
	)

	log.Debug("debug oculto")
	log.Info("info oculto")
	log.Warn("warn visible")
	log.Error("error visible")

	records := decodeLines(t, &buf)
	if len(records) != 2 {
		t.Fatalf("esperaba 2 registros (warn+error), obtuve %d: %v", len(records), records)
	}
	if records[0]["level"] != "WARN" {
		t.Errorf("primer level = %v, esperaba 'WARN'", records[0]["level"])
	}
	if records[1]["level"] != "ERROR" {
		t.Errorf("segundo level = %v, esperaba 'ERROR'", records[1]["level"])
	}
}

func TestTextFormatByDefault(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New(logger.WithWriter(&buf))

	log.Info("texto plano", "k", "v")

	out := buf.String()
	if len(out) == 0 {
		t.Fatal("esperaba salida en formato texto")
	}
	// El handler de texto incluye msg=... y los pares clave/valor.
	if !bytes.Contains(buf.Bytes(), []byte("msg=")) {
		t.Errorf("salida de texto sin 'msg=': %q", out)
	}
	if !bytes.Contains(buf.Bytes(), []byte("k=v")) {
		t.Errorf("salida de texto sin 'k=v': %q", out)
	}
}

func TestWithContextAndFromContext(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New(logger.WithJSON(true), logger.WithWriter(&buf))

	ctx := logger.WithContext(t.Context(), log)
	fromCtx := logger.FromContext(ctx)

	if fromCtx == nil {
		t.Fatal("FromContext devolvió nil")
	}

	fromCtx.Info("mensaje desde contexto", "ctx", "ok")

	records := decodeLines(t, &buf)
	if len(records) != 1 {
		t.Fatalf("esperaba 1 registro, obtuve %d", len(records))
	}
	if records[0]["msg"] != "mensaje desde contexto" {
		t.Errorf("msg = %v, esperaba 'mensaje desde contexto'", records[0]["msg"])
	}

	// Contexto vacio devuelve Default()
	var nilCtx context.Context
	defaultLog := logger.FromContext(nilCtx)
	if defaultLog == nil {
		t.Error("FromContext(nilCtx) devolvió nil")
	}
}
