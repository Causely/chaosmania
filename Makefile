SHELL := /bin/bash

GO_SRC = cmd/chaosmania/main.go
GO_BIN = chaosmania
DOCKER_REGISTRY = quay.io/causely
DOCKER_IMAGE = chaosmania
LINUX_ARCH=amd64 arm64

.PHONY: build

build: $(LINUX_ARCH)

$(LINUX_ARCH): $(GO_SRC)
	GOOS=linux GOARCH=$@ go build -o ./out/$(GO_BIN)-linux-$@ ./cmd/chaosmania

image: build Dockerfile
	docker buildx build --platform linux/amd64,linux/arm64 --push -t $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):latest .
