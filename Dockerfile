FROM golang:1.23.4-alpine3.21 AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/user-points-app ./cmd/main.go

FROM alpine:3.18

RUN apk add --no-cache tzdata

COPY --from=builder /app/user-points-app /app/user-points-app

COPY config /app/config

WORKDIR /app

EXPOSE 8080

CMD ["/app/user-points-app"]