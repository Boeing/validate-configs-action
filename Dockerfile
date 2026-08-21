FROM golang:1.26-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2 AS builder
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
