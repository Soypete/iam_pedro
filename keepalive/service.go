package keepalive

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Soypete/twitch-llm-bot/logging"
	"golang.org/x/sync/errgroup"
)

// ServiceConfig represents configuration for a service to monitor
type ServiceConfig struct {
	Name         string
	HealthURL    string
	AuthHealthURL string // Optional: URL to check auth token health (e.g., /healthz/auth)
}

// AuthHealthResponse represents the response from /healthz/auth endpoint
type AuthHealthResponse struct {
	HasToken         bool      `json:"has_token"`
	LastRefreshTime  time.Time `json:"last_refresh_time"`
	ExpirationTime   time.Time `json:"expiration_time"`
	IsExpired        bool      `json:"is_expired"`
	HoursUntilExpiry float64   `json:"hours_until_expiry"`
}

// ServiceState tracks the state of a monitored service
type ServiceState struct {
	Name                string
	HealthURL           string
	AuthHealthURL       string
	LastCheckTime       time.Time
	LastAlertTime       time.Time
	LastAuthAlertTime   time.Time // Track when we last alerted about auth expiry
	ConsecutiveFailures int
	IsHealthy           bool
	AuthHealth          *AuthHealthResponse
	mu                  sync.RWMutex
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
			Name:                svc.Name,
			HealthURL:           svc.HealthURL,
			AuthHealthURL:       svc.AuthHealthURL,
			LastCheckTime:       time.Time{},
			LastAlertTime:       time.Time{},
			LastAuthAlertTime:   time.Time{},
			ConsecutiveFailures: 0,
			IsHealthy:           true,
			AuthHealth:          nil,
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

// checkAllServices checks all monitored services in parallel
func (kas *KeepAliveService) checkAllServices(ctx context.Context) {
	kas.mu.RLock()
	services := make([]*ServiceState, 0, len(kas.services))
	for _, svc := range kas.services {
		services = append(services, svc)
	}
	kas.mu.RUnlock()

	// Check all services in parallel using errgroup
	var eg errgroup.Group
	for _, svc := range services {
		svc := svc // capture loop variable
		eg.Go(func() error {
			kas.checkService(ctx, svc)
			return nil
		})
	}

	// Wait for all checks to complete
	// We ignore errors since checkService doesn't return errors,
	// it handles failures internally
	_ = eg.Wait()
}

// checkService checks a single service and handles alerting
func (kas *KeepAliveService) checkService(ctx context.Context, state *ServiceState) {
	state.mu.Lock()
	state.LastCheckTime = time.Now()
	state.mu.Unlock()

	healthy := kas.performHealthCheck(ctx, state.HealthURL)

	// If AuthHealthURL is configured, check auth token health
	if state.AuthHealthURL != "" {
		kas.checkAuthHealth(ctx, state)
	}

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

	attempt := 0
	for range 3 {
		// Exponential backoff: 1s, 2s, 4s
		time.Sleep(backoffDuration)
		backoffDuration *= 2

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

// checkAuthHealth checks the auth token health and alerts if expiring soon or expired
func (kas *KeepAliveService) checkAuthHealth(ctx context.Context, state *ServiceState) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, state.AuthHealthURL, nil)
	if err != nil {
		kas.logger.Error("failed to create auth health check request",
			"error", err.Error(),
			"url", state.AuthHealthURL)
		return
	}

	resp, err := kas.httpClient.Do(req)
	if err != nil {
		kas.logger.Debug("auth health check request failed",
			"error", err.Error(),
			"url", state.AuthHealthURL)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		kas.logger.Debug("auth health check returned non-OK status",
			"status", resp.StatusCode,
			"url", state.AuthHealthURL)
		return
	}

	var authHealth AuthHealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authHealth); err != nil {
		kas.logger.Error("failed to decode auth health response",
			"error", err.Error(),
			"url", state.AuthHealthURL)
		return
	}

	state.mu.Lock()
	state.AuthHealth = &authHealth
	state.mu.Unlock()

	// Alert if token is expired
	if authHealth.IsExpired {
		state.mu.Lock()
		// Only alert once per hour
		if time.Since(state.LastAuthAlertTime) >= kas.alertInterval {
			state.mu.Unlock()
			msg := fmt.Sprintf("⚠️ Auth token for %s has EXPIRED! Last refreshed: %s",
				state.Name,
				authHealth.LastRefreshTime.Format(time.RFC3339))
			go func() {
				if err := kas.alerter.SendAlert(ctx, state.Name, msg); err != nil {
					kas.logger.Error("failed to send auth expiry alert", "error", err.Error())
				}
			}()
			state.mu.Lock()
			state.LastAuthAlertTime = time.Now()
			state.mu.Unlock()
		} else {
			state.mu.Unlock()
		}
		return
	}

	// Alert if token will expire within 12 hours
	if authHealth.HoursUntilExpiry <= 12 && authHealth.HoursUntilExpiry > 0 {
		state.mu.Lock()
		// Only alert once per hour
		if time.Since(state.LastAuthAlertTime) >= kas.alertInterval {
			state.mu.Unlock()
			msg := fmt.Sprintf("⚠️ Auth token for %s will expire in %.1f hours (at %s). Please refresh the token.",
				state.Name,
				authHealth.HoursUntilExpiry,
				authHealth.ExpirationTime.Format(time.RFC3339))
			go func() {
				if err := kas.alerter.SendAlert(ctx, state.Name, msg); err != nil {
					kas.logger.Error("failed to send auth expiry warning", "error", err.Error())
				}
			}()
			state.mu.Lock()
			state.LastAuthAlertTime = time.Now()
			state.mu.Unlock()
		} else {
			state.mu.Unlock()
		}
	}
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
