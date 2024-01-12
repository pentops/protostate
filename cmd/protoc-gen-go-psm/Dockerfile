FROM golang:1.21.4 AS builder

WORKDIR /src

LABEL org.opencontainers.image.source = "https://github.com/pentops/protostate"

ADD . .
ARG VERSION
RUN CGO_ENABLED=0 \
    GOOS=linux \
    go build \
    -ldflags="-X main.Version=$VERSION" \
    -o /protoc-gen-go-psm ./cmd/protoc-gen-go-psm

FROM scratch
COPY --from=builder /protoc-gen-go-psm /
ENTRYPOINT ["/protoc-gen-go-psm"]

