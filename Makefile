#!/usr/bin/make -f

VERSION := $(shell echo $(shell git describe --tags) | sed 's/^v//')
COMMIT := $(shell git log -1 --format='%H')

BUILDDIR ?= $(CURDIR)/build
export PROJECT_HOME=$(shell git rev-parse --show-toplevel)
export GO_PKG_PATH=$(HOME)/go/pkg
export GO111MODULE = on

# process build tags

LEDGER_ENABLED ?= true
build_tags = netgo
ifeq ($(LEDGER_ENABLED),true)
	ifeq ($(OS),Windows_NT)
		GCCEXE = $(shell where gcc.exe 2> NUL)
		ifeq ($(GCCEXE),)
			$(error gcc.exe not installed for ledger support, please install or set LEDGER_ENABLED=false)
		else
			build_tags += ledger
		endif
	else
		UNAME_S = $(shell uname -s)
		ifeq ($(UNAME_S),OpenBSD)
			$(warning OpenBSD detected, disabling ledger support (https://github.com/cosmos/cosmos-sdk/issues/1988))
		else
			GCC = $(shell command -v gcc 2> /dev/null)
			ifeq ($(GCC),)
				$(error gcc not installed for ledger support, please install or set LEDGER_ENABLED=false)
			else
				build_tags += ledger
			endif
		endif
	endif
endif

build_tags += $(BUILD_TAGS)
build_tags := $(strip $(build_tags))

whitespace :=
whitespace += $(whitespace)
comma := ,
build_tags_comma_sep := $(subst $(whitespace),$(comma),$(build_tags))

# process linker flags

ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=coolcat \
			-X github.com/cosmos/cosmos-sdk/version.ServerName=coolcat \
			-X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
			-X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT) \
			-X "github.com/cosmos/cosmos-sdk/version.BuildTags=$(build_tags_comma_sep)"

ifeq ($(LINK_STATICALLY),true)
	ldflags += -linkmode=external -extldflags "-Wl,-z,muldefs -static"
endif
ldflags += $(LDFLAGS)
ldflags := $(strip $(ldflags))

# BUILD_FLAGS := -tags "$(build_tags)" -ldflags '$(ldflags)' -race
BUILD_FLAGS := -tags "$(build_tags)" -ldflags '$(ldflags)'

#### Command List ####

all: lint install

install: go.sum
		go install $(BUILD_FLAGS) ./cmd/coolcat

go.sum: go.mod
		@echo "--> Ensure dependencies have not been modified"
		@go mod verify

lint:
	golangci-lint run
	find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" | xargs gofmt -d -s
	go mod verify

build:
	go build $(BUILD_FLAGS) -o ./build/coolcat ./cmd/coolcat

clean:
	rm -rf ./build

###############################################################################
###                                  Proto                                  ###
###############################################################################

proto-all: proto-format proto-gen

proto:
	@echo
	@echo "=========== Generate Message ============"
	@echo
	./scripts/protocgen.sh
	@echo
	@echo "=========== Generate Complete ============"
	@echo

test:
	@go test -v ./x/...

docs:
	@echo
	@echo "=========== Generate Message ============"
	@echo
	./scripts/generate-docs.sh

	statik -src=client/docs/static -dest=client/docs -f -m
	@if [ -n "$(git status --porcelain)" ]; then \
        echo "\033[91mSwagger docs are out of sync!!!\033[0m";\
        exit 1;\
    else \
        echo "\033[92mSwagger docs are in sync\033[0m";\
    fi
	@echo
	@echo "=========== Generate Complete ============"
	@echo
.PHONY: docs

protoVer=v0.9
protoImageName=osmolabs/osmo-proto-gen:$(protoVer)
containerProtoGen=cosmos-sdk-proto-gen-$(protoVer)
containerProtoFmt=cosmos-sdk-proto-fmt-$(protoVer)

proto-gen:
	@echo "Generating Protobuf files"
	@if docker ps -a --format '{{.Names}}' | grep -Eq "^${containerProtoGen}$$"; then docker start -a $(containerProtoGen); else docker run --name $(containerProtoGen) -v $(CURDIR):/workspace --workdir /workspace $(protoImageName) \
		sh ./scripts/protocgen.sh; fi

proto-format:
	@echo "Formatting Protobuf files"
	@if docker ps -a --format '{{.Names}}' | grep -Eq "^${containerProtoFmt}$$"; then docker start -a $(containerProtoFmt); else docker run --name $(containerProtoFmt) -v $(CURDIR):/workspace --workdir /workspace tendermintdev/docker-build-proto \
		find ./ -not -path "./third_party/*" -name "*.proto" -exec clang-format -i {} \; ; fi

proto-image-build:
	@DOCKER_BUILDKIT=1 docker build -t $(protoImageName) -f ./proto/Dockerfile ./proto

proto-image-push:
	docker push $(protoImageName)

###############################################################################
###                       Local testing using docker container              ###
###############################################################################
# To start a 4-node cluster from scratch:
# make clean && make docker-cluster-start
# To stop the 4-node cluster:
# make docker-cluster-stop
# If you have already built the binary, you can skip the build:
# make docker-cluster-start-skipbuild
###############################################################################

# Build docker image
build-docker-node:
	@cd docker && docker build --tag sei-chain/localnode localnode --platform linux/x86_64
.PHONY: build-docker-node

build-rpc-node:
	@cd docker && docker build --tag sei-chain/rpcnode rpcnode --platform linux/x86_64
.PHONY: build-rpc-node

# Run a single node docker container
run-local-node: kill-sei-node build-docker-node
	@rm -rf $(PROJECT_HOME)/build/generated
	docker run --rm \
	--name sei-node \
	--network host \
	-v $(PROJECT_HOME):/sei-protocol/sei-chain:Z \
	-v $(GO_PKG_PATH)/mod:/root/go/pkg/mod:Z \
	--platform linux/x86_64 \
	sei-chain/localnode
.PHONY: run-local-node

# Run a single rpc state sync node docker container
run-rpc-node: kill-rpc-node build-rpc-node
	docker run --rm \
	--name sei-rpc-node \
	--network docker_localnet \
	-v $(PROJECT_HOME):/sei-protocol/sei-chain:Z \
	-v $(PROJECT_HOME)/../sei-tendermint:/sei-protocol/sei-tendermint:Z \
    -v $(PROJECT_HOME)/../sei-cosmos:/sei-protocol/sei-cosmos:Z \
	-v $(GO_PKG_PATH)/mod:/root/go/pkg/mod:Z \
	-p 26668-26670:26656-26658 \
	--platform linux/x86_64 \
	sei-chain/rpcnode
.PHONY: run-rpc-node

# Run a 4-node docker containers
docker-cluster-start: docker-cluster-stop build-docker-node
	@rm -rf $(PROJECT_HOME)/build/generated
	@cd docker && NUM_ACCOUNTS=10 docker-compose up

.PHONY: localnet-start

# Use this to skip the seid build process
docker-cluster-start-skipbuild: docker-cluster-stop build-docker-node
	@rm -rf $(PROJECT_HOME)/build/generated
	@cd docker && NUM_ACCOUNTS=10 SKIP_BUILD=true docker-compose up
.PHONY: localnet-start

# Stop 4-node docker containers
docker-cluster-stop:
	@cd docker && docker-compose down
.PHONY: localnet-stop


# Implements test splitting and running. This is pulled directly from
# the github action workflows for better local reproducibility.

GO_TEST_FILES != find $(CURDIR) -name "*_test.go"

# default to four splits by default
NUM_SPLIT ?= 4

$(BUILDDIR):
	mkdir -p $@

# The format statement filters out all packages that don't have tests.
# Note we need to check for both in-package tests (.TestGoFiles) and
# out-of-package tests (.XTestGoFiles).
$(BUILDDIR)/packages.txt:$(GO_TEST_FILES) $(BUILDDIR)
	go list -f "{{ if (or .TestGoFiles .XTestGoFiles) }}{{ .ImportPath }}{{ end }}" ./... | sort > $@

split-test-packages:$(BUILDDIR)/packages.txt
	split -d -n l/$(NUM_SPLIT) $< $<.
test-group-%:split-test-packages
	cat $(BUILDDIR)/packages.txt.$* | xargs go test -mod=readonly -timeout=10m -race -coverprofile=$(BUILDDIR)/$*.profile.out