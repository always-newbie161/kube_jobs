FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o kubejobs ./cmd/server/main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/kubejobs .

CMD ["./kubejobs"]