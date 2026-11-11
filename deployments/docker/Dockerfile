# build the shared React frontend
FROM node:24-alpine AS web
WORKDIR /src/apps/frontend
COPY apps/frontend/package.json apps/frontend/package-lock.json ./
RUN npm ci
COPY apps/frontend/ ./
RUN npm run build

# build the Go server
FROM golang:1.27.4-alpine AS server
ARG VERSION=dev
WORKDIR /src
COPY apps/server/go.mod apps/server/go.sum ./apps/server/
RUN cd apps/server && GOWORK=off go mod download
COPY apps/server/ ./apps/server/
RUN cd apps/server && GOWORK=off CGO_ENABLED=0 go build -trimpath \
    -ldflags "-s -w -X main.version=${VERSION}" \
    -o /out/oryn-server ./cmd/oryn-server

# minimal runtime
FROM alpine:3.22
RUN addgroup -g 1000 -S oryn && adduser -u 1000 -G oryn -S oryn
WORKDIR /app
COPY --from=server /out/oryn-server /app/oryn-server
COPY --from=web /src/apps/frontend/dist /app/web
ENV ORYN_HTTP_ADDR=:8080 \
    ORYN_DATA_DIR=/data \
    ORYN_WEB_DIR=/app/web \
    ORYN_LOG_LEVEL=info
VOLUME ["/data"]
EXPOSE 8080
USER oryn
ENTRYPOINT ["/app/oryn-server"]
