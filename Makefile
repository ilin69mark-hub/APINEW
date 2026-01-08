.PHONY: build run test clean docker-build docker-run docker-down

# Build all services
build:
	@echo "Building API Gateway..."
	cd api-gateway && go build -o ../bin/api-gateway .
	@echo "Building Comment Service..."
	cd comment-service && go build -o ../bin/comment-service .
	@echo "Building Censor Service..."
	cd censor-service && go build -o ../bin/censor-service .
	@echo "Building News Aggregator..."
	cd news-aggregator && go build -o ../bin/news-aggregator .
	@echo "All services built successfully!"

# Run all services (in background)
run: build
	@echo "Starting all services..."
	./bin/api-gateway > api-gateway.log 2>&1 &
	./bin/comment-service > comment-service.log 2>&1 &
	./bin/censor-service > censor-service.log 2>&1 &
	./bin/news-aggregator > news-aggregator.log 2>&1 &
	@echo "All services started in background!"

# Test all services
test:
	@echo "Running tests for API Gateway..."
	cd api-gateway && go test -v
	@echo "Running tests for Comment Service..."
	cd comment-service && go test -v
	@echo "Running tests for Censor Service..."
	cd censor-service && go test -v
	@echo "Running tests for News Aggregator..."
	cd news-aggregator && go test -v

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f api-gateway.log comment-service.log censor-service.log news-aggregator.log

# Docker build
docker-build:
	docker-compose build

# Docker run
docker-run: docker-build
	docker-compose up -d

# Docker down
docker-down:
	docker-compose down

# Install dependencies
deps:
	cd api-gateway && go mod tidy
	cd comment-service && go mod tidy
	cd censor-service && go mod tidy
	cd news-aggregator && go mod tidy