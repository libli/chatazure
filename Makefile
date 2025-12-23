.DEFAULT_GOAL := build
.PHONY: login build clean

IMAGE_NAME := libli/chatazure
DOCKER_REPO := docker.io
VERSION := 1.3
BUILDER_NAME := multiarch-builder

# 定义要支持的平台
PLATFORMS := linux/amd64,linux/arm64/v8

login:
	@docker login -u libli -p $(DOCKER_PASSWORD) $(DOCKER_REPO)
create-builder:
	@if ! docker buildx inspect $(BUILDER_NAME) > /dev/null 2>&1; then\
		docker buildx create --name $(BUILDER_NAME) --use;\
	fi
build: login create-builder
	docker buildx build --platform $(PLATFORMS) \
		-t $(DOCKER_REPO)/$(IMAGE_NAME):latest \
		-t $(DOCKER_REPO)/$(IMAGE_NAME):$(VERSION) \
		--progress=plain --push .
clean:
	-docker rmi $(docker images -q $(DOCKER_REPO)/$(IMAGE_NAME)) 2>/dev/null || true
