// Package health ofrece un framework minimo y extensible de health checks para el
// ecosistema wApp, construido solo sobre la libreria estandar.
//
// El patron es Checker + HealthCheck: cada componente (base de datos, cache, socket,
// etc.) implementa la interfaz [HealthCheck] (Name + Check), se registra en un
// [Checker] con [Checker.Register], y [Checker.CheckAll] ejecuta todos y agrega los
// resultados indexados por nombre de componente.
//
//	c := health.NewChecker()
//	c.Register(miCheck)
//	results := c.CheckAll(ctx)        // map[string]CheckResult
//	ok := c.IsHealthy(ctx)            // false si algun check esta unhealthy
//
// Cada resultado ([CheckResult]) lleva un [Status] (healthy/unhealthy/degraded), el
// nombre del componente, un mensaje opcional, un timestamp y metadata libre. Un estado
// degraded no marca al conjunto como unhealthy; solo unhealthy lo hace.
//
// El paquete no depende de drivers concretos: define unicamente el contrato. Los checks
// especificos (Postgres, Mongo, socket de whatsmeow, etc.) se implementan donde se usan.
package health
