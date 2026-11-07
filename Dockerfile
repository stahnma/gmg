# Web UI
FROM node:24-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY web/ ./
RUN npm run build

# Server
FROM golang:1.26-alpine AS build
RUN apk add --no-cache libcap
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY web/embed.go ./web/
COPY --from=web /web/dist ./web/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/gmg ./cmd/gmg \
    # let the non-root user listen on port 80, also with --network host
    && setcap cap_net_bind_service=+ep /out/gmg \
    && mkdir -p /out/data

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/gmg /gmg
COPY --from=build --chown=nonroot:nonroot /out/data /data
ENV PORT=80 \
    GMG_DB_PATH=/data
VOLUME /data
EXPOSE 80
ENTRYPOINT ["/gmg"]
