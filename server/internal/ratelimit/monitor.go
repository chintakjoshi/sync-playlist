package ratelimit

import (
	"log"
	"sync"
	"time"
)

type RequestMetrics struct {
	TotalRequests   int64
	RateLimited     int64
	Errors          int64
	LastRequestTime time.Time
	mu              sync.RWMutex
}

type RateLimitMonitor struct {
	metrics     map[ServiceType]*RequestMetrics
	rateLimiter *RateLimiter
	mu          sync.RWMutex
}

func NewRateLimitMonitor(rateLimiter *RateLimiter) *RateLimitMonitor {
	return &RateLimitMonitor{
		metrics:     make(map[ServiceType]*RequestMetrics),
		rateLimiter: rateLimiter,
	}
}

// RecordRequest records a request attempt
func (m *RateLimitMonitor) RecordRequest(service ServiceType, wasRateLimited bool, hadError bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.metrics[service]; !exists {
		m.metrics[service] = &RequestMetrics{}
	}

	metrics := m.metrics[service]
	metrics.mu.Lock()
	defer metrics.mu.Unlock()

	metrics.TotalRequests++
	if wasRateLimited {
		metrics.RateLimited++
	}
	if hadError {
		metrics.Errors++
	}
	metrics.LastRequestTime = time.Now()
}

// GetMetrics returns current metrics for all services
func (m *RateLimitMonitor) GetMetrics() map[ServiceType]map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[ServiceType]map[string]interface{})
	for service, metrics := range m.metrics {
		metrics.mu.RLock()
		result[service] = map[string]interface{}{
			"total_requests":    metrics.TotalRequests,
			"rate_limited":      metrics.RateLimited,
			"errors":            metrics.Errors,
			"last_request_time": metrics.LastRequestTime,
			"rate_limit_stats":  m.rateLimiter.GetLimiterStats(service),
		}
		metrics.mu.RUnlock()
	}

	return result
}

// StartMonitoring starts periodic monitoring and logging
func (m *RateLimitMonitor) StartMonitoring() {
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for range ticker.C {
			m.logMetrics()
		}
	}()
}

func (m *RateLimitMonitor) logMetrics() {
	metrics := m.GetMetrics()
	for service, stats := range metrics {
		log.Printf("[RATE LIMIT MONITOR] %s: %+v", service, stats)
	}
}
