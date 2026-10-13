.PHONY: run migration build test vet tidy up down clean

up:
	docker compose up -d

down:
	docker compose down

run:
	go run ./cmd/server -conf config/local.yml

migration:
	go run ./cmd/migration -conf config/local.yml

build:
	CGO_ENABLED=0 go build -o bin/server ./cmd/server

test:
	go test ./...

vet:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -rf bin storage/do-template.db storage/do-template.db-*
