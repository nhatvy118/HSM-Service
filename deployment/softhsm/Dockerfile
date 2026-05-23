FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive \
    SOFTHSM2_CONF=/etc/softhsm/softhsm2.conf \
    PKCS11_MODULE_PATH=/usr/lib/softhsm/libsofthsm2.so \
    TOKEN_DIR=/var/lib/softhsm/tokens

RUN apt-get update && apt-get install -y --no-install-recommends \
        softhsm2 \
        opensc \
        ca-certificates \
        tini \
    && rm -rf /var/lib/apt/lists/* \
    && (getent group softhsm >/dev/null || groupadd -r softhsm) \
    && (getent passwd softhsm >/dev/null || \
        useradd -r -g softhsm -M -s /sbin/nologin -d /var/lib/softhsm softhsm) \
    && mkdir -p ${TOKEN_DIR} /etc/softhsm \
    && chown -R softhsm:softhsm /var/lib/softhsm /etc/softhsm \
    && chmod 750 ${TOKEN_DIR}

COPY --chown=softhsm:softhsm softhsm2.conf /etc/softhsm/softhsm2.conf
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

USER softhsm
VOLUME ["/var/lib/softhsm/tokens"]

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD softhsm2-util --show-slots >/dev/null 2>&1 || exit 1

ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/entrypoint.sh"]
CMD ["sleep", "infinity"]
