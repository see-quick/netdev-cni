FROM golang:1.24 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /cni-plugin ./cmd/cni-plugin/

FROM alpine:3.21
COPY --from=builder /cni-plugin /netdev-cni
ENTRYPOINT ["/netdev-cni"]