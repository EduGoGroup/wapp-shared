package health

import (
	"context"
	"sync"
	"time"
)

// Status representa el estado de salud de un componente.
type Status string

const (
	// StatusHealthy indica que el componente opera con normalidad.
	StatusHealthy Status = "healthy"
	// StatusUnhealthy indica que el componente no responde o falla.
	StatusUnhealthy Status = "unhealthy"
	// StatusDegraded indica que el componente responde pero con limitaciones.
	StatusDegraded Status = "degraded"
)

// CheckResult representa el resultado de un health check individual.
type CheckResult struct {
	Status    Status         `json:"status"`
	Component string         `json:"component"`
	Message   string         `json:"message,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// HealthCheck representa un health check individual de un componente.
//
//nolint:revive // nombre fijado por la especificacion del modulo (HealthCheck), no se renombra a Check.
type HealthCheck interface {
	// Name devuelve el identificador del componente chequeado.
	Name() string
	// Check ejecuta la verificacion y devuelve su resultado.
	Check(ctx context.Context) CheckResult
}

// Checker gestiona y agrega multiples health checks.
//
// Es seguro para uso concurrente: Register y las operaciones de lectura
// (CheckAll/IsHealthy/IsReady) se sincronizan con un RWMutex. El contrato
// recomendado sigue siendo registrar los checks en el arranque y solo leer
// despues, pero registrar en caliente es seguro.
type Checker struct {
	mu     sync.RWMutex
	checks []HealthCheck
}

// NewChecker crea un Checker vacio listo para registrar checks.
func NewChecker() *Checker {
	return &Checker{
		checks: make([]HealthCheck, 0),
	}
}

// Register agrega un health check al Checker.
// Si check es nil, se ignora silenciosamente.
func (c *Checker) Register(check HealthCheck) {
	if check == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.checks = append(c.checks, check)
}

// CheckAll ejecuta todos los health checks registrados y devuelve un mapa
// indexado por nombre de componente con su resultado.
//
// Toma una instantanea de los checks bajo lock de lectura y los ejecuta fuera
// del lock, para no bloquear Register mientras corren chequeos potencialmente
// lentos.
func (c *Checker) CheckAll(ctx context.Context) map[string]CheckResult {
	c.mu.RLock()
	checks := make([]HealthCheck, len(c.checks))
	copy(checks, c.checks)
	c.mu.RUnlock()

	results := make(map[string]CheckResult, len(checks))
	for _, check := range checks {
		results[check.Name()] = check.Check(ctx)
	}
	return results
}

// IsHealthy devuelve true si ningun check registrado esta unhealthy.
// Un estado degraded no se considera fallo de salud.
func (c *Checker) IsHealthy(ctx context.Context) bool {
	for _, result := range c.CheckAll(ctx) {
		if result.Status == StatusUnhealthy {
			return false
		}
	}
	return true
}

// IsReady devuelve true si el conjunto esta listo para recibir trafico
// (equivalente a IsHealthy, semantica de readiness).
func (c *Checker) IsReady(ctx context.Context) bool {
	return c.IsHealthy(ctx)
}

// IsLive devuelve true si la aplicacion esta viva (liveness basica): si puede
// responder a esta llamada, esta viva.
func (c *Checker) IsLive() bool {
	return true
}
