package logger

import "context"

type loggerKey struct{}

// WithContext inyecta un [Logger] en el contexto de Go.
func WithContext(ctx context.Context, l Logger) context.Context {
	if ctx == nil || l == nil {
		return ctx
	}
	return context.WithValue(ctx, loggerKey{}, l)
}

// FromContext recupera el [Logger] inyectado en el contexto de Go.
// Si el contexto es nil o no contiene un logger, devuelve [Default].
func FromContext(ctx context.Context) Logger {
	if ctx == nil {
		return Default()
	}
	if l, ok := ctx.Value(loggerKey{}).(Logger); ok && l != nil {
		return l
	}
	return Default()
}
