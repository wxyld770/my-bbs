.PHONY: run test vet fmt up down logs

run:
	go run ./cmd

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

up:
	docker compose up --build -d

down:
	docker compose down

logs:
	docker compose logs -f app
