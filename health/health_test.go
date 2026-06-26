package health_test

import (
	"context"
	"testing"
	"time"

	"github.com/EduGoGroup/wapp-shared/health"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubCheck es un HealthCheck de prueba con nombre y resultado fijos.
type stubCheck struct {
	name   string
	status health.Status
}

func (s stubCheck) Name() string { return s.name }

func (s stubCheck) Check(_ context.Context) health.CheckResult {
	return health.CheckResult{
		Status:    s.status,
		Component: s.name,
		Timestamp: time.Now(),
	}
}

func TestChecker_CheckAll_AggregatesResults(t *testing.T) {
	c := health.NewChecker()
	c.Register(stubCheck{name: "postgres", status: health.StatusHealthy})
	c.Register(stubCheck{name: "mongo", status: health.StatusUnhealthy})
	c.Register(stubCheck{name: "cache", status: health.StatusDegraded})

	results := c.CheckAll(context.Background())
	require.Len(t, results, 3)
	assert.Equal(t, health.StatusHealthy, results["postgres"].Status)
	assert.Equal(t, health.StatusUnhealthy, results["mongo"].Status)
	assert.Equal(t, health.StatusDegraded, results["cache"].Status)
	assert.Equal(t, "postgres", results["postgres"].Component)
}

func TestChecker_Register_IgnoresNil(t *testing.T) {
	c := health.NewChecker()
	c.Register(nil)
	results := c.CheckAll(context.Background())
	assert.Empty(t, results)
}

func TestChecker_IsHealthy_AllHealthy(t *testing.T) {
	c := health.NewChecker()
	c.Register(stubCheck{name: "a", status: health.StatusHealthy})
	c.Register(stubCheck{name: "b", status: health.StatusHealthy})
	assert.True(t, c.IsHealthy(context.Background()))
	assert.True(t, c.IsReady(context.Background()))
}

func TestChecker_IsHealthy_DegradedStillHealthy(t *testing.T) {
	c := health.NewChecker()
	c.Register(stubCheck{name: "a", status: health.StatusHealthy})
	c.Register(stubCheck{name: "b", status: health.StatusDegraded})
	// Un componente degraded no marca al conjunto como unhealthy.
	assert.True(t, c.IsHealthy(context.Background()))
}

func TestChecker_IsHealthy_OneUnhealthy(t *testing.T) {
	c := health.NewChecker()
	c.Register(stubCheck{name: "a", status: health.StatusHealthy})
	c.Register(stubCheck{name: "b", status: health.StatusUnhealthy})
	assert.False(t, c.IsHealthy(context.Background()))
	assert.False(t, c.IsReady(context.Background()))
}

func TestChecker_IsLive(t *testing.T) {
	c := health.NewChecker()
	assert.True(t, c.IsLive())
}

func TestChecker_Empty_IsHealthy(t *testing.T) {
	c := health.NewChecker()
	// Sin checks registrados no hay nada que falle.
	assert.True(t, c.IsHealthy(context.Background()))
}
