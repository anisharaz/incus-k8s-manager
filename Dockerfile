FROM node:26-bookworm AS frontend
WORKDIR /src/fe
RUN npm install -g pnpm
COPY fe/package.json fe/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY fe/ ./
RUN pnpm build

FROM golang:1.26-bookworm AS backend
WORKDIR /src/be
COPY be/go.mod be/go.sum ./
RUN go mod download
COPY be/ ./
COPY --from=frontend /src/fe/dist ./internal/webui/dist
RUN CGO_ENABLED=0 go build -o /out/koi ./cmd/server

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*
COPY --from=backend /out/koi /usr/local/bin/koi
EXPOSE 8000
ENTRYPOINT ["/usr/local/bin/koi"]
