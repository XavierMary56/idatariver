FROM golang:1.23 AS build
WORKDIR /app

COPY . .

RUN go mod tidy
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/worker ./cmd/worker
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/migrate ./cmd/migrate

# ---- api image ----
FROM gcr.io/distroless/static-debian12:latest AS api
WORKDIR /
COPY --from=build /out/api /api
ENTRYPOINT ["/api"]

# ---- worker image ----
FROM gcr.io/distroless/static-debian12:latest AS worker
WORKDIR /
COPY --from=build /out/worker /worker
ENTRYPOINT ["/worker"]

# ---- migrate image ----
FROM gcr.io/distroless/static-debian12:latest AS migrate
WORKDIR /app
COPY --from=build /out/migrate /migrate
COPY --from=build /app/migrations /app/migrations
ENTRYPOINT ["/migrate"]