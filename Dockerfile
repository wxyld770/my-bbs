FROM golang:1.26 AS build

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /my-bbs ./cmd

FROM alpine:3.22
WORKDIR /app
COPY --from=build /my-bbs /usr/local/bin/my-bbs

EXPOSE 8080
CMD ["my-bbs"]
