# Multi-stage build for CN DMS
FROM golang:1.24-alpine AS builder

WORKDIR /build

COPY go.mod go.sum* ./
RUN go mod download 2>/dev/null || true

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o dms ./cn-dms/cmd/dms/

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /build/dms ./cn-dms

EXPOSE 8082 9092

CMD ["./cn-dms"]