# Build
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/hefai ./cmd/hefai

# Run
FROM alpine:3.20
RUN adduser -D -u 1000 hefai && apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/hefai /app/hefai
COPY db/migrations /app/db/migrations
COPY ai-docs /app/ai-docs
RUN mkdir -p /app/data/files && chown -R hefai /app/data
USER hefai
ENV MIGRATIONS_DIR=/app/db/migrations \
    AI_DOCS_DIR=/app/ai-docs \
    FILE_STORE_DIR=/app/data/files \
    HTTP_ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["/app/hefai"]
