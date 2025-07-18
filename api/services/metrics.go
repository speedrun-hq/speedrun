package services

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/speedrun-hq/speedrun/api/logger"
)

// MetricsService handles Prometheus metrics collection and exposition
type MetricsService struct {
	// Prometheus metrics
	intentServicesUp         *prometheus.GaugeVec
	activeGoroutines         *prometheus.GaugeVec
	subscriptionCount        *prometheus.GaugeVec
	eventsProcessedTotal     *prometheus.GaugeVec
	eventsSkippedTotal       *prometheus.GaugeVec
	processingErrorsTotal    *prometheus.GaugeVec
	reconnectionCount        *prometheus.GaugeVec
	lastEventTimestamp       *prometheus.GaugeVec
	timeSinceLastEvent       *prometheus.GaugeVec
	lastHealthCheckTimestamp *prometheus.GaugeVec

	// Service references
	intentServices      map[uint64]*IntentService
	fulfillmentServices map[uint64]*FulfillmentService
	settlementServices  map[uint64]*SettlementService
	mu                  sync.RWMutex
	logger              logger.Logger
	registry            *prometheus.Registry
}

// NewMetricsService creates a new metrics service
func NewMetricsService(logger logger.Logger) *MetricsService {
	registry := prometheus.NewRegistry()

	// Create metrics
	intentServicesUp := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "speedrun_intent_services_up",
			Help: "Whether intent services are healthy (1 = healthy, 0 = unhealthy)",
		},
		[]string{"chain_id", "chain_name"},
	)

	activeGoroutines := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "speedrun_active_goroutines",
			Help: "Number of active goroutines per chain",
		},
		[]string{"chain_id", "chain_name"},
	)

	subscriptionCount := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "speedrun_subscriptions_active",
			Help: "Number of active blockchain subscriptions per chain",
		},
		[]string{"chain_id", "chain_name"},
	)

	eventsProcessedTotal := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "speedrun_events_processed_total",
			Help: "Total number of events processed per chain",
		},
		[]string{"chain_id", "chain_name"},
	)

	eventsSkippedTotal := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "speedrun_events_skipped_total",
			Help: "Total number of events skipped (duplicates) per chain",
		},
		[]string{"chain_id", "chain_name"},
	)

	processingErrorsTotal := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "speedrun_processing_errors_total",
			Help: "Total number of processing errors per chain",
		},
		[]string{"chain_id", "chain_name"},
	)

	reconnectionCount := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "speedrun_reconnections_total",
			Help: "Total number of reconnections per chain",
		},
		[]string{"chain_id", "chain_name"},
	)

	lastEventTimestamp := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "speedrun_last_event_timestamp",
			Help: "Timestamp of the last processed event per chain",
		},
		[]string{"chain_id", "chain_name"},
	)

	timeSinceLastEvent := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "speedrun_time_since_last_event_seconds",
			Help: "Time in seconds since the last processed event per chain",
		},
		[]string{"chain_id", "chain_name"},
	)

	lastHealthCheckTimestamp := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "speedrun_last_health_check_timestamp",
			Help: "Timestamp of the last health check per chain",
		},
		[]string{"chain_id", "chain_name"},
	)

	// Register metrics
	registry.MustRegister(intentServicesUp)
	registry.MustRegister(activeGoroutines)
	registry.MustRegister(subscriptionCount)
	registry.MustRegister(eventsProcessedTotal)
	registry.MustRegister(eventsSkippedTotal)
	registry.MustRegister(processingErrorsTotal)
	registry.MustRegister(reconnectionCount)
	registry.MustRegister(lastEventTimestamp)
	registry.MustRegister(timeSinceLastEvent)
	registry.MustRegister(lastHealthCheckTimestamp)

	return &MetricsService{
		intentServicesUp:         intentServicesUp,
		activeGoroutines:         activeGoroutines,
		subscriptionCount:        subscriptionCount,
		eventsProcessedTotal:     eventsProcessedTotal,
		eventsSkippedTotal:       eventsSkippedTotal,
		processingErrorsTotal:    processingErrorsTotal,
		reconnectionCount:        reconnectionCount,
		lastEventTimestamp:       lastEventTimestamp,
		timeSinceLastEvent:       timeSinceLastEvent,
		lastHealthCheckTimestamp: lastHealthCheckTimestamp,
		intentServices:           make(map[uint64]*IntentService),
		fulfillmentServices:      make(map[uint64]*FulfillmentService),
		settlementServices:       make(map[uint64]*SettlementService),
		logger:                   logger,
		registry:                 registry,
	}
}

// RegisterIntentService registers an intent service for metrics collection
func (m *MetricsService) RegisterIntentService(chainID uint64, service *IntentService) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.intentServices[chainID] = service
	m.logger.Info("Registered intent service for chain %d in metrics collector", chainID)
}

// RegisterFulfillmentService registers a fulfillment service for metrics collection
func (m *MetricsService) RegisterFulfillmentService(chainID uint64, service *FulfillmentService) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.fulfillmentServices[chainID] = service
	m.logger.Info("Registered fulfillment service for chain %d in metrics collector", chainID)
}

// RegisterSettlementService registers a settlement service for metrics collection
func (m *MetricsService) RegisterSettlementService(chainID uint64, service *SettlementService) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.settlementServices[chainID] = service
	m.logger.Info("Registered settlement service for chain %d in metrics collector", chainID)
}

// UnregisterIntentService removes an intent service from metrics collection
func (m *MetricsService) UnregisterIntentService(chainID uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.intentServices, chainID)
	m.logger.Info("Unregistered intent service for chain %d from metrics collector", chainID)
}

// GetChainName returns a human-readable chain name for metrics labels
func (m *MetricsService) GetChainName(chainID uint64) string {
	switch chainID {
	case 1:
		return "ethereum"
	case 42161:
		return "arbitrum"
	case 8453:
		return "base"
	case 137:
		return "polygon"
	case 56:
		return "bsc"
	case 43114:
		return "avalanche"
	case 7000:
		return "zetachain"
	default:
		return fmt.Sprintf("chain_%d", chainID)
	}
}

// UpdateMetrics collects and updates all metrics from registered intent services
func (m *MetricsService) UpdateMetrics() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now()

	// Create a map to track total goroutines per chain
	chainGoroutines := make(map[uint64]int32)

	// Collect goroutines from intent services
	for chainID, service := range m.intentServices {
		if service != nil {
			chainGoroutines[chainID] += service.ActiveGoroutines()
		}
	}

	// Collect goroutines from fulfillment services
	for chainID, service := range m.fulfillmentServices {
		if service != nil {
			chainGoroutines[chainID] += service.ActiveGoroutines()
		}
	}

	// Collect goroutines from settlement services
	for chainID, service := range m.settlementServices {
		if service != nil {
			chainGoroutines[chainID] += service.ActiveGoroutines()
		}
	}

	// Update metrics for each chain
	for chainID, totalGoroutines := range chainGoroutines {
		chainName := m.GetChainName(chainID)
		chainIDStr := fmt.Sprintf("%d", chainID)

		// Update intent service metrics (for backward compatibility)
		if intentService, exists := m.intentServices[chainID]; exists && intentService != nil {
			metrics := intentService.GetMetrics()

			// Update gauge metrics
			if metrics.IsHealthy {
				m.intentServicesUp.WithLabelValues(chainIDStr, chainName).Set(1)
			} else {
				m.intentServicesUp.WithLabelValues(chainIDStr, chainName).Set(0)
			}

			// For ZetaChain, subscription count is always 0 since it uses HTTP polling
			// For other chains, report actual subscription count
			if metrics.IsZetaChain {
				// ZetaChain uses polling, so report 1 if polling is healthy, 0 if not
				if metrics.PollingHealthy {
					m.subscriptionCount.WithLabelValues(chainIDStr, chainName).Set(1)
				} else {
					m.subscriptionCount.WithLabelValues(chainIDStr, chainName).Set(0)
				}
			} else {
				m.subscriptionCount.WithLabelValues(chainIDStr, chainName).Set(float64(metrics.SubscriptionCount))
			}

			// Update counter metrics - we need to track the current values and set them
			// Note: These are counters that reset on service restart, so we use gauges to track current values
			m.eventsProcessedTotal.WithLabelValues(chainIDStr, chainName).Set(float64(metrics.EventsProcessed))
			m.eventsSkippedTotal.WithLabelValues(chainIDStr, chainName).Set(float64(metrics.EventsSkipped))
			m.processingErrorsTotal.WithLabelValues(chainIDStr, chainName).Set(float64(metrics.ProcessingErrors))
			m.reconnectionCount.WithLabelValues(chainIDStr, chainName).Set(float64(metrics.ReconnectionCount))

			// Update timestamp metrics
			if !metrics.LastEventTime.IsZero() {
				m.lastEventTimestamp.WithLabelValues(chainIDStr, chainName).Set(float64(metrics.LastEventTime.Unix()))
				timeSinceLastEvent := now.Sub(metrics.LastEventTime).Seconds()
				m.timeSinceLastEvent.WithLabelValues(chainIDStr, chainName).Set(timeSinceLastEvent)
			}

			if !metrics.LastHealthCheck.IsZero() {
				m.lastHealthCheckTimestamp.WithLabelValues(chainIDStr, chainName).Set(float64(metrics.LastHealthCheck.Unix()))
			}
		}

		// Update total goroutines metric (sum of all services for this chain)
		m.activeGoroutines.WithLabelValues(chainIDStr, chainName).Set(float64(totalGoroutines))
	}
}

// StartMetricsUpdater starts a goroutine that periodically updates metrics
func (m *MetricsService) StartMetricsUpdater(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(15 * time.Second) // Update every 15 seconds
		defer ticker.Stop()

		m.logger.Info("Started Prometheus metrics updater")

		for {
			select {
			case <-ticker.C:
				m.UpdateMetrics()
			case <-ctx.Done():
				m.logger.Info("Stopped Prometheus metrics updater")
				return
			}
		}
	}()
}

// GetHandler returns the Prometheus metrics HTTP handler
func (m *MetricsService) GetHandler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

// GetMetricsSummary returns a summary of all metrics for debugging
func (m *MetricsService) GetMetricsSummary() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	summary := make(map[string]interface{})
	chainMetrics := make(map[string]interface{})

	// Create a map to track total goroutines per chain
	chainGoroutines := make(map[uint64]int32)

	// Collect goroutines from intent services
	for chainID, service := range m.intentServices {
		if service != nil {
			chainGoroutines[chainID] += service.ActiveGoroutines()
		}
	}

	// Collect goroutines from fulfillment services
	for chainID, service := range m.fulfillmentServices {
		if service != nil {
			chainGoroutines[chainID] += service.ActiveGoroutines()
		}
	}

	// Collect goroutines from settlement services
	for chainID, service := range m.settlementServices {
		if service != nil {
			chainGoroutines[chainID] += service.ActiveGoroutines()
		}
	}

	// Build metrics summary for each chain
	for chainID, totalGoroutines := range chainGoroutines {
		chainName := m.GetChainName(chainID)

		// Get intent service metrics for backward compatibility
		var intentMetrics *ServiceMetrics
		if intentService, exists := m.intentServices[chainID]; exists && intentService != nil {
			metrics := intentService.GetMetrics()
			intentMetrics = &metrics
		}

		chainMetrics[chainName] = map[string]interface{}{
			"chain_id":          chainID,
			"active_goroutines": totalGoroutines, // Total from all services
			"intent_goroutines": func() int32 {
				if intentMetrics != nil {
					return intentMetrics.ActiveGoroutines
				}
				return 0
			}(),
			"fulfillment_goroutines": func() int32 {
				if service, exists := m.fulfillmentServices[chainID]; exists && service != nil {
					return service.ActiveGoroutines()
				}
				return 0
			}(),
			"settlement_goroutines": func() int32 {
				if service, exists := m.settlementServices[chainID]; exists && service != nil {
					return service.ActiveGoroutines()
				}
				return 0
			}(),
		}

		// Add intent service specific metrics if available
		if intentMetrics != nil {
			chainMetrics[chainName].(map[string]interface{})["is_healthy"] = intentMetrics.IsHealthy
			chainMetrics[chainName].(map[string]interface{})["subscription_count"] = intentMetrics.SubscriptionCount
			chainMetrics[chainName].(map[string]interface{})["events_processed"] = intentMetrics.EventsProcessed
			chainMetrics[chainName].(map[string]interface{})["events_skipped"] = intentMetrics.EventsSkipped
			chainMetrics[chainName].(map[string]interface{})["processing_errors"] = intentMetrics.ProcessingErrors
			chainMetrics[chainName].(map[string]interface{})["reconnection_count"] = intentMetrics.ReconnectionCount
			chainMetrics[chainName].(map[string]interface{})["last_event_time"] = intentMetrics.LastEventTime
			chainMetrics[chainName].(map[string]interface{})["last_health_check"] = intentMetrics.LastHealthCheck
			chainMetrics[chainName].(map[string]interface{})["time_since_last_event"] = intentMetrics.TimeSinceLastEvent
		}
	}

	summary["chains"] = chainMetrics
	summary["total_chains"] = len(chainGoroutines)
	summary["timestamp"] = time.Now()

	return summary
}
