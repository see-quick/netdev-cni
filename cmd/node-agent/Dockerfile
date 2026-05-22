FROM golang:1.24 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /node-agent ./cmd/node-agent/

FROM alpine:3.21
RUN apk add --no-cache iproute2
COPY --from=builder /node-agent /node-agent
ENTRYPOINT ["/node-agent"]