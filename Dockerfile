FROM golang:1.20 AS builder

WORKDIR /app

COPY src/go.mod src/go.sum ./src/
WORKDIR /app/src
RUN go mod download

COPY src/ /app/src/
COPY bots/ /app/bots/

WORKDIR /app/src
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o /app/main .

FROM arm64v8/alpine

WORKDIR /app

COPY --from=builder /app/main .
COPY --from=builder /app/bots /app/bots/

ENTRYPOINT ["./main"]
