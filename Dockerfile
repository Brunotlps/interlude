# First stage: builder 
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o gateway ./cmd/gateway

# Second stage: runtime
FROM scratch 
# scratch -> from 0 

COPY --from=builder /app/gateway            /gateway
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# TLS certificate from server
COPY config.yaml                            /config.yaml

ENV CONFIG_PATH=/config.yaml

EXPOSE 8080
EXPOSE 9091

# CMD allows overwriting in runtime, ENTRYPOINT don't
CMD ["/gateway"]
