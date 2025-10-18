package keepalive

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Soypete/twitch-llm-bot/logging"
)

// ServiceConfig represents configuration for a service to monitor
type ServiceConfig struct {
	Name      string
	HealthURL string
}

// ServiceState tracks the state of a monitored service
type ServiceState struct {
	Name              string
	HealthURL         string
	LastCheckTime     time.Time
	LastAlertTime     time.Time
	ConsecutiveFailures int
	IsHealthy         bool
	mu                sync.RWMutex
}

// KeepAliveService monitors multiple services and alerts on failures
type KeepAliveService struct {
	services      map[string]*ServiceState
	checkInterval time.Duration
	alertInterval time.Duration
	httpClient    *http.Client
	alerter       Alerter
	logger        *logging.Logger
	mu            sync.RWMutex
}

// Alerter defines the interface for sending alerts
type Alerter interface {
	SendAlert(ctx context.Context, serviceName string, message string) error
}

// NewKeepAliveService creates a new keepalive service
func NewKeepAliveService(
	services []ServiceConfig,
	checkInterval time.Duration,
	alertInterval time.Duration,
	alerter Alerter,
	logger *logging.Logger,
) *KeepAliveService {
	kas := &KeepAliveService{
		services:      make(map[string]*ServiceState),
		checkInterval: checkInterval,
		alertInterval: alertInterval,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		alerter: alerter,
		logger:  logger,
	}

	for _, svc := range services {
		kas.services[svc.Name] = &ServiceState{
			Name:              svc.Name,
			HealthURL:         svc.HealthURL,
			LastCheckTime:     time.Time{},
			LastAlertTime:     time.Time{},
			ConsecutiveFailures: 0,
			IsHealthy:         true,
		}
	}

	return kas
}

// Start begins the monitoring loop
func (kas *KeepAliveService) Start(ctx context.Context) error {
	ticker := time.NewTicker(kas.checkInterval)
	defer ticker.Stop()

	// Do an initial check immediately
	kas.checkAllServices(ctx)

	for {
		select {
		case <-ctx.Done():
			kas.logger.Info("KeepAlive service shutting down")
			return ctx.Err()
		case <-ticker.C:
			kas.checkAllServices(ctx)
		}
	}
}

// checkAllServices checks all monitored services
func (kas *KeepAliveService) checkAllServices(ctx context.Context) {
	kas.mu.RLock()
	services := make([]*ServiceState, 0, len(kas.services))
	for _, svc := range kas.services {
		services = append(services, svc)
	}
	kas.mu.RUnlock()

	for _, svc := range services {
		kas.checkService(ctx, svc)
	}
}

// checkService checks a single service and handles alerting
func (kas *KeepAliveService) checkService(ctx context.Context, state *ServiceState) {
	state.mu.Lock()
	state.LastCheckTime = time.Now()
	state.mu.Unlock()

	healthy := kas.performHealthCheck(ctx, state.HealthURL)

	state.mu.Lock()
	defer state.mu.Unlock()

	if healthy {
		if !state.IsHealthy {
			// Service recovered
			kas.logger.Info("service recovered",
				"service", state.Name,
				"after_failures", state.ConsecutiveFailures)

			// Send recovery alert
			recoveryMsg := fmt.Sprintf("Service %s has recovered after %d failed checks",
				state.Name, state.ConsecutiveFailures)
			go func() {
				if err := kas.alerter.SendAlert(ctx, state.Name, recoveryMsg); err != nil {
					kas.logger.Error("failed to send recovery alert", "error", err.Error())
				}
			}()
		}
		state.IsHealthy = true
		state.ConsecutiveFailures = 0
	} else {
		state.ConsecutiveFailures++
		state.IsHealthy = false

		kas.logger.Warn("service health check failed",
			"service", state.Name,
			"consecutive_failures", state.ConsecutiveFailures,
			"url", state.HealthURL)

		// Alert after 3 consecutive failures
		if state.ConsecutiveFailures == 3 {
			msg := fmt.Sprintf("Service %s is offline after 3 failed health checks", state.Name)
			go func() {
				if err := kas.alerter.SendAlert(ctx, state.Name, msg); err != nil {
					kas.logger.Error("failed to send initial alert", "error", err.Error())
				}
			}()
			state.LastAlertTime = time.Now()
		} else if state.ConsecutiveFailures > 3 {
			// Alert once per hour after the initial alert
			if time.Since(state.LastAlertTime) >= kas.alertInterval {
				msg := fmt.Sprintf("Service %s is still offline (consecutive failures: %d)",
					state.Name, state.ConsecutiveFailures)
				go func() {
					if err := kas.alerter.SendAlert(ctx, state.Name, msg); err != nil {
						kas.logger.Error("failed to send repeat alert", "error", err.Error())
					}
				}()
				state.LastAlertTime = time.Now()
			}
		}
	}
}

// performHealthCheck performs the actual HTTP health check with exponential backoff
func (kas *KeepAliveService) performHealthCheck(ctx context.Context, url string) bool {
	backoffDuration := 1 * time.Second
	maxRetries := 3

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			time.Sleep(backoffDuration)
			backoffDuration *= 2
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			kas.logger.Error("failed to create health check request",
				"error", err.Error(),
				"url", url,
				"attempt", attempt+1)
			continue
		}

		resp, err := kas.httpClient.Do(req)
		if err != nil {
			kas.logger.Debug("health check request failed",
				"error", err.Error(),
				"url", url,
				"attempt", attempt+1)
			continue
		}

		if err := resp.Body.Close(); err != nil {
			kas.logger.Debug("failed to close response body", "error", err.Error())
		}

		if resp.StatusCode == http.StatusOK {
			return true
		}

		kas.logger.Debug("health check returned non-OK status",
			"status", resp.StatusCode,
			"url", url,
			"attempt", attempt+1)
	}

	return false
}

// ServiceStateSnapshot is a snapshot of a service state without locks
type ServiceStateSnapshot struct {
	Name                string
	HealthURL           string
	LastCheckTime       time.Time
	LastAlertTime       time.Time
	ConsecutiveFailures int
	IsHealthy           bool
}

// GetServiceStates returns the current state of all services
func (kas *KeepAliveService) GetServiceStates() map[string]ServiceStateSnapshot {
	kas.mu.RLock()
	defer kas.mu.RUnlock()

	states := make(map[string]ServiceStateSnapshot)
	for name, svc := range kas.services {
		svc.mu.RLock()
		states[name] = ServiceStateSnapshot{
			Name:                svc.Name,
			HealthURL:           svc.HealthURL,
			LastCheckTime:       svc.LastCheckTime,
			LastAlertTime:       svc.LastAlertTime,
			ConsecutiveFailures: svc.ConsecutiveFailures,
			IsHealthy:           svc.IsHealthy,
		}
		svc.mu.RUnlock()
	}
	return states
}
