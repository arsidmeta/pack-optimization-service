.PHONY: run build test docker-build docker-run docker-up docker-down

## Run the server locally (requires Go)
run:
	go run .

## Build the binary
build:
	go build -o pack-optimization-service .

## Run unit tests
test:
	go test ./... -v

## Build the Docker image
docker-build:
	docker build -t pack-optimization-service .

## Run the Docker container directly
docker-run:
	docker run -p 8080:8080 pack-optimization-service

## Start with docker compose (recommended — persists pack sizes)
docker-up:
	docker compose up --build

## Stop docker compose
docker-down:
	docker compose down
