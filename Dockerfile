# ----- Build stage -----
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /src

# Cache module downloads.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev

RUN CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /minicloud \
    ./cmd/minicloud

# ----- Runtime stage -----
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S minicloud \
    && adduser -S -G minicloud minicloud

COPY --from=builder /minicloud /usr/local/bin/minicloud

# Default data directory.
RUN mkdir -p /data && chown minicloud:minicloud /data
VOLUME /data

USER minicloud

EXPOSE 8080

ENTRYPOINT ["minicloud"]
CMD ["-data-dir", "/data"]
