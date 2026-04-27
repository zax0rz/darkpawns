# Dark Pawns Monitoring Stack

This document describes the monitoring infrastructure for Dark Pawns, including Prometheus metrics, Grafana dashboards, and alerting.

## Overview

The monitoring stack consists of:
- **Prometheus** - Metrics collection and alerting
- **Grafana** - Visualization and dashboards
- **Alertmanager** - Alert routing and notification
- **Node Exporter** - System metrics

## Metrics Exposed

Dark Pawns server exposes Prometheus metrics at `/metrics` endpoint:

### Connection Metrics
- `darkpawns_connections_active` - Active WebSocket connections
- `darkpawns_connections_total` - Total connections established
- `darkpawns_connection_errors_total` - Connection errors

### Command Metrics
- `darkpawns_commands_processed_total` - Commands processed by type
- `darkpawns_command_duration_seconds` - Command processing latency

### Game State Metrics
- `darkpawns_players_online` - Players currently online
- `darkpawns_rooms_active` - Active rooms with players
- `darkpawns_mobs_active` - Active mobs in world

### Combat Metrics
- `darkpawns_combat_rounds_total` - Combat rounds processed
- `darkpawns_damage_dealt_total` - Damage dealt by source type
- `darkpawns_deaths_total` - Player/mob deaths

### Error Metrics
- `darkpawns_errors_total` - Errors by type

### Database Metrics
- `darkpawns_db_queries_total` - Database queries
- `darkpawns_db_query_duration_seconds` - Query latency

### Memory Metrics
- `darkpawns_memory_writes_total` - Narrative memory writes
- `darkpawns_memory_reads_total` - Narrative memory reads

## Getting Started

### 1. Start the Monitoring Stack

```bash
docker-compose -f docker-compose.monitoring.yml up -d
```

This will start:
- Dark Pawns server on port 8080
- Prometheus on port 9090
- Grafana on port 3000
- Alertmanager on port 9093
- Node Exporter on port 9100

### 2. Access Dashboards

- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090
- **Alertmanager**: http://localhost:9093
- **Dark Pawns**: http://localhost:8080

### 3. Import Grafana Dashboard

The Dark Pawns dashboard is automatically provisioned. You can also manually import the dashboard JSON from `grafana/dashboards/darkpawns-overview.json`.

## Alerting Rules

Prometheus includes the following alert rules:

### Critical Alerts
- `DarkPawnsDown` - Server is down for >1 minute
- `HighMemoryUsage` - Memory usage >80% for 5 minutes

### Warning Alerts
- `HighErrorRate` - Error rate >10/s for 2 minutes
- `LowConnections` - No active connections for 5 minutes
- `HighCommandLatency` - 95th percentile latency >1s for 2 minutes
- `HighDBLatency` - 95th percentile DB latency >0.5s for 2 minutes

## Configuration

### Prometheus
- Configuration: `prometheus/prometheus.yml`
- Alert rules: `prometheus/alert_rules.yml`

### Alertmanager
- Configuration: `prometheus/alertmanager.yml`

### Grafana
- Datasources: `grafana/datasources/prometheus.yml`
- Dashboards: `grafana/dashboards/`

## Adding Custom Metrics

To add new metrics to the Dark Pawns server:

1. Add metric definition in `pkg/metrics/metrics.go`
2. Use the metric in your code:
   ```go
   metrics.CommandProcessed("look", duration)
   metrics.ErrorOccurred("database")
   ```

3. The metric will automatically appear in Prometheus and can be added to Grafana dashboards.

## Health Checks

The server includes health endpoints:
- `/health` - Basic health check
- `/metrics` - Prometheus metrics

## Troubleshooting

### Metrics Not Appearing
1. Verify server is running: `curl http://localhost:8080/health`
2. Check metrics endpoint: `curl http://localhost:8080/metrics`
3. Verify Prometheus is scraping: Check targets in Prometheus UI

### Grafana No Data
1. Verify datasource is configured correctly
2. Check Prometheus is accessible from Grafana
3. Verify time range in dashboard

### Alerts Not Firing
1. Check alert rules in Prometheus UI
2. Verify Alertmanager is running
3. Check Alertmanager configuration

## Monitoring Best Practices

1. **Set up alerts** for critical metrics
2. **Monitor trends** not just absolute values
3. **Use dashboards** for quick visibility
4. **Regularly review** alert thresholds
5. **Document** monitoring setup and procedures

## Extending Monitoring

### Additional Metrics to Consider
- Lua script execution time
- WebSocket message throughput
- Room transition frequency
- Player session duration
- Combat balance metrics

### Integration with External Systems
- Slack/Discord notifications via webhooks
- PagerDuty for critical alerts
- Email notifications for warnings
- Custom webhook receivers