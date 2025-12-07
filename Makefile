BIN_DIR := bin
BIN_LOADER := $(BIN_DIR)/loader
BIN_CFG2JSON := $(BIN_DIR)/cfg2json
BIN_SERVER := $(BIN_DIR)/server

# Go Build settings
export CGO_ENABLED=1
LDVARS  :=
LDFLAGS := -s -w -linkmode external -extldflags -static $(LDVARS)
GOFLAGS := -buildvcs=false -trimpath
TAGS    := forceposix

# Container settings
VERSION ?= dev
GH_USER ?= woozymasta
DOCKER_USER ?= woozymasta

REGISTRY_GHCR := ghcr.io/$(GH_USER)
REGISTRY_DOCKER := docker.io/$(DOCKER_USER)
IMAGE_NAME := dzmap

IMG_BASE_GHCR := $(REGISTRY_GHCR)/$(IMAGE_NAME)
IMG_BASE_DOCKER := $(REGISTRY_DOCKER)/$(IMAGE_NAME)

# exe extension
ifeq ($(OS),Windows_NT)
	BIN_LOADER := $(BIN_LOADER).exe
	BIN_CFG2JSON := $(BIN_CFG2JSON).exe
	BIN_SERVER := $(BIN_SERVER).exe
endif

.PHONY: all build containers push-containers release clean fmt vet align lint check deps tools generate

all: check build

build: deps generate
	@echo ">> Building binaries..."
	@mkdir -p $(BIN_DIR)
	@echo "   [loader]   -> $(BIN_LOADER)"
	@go build $(GOFLAGS) -tags '$(TAGS)' -ldflags '$(LDFLAGS)' -o $(BIN_LOADER) ./cmd/loader
	@echo "   [cfg2json] -> $(BIN_CFG2JSON)"
	@go build $(GOFLAGS) -tags '$(TAGS)' -ldflags '$(LDFLAGS)' -o $(BIN_CFG2JSON) ./cmd/cfg2json
	@echo "   [server]   -> $(BIN_SERVER)"
	@go build $(GOFLAGS) -tags '$(TAGS)' -ldflags '$(LDFLAGS)' -o $(BIN_SERVER) ./cmd/server
	@echo ">> Build finished."

containers:
	@echo ">> Building containers (VERSION: $(VERSION))..."

	@echo "   [base]"
	@docker build -f Dockerfile \
		-t $(IMG_BASE_GHCR):$(VERSION) \
		-t $(IMG_BASE_GHCR):latest \
		-t $(IMG_BASE_DOCKER):$(VERSION) \
		-t $(IMG_BASE_DOCKER):latest \
		.

	@echo "   [vanilla]"
	@docker build -f Dockerfile.vanila \
		--build-arg DZMAP_IMAGE=$(IMG_BASE_GHCR) --build-arg DZMAP_TAG=$(VERSION) \
		-t $(IMG_BASE_GHCR):vanilla \
		-t $(IMG_BASE_DOCKER):vanilla \
		.

	@echo "   [slim]"
	@docker build -f Dockerfile.slim \
		--build-arg DZMAP_IMAGE=$(IMG_BASE_GHCR) --build-arg DZMAP_TAG=$(VERSION) \
		-t $(IMG_BASE_GHCR):slim \
		-t $(IMG_BASE_DOCKER):slim \
		.

	@echo "   [full]"
	@docker build -f Dockerfile.full \
		--build-arg DZMAP_IMAGE=$(IMG_BASE_GHCR) --build-arg DZMAP_TAG=slim \
		-t $(IMG_BASE_GHCR):full \
		-t $(IMG_BASE_DOCKER):full \
		.

push-containers:
	@echo ">> Pushing images..."
	docker push $(IMG_BASE_GHCR):$(VERSION)
	docker push $(IMG_BASE_GHCR):latest
	docker push $(IMG_BASE_DOCKER):$(VERSION)
	docker push $(IMG_BASE_DOCKER):latest

	docker push $(IMG_BASE_GHCR):vanilla
	docker push $(IMG_BASE_DOCKER):vanilla

	docker push $(IMG_BASE_GHCR):slim
	docker push $(IMG_BASE_DOCKER):slim

	docker push $(IMG_BASE_GHCR):full
	docker push $(IMG_BASE_DOCKER):full

release: containers push-containers
	@echo ">> Release $(VERSION) finished."

generate:
	@echo ">> Running generate..."
	@echo > assets/index.html
	@go run ./cmd/minify/main.go

deps:
	@echo ">> Downloading dependencies..."
	@go mod tidy
	@go mod download

clean:
	@echo ">> Cleaning..."
	@rm -rf $(BIN_DIR)

fmt:
	@echo ">> Running go fmt..."
	@go fmt ./...

vet: generate
	@echo ">> Running go vet..."
	@go vet ./...

align: generate
	@echo ">> Checking struct alignment..."
	@betteralign ./...

align-fix: generate
	@echo ">> Optimizing struct alignment..."
	@betteralign -apply ./...

lint: generate
	@echo ">> Running golangci-lint..."
	@golangci-lint run

check: fmt vet align lint
	@echo ">> All checks passed."

tools:
	@echo ">> Installing tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/dkorunic/betteralign/cmd/betteralign@latest
