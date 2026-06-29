FROM golang:1.26-alpine@sha256:3ad57304ad93bbec8548a0437ad9e06a455660655d9af011d58b993f6f615648 AS builder
RUN apk --no-cache add git
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
RUN CGO_ENABLED=0 go build -ldflags='-w -s' -o /entrypoint cmd/entrypoint/main.go

FROM alpine:3.24@sha256:a2d49ea686c2adfe3c992e47dc3b5e7fa6e6b5055609400dc2acaeb241c829f4
RUN apk --no-cache add git
COPY --from=builder /entrypoint /entrypoint
RUN chmod 0755 /entrypoint
WORKDIR /github/workspace
ENTRYPOINT ["/entrypoint"]
