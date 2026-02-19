# Multi-stage Dockerfile for beads_viewer web UI.
#
# Build: docker build -t beads-viewer .
# Run:   docker run -p 9000:9000 \
#          -e BEADS_URL=http://bd-daemon:8443 \
#          -e BD_API_KEY=<token> \
#          beads-viewer
#
# The container:
# 1. Builds the bv binary with FTS5 support
# 2. At runtime, loads issues from the daemon via --beads-url
# 3. Exports a static site to /app/pages
# 4. Serves the static site on 0.0.0.0:9000

# --- Build stage ---
FROM golang:1.25-bookworm AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build with SQLite FTS5 support (same as Makefile)
ENV CGO_CFLAGS="-DSQLITE_ENABLE_FTS5"
RUN go build -o /bv ./cmd/bv

# --- Runtime stage ---
FROM debian:bookworm-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*

RUN useradd -r -m -s /bin/false bv

COPY --from=builder /bv /usr/local/bin/bv
COPY scripts/docker-entrypoint.sh /app/entrypoint.sh

WORKDIR /app
RUN mkdir -p /app/pages && chown -R bv:bv /app

USER bv

ENV BEADS_URL=""
ENV BD_API_KEY=""
ENV BV_PORT="9000"

EXPOSE 9000

ENTRYPOINT ["/app/entrypoint.sh"]
