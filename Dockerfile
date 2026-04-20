FROM golang:1.26-alpine@sha256:f85330846cde1e57ca9ec309382da3b8e6ae3ab943d2739500e08c86393a21b1 AS builder
RUN apk --no-cache add git
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
RUN CGO_ENABLED=0 go build -ldflags='-w -s' -o /entrypoint cmd/entrypoint/main.go

FROM alpine:3.23@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11
COPY --from=builder /entrypoint /entrypoint
RUN chmod 0755 /entrypoint
WORKDIR /github/workspace
ENTRYPOINT ["/entrypoint"]
