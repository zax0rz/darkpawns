# Native server monitoring

Start the server using [Running Dark Pawns](../../DEPLOYMENT.md). Then inspect
the HTTP endpoints on your instance (default port 4350):

```bash
curl --fail http://localhost:4350/health
curl --fail http://localhost:4350/metrics
```

A healthy HTTP endpoint does not prove database persistence; also verify a saved
login after restart as described in the installation guide.

The inherited Compose monitoring stack and its lifecycle commands are retired.
Prometheus and Grafana files in the repository remain reference assets; they do
not install or provision services. Configure scrape targets, dashboards, and
alerts for your own native deployment. Official monitoring configuration belongs
in the private ops repo. See [MONITORING.md](MONITORING.md) for historical metric
and dashboard notes, which should be checked against the running endpoint.
