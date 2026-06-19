# syntax=docker/dockerfile:1

# Stage 1: Build the Go helpers statically linked.
FROM golang:1.23-bookworm AS go-builder

WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd

RUN CGO_ENABLED=0 go build -ldflags='-s -w -extldflags=-static' \
        -o /out/entrypoint ./cmd/entrypoint && \
    CGO_ENABLED=0 go build -ldflags='-s -w -extldflags=-static' \
        -o /out/sms-telegram ./cmd/sms-telegram


# Stage 2: Extract gammu runtime dependencies.
FROM debian:bookworm-slim AS gammu-builder

RUN apt-get update && apt-get install -y --no-install-recommends \
        gammu gammu-smsd ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p /var/spool/gammu/inbox /var/spool/gammu/outbox \
                /var/spool/gammu/sent /var/spool/gammu/error

# Collect all shared libraries required by the gammu binaries.
RUN mkdir -p /out/usr/bin /out/etc/ssl /out/lib /out/lib64 /out/usr/lib /out/var/spool/gammu && \
    cp /usr/bin/gammu /usr/bin/gammu-smsd /usr/bin/gammu-smsd-inject /out/usr/bin/ && \
    cp -r /etc/ssl/certs /out/etc/ssl/ && \
    cp -r /var/spool/gammu/* /out/var/spool/gammu/ && \
    for bin in /usr/bin/gammu /usr/bin/gammu-smsd /usr/bin/gammu-smsd-inject; do \
        ldd "$bin" | awk '/=>/ {print $3} /ld-linux/ {print $1}' | \
        while read -r lib; do \
            [ -f "$lib" ] || continue; \
            cp -v "$lib" /out/lib/ 2>/dev/null || cp -v "$lib" /out/lib64/ 2>/dev/null || cp -v "$lib" /out/usr/lib/ 2>/dev/null || true; \
        done; \
    done && \
    # Resolve transitive dependencies one more time for anything we copied.
    for lib in /out/lib/*.so* /out/lib64/*.so* /out/usr/lib/*.so*; do \
        [ -f "$lib" ] || continue; \
        ldd "$lib" 2>/dev/null | awk '/=>/ {print $3} /ld-linux/ {print $1}' | \
        while read -r dep; do \
            [ -f "$dep" ] || continue; \
            cp -v "$dep" /out/lib/ 2>/dev/null || cp -v "$dep" /out/lib64/ 2>/dev/null || cp -v "$dep" /out/usr/lib/ 2>/dev/null || true; \
        done; \
    done

# Stage 3: Final distroless image.
FROM gcr.io/distroless/cc-debian12

# gammu-smsd uses system() for RunOnReceive, which requires /bin/sh.
COPY --from=busybox:stable-musl /bin/busybox /bin/busybox
RUN ["/bin/busybox", "ln", "-s", "/bin/busybox", "/bin/sh"]

# Copy gammu binaries and their libraries.
COPY --from=gammu-builder /out/usr/bin/gammu /usr/bin/gammu
COPY --from=gammu-builder /out/usr/bin/gammu-smsd /usr/bin/gammu-smsd
COPY --from=gammu-builder /out/usr/bin/gammu-smsd-inject /usr/bin/gammu-smsd-inject
COPY --from=gammu-builder /out/lib/ /lib/
COPY --from=gammu-builder /out/lib64/ /lib64/
COPY --from=gammu-builder /out/usr/lib/ /usr/lib/
COPY --from=gammu-builder /out/etc/ssl/certs /etc/ssl/certs

# Copy Go helpers.
COPY --from=go-builder /out/entrypoint /opt/entrypoint
COPY --from=go-builder /out/sms-telegram /opt/sms-telegram

# Copy config and create spool directories.
COPY gammu-smsdrc /etc/gammu-smsdrc.template
COPY --from=gammu-builder --chown=65532:65532 /out/var/spool/gammu /var/spool/gammu

# gammu-smsd needs a writable spool and config file.
VOLUME ["/var/spool/gammu", "/etc"]

USER 65532:65532

ENTRYPOINT ["/opt/entrypoint"]

