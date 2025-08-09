package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	QueueMessages               *prometheus.GaugeVec
	QueueMessagesReady          *prometheus.GaugeVec
	QueueMessagesUnacknowledged *prometheus.GaugeVec

	QueueMessagePublishRate   *prometheus.GaugeVec
	QueueMessageDeliverRate   *prometheus.GaugeVec
	QueueMessageAckRate       *prometheus.GaugeVec
	QueueMessageRedeliverRate *prometheus.GaugeVec

	QueueConsumers           *prometheus.GaugeVec
	QueueConsumerUtilisation *prometheus.GaugeVec
	QueueConsumerCapacity    *prometheus.GaugeVec

	QueueState        *prometheus.GaugeVec
	QueueIsDeadLetter *prometheus.GaugeVec

	QueueHealthScore      *prometheus.GaugeVec
	QueueDepthAlert       *prometheus.GaugeVec
	QueueUtilizationAlert *prometheus.GaugeVec

	ScrapeDurationSeconds prometheus.Gauge
	ScrapeErrorsTotal     *prometheus.CounterVec

	CircuitBreakerState    *prometheus.GaugeVec
	CircuitBreakerFailures *prometheus.CounterVec
}

func NewMetrics() *Metrics {
	return &Metrics{
		// Queue message counts
		QueueMessages: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "rabbitmq_custom_queue_messages",
				Help: "Total number of messages in the queue",
			},
			[]string{"queue_name", "vhost", "state"},
		),
		QueueMessagesReady: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "rabbitmq_custom_queue_messages_ready",
				Help: "Number of messages ready to be delivered",
			},
			[]string{"queue_name", "vhost"},
		),
		QueueMessagesUnacknowledged: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "rabbitmq_custom_queue_messages_unacknowledged",
				Help: "Number of messages that have been delivered but not yet acknowledged",
			},
			[]string{"queue_name", "vhost"},
		),

		// Message rates (per second)
		QueueMessagePublishRate: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "rabbitmq_custom_queue_message_publish_rate",
				Help: "Message publish rate per second",
			},
			[]string{"queue_name", "vhost"},
		),
		QueueMessageDeliverRate: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "rabbitmq_custom_queue_message_deliver_rate",
				Help: "Message delivery rate per second",
			},
			[]string{"queue_name", "vhost"},
		),
		QueueMessageAckRate: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "rabbitmq_custom_queue_message_ack_rate",
				Help: "Message acknowledgment rate per second",
			},
			[]string{"queue_name", "vhost"},
		),
		QueueMessageRedeliverRate: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "rabbitmq_custom_queue_message_redeliver_rate",
				Help: "Message redelivery rate per second",
			},
			[]string{"queue_name", "vhost"},
		),

		// Consumer metrics
		QueueConsumers: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "rabbitmq_custom_queue_consumers",
				Help: "Number of consumers connected to the queue",
			},
			[]string{"queue_name", "vhost"},
		),
		QueueConsumerUtilisation: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "rabbitmq_custom_queue_consumer_utilisation",
				Help: "Consumer utilisation as a percentage (0-1)",
			},
			[]string{"queue_name", "vhost"},
		),
		QueueConsumerCapacity: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "rabbitmq_custom_queue_consumer_capacity",
				Help: "Consumer capacity as a percentage (0-1)",
			},
			[]string{"queue_name", "vhost"},
		),

		// Queue state indicators
		QueueState: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "rabbitmq_custom_queue_state",
				Help: "Queue state indicator (1 for current state, 0 otherwise)",
			},
			[]string{"queue_name", "vhost", "state"},
		),
		QueueIsDeadLetter: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "rabbitmq_custom_queue_is_dead_letter",
				Help: "Indicates if the queue is a dead letter queue (1 if true, 0 if false)",
			},
			[]string{"queue_name", "vhost"},
		),

		// Queue health indicators
		QueueHealthScore: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "rabbitmq_custom_queue_health_score",
				Help: "Queue health score (0-100, higher is better)",
			},
			[]string{"queue_name", "vhost"},
		),
		QueueDepthAlert: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "rabbitmq_custom_queue_depth_alert",
				Help: "Queue depth alert indicator (1 if depth > threshold, 0 otherwise)",
			},
			[]string{"queue_name", "vhost", "severity"},
		),
		QueueUtilizationAlert: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "rabbitmq_custom_queue_utilization_alert",
				Help: "Queue utilization alert indicator (1 if utilization < threshold, 0 otherwise)",
			},
			[]string{"queue_name", "vhost", "severity"},
		),

		// Health metrics
		ScrapeDurationSeconds: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "rabbitmq_custom_scrape_duration_seconds",
				Help: "Duration of the last scrape in seconds",
			},
		),
		ScrapeErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "rabbitmq_custom_scrape_errors_total",
				Help: "Total number of scrape errors",
			},
			[]string{"error_type"},
		),

		// Circuit breaker metrics
		CircuitBreakerState: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "rabbitmq_custom_circuit_breaker_state",
				Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
			},
			[]string{"endpoint"},
		),
		CircuitBreakerFailures: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "rabbitmq_custom_circuit_breaker_failures_total",
				Help: "Total number of circuit breaker failures",
			},
			[]string{"endpoint"},
		),
	}
}

// GetAllCollectors returns all metrics as collectors for consistent iteration
func (m *Metrics) GetAllCollectors() []prometheus.Collector {
	return []prometheus.Collector{
		m.QueueMessages,
		m.QueueMessagesReady,
		m.QueueMessagesUnacknowledged,
		m.QueueMessagePublishRate,
		m.QueueMessageDeliverRate,
		m.QueueMessageAckRate,
		m.QueueMessageRedeliverRate,
		m.QueueConsumers,
		m.QueueConsumerUtilisation,
		m.QueueConsumerCapacity,
		m.QueueState,
		m.QueueIsDeadLetter,
		m.QueueHealthScore,
		m.QueueDepthAlert,
		m.QueueUtilizationAlert,
		m.ScrapeDurationSeconds,
		m.ScrapeErrorsTotal,
		m.CircuitBreakerState,
		m.CircuitBreakerFailures,
	}
}

// GetQueueCollectors returns only queue-related metrics for reset operations
func (m *Metrics) GetQueueCollectors() []prometheus.Collector {
	return []prometheus.Collector{
		m.QueueMessages,
		m.QueueMessagesReady,
		m.QueueMessagesUnacknowledged,
		m.QueueMessagePublishRate,
		m.QueueMessageDeliverRate,
		m.QueueMessageAckRate,
		m.QueueMessageRedeliverRate,
		m.QueueConsumers,
		m.QueueConsumerUtilisation,
		m.QueueConsumerCapacity,
		m.QueueState,
		m.QueueIsDeadLetter,
		m.QueueHealthScore,
		m.QueueDepthAlert,
		m.QueueUtilizationAlert,
	}
}

// ResetQueueMetrics resets all queue-related metrics to zero
func (m *Metrics) ResetQueueMetrics() {
	collectors := m.GetQueueCollectors()
	for _, collector := range collectors {
		if gaugeVec, ok := collector.(*prometheus.GaugeVec); ok {
			gaugeVec.Reset()
		}
	}
}
