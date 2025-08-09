package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type Client struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
	mu         sync.RWMutex

	// Circuit breaker state
	failureCount    int
	lastFailureTime time.Time
	circuitOpen     bool

	// Configuration
	maxFailures    int
	resetTimeout   time.Duration
	requestTimeout time.Duration
}

func NewClient(baseURL, username, password string, timeout time.Duration) *Client {
	if baseURL == "" {
		baseURL = "http://localhost:15672"
	}
	if username == "" {
		username = "guest"
	}
	if password == "" {
		password = "guest"
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 50,
		IdleConnTimeout:     90 * time.Second,
		MaxConnsPerHost:     100,

		DisableCompression: true,
		DisableKeepAlives:  false,

		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	return &Client{
		baseURL:        baseURL,
		username:       username,
		password:       password,
		httpClient:     &http.Client{Timeout: timeout, Transport: transport},
		maxFailures:    5,
		resetTimeout:   60 * time.Second,
		requestTimeout: timeout,
	}
}

func (c *Client) isCircuitOpen() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.circuitOpen {
		return false
	}

	if time.Since(c.lastFailureTime) > c.resetTimeout {
		c.mu.RUnlock()
		c.mu.Lock()
		c.circuitOpen = false
		c.failureCount = 0
		c.mu.Unlock()
		c.mu.RLock()
		return false
	}

	return true
}

func (c *Client) recordFailure() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.failureCount++
	c.lastFailureTime = time.Now()

	if c.failureCount >= c.maxFailures {
		c.circuitOpen = true
	}
}

func (c *Client) recordSuccess() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.failureCount = 0
	c.circuitOpen = false
}

func (c *Client) GetQueues(ctx context.Context) ([]Queue, error) {
	if c.isCircuitOpen() {
		return nil, fmt.Errorf("circuit breaker is open - too many recent failures")
	}

	url := fmt.Sprintf("%s/api/queues", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		c.recordFailure()
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Connection", "keep-alive")

	var resp *http.Response
	var lastErr error

	for attempt := 0; attempt < 2; attempt++ {
		resp, err = c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed (attempt %d): %w", attempt+1, err)

			if attempt < 1 {
				backoff := time.Duration(attempt+1) * 500 * time.Millisecond
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(backoff):
					continue
				}
			}
			continue
		}
		break
	}

	if resp == nil {
		c.recordFailure()
		return nil, lastErr
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		c.recordFailure()
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		c.recordFailure()
		var apiErr APIError
		if json.Unmarshal(body, &apiErr) == nil {
			return nil, &apiErr
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var queues []Queue
	if err := json.Unmarshal(body, &queues); err != nil {
		c.recordFailure()
		return nil, fmt.Errorf("failed to unmarshal queues: %w", err)
	}

	c.recordSuccess()
	return queues, nil
}

func (c *Client) HealthCheck(ctx context.Context) error {
	if c.isCircuitOpen() {
		return fmt.Errorf("circuit breaker is open - too many recent failures")
	}

	url := fmt.Sprintf("%s/api/overview", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		c.recordFailure()
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.recordFailure()
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.recordFailure()
		return fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}

	c.recordSuccess()
	return nil
}

func (c *Client) GetCircuitBreakerStatus() (bool, int, time.Time) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.circuitOpen, c.failureCount, c.lastFailureTime
}

func (c *Client) Close() {
	c.httpClient.CloseIdleConnections()
}
