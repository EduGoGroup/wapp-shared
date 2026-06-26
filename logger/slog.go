package logger

import (
	"io"
	"log/slog"
	"os"
)

// options agrupa la configuracion de construccion de un [slogLogger].
type options struct {
	level  slog.Level
	json   bool
	writer io.Writer
}

// Option configura la construccion de un Logger mediante [New].
type Option func(*options)

// WithLevel fija el nivel minimo de logging. Los mensajes por debajo del nivel
// se descartan.
func WithLevel(level slog.Level) Option {
	return func(o *options) {
		o.level = level
	}
}

// WithJSON selecciona el formato de salida: JSON cuando enabled es true, texto
// en caso contrario.
func WithJSON(enabled bool) Option {
	return func(o *options) {
		o.json = enabled
	}
}

// WithWriter define el destino de escritura del logger. Por defecto os.Stdout.
func WithWriter(w io.Writer) Option {
	return func(o *options) {
		if w != nil {
			o.writer = w
		}
	}
}

// slogLogger es la implementacion de [Logger] sobre log/slog.
type slogLogger struct {
	l *slog.Logger
}

// New construye un [Logger] sobre log/slog aplicando las opciones dadas.
//
// Por defecto: nivel Info, formato texto y salida a os.Stdout.
func New(opts ...Option) Logger {
	cfg := options{
		level:  slog.LevelInfo,
		json:   false,
		writer: os.Stdout,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	handlerOpts := &slog.HandlerOptions{Level: cfg.level}

	var handler slog.Handler
	if cfg.json {
		handler = slog.NewJSONHandler(cfg.writer, handlerOpts)
	} else {
		handler = slog.NewTextHandler(cfg.writer, handlerOpts)
	}

	return &slogLogger{l: slog.New(handler)}
}

// Default devuelve un [Logger] respaldado por el slog.Logger por defecto del
// proceso (slog.Default).
func Default() Logger {
	return &slogLogger{l: slog.Default()}
}

func (s *slogLogger) Debug(msg string, args ...any) { s.l.Debug(msg, args...) }
func (s *slogLogger) Info(msg string, args ...any)  { s.l.Info(msg, args...) }
func (s *slogLogger) Warn(msg string, args ...any)  { s.l.Warn(msg, args...) }
func (s *slogLogger) Error(msg string, args ...any) { s.l.Error(msg, args...) }

// With devuelve un Logger hijo que arrastra los args (pares clave/valor) en
// todas sus emisiones.
func (s *slogLogger) With(args ...any) Logger {
	return &slogLogger{l: s.l.With(args...)}
}
