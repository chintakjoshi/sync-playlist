package ratelimit

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type RateLimitedHTTPClient struct {
	client      *http.Client
	rateLimiter *RateLimiter
	service     ServiceType
	maxRetries  int
}

func NewRateLimitedHTTPClient(service ServiceType, rateLimiter *RateLimiter) *RateLimitedHTTPClient {
	return &RateLimitedHTTPClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		rateLimiter: rateLimiter,
		service:     service,
		maxRetries:  3,
	}
}

// Do executes an HTTP request with rate limiting and retry logic
func (c *RateLimitedHTTPClient) Do(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		// Wait for rate limit
		if err := c.rateLimiter.Wait(c.service); err != nil {
			return nil, fmt.Errorf("rate limit error: %v", err)
		}

		// Execute request
		resp, err = c.client.Do(req)
		if err != nil {
			log.Printf("HTTP request error (attempt %d/%d): %v", attempt+1, c.maxRetries+1, err)
			if attempt == c.maxRetries {
				return nil, err
			}
			time.Sleep(time.Duration(attempt+1) * time.Second)
			continue
		}

		// Check for rate limit headers
		if c.isRateLimited(resp) {
			c.handleRateLimitResponse(resp, attempt)
			if attempt == c.maxRetries {
				resp.Body.Close()
				return nil, fmt.Errorf("rate limited after %d retries", c.maxRetries)
			}
			continue
		}

		// Success
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, nil
		}

		// Handle other errors
		if attempt == c.maxRetries {
			return resp, nil // Return the error response
		}

		// For server errors, retry with backoff
		if resp.StatusCode >= 500 {
			log.Printf("Server error %d (attempt %d/%d)", resp.StatusCode, attempt+1, c.maxRetries+1)
			resp.Body.Close()
			time.Sleep(time.Duration(attempt+1) * time.Second)
			continue
		}

		// For client errors, don't retry
		return resp, nil
	}

	return resp, err
}

// isRateLimited checks if the response indicates rate limiting
func (c *RateLimitedHTTPClient) isRateLimited(resp *http.Response) bool {
	return resp.StatusCode == http.StatusTooManyRequests ||
		resp.StatusCode == 429 ||
		resp.Header.Get("X-RateLimit-Remaining") == "0"
}

// handleRateLimitResponse handles rate limit responses with proper backoff
func (c *RateLimitedHTTPClient) handleRateLimitResponse(resp *http.Response, attempt int) {
	resp.Body.Close()

	retryAfter := c.getRetryAfter(resp)
	if retryAfter > 0 {
		log.Printf("Rate limited for %s. Retrying after %v (attempt %d/%d)",
			c.service, retryAfter, attempt+1, c.maxRetries+1)
		time.Sleep(retryAfter)
	} else {
		// Exponential backoff
		backoff := time.Duration(attempt+1) * 5 * time.Second
		log.Printf("Rate limited for %s. Retrying after %v (attempt %d/%d)",
			c.service, backoff, attempt+1, c.maxRetries+1)
		time.Sleep(backoff)
	}
}

// getRetryAfter extracts Retry-After header from response
func (c *RateLimitedHTTPClient) getRetryAfter(resp *http.Response) time.Duration {
	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter == "" {
		return 0
	}

	// Try to parse as seconds
	if seconds, err := time.ParseDuration(retryAfter + "s"); err == nil {
		return seconds
	}

	// Try to parse as RFC1123 date
	if retryTime, err := time.Parse(time.RFC1123, retryAfter); err == nil {
		return time.Until(retryTime)
	}

	return 0
}

// Get makes a GET request with rate limiting
func (c *RateLimitedHTTPClient) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Post makes a POST request with rate limiting
func (c *RateLimitedHTTPClient) Post(url, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return c.Do(req)
}
