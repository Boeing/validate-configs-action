FROM golang:1.26-alpine@sha256:f23e8b227fb4493eabe03bede4d5a32d04092da71962f1fb79b5f7d1e6c2a17f AS builder
RUN apk --no-cache add git
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
RUN CGO_ENABLED=0 go build -ldflags='-w -s' -o /entrypoint cmd/entrypoint/main.go

FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b
RUN apk --no-cache add git
COPY --from=builder /entrypoint /entrypoint
RUN chmod 0755 /entrypoint
WORKDIR /github/workspace
ENTRYPOINT ["/entrypoint"]
