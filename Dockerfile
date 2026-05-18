FROM golang:1.26-alpine@sha256:91eda9776261207ea25fd06b5b7fed8d397dd2c0a283e77f2ab6e91bfa71079d AS builder
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
