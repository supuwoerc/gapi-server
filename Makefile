APP_NAME    := gapi-server
VERSION     := $(shell git describe --tags --always --dirty)
BUILD_TIME  := $(shell date -u '+%Y-%m-%d %H:%M:%S')
APP_ENV     ?= dev
GIN_MODE    ?= debug
GO_LDFLAGS  := -s -w \
	-X 'main.Version=$(VERSION)' \
	-X 'main.BuildTime=$(BUILD_TIME)' \
	-X 'main.AppEnv=$(APP_ENV)' \
	-X 'main.GinMode=$(GIN_MODE)'

.PHONY: build build-cli run run-cli clean docker wire wire-cli swagger

build:
	CGO_ENABLED=0 go build -ldflags "$(GO_LDFLAGS)" -o bin/$(APP_NAME) ./cmd/server

build-cli:
	CGO_ENABLED=0 go build -ldflags "$(GO_LDFLAGS)" -o bin/$(APP_NAME)-cli ./cmd/cli

run:
	go run ./cmd/server

run-cli:
	go run ./cmd/cli

clean:
	rm -rf bin/

docker:
	docker build \
		--build-arg APP_ENV=$(APP_ENV) \
		--build-arg GIN_MODE=$(GIN_MODE) \
		-t $(APP_NAME):$(VERSION) \
		-t $(APP_NAME):latest .

wire:
	wire ./cmd/server/ && wire ./cmd/cli/

wire-cli:
	wire ./cmd/cli/

swagger:
	swag init -d ./cmd/server,./internal/handler/v1,./pkg/response -g main.go -o docs/ --parseDependency
