############################################
# Healthcheck binary
############################################
FROM golang:1.23-bookworm AS build_healthcheck
WORKDIR /src
ARG TARGETOS=linux
ARG TARGETARCH=amd64
COPY ./extra/healthcheck.go ./healthcheck.go
RUN CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" \
    go build -trimpath -ldflags="-s -w" -o /out/healthcheck ./healthcheck.go

############################################
# Base runtime/build image
############################################
FROM node:22-bookworm-slim AS base
RUN apt-get update \
    && apt-get install --yes --no-install-recommends \
       curl \
       ca-certificates \
       gnupg \
       unzip \
       dumb-init \
    && install -m 0755 -d /etc/apt/keyrings \
    && curl -fsSL https://download.docker.com/linux/debian/gpg \
       | gpg --dearmor -o /etc/apt/keyrings/docker.gpg \
    && chmod a+r /etc/apt/keyrings/docker.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian $(. /etc/os-release && echo "$VERSION_CODENAME") stable" \
       > /etc/apt/sources.list.d/docker.list \
    && apt-get update \
    && apt-get install --yes --no-install-recommends docker-ce-cli docker-compose-plugin \
    && rm -rf /var/lib/apt/lists/* \
    && npm install -g tsx

############################################
# Node dependencies
############################################
FROM base AS build
WORKDIR /app
COPY --chown=node:node ./package.json ./package.json
COPY --chown=node:node ./package-lock.json ./package-lock.json
RUN npm ci --omit=dev

############################################
# Main image
############################################
FROM base AS release
WORKDIR /app

LABEL org.opencontainers.image.source="https://github.com/wkarts/dockge" \
      org.opencontainers.image.documentation="https://github.com/wkarts/dockge" \
      org.opencontainers.image.licenses="MIT"

COPY --chown=node:node --from=build_healthcheck /out/healthcheck /app/extra/healthcheck
COPY --from=build /app/node_modules /app/node_modules
COPY --chown=node:node . .
RUN mkdir -p /app/data \
    && chmod +x /app/extra/healthcheck

# Node/node-pty compatibility safeguard.
ENV UV_USE_IO_URING=0
ENV DOCKGE_IS_CONTAINER=1

VOLUME ["/app/data"]
EXPOSE 5001
HEALTHCHECK --interval=60s --timeout=30s --start-period=60s --retries=5 CMD ["/app/extra/healthcheck"]
ENTRYPOINT ["/usr/bin/dumb-init", "--"]
CMD ["tsx", "./backend/index.ts"]
