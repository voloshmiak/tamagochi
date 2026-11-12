# syntax=docker/dockerfile:1
# Збірка фронтенду та образ Caddy, що віддає статику й проксіює /api на бекенд.
# Контекст збірки — корінь репозиторію.

FROM node:22-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY web/ ./
RUN npm run build

FROM caddy:2-alpine
COPY deploy/Caddyfile /etc/caddy/Caddyfile
COPY --from=web /web/dist /srv