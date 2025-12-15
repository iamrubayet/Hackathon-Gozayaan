.PHONY: help migrate-up migrate-down migrate-create run build clean

DB_URL=postgres://root:password@localhost:5432/rickshaw?sslmode=disable

help:
	@echo "Available commands:"
	@echo "  make run          - Run the application"
	@echo "  make build        - Build the application"
	@echo "  make migrate-up   - Run database migrations"
	@echo "  make migrate-down - Rollback database migrations"
	@echo "  make migrate-create NAME=migration_name - Create new migration"
	@echo "  make clean        - Clean build artifacts"

run:
	@echo "Starting server..."
	go run main.go

build:
	@echo "Building application..."
	go build -o bin/rickshaw-app main.go

migrate-up:
	@echo "Running migrations..."
	migrate -path migrations -database "$(DB_URL)" up

migrate-down:
	@echo "Rolling back migrations..."
	migrate -path migrations -database "$(DB_URL)" down

migrate-create:
	@echo "Creating migration: $(NAME)"
	migrate create -ext sql -dir migrations -seq $(NAME)

clean:
	@echo "Cleaning..."
	rm -rf bin/