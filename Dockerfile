FROM alpine:3.23@sha256:25109184c71bdad752c8312a8623239686a9a2071e8825f20acb8f2198c3f659
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
