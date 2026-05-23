# ---- Build Stage ----
FROM golang:1.26-alpine AS builder

ARG APP_ENV=dev
ARG GIN_MODE=debug

RUN apk add --no-cache git
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w -X 'main.AppEnv=${APP_ENV}' -X 'main.GinMode=${GIN_MODE}'" \
    -o /app/gapi-server ./cmd/server

# ---- Runtime Stage ----
FROM alpine:3.20

ARG APP_ENV=dev
ARG GIN_MODE=debug

RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai
ENV APP_ENV=${APP_ENV}
ENV GIN_MODE=${GIN_MODE}

WORKDIR /app
COPY --from=builder /app/gapi-server .
COPY configs/ ./configs/
COPY pkg/locale/ ./pkg/locale/
EXPOSE 8080
ENTRYPOINT ["./gapi-server"]
