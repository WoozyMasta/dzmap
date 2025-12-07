# binaries build
FROM docker.io/golang:1.25-alpine AS build

# hadolint ignore=DL3018
RUN ["apk", "add", "--no-cache", "build-base", "make", "libwebp-dev"]
WORKDIR /src
COPY go.mod go.sum ./
RUN ["go", "mod", "download"]
COPY . ./
RUN ["make", "build"]
RUN echo "dzmap:x:1000:1000:dzmap:/:/sbin/nologin" > ./passwd && \
    echo "dzmap:x:1000:" > ./group

# create final root fs
FROM scratch AS rootfs

COPY --from=build /src/group /etc/group
COPY --from=build /src/passwd /etc/passwd
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /src/bin /bin
COPY --from=build /src/config.yaml /
WORKDIR /maps

# final binaries image
FROM scratch

USER 1000
WORKDIR /
ENV PATH=/bin \
    CONFIG_FILE=/config.yaml \
    LISTEN_ADDRESS=0.0.0.0 \
    LISTEN_PORT=8080 \
    ZOOM_LIMIT=6 \
    LOG_LEVEL=info
COPY --from=rootfs --chown=1000:1000 / /
ENTRYPOINT ["/bin/server"]
