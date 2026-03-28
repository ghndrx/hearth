# Hearth Grafana Dashboards

This directory contains Grafana dashboards for monitoring Hearth services.

## Available Dashboards

### hearth-websocket.json
Real-time monitoring of WebSocket connections and message throughput.

**Panels include:**
- Total Active Connections (across all pods)
- Connections Per Pod (per-instance breakdown)
- Connection Rate (connections/second)
- Message Sent/Received Rate
- Message Latency Percentiles (p50, p90, p95, p99)
- Heartbeat Rate
- Channel & Server Subscriptions
- Active Sessions
- Connections by Client Type (web, desktop, mobile)
- Messages by Event Type

## Installation

### Option 1: Import via Grafana UI

1. Open Grafana → Dashboards → Import
2. Upload the JSON file or paste its contents
3. Select your Prometheus datasource
4. Click Import

### Option 2: Provisioning (Recommended for Production)

Add to your Grafana provisioning configuration:

```yaml
# /etc/grafana/provisioning/dashboards/hearth.yaml
apiVersion: 1
providers:
  - name: 'Hearth'
    orgId: 1
    folder: 'Hearth'
    type: file
    disableDeletion: false
    updateIntervalSeconds: 30
    options:
      path: /var/lib/grafana/dashboards/hearth
```

Then copy the JSON files to `/var/lib/grafana/dashboards/hearth/`.

### Option 3: Kubernetes ConfigMap

```bash
kubectl create configmap hearth-grafana-dashboards \
  --from-file=dashboards/ \
  -n monitoring
```

Then mount the ConfigMap in your Grafana deployment.

## Prometheus Configuration

Ensure your Prometheus is scraping the Hearth backend's `/metrics` endpoint:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'hearth-backend'
    kubernetes_sd_configs:
      - role: endpoints
        namespaces:
          names: ['hearth']
    relabel_configs:
      - source_labels: [__meta_kubernetes_service_label_app]
        regex: hearth
        action: keep
      - source_labels: [__meta_kubernetes_endpoint_port_name]
        regex: http
        action: keep
```

Or if using ServiceMonitor (Prometheus Operator):

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: hearth-backend
  namespace: hearth
spec:
  selector:
    matchLabels:
      app: hearth
      component: backend
  endpoints:
    - port: http
      path: /metrics
      interval: 15s
```

## Alerting

Consider adding alerts for:

| Alert | Condition | Severity |
|-------|-----------|----------|
| High Connection Count | `sum(hearth_websocket_connections_active) > 5000` | Warning |
| Connection Spike | `rate(hearth_websocket_connections_total[5m]) > 50` | Warning |
| High Message Latency | `histogram_quantile(0.95, ...) > 0.5` | Critical |
| Pod Imbalance | `max - min connections > 500` | Warning |

## Variables

The dashboard supports the following template variables:

- `$datasource`: Prometheus datasource selector
- `$instance`: Filter by pod/instance name

## Metrics Reference

| Metric | Type | Description |
|--------|------|-------------|
| `hearth_websocket_connections_active` | Gauge | Currently active connections |
| `hearth_websocket_connections_total` | Counter | Total connections ever established |
| `hearth_websocket_messages_sent_total` | Counter | Messages sent to clients |
| `hearth_websocket_messages_received_total` | Counter | Messages received from clients |
| `hearth_websocket_message_latency_seconds` | Histogram | Message processing latency |
| `hearth_websocket_sessions_active` | Gauge | Active WebSocket sessions |
| `hearth_websocket_channel_subscriptions_active` | Gauge | Active channel subscriptions |
| `hearth_websocket_server_subscriptions_active` | Gauge | Active server subscriptions |
| `hearth_websocket_heartbeats_total` | Counter | Heartbeat messages processed |
| `hearth_websocket_connection_duration_seconds` | Histogram | Connection duration distribution |
