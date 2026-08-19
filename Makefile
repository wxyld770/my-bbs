.PHONY: run test test-stability vet fmt up down logs

run:
	go run ./cmd

test:
	go test ./...

test-stability:
	go test ./... -shuffle=on -count=2

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
