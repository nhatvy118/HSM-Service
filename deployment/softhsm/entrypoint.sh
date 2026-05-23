#!/usr/bin/env bash
set -euo pipefail

: "${TOKEN_LABEL:?TOKEN_LABEL is required}"
: "${USER_PIN:?USER_PIN is required}"
: "${SO_PIN:?SO_PIN is required}"

if softhsm2-util --show-slots 2>/dev/null | grep -q "Label:[[:space:]]*${TOKEN_LABEL}$"; then
    echo "[entrypoint] Token '${TOKEN_LABEL}' already initialized, skipping."
else
    echo "[entrypoint] Initializing token '${TOKEN_LABEL}'..."
    softhsm2-util --init-token --free \
        --label "${TOKEN_LABEL}" \
        --pin "${USER_PIN}" \
        --so-pin "${SO_PIN}"
fi

echo "[entrypoint] Current slots:"
softhsm2-util --show-slots | grep -E "Slot|Label|Token" || true

exec "$@"
