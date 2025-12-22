FROM alpine:3.23@sha256:865b95f46d98cf867a156fe4a135ad3fe50d2056aa3f25ed31662dff6da4eb62
COPY entrypoint.sh /entrypoint.sh
ENV CFV_VERSION=v1.8.1
RUN apk --no-cache add curl tar && \
  curl https://github.com/Boeing/config-file-validator/releases/download/${CFV_VERSION}/validator-${CFV_VERSION}-linux-386.tar.gz \
  -o /tmp/validator-${CFV_VERSION}-linux-386.tar.gz  -s -L && \
  tar -xvf /tmp/validator-${CFV_VERSION}-linux-386.tar.gz -C /tmp && \
  mv /tmp/validator /usr/local/bin && \
  rm -rf /tmp/* && \
  chmod 0755 /usr/local/bin/validator && \
  chmod 0755 /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
