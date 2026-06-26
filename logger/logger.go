package logger

// Logger es la interfaz minima de logging estructurado del ecosistema wApp.
//
// Los metodos por nivel reciben un mensaje y una lista variadic de argumentos
// con semantica clave/valor de slog: pares (clave string, valor any). Por
// ejemplo: log.Info("usuario creado", "id", 42, "rol", "admin").
type Logger interface {
	// Debug registra un mensaje en nivel Debug.
	Debug(msg string, args ...any)
	// Info registra un mensaje en nivel Info.
	Info(msg string, args ...any)
	// Warn registra un mensaje en nivel Warn.
	Warn(msg string, args ...any)
	// Error registra un mensaje en nivel Error.
	Error(msg string, args ...any)
	// With devuelve un Logger hijo que arrastra los args dados (pares
	// clave/valor) en todas sus emisiones posteriores.
	With(args ...any) Logger
}
