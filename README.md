# Weather Aggregation Service

It is a prototype of the production grade weather aggregation 
service.  It is dockerized and could be run on kubernetes clusters
It has the observability stack based on Grafana (not yet fully working due to broken configuration)

## Desing ideas

- extantable weather services through configuration
- configuration from diferent sources with env variable priority
- request id support for better debuging
- server performance optimization through:
    - throtling mechanics
    - response caching
    - aggregator with workers
- open telemetry observability stack


## Quick Start: Run & Test Locally

### 1. Clone the Repository 
```sh
cd weather-demo-app
```

### 2. Build the Application
```sh
make build
```

### 3. Run with Docker Compose (Recommended)
This will start the app and the observability stack (Grafana, Prometheus, etc.):
```sh
docker compose up --build
```
- The API will be available at: [http://localhost:8080](http://localhost:8080)
- Grafana (if configured) at: [http://localhost:3002](http://localhost:3002)

### 4. Run Locally Without Docker
```sh
go run main.go server --config config.yaml
```

### 5. Test the API
You can use the provided curl command:
```sh
curl -X GET "http://localhost:8080/weather?lat=54.52&lon=13.41" \
  -H "X-Request-ID: test-request-12345" \
  -H "Accept: application/json" \
  -v
```

### 6. Run Unit Tests
```sh
make test
```
or
```sh
go test -v ./...`


