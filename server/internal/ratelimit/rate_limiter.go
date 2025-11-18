package ratelimit

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type ServiceType string

const (
	SpotifyService ServiceType = "spotify"
	YouTubeService ServiceType = "youtube"
)

// Rate limits based on official API documentation
var serviceLimits = map[ServiceType]struct {
	requestsPerSecond int
	burst             int
}{
	SpotifyService: {requestsPerSecond: 10, burst: 20}, // Spotify: 10 req/sec, burst to 20
	YouTubeService: {requestsPerSecond: 1, burst: 5},   // YouTube: 1 req/sec, burst to 5 (conservative)
}

type RateLimiter struct {
	limiters map[ServiceType]*rate.Limiter
	mutex    sync.RWMutex
}

func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		limiters: make(map[ServiceType]*rate.Limiter),
	}

	// Initialize limiters for each service
	for serviceType, limits := range serviceLimits {
		rl.limiters[serviceType] = rate.NewLimiter(
			rate.Limit(limits.requestsPerSecond),
			limits.burst,
		)
	}

	return rl
}

// Wait blocks until the request is allowed for the service
func (rl *RateLimiter) Wait(service ServiceType) error {
	rl.mutex.RLock()
	limiter, exists := rl.limiters[service]
	rl.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("no rate limiter configured for service: %s", service)
	}

	// Use context with timeout to avoid infinite waiting
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := limiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limit wait timeout for %s: %v", service, err)
	}

	return nil
}

// Allow checks if a request is allowed without waiting
func (rl *RateLimiter) Allow(service ServiceType) bool {
	rl.mutex.RLock()
	limiter, exists := rl.limiters[service]
	rl.mutex.RUnlock()

	if !exists {
		return false
	}

	return limiter.Allow()
}

// GetLimiterStats returns current rate limiter statistics
func (rl *RateLimiter) GetLimiterStats(service ServiceType) map[string]interface{} {
	rl.mutex.RLock()
	limiter, exists := rl.limiters[service]
	rl.mutex.RUnlock()

	if !exists {
		return nil
	}

	// Note: rate.Limiter doesn't expose internal stats directly
	// We can track our own metrics
	return map[string]interface{}{
		"service":        service,
		"limit":          serviceLimits[service].requestsPerSecond,
		"burst":          serviceLimits[service].burst,
		"current_tokens": limiter.Tokens(),
	}
}

// SetCustomLimit allows dynamic adjustment of rate limits
func (rl *RateLimiter) SetCustomLimit(service ServiceType, requestsPerSecond int, burst int) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	rl.limiters[service] = rate.NewLimiter(
		rate.Limit(requestsPerSecond),
		burst,
	)

	log.Printf("Updated rate limit for %s: %d req/sec, burst %d",
		service, requestsPerSecond, burst)
}
