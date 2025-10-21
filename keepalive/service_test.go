package keepalive

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Soypete/twitch-llm-bot/logging"
)

// mockAlerter implements the Alerter interface for testing
type mockAlerter struct {
	alerts []string
}

func (m *mockAlerter) SendAlert(ctx context.Context, serviceName string, message string) error {
	m.alerts = append(m.alerts, message)
	return nil
}

func TestKeepAliveService_HealthyService(t *testing.T) {
	// Create a test HTTP server that returns 200 OK
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	alerter := &mockAlerter{}
	logger := logging.NewLogger("error", nil)

	services := []ServiceConfig{
		{Name: "Test Service", HealthURL: server.URL},
	}

	kas := NewKeepAliveService(services, 100*time.Millisecond, 1*time.Second, alerter, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Run one check cycle
	kas.checkAllServices(ctx)

	// Verify service is healthy
	states := kas.GetServiceStates()
	if len(states) != 1 {
		t.Fatalf("expected 1 service, got %d", len(states))
	}

	state := states["Test Service"]
	if !state.IsHealthy {
		t.Error("expected service to be healthy")
	}
	if state.ConsecutiveFailures != 0 {
		t.Errorf("expected 0 consecutive failures, got %d", state.ConsecutiveFailures)
	}
	if len(alerter.alerts) != 0 {
		t.Errorf("expected no alerts, got %d", len(alerter.alerts))
	}
}

func TestKeepAliveService_FailingService(t *testing.T) {
	// Create a test HTTP server that returns 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	alerter := &mockAlerter{}
	logger := logging.NewLogger("error", nil)

	services := []ServiceConfig{
		{Name: "Failing Service", HealthURL: server.URL},
	}

	kas := NewKeepAliveService(services, 100*time.Millisecond, 1*time.Second, alerter, logger)
	ctx := context.Background()

	// Run check 3 times to trigger alert
	for i := 0; i < 3; i++ {
		kas.checkAllServices(ctx)
	}

	// Wait for goroutine alerts to complete
	time.Sleep(100 * time.Millisecond)

	// Verify service is unhealthy
	states := kas.GetServiceStates()
	state := states["Failing Service"]
	if state.IsHealthy {
		t.Error("expected service to be unhealthy")
	}
	if state.ConsecutiveFailures != 3 {
		t.Errorf("expected 3 consecutive failures, got %d", state.ConsecutiveFailures)
	}

	// Should have sent one alert after 3 failures
	if len(alerter.alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerter.alerts))
	}
	if alerter.alerts[0] != "Service Failing Service is offline after 3 failed health checks" {
		t.Errorf("unexpected alert message: %s", alerter.alerts[0])
	}
}

func TestKeepAliveService_ServiceRecovery(t *testing.T) {
	checkCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Each performHealthCheck has maxRetries=3, so each checkAllServices
		// will result in up to 3 requests. We want 3 check cycles to fail,
		// then the 4th to succeed.
		checkCount++
		if checkCount <= 9 { // 3 failed check cycles * 3 retries = 9 failed requests
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	alerter := &mockAlerter{}
	logger := logging.NewLogger("error", nil)

	services := []ServiceConfig{
		{Name: "Recovery Service", HealthURL: server.URL},
	}

	kas := NewKeepAliveService(services, 100*time.Millisecond, 1*time.Second, alerter, logger)
	ctx := context.Background()

	// Fail 3 times
	for i := 0; i < 3; i++ {
		kas.checkAllServices(ctx)
		time.Sleep(10 * time.Millisecond) // Small delay between checks
	}

	// Wait for failure alert goroutine
	time.Sleep(100 * time.Millisecond)

	// Then recover
	kas.checkAllServices(ctx)

	// Wait for recovery alert goroutine
	time.Sleep(100 * time.Millisecond)

	// Should have 2 alerts: failure and recovery
	if len(alerter.alerts) != 2 {
		t.Fatalf("expected 2 alerts (failure + recovery), got %d: %v", len(alerter.alerts), alerter.alerts)
	}

	states := kas.GetServiceStates()
	state := states["Recovery Service"]
	if !state.IsHealthy {
		t.Error("expected service to be healthy after recovery")
	}
	if state.ConsecutiveFailures != 0 {
		t.Errorf("expected 0 consecutive failures after recovery, got %d", state.ConsecutiveFailures)
	}
}

func TestKeepAliveService_ParallelChecks(t *testing.T) {
	// Create 3 slow servers that take 100ms each
	servers := make([]*httptest.Server, 3)
	for i := 0; i < 3; i++ {
		servers[i] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer servers[i].Close()
	}

	alerter := &mockAlerter{}
	logger := logging.NewLogger("error", nil)

	services := []ServiceConfig{
		{Name: "Service 1", HealthURL: servers[0].URL},
		{Name: "Service 2", HealthURL: servers[1].URL},
		{Name: "Service 3", HealthURL: servers[2].URL},
	}

	kas := NewKeepAliveService(services, 100*time.Millisecond, 1*time.Second, alerter, logger)
	ctx := context.Background()

	start := time.Now()
	kas.checkAllServices(ctx)
	elapsed := time.Since(start)

	// If parallel, should take ~100ms. If sequential, would take ~300ms
	// Allow some overhead, so check for < 200ms
	if elapsed > 200*time.Millisecond {
		t.Errorf("parallel checks took too long: %v (expected < 200ms)", elapsed)
	}

	// Verify all services are healthy
	states := kas.GetServiceStates()
	if len(states) != 3 {
		t.Fatalf("expected 3 services, got %d", len(states))
	}

	for name, state := range states {
		if !state.IsHealthy {
			t.Errorf("service %s should be healthy", name)
		}
	}
}
