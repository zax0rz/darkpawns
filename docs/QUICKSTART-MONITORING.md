# Quick Start: Dark Pawns Monitoring

Get the monitoring stack up and running in 5 minutes.

## Prerequisites

- Docker and Docker Compose
- Go 1.25+ (for building the server)

## Step 1: Build the Server

```bash
make install
make build
```

## Step 2: Start Monitoring Stack

```bash
make monitoring-up
```

Or manually:
```bash
docker-compose -f docker-compose.monitoring.yml up -d
```

## Step 3: Verify Services

Check that all services are running:
```bash
docker-compose -f docker-compose.monitoring.yml ps
```

## Step 4: Access Dashboards

1. **Grafana**: http://localhost:3000
   - Username: `admin`
   - Password: `admin`

2. **Prometheus**: http://localhost:9090
   - Check targets: http://localhost:9090/targets

3. **Dark Pawns Server**: http://localhost:8080
   - Health check: http://localhost:8080/health
   - Metrics: http://localhost:8080/metrics

## Step 5: Start the Game Server

```bash
make run
```

Or with custom world directory:
```bash
./darkpawns -world /path/to/world/files
```

## Step 6: View Metrics

1. Open Grafana at http://localhost:3000
2. Navigate to "Dashboards" → "Dark Pawns - Overview"
3. Connect to the game and watch metrics update in real-time

## Common Commands

### Start/Stop Monitoring
```bash
make monitoring-up    # Start monitoring stack
make monitoring-down  # Stop monitoring stack
make monitoring-logs  # View logs
```

### Build and Run
```bash
make build           # Build server
make run             # Run server
make test            # Run tests
```

### Clean Up
```bash
make clean           # Remove build artifacts
docker-compose -f docker-compose.monitoring.yml down -v  # Remove volumes
```

## Troubleshooting

### No Metrics Showing
1. Verify server is running: `curl http://localhost:8080/health`
2. Check metrics endpoint: `curl http://localhost:8080/metrics`
3. Verify Prometheus is scraping: Check http://localhost:9090/targets

### Grafana Login Issues
Default credentials: `admin` / `admin`
If locked out, reset password:
```bash
docker exec -it grafana grafana-cli admin reset-admin-password admin
```

### Port Conflicts
If ports are already in use, modify `docker-compose.monitoring.yml` to use different ports.

## Next Steps

1. **Configure Alerts**: Edit `prometheus/alert_rules.yml` for custom alert thresholds
2. **Add Notifications**: Configure `prometheus/alertmanager.yml` for Slack/Email alerts
3. **Custom Dashboards**: Create additional Grafana dashboards in `grafana/dashboards/`
4. **Add Metrics**: Extend `pkg/metrics/metrics.go` with game-specific metrics

## Support

- Full documentation: `docs/MONITORING.md`
- Metric reference: `pkg/metrics/metrics.go`
- Example integration: `examples/metrics_integration.go`