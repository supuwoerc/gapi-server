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

.PHONY: build run clean docker wire

build:
	CGO_ENABLED=0 go build -ldflags "$(GO_LDFLAGS)" -o bin/$(APP_NAME) ./cmd/server

run:
	go run ./cmd/server

clean:
	rm -rf bin/

docker:
	docker build \
		--build-arg APP_ENV=$(APP_ENV) \
		--build-arg GIN_MODE=$(GIN_MODE) \
		-t $(APP_NAME):$(VERSION) \
		-t $(APP_NAME):latest .

wire:
	wire ./cmd/server/
