# Build-from-source image. The release images are built by GoReleaser from a
# pre-compiled binary (Dockerfile.goreleaser); this one is for `docker build`
# straight off a checkout. Multi-stage so the final image is scratch + the
# static binary + a CA bundle (~15 MB).
FROM golang:1.25-alpine AS build
WORKDIR /src
RUN apk add --no-cache make ca-certificates git
COPY . .
# The admin SPA and /docs content are committed and embedded, so no Node
# toolchain is needed here — just compile the CGO-free binary.
RUN CGO_ENABLED=0 make build VERSION=${VERSION:-docker}

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /src/bin/moth /moth
VOLUME ["/data"]
EXPOSE 8080
ENTRYPOINT ["/moth", "serve", "--data-dir", "/data"]
