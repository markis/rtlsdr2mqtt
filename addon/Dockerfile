# This Dockerfile is only used for local development/testing.
# Production add-on uses the pre-built image from config.yaml.

ARG BUILD_FROM
FROM ${BUILD_FROM}

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    librtlsdr \
    libusb \
    tzdata

# Copy rootfs (run.sh script)
COPY rootfs /

# The binary must be manually copied for local testing:
# cp /path/to/rtlsdr2mqtt addon/rootfs/usr/bin/rtlsdr2mqtt

RUN chmod a+x /run.sh && \
    if [ -f /usr/bin/rtlsdr2mqtt ]; then chmod a+x /usr/bin/rtlsdr2mqtt; fi

CMD [ "/run.sh" ]
