FROM golang:1.25-alpine AS builder

ARG VERSION=dev

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags "-X github.com/Sudhan30/freshprobe/internal/version.Version=${VERSION}" \
    -o /freshprobe ./cmd/freshprobe

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=builder /freshprobe /usr/local/bin/freshprobe
COPY policies/ /etc/freshprobe/policies/

LABEL org.opencontainers.image.source="https://github.com/Sudhan30/freshprobe"
LABEL org.opencontainers.image.description="Data freshness verification for AI agents"
LABEL org.opencontainers.image.licenses="MIT"

ENTRYPOINT ["freshprobe"]
CMD ["serve", "--mode", "mcp", "--policy-dir", "/etc/freshprobe/policies", "--stateless"]
