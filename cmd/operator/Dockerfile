FROM golang:1.24 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /operator ./cmd/operator/

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /operator /operator
ENTRYPOINT ["/operator"]