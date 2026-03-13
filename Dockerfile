FROM golang:1.26-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -o discordiance ./cmd/discordiance

FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /app/discordiance .

EXPOSE 8067

ENTRYPOINT ["./discordiance"]
