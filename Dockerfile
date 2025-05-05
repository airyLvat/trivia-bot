FROM arm64v8/golang:1.23-alpine AS builder
RUN apk add --no-cache git gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux GOARCH=arm64 GOARM=8 go build -o trivia-bot main.go

FROM arm64v8/alpine:3.18
RUN apk add --no-cache sqlite
WORKDIR /app
COPY --from=builder /app/trivia-bot .
COPY .env .
VOLUME /app/data
ENV DATABASE_PATH=/app/data/trivia.db
RUN mkdir -p /app/data && chown -R 1000:1000 /app/data
ENTRYPOINT ["./trivia-bot"]
