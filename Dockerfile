FROM golang:1.26-alpine@sha256:7667e40de01ef81b25dea1a0a4e3a2d0e3b93e85bcf64a6e6e69a1e8badf323a AS builder
RUN apk --no-cache add git
RUN git clone --branch feat/action-integration --depth=1 https://github.com/Boeing/config-file-validator.git /src
WORKDIR /src
RUN CGO_ENABLED=0 go build -ldflags='-w -s' -o /validator cmd/validator/validator.go

FROM alpine:3.23@sha256:25109184c71bdad752c8312a8623239686a9a2071e8825f20acb8f2198c3f659
COPY --from=builder /validator /usr/local/bin/validator
COPY entrypoint.sh /entrypoint.sh
RUN chmod 0755 /usr/local/bin/validator && chmod 0755 /entrypoint.sh
WORKDIR /github/workspace
ENTRYPOINT ["/entrypoint.sh"]
