# Weather Demo App
BINARY_NAME=weather-demo-app
VERSION?=1.0.0

GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod

LDFLAGS=-ldflags "-X main.Version=$(VERSION)"
BUILD_DIR=./build

.PHONY: build clean test run docker help

all: clean build

build:
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./main.go

clean:
	$(GOCMD) clean
	rm -rf $(BUILD_DIR)

test:
	$(GOTEST) -v ./...

deps:
	$(GOMOD) download
	$(GOMOD) tidy

run:
	$(GOCMD) run main.go server --config config.yaml

fmt:
	$(GOCMD) fmt ./...

docker:
	docker build -t $(BINARY_NAME):$(VERSION) .

docker-run:
	docker run --rm -p 8080:8080 -v $(PWD)/config.yaml:/app/config.yaml $(BINARY_NAME):$(VERSION)

up:
	docker compose -f compose.yml up -d

down:
	docker compose -f compose.yml down

logs:
	docker compose -f compose.yml logs -f weather-app

dev: fmt build run

quick: build run

help:
	@echo "Available commands:"
	@echo "  build    - Build the application"
	@echo "  clean    - Clean build files"
	@echo "  test     - Run tests"
	@echo "  deps     - Update dependencies"
	@echo "  run      - Run locally"
	@echo "  fmt      - Format code"
	@echo "  docker   - Build Docker image"
	@echo "  docker-run - Run with Docker"
	@echo "  up       - Start with docker-compose"
	@echo "  down     - Stop docker-compose"
	@echo "  logs     - Show app logs"
	@echo "  dev      - Format, build and run"
	@echo "  quick    - Build and run" 