# Weather Demo App Dockerfile

FROM golang:1.24-alpine AS builder

ARG VERSION=1.0.0
ARG BUILD_TIME
ARG GIT_COMMIT

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download && go mod verify

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-s -w -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}" \
    -a -installsuffix cgo \
    -o weather-demo-app \
    ./main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/weather-demo-app .

COPY --from=builder /app/config.yaml .

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

RUN mkdir -p /app/logs && \
    chown -R appuser:appgroup /app

USER appuser

HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health/ready || exit 1

EXPOSE 8080

ENV GIN_MODE=release
ENV WDP_LOGGING_LEVEL=info
ENV WDP_LOGGING_FORMAT=json


# Run the application
ENTRYPOINT ["./weather-demo-app"]
CMD ["server", "--config", "config.yaml"]

