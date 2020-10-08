PLATFORM ?= local
DOCKER := DOCKER_BUILDKIT=1 docker

BIN_DIR ?= bin/

.PHONY: build
build:
	@$(DOCKER) build --target bin --output $(BIN_DIR) --platform ${PLATFORM} .

.PHONY: unit-test
unit-test:
	@$(DOCKER) build . --target unit-test

.PHONY: unit-test-coverage
unit-test-coverage:
	@$(DOCKER) build . --target unit-test-coverage --output coverage/
	@cat coverage/cover.out

.PHONY: lint
lint:
	@$(DOCKER) build . --target lint

.PHONY: test
test: lint unit-test

.PHONY: clean
clean:
	@rm -rf $(BIN_DIR)

REPO ?= docker.pkg.github.com/superbrothers-sandbox/try-envoy-grpc
NAME ?= hello
IMAGE := $(REPO)/$(NAME)
.PHONY: image-build
image-build:
	$(DOCKER) build -t $(IMAGE) .

.PHONY: image-push
image-push:
	@$(DOCKER) push $(IMAGE) 
