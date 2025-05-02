FROM apache/apisix:3.9.0-debian

USER root

RUN adduser --home /sandbox --disabled-password --gecos '' sandbox \
    && chown -R sandbox:sandbox /usr/local/apisix/ \
    && apt-get update -y \
    && apt-get install -y curl \
    && rm -rf /var/lib/apt/lists/*

USER sandbox
WORKDIR /sandbox

COPY apisix.yaml /usr/local/apisix/conf/apisix.yaml
COPY config.yaml /usr/local/apisix/conf/config.yaml
