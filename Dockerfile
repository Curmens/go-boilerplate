# Production image: static binary on distroless
FROM golang:1.23-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api . \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=build /out/api /out/migrate /app/

ENV GIN_MODE=release \
    SERVER_PORT=8080 \
    LOG_FILE_PATH=/tmp/logs

EXPOSE 8080
USER nonroot:nonroot

ENTRYPOINT ["/app/api"]
