FROM golang:1.21-alpine AS builder

WORKDIR /app

# Download dependencies first so this layer is cached
COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

# -------- final image --------
FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/server .
COPY --from=builder /app/static ./static

EXPOSE 8080

CMD ["./server"]
