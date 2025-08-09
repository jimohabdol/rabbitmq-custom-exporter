package main

import (
	"context"
	"log"
	"sync"
	"time"

	"rabbitmq-exporter/metrics"
	"rabbitmq-exporter/rabbitmq"

	"github.com/prometheus/client_golang/prometheus"
)

type Collector struct {
	client  *rabbitmq.Client
	metrics *metrics.Metrics
	scrapeInterval time.Duration
	lastScrape     time.Time

	mu              sync.RWMutex
	cachedQueues    []rabbitmq.Queue
	cacheTimestamp  time.Time
	cacheValid      bool
	collectionError error

	stopChan       chan struct{}
	collectionDone chan struct{}
}

func NewCollector(client *rabbitmq.Client, metrics *metrics.Metrics, scrapeInterval time.Duration) *Collector {
	c := &Collector{
		client:         client,
		metrics:        metrics,
		scrapeInterval: scrapeInterval,
		stopChan:       make(chan struct{}),
		collectionDone: make(chan struct{}),
	}

	go c.backgroundCollection()

	return c
}

func (c *Collector) backgroundCollection() {
	ticker := time.NewTicker(c.scrapeInterval)
	defer ticker.Stop()
	defer close(c.collectionDone)

	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			c.collectQueueData()
		}
	}
}

func (c *Collector) collectQueueData() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	queues, err := c.client.GetQueues(ctx)

	c.mu.Lock()
	defer c.mu.Unlock()

	if err != nil {
		c.collectionError = err
		c.cacheValid = false
		if time.Since(c.lastScrape) > time.Minute {
			log.Printf("Background collection error: %v", err)
		}
		return
	}

	c.cachedQueues = queues
	c.cacheTimestamp = time.Now()
	c.cacheValid = true
	c.collectionError = nil
	c.lastScrape = time.Now()

	c.updateCircuitBreakerMetrics()
}

func (c *Collector) updateCircuitBreakerMetrics() {
	isOpen, failureCount, _ := c.client.GetCircuitBreakerStatus()

	state := 0.0
	if isOpen {
		state = 1.0
	}
	c.metrics.CircuitBreakerState.WithLabelValues("rabbitmq_api").Set(state)

	c.metrics.CircuitBreakerFailures.WithLabelValues("rabbitmq_api").Add(float64(failureCount))
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	collectors := c.metrics.GetAllCollectors()
	for _, collector := range collectors {
		collector.Describe(ch)
	}
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	start := time.Now()

	c.metrics.ResetQueueMetrics()

	c.mu.RLock()
	queues := c.cachedQueues
	cacheValid := c.cacheValid
	collectionError := c.collectionError
	c.mu.RUnlock()

	if !cacheValid || time.Since(c.cacheTimestamp) > c.scrapeInterval*2 {
		if collectionError != nil {
			c.metrics.ScrapeErrorsTotal.WithLabelValues("api_error").Inc()
		}
		c.metrics.ScrapeDurationSeconds.Set(time.Since(start).Seconds())
		c.collectMetrics(ch)
		return
	}

	for _, queue := range queues {
		c.updateQueueMetrics(queue)
	}

	c.metrics.ScrapeDurationSeconds.Set(time.Since(start).Seconds())
	c.collectMetrics(ch)
}

func (c *Collector) updateQueueMetrics(queue rabbitmq.Queue) {
	state := queue.GetQueueState()
	stateStr := string(state)
	labels := []string{queue.Name, queue.Vhost}
	labelsWithState := append(labels, stateStr)

	c.metrics.QueueMessages.WithLabelValues(labelsWithState...).Set(float64(queue.Messages))
	c.metrics.QueueMessagesReady.WithLabelValues(labels...).Set(float64(queue.MessagesReady))
	c.metrics.QueueMessagesUnacknowledged.WithLabelValues(labels...).Set(float64(queue.MessagesUnacknowledged))


	c.metrics.QueueMessagePublishRate.WithLabelValues(labels...).Set(queue.GetPublishRate())
	c.metrics.QueueMessageDeliverRate.WithLabelValues(labels...).Set(queue.GetDeliverRate())
	c.metrics.QueueMessageAckRate.WithLabelValues(labels...).Set(queue.GetAckRate())
	c.metrics.QueueMessageRedeliverRate.WithLabelValues(labels...).Set(queue.GetRedeliverRate())

	c.metrics.QueueConsumers.WithLabelValues(labels...).Set(float64(queue.Consumers))
	c.metrics.QueueConsumerUtilisation.WithLabelValues(labels...).Set(queue.ConsumerUtilisation)
	c.metrics.QueueConsumerCapacity.WithLabelValues(labels...).Set(queue.ConsumerUtilisation) // Capacity is same as utilization for now

	states := []string{"idle", "active", "blocked"}
	for _, s := range states {
		value := 0.0
		if s == stateStr {
			value = 1.0
		}
		stateLabels := append(labels, s)
		c.metrics.QueueState.WithLabelValues(stateLabels...).Set(value)
	}

	dlqValue := 0.0
	if queue.IsDeadLetterQueue() {
		dlqValue = 1.0
	}
	c.metrics.QueueIsDeadLetter.WithLabelValues(labels...).Set(dlqValue)

	c.calculateHealthMetrics(queue, labels)
}

func (c *Collector) calculateHealthMetrics(queue rabbitmq.Queue, labels []string) {
	healthScore := 100.0

	if queue.Messages > 1000 {
		healthScore -= 20
	}
	if queue.Messages > 10000 {
		healthScore -= 30
	}

	if queue.ConsumerUtilisation < 0.1 {
		healthScore -= 25
	}
	if queue.ConsumerUtilisation < 0.01 {
		healthScore -= 40
	}

	redeliverRate := queue.GetRedeliverRate()
	if redeliverRate > 1.0 {
		healthScore -= 15
	}
	if redeliverRate > 5.0 {
		healthScore -= 25
	}

	if healthScore < 0 {
		healthScore = 0
	}

	c.metrics.QueueHealthScore.WithLabelValues(labels...).Set(healthScore)

	if queue.Messages > 1000 {
		c.metrics.QueueDepthAlert.WithLabelValues(append(labels, "warning")...).Set(1.0)
	} else {
		c.metrics.QueueDepthAlert.WithLabelValues(append(labels, "warning")...).Set(0.0)
	}
	if queue.Messages > 10000 {
		c.metrics.QueueDepthAlert.WithLabelValues(append(labels, "critical")...).Set(1.0)
	} else {
		c.metrics.QueueDepthAlert.WithLabelValues(append(labels, "critical")...).Set(0.0)
	}

	if queue.ConsumerUtilisation < 0.1 {
		c.metrics.QueueUtilizationAlert.WithLabelValues(append(labels, "warning")...).Set(1.0)
	} else {
		c.metrics.QueueUtilizationAlert.WithLabelValues(append(labels, "warning")...).Set(0.0)
	}
	if queue.ConsumerUtilisation < 0.01 {
		c.metrics.QueueUtilizationAlert.WithLabelValues(append(labels, "critical")...).Set(1.0)
	} else {
		c.metrics.QueueUtilizationAlert.WithLabelValues(append(labels, "critical")...).Set(0.0)
	}
}

func (c *Collector) collectMetrics(ch chan<- prometheus.Metric) {
	collectors := c.metrics.GetAllCollectors()
	for _, collector := range collectors {
		collector.Collect(ch)
	}
}

func (c *Collector) Stop() {
	close(c.stopChan)
	<-c.collectionDone
}
