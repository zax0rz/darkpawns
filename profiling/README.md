# Profiling utilities

This directory contains the standalone `profiler` tool used for runtime
profiling and pprof serving.

## pprof server

Start an HTTP pprof endpoint:

```bash
PPROF_USER=admin PPROF_PASS=secret go run ./profiling pprof
```

### Bind address

By default the pprof server binds to **localhost only**:

```
127.0.0.1:6060
```

To bind to a different address (for example, remote debugging on a LAN),
set the `PPROF_BIND_ADDR` environment variable:

```bash
PPROF_BIND_ADDR=0.0.0.0:6060 go run ./profiling pprof
```

An explicit address passed as a command-line argument takes precedence over
the environment variable:

```bash
go run ./profiling pprof :7070
```

The server requires HTTP Basic Auth credentials via `PPROF_USER` and
`PPROF_PASS`.
