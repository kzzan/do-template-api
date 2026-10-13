.PHONY: run migration build test vet tidy up down proto clean

up:
	docker compose up -d

down:
	docker compose down

proto:
	protoc --go_out=. --go_opt=module=do-template-api \
		--go-grpc_out=. --go-grpc_opt=module=do-template-api \
		api/proto/userv1/user.proto

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
