# Prometheus Metrics for SPEEDRUN Intent Services

This document describes the Prometheus metrics exposed by the SPEEDRUN intent services for monitoring and alerting.

## Metrics Endpoints

### `/metrics`
Standard Prometheus metrics endpoint that exposes all metrics in Prometheus format for scraping.

### `/api/v1/metrics`
Human-readable JSON endpoint that provides a summary of all metrics for debugging and inspection.

## Available Metrics

All metrics are labeled with `chain_id` and `chain_name` for multi-chain monitoring.

### Service Health Metrics

#### `speedrun_intent_services_up`
- **Type:** Gauge
- **Description:** Whether intent services are healthy (1 = healthy, 0 = unhealthy)
- **Labels:** `chain_id`, `chain_name`
- **Use Case:** Primary health check for alerting

#### `speedrun_active_goroutines`
- **Type:** Gauge
- **Description:** Number of active goroutines per chain
- **Labels:** `chain_id`, `chain_name`
- **Use Case:** Monitor resource usage and detect goroutine leaks

#### `speedrun_subscriptions_active`
- **Type:** Gauge
- **Description:** Number of active blockchain subscriptions per chain (for ZetaChain: 1 = polling healthy, 0 = polling unhealthy)
- **Labels:** `chain_id`, `chain_name`
- **Use Case:** Monitor WebSocket connection health (or HTTP polling health for ZetaChain)

### Event Processing Metrics

#### `speedrun_events_processed_total`
- **Type:** Gauge
- **Description:** Total number of events processed per chain
- **Labels:** `chain_id`, `chain_name`
- **Use Case:** Monitor event processing throughput

#### `speedrun_events_skipped_total`
- **Type:** Gauge
- **Description:** Total number of events skipped (duplicates) per chain
- **Labels:** `chain_id`, `chain_name`
- **Use Case:** Monitor duplicate event detection

#### `speedrun_processing_errors_total`
- **Type:** Gauge
- **Description:** Total number of processing errors per chain
- **Labels:** `chain_id`, `chain_name`
- **Use Case:** Monitor error rates and service reliability

#### `speedrun_reconnections_total`
- **Type:** Gauge
- **Description:** Total number of reconnections per chain
- **Labels:** `chain_id`, `chain_name`
- **Use Case:** Monitor WebSocket connection stability

### Timing Metrics

#### `speedrun_last_event_timestamp`
- **Type:** Gauge
- **Description:** Timestamp of the last processed event per chain
- **Labels:** `chain_id`, `chain_name`
- **Use Case:** Monitor event processing latency

#### `speedrun_time_since_last_event_seconds`
- **Type:** Gauge
- **Description:** Time in seconds since the last processed event per chain
- **Labels:** `chain_id`, `chain_name`
- **Use Case:** Alert on stale event processing

#### `speedrun_last_health_check_timestamp`
- **Type:** Gauge
- **Description:** Timestamp of the last health check per chain
- **Labels:** `chain_id`, `chain_name`
- **Use Case:** Monitor health check system

## Supported Chains

The metrics service automatically recognizes these chains and provides human-readable names:

- **ethereum** (chain_id: 1)
- **arbitrum** (chain_id: 42161)
- **base** (chain_id: 8453)
- **polygon** (chain_id: 137)
- **bsc** (chain_id: 56)
- **avalanche** (chain_id: 43114)
- **zetachain** (chain_id: 7000) - *Uses HTTP polling instead of WebSocket subscriptions*

## ZetaChain Special Behavior

ZetaChain (chain_id: 7000) operates differently from other chains:

### **HTTP Polling vs WebSocket Subscriptions**
- **Other chains**: Use WebSocket subscriptions for real-time event monitoring
- **ZetaChain**: Uses HTTP polling every 15 seconds due to WebSocket limitations

### **Health Check Differences**
- **Other chains**: Healthy = ≥3 goroutines + ≥1 subscription
- **ZetaChain**: Healthy = HTTP client can reach chain + polling working

### **Metrics Interpretation**
- **`speedrun_subscriptions_active`**: 
  - Other chains: Actual WebSocket subscription count
  - ZetaChain: 1 = polling healthy, 0 = polling unhealthy
- **`speedrun_active_goroutines`**:
  - Other chains: 3+ expected (error monitor + health monitor + subscription)
  - ZetaChain: May be lower since no subscription goroutines needed

### **Polling Health Tracking**
ZetaChain metrics include additional fields:
- **`is_zetachain`**: Always true for ZetaChain
- **`polling_healthy`**: Whether HTTP polling is working
- **`last_polling_check`**: Timestamp of last polling health verification
- **`time_since_polling_check`**: Time since last polling health check

## Prometheus Configuration

Add this to your `prometheus.yml` configuration:

```yaml
scrape_configs:
  - job_name: 'speedrun-intent-services'
    static_configs:
      - targets: ['localhost:8080']  # Replace with your API server address
    scrape_interval: 15s
    metrics_path: '/metrics'
    scrape_timeout: 10s
```

## Example Queries

### Service Health
```promql
# Check if all intent services are healthy
speedrun_intent_services_up == 1

# Count unhealthy services
count(speedrun_intent_services_up == 0)

# Services with no active subscriptions (excludes ZetaChain which uses polling)
speedrun_subscriptions_active{chain_name!="zetachain"} == 0

# ZetaChain polling unhealthy
speedrun_subscriptions_active{chain_name="zetachain"} == 0
```

### Event Processing
```promql
# Event processing rate (events per second)
rate(speedrun_events_processed_total[5m])

# Error rate percentage
(speedrun_processing_errors_total / speedrun_events_processed_total) * 100

# Events not processed in the last 5 minutes
speedrun_time_since_last_event_seconds > 300
```

### Resource Monitoring
```promql
# High goroutine count (potential leak)
speedrun_active_goroutines > 10

# Frequent reconnections (connection issues)
increase(speedrun_reconnections_total[1h]) > 5
```

## Recommended Alerts

### Critical Alerts

```yaml
- alert: SpeedrunIntentServiceDown
  expr: speedrun_intent_services_up == 0
  for: 1m
  labels:
    severity: critical
  annotations:
    summary: "Intent service is down for {{ $labels.chain_name }}"
    description: "Intent service for chain {{ $labels.chain_id }} ({{ $labels.chain_name }}) has been down for more than 1 minute"

- alert: SpeedrunNoActiveSubscriptions
  expr: speedrun_subscriptions_active{chain_name!="zetachain"} == 0
  for: 2m
  labels:
    severity: critical
  annotations:
    summary: "No active subscriptions for {{ $labels.chain_name }}"
    description: "Chain {{ $labels.chain_id }} ({{ $labels.chain_name }}) has no active subscriptions"

- alert: SpeedrunZetaChainPollingUnhealthy
  expr: speedrun_subscriptions_active{chain_name="zetachain"} == 0
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "ZetaChain HTTP polling is unhealthy"
    description: "ZetaChain HTTP polling has been unhealthy for more than 5 minutes"

- alert: SpeedrunStaleEventProcessing
  expr: speedrun_time_since_last_event_seconds > 600
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Event processing is stale for {{ $labels.chain_name }}"
    description: "No events processed for chain {{ $labels.chain_id }} ({{ $labels.chain_name }}) in the last 10 minutes"
```

### Warning Alerts

```yaml
- alert: SpeedrunHighErrorRate
  expr: (speedrun_processing_errors_total / speedrun_events_processed_total) * 100 > 5
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "High error rate for {{ $labels.chain_name }}"
    description: "Error rate for chain {{ $labels.chain_id }} ({{ $labels.chain_name }}) is above 5%"

- alert: SpeedrunFrequentReconnections
  expr: increase(speedrun_reconnections_total[1h]) > 3
  for: 0m
  labels:
    severity: warning
  annotations:
    summary: "Frequent reconnections for {{ $labels.chain_name }}"
    description: "Chain {{ $labels.chain_id }} ({{ $labels.chain_name }}) has reconnected more than 3 times in the last hour"

- alert: SpeedrunHighGoroutineCount
  expr: speedrun_active_goroutines > 15
  for: 10m
  labels:
    severity: warning
  annotations:
    summary: "High goroutine count for {{ $labels.chain_name }}"
    description: "Chain {{ $labels.chain_id }} ({{ $labels.chain_name }}) has more than 15 active goroutines"
```

## Grafana Dashboard

Create a Grafana dashboard with these panels:

1. **Service Health Overview**
   - Single stat panels showing healthy/unhealthy services
   - Table with all chain statuses

2. **Event Processing**
   - Time series graph of event processing rates
   - Error rate percentage over time
   - Time since last event processed

3. **Resource Usage**
   - Goroutine count per chain
   - Subscription count per chain
   - Reconnection rate

4. **Alerts Summary**
   - List of active alerts
   - Alert history

## Troubleshooting

### No Metrics Data
- Check if the `/metrics` endpoint is accessible
- Verify Prometheus is scraping the correct endpoint
- Check API server logs for metrics service errors

### Metrics Not Updating
- Verify intent services are registered with metrics service
- Check if the metrics updater goroutine is running
- Look for errors in the metrics service logs

### Missing Chain Data
- Ensure intent services are properly started for all chains
- Check if chain clients are connected
- Verify chain configuration is correct

## Metrics Service Architecture

The metrics service:
1. Registers intent services from all configured chains
2. Periodically collects metrics from each service (every 15 seconds)
3. Exposes metrics in Prometheus format
4. Provides human-readable summaries for debugging

This architecture ensures that metrics are always up-to-date and reflect the current state of all intent services across all supported chains. 