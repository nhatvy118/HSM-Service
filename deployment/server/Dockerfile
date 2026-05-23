FROM golang:1.25-bookworm AS builder

WORKDIR /src
COPY go.mod ./
COPY go.sum* ./
RUN go mod download || true

COPY . .
RUN CGO_ENABLED=1 go build -o /out/server .

# ---------- Runtime ----------
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
        softhsm2 \
        opensc \
        ca-certificates \
        tini \
        wget \
    && rm -rf /var/lib/apt/lists/*

ENV SOFTHSM2_CONF=/etc/softhsm/softhsm2.conf \
    PKCS11_MODULE_PATH=/usr/lib/softhsm/libsofthsm2.so \
    PORT=8080

COPY --from=builder /out/server /usr/local/bin/server

EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:${PORT}/health || exit 1

ENTRYPOINT ["/usr/bin/tini", "--"]
CMD ["/usr/local/bin/server"]
