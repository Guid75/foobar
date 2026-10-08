.PHONY: generate
generate:
	templ generate

.PHONY: dev
dev:
	templ generate --watch --proxy="http://localhost:8080" --cmd="go run ./cmd/app"

.PHONY: build
build: generate
	go build -o bin/app ./cmd/app

.PHONY: test
test:
	go test ./...