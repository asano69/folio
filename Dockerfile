# Stage 1: Go
FROM golang:1.26.1-alpine AS builder
WORKDIR /build
COPY go.mod go.sum* ./
RUN go mod download || true
# Copy only Go source so the build cache is not invalidated by changes to
# static assets or other non-Go files.
COPY cmd/ ./cmd/
COPY internal/ ./internal/
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o folio ./cmd/folio

# Stage 2: runtime
FROM alpine:3.23
RUN apk add --no-cache ca-certificates su-exec busybox-extras tzdata

COPY --from=builder /build/folio /usr/local/bin/folio
COPY static/ /folio/static/
COPY templates/ /folio/templates/

WORKDIR /folio
EXPOSE 3000
ENTRYPOINT ["folio", "serve"]
