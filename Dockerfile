FROM ghcr.io/rss3-network/go-image/go-builder:main-4782eed AS builder

WORKDIR /root/follow-image-proxy
ENV CGO_ENABLED=0
COPY . .

RUN go mod download -x
RUN go build -o build/follow-image-proxy main.go

FROM ghcr.io/rss3-network/go-image/go-runtime:main-4782eed AS runner

WORKDIR /root/follow-image-proxy

COPY --from=builder /root/follow-image-proxy/build/follow-image-proxy .

EXPOSE 8080

ENTRYPOINT ["./follow-image-proxy"]
