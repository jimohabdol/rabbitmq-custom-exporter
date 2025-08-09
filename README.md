# RabbitMQ Custom Prometheus Exporter

A high-performance, production-ready RabbitMQ Prometheus exporter written in Go that provides granular queue-level metrics with zero impact on RabbitMQ performance.

## 🚀 Features

- **Zero Impact**: Asynchronous collection with circuit breaker protection
- **Queue-Level Metrics**: Detailed metrics for every queue with state detection
- **Production Ready**: Connection pooling, graceful shutdown, comprehensive error handling
- **Health Monitoring**: Built-in health scoring and alert indicators
- **High Performance**: Sub-millisecond Prometheus scrape times

## 📊 Metrics

### Queue Metrics
- `rabbitmq_custom_queue_messages` - Total messages in queue (with state labels)
- `rabbitmq_custom_queue_messages_ready` - Messages ready for delivery
- `rabbitmq_custom_queue_messages_unacknowledged` - Unacknowledged messages
- `rabbitmq_custom_queue_message_publish_rate` - Message publish rate per second
- `rabbitmq_custom_queue_message_deliver_rate` - Message delivery rate per second
- `rabbitmq_custom_queue_message_ack_rate` - Message acknowledgment rate per second
- `rabbitmq_custom_queue_message_redeliver_rate` - Message redelivery rate per second

### Consumer Metrics
- `rabbitmq_custom_queue_consumers` - Number of consumers
- `rabbitmq_custom_queue_consumer_utilisation` - Consumer utilization percentage
- `rabbitmq_custom_queue_consumer_capacity` - Consumer capacity percentage

### Queue State & Health
- `rabbitmq_custom_queue_state` - Queue state indicators (idle/active/blocked)
- `rabbitmq_custom_queue_is_dead_letter` - Dead letter queue indicator
- `rabbitmq_custom_queue_health_score` - Queue health score (0-100)
- `rabbitmq_custom_queue_depth_alert` - Queue depth alerts (warning/critical)
- `rabbitmq_custom_queue_utilization_alert` - Utilization alerts (warning/critical)

### System Metrics
- `rabbitmq_custom_scrape_duration_seconds` - Scrape duration
- `rabbitmq_custom_scrape_errors_total` - Error counters
- `rabbitmq_custom_circuit_breaker_state` - Circuit breaker state
- `rabbitmq_custom_circuit_breaker_failures_total` - Circuit breaker failures

## 🏗️ Architecture

### Production Optimizations
- **Connection Pooling**: 100 connections, 50 per host, 90s timeout
- **Circuit Breaker**: 5 failure threshold, 60s reset time
- **Asynchronous Collection**: Background data fetching, non-blocking scrapes
- **Memory Safety**: 10MB response limits, efficient caching
- **Graceful Shutdown**: Proper resource cleanup

### Performance Characteristics
- **Scrape Time**: < 1ms (typically 28μs)
- **Memory Usage**: Minimal, efficient caching
- **RabbitMQ Impact**: Zero active connections during scrapes
- **Scalability**: Supports high-volume RabbitMQ clusters

## 🚀 Quick Start

### Prerequisites
- Go 1.21+
- RabbitMQ with Management Plugin enabled

### Installation

1. **Clone the repository**
```bash
git clone git@github.com:jimohabdol/rabbitmq-custom-exporter.git
cd rabbitmq-exporter
```

2. **Build the exporter**
```bash
make build
```

3. **Run the exporter**
```bash
./rabbitmq-exporter --rabbitmq-url=http://localhost:15672 --username=guest --password=guest
```

### Docker

```bash
# Build image
docker build -t rabbitmq-exporter .

# Run container
docker run -p 9419:9419 \
  -e RABBITMQ_EXPORTER_RABBITMQ_URL=http://rabbitmq:15672 \
  -e RABBITMQ_EXPORTER_RABBITMQ_USERNAME=guest \
  -e RABBITMQ_EXPORTER_RABBITMQ_PASSWORD=guest \
  rabbitmq-exporter
```

## ⚙️ Configuration

### Environment Variables
- `RABBITMQ_EXPORTER_RABBITMQ_URL` - RabbitMQ Management API URL (default: http://localhost:15672)
- `RABBITMQ_EXPORTER_RABBITMQ_USERNAME` - RabbitMQ username (default: guest)
- `RABBITMQ_EXPORTER_RABBITMQ_PASSWORD` - RabbitMQ password (default: guest)
- `RABBITMQ_EXPORTER_SCRAPE_INTERVAL` - Scrape interval (default: 15s)
- `RABBITMQ_EXPORTER_LISTEN_PORT` - HTTP server port (default: 9419)
- `RABBITMQ_EXPORTER_TIMEOUT` - Request timeout (default: 10s)

### Configuration File
```yaml
rabbitmq_url: "http://localhost:15672"
rabbitmq_username: "guest"
rabbitmq_password: "guest"
scrape_interval: "15s"
listen_port: 9419
timeout: "10s"
```

## 📈 Prometheus Configuration

Add to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'rabbitmq-custom'
    static_configs:
      - targets: ['localhost:9419']
    scrape_interval: 15s
    metrics_path: /metrics
```

## 🚨 Alerting Rules

```yaml
groups:
  - name: rabbitmq-custom
    rules:
      # High Queue Depth
      - alert: HighQueueDepth
        expr: rabbitmq_custom_queue_depth_alert{severity="critical"} == 1
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "High queue depth detected"
          description: "Queue {{ $labels.queue_name }} has {{ $value }} messages"

      # Low Consumer Utilization
      - alert: LowConsumerUtilization
        expr: rabbitmq_custom_queue_utilization_alert{severity="critical"} == 1
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "Low consumer utilization"
          description: "Queue {{ $labels.queue_name }} has low consumer utilization"

      # Circuit Breaker Open
      - alert: RabbitMQCircuitBreakerOpen
        expr: rabbitmq_custom_circuit_breaker_state{endpoint="rabbitmq_api"} == 1
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "RabbitMQ circuit breaker is open"
          description: "Too many API failures, circuit breaker has opened"

      # Poor Queue Health
      - alert: PoorQueueHealth
        expr: rabbitmq_custom_queue_health_score < 50
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Poor queue health detected"
          description: "Queue {{ $labels.queue_name }} has health score {{ $value }}"
```

## 🧪 Development

### Prerequisites
- Go 1.21+
- Make

### Build Commands
```bash
make build      # Build binary
make test       # Run tests
make clean      # Clean build artifacts
make docker     # Build Docker image
make run        # Run locally
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific test
go test -v ./rabbitmq -run TestClient
```

## 📋 API Endpoints

- `GET /metrics` - Prometheus metrics
- `GET /health` - Health check
- `GET /` - Basic information

## 🔧 Troubleshooting

### Common Issues

1. **Connection Refused**
   - Verify RabbitMQ Management Plugin is enabled
   - Check RabbitMQ URL and credentials
   - Ensure firewall allows connections

2. **High Scrape Duration**
   - Check RabbitMQ performance
   - Verify network connectivity
   - Review circuit breaker metrics

3. **Missing Metrics**
   - Check RabbitMQ API permissions
   - Verify queue names and vhosts
   - Review exporter logs

### Logs
```bash
# View exporter logs
docker logs rabbitmq-exporter

# Check RabbitMQ Management API
curl -u guest:guest http://localhost:15672/api/overview
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Prometheus Client Go](https://github.com/prometheus/client_golang)
- [RabbitMQ Management API](https://www.rabbitmq.com/management.html)
- [Circuit Breaker Pattern](https://martinfowler.com/bliki/CircuitBreaker.html) 