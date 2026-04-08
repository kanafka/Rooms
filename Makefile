.PHONY: up down seed lint test swag

up:
	docker-compose up --build -d

down:
	docker-compose down

seed:
	go run seed/seed.go

lint:
	golangci-lint run

test:
	go test ./...

swag:
	swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal
