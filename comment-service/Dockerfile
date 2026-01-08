FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git gcc musl-dev sqlite-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates sqlite-dev
WORKDIR /root/

COPY --from=builder /app/main .

EXPOSE 8081

CMD ["./main"]