// Package logger ofrece logging estructurado para el ecosistema wApp.
//
// La API se reduce a una interfaz [Logger] minima con metodos por nivel
// (Debug, Info, Warn, Error) y un metodo [Logger.With] para derivar loggers
// hijos que arrastran campos clave/valor.
//
// La implementacion por defecto se apoya exclusivamente en la libreria estandar
// (log/slog), sin dependencias externas. Se construye con [New] y un conjunto de
// opciones funcionales:
//
//	log := logger.New(
//		logger.WithLevel(slog.LevelDebug),
//		logger.WithJSON(true),
//	)
//	log.Info("servidor iniciado", "puerto", 8080)
//
//	reqLog := log.With("request_id", "abc-123")
//	reqLog.Warn("latencia alta", "ms", 250)
//
// Los argumentos variadics siguen la semantica clave/valor de slog: se
// interpretan en pares (clave string, valor any).
package logger
