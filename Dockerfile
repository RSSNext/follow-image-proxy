FROM ghcr.io/rss3-network/go-image/go-builder:main-4782eed AS base

WORKDIR /root/follow-image-proxy

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download -x

COPY . .

FROM base AS builder

ENV CGO_ENABLED=0
RUN --mount=type=cache,target=/go/pkg/mod/ \
    go build -o build/follow-image-proxy main.go

FROM ghcr.io/rss3-network/go-image/go-runtime:main-4782eed AS runner

WORKDIR /root/follow-image-proxy

COPY --from=builder /root/follow-image-proxy/build/follow-image-proxy .

EXPOSE 8080

ENTRYPOINT ["./follow-image-proxy"]
