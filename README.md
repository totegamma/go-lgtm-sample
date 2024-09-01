# go lgtm sample

A sample project that enables you to quickly leverage the LGTM stack in local development.

## Features

- Collection of logs, metrics, and traces
- Built-in, lightweight dashboard
- Hot reload using air
- Enable log to trace and trace to log
- Add trace IDs to response headers
- Display traces in tests
- Load testing with k6

## Requirements
Ensure that Docker is set up with the Loki plugin.

```bash
docker plugin install grafana/loki-docker-driver:latest --alias loki --grant-all-permissions
```

## Usage

Start the application with Docker Compose and access it via localhost:3000.

query example
```
curl -X"POST" -H"content-type: application/json" localhost:8000/calc -d '{"op": "/", "a": 30, "b": {"op": "-", "a": 10, "b": 10}}' -i
```

start k6
```
docker compose run --rm k6 run /scripts/loadtest.js
```
