VERSION := $(shell echo $(shell git describe --tags) | sed 's/^v//')
COMMIT  := $(shell git log -1 --format='%H')
TMVERSION := $(shell go list -m -u -f '{{.Version}}' github.com/tendermint/tendermint)
all: install

###############################################################################
# Build / Install
###############################################################################

LD_FLAGS = -X github.com/jackzampolin/cosmos-registrar/cmd.Version=$(VERSION) \
	-X github.com/jackzampolin/cosmos-registrar/cmd.Commit=$(COMMIT) \
	-X github.com/jackzampolin/cosmos-registrar/cmd.TMVersion=$(TMVERSION) 

BUILD_FLAGS := -ldflags '$(LD_FLAGS)'

build: go.sum
ifeq ($(OS),Windows_NT)
	@echo "building registrar binary..."
	@go build -mod=readonly $(BUILD_FLAGS) -o build/registrar.exe main.go
else
	@echo "building registrar binary..."
	@go build -mod=readonly $(BUILD_FLAGS) -o build/registrar main.go
endif

install: go.sum
	@echo "installing registrar binary..."
	@go build -mod=readonly $(BUILD_FLAGS) -o $${GOBIN-$${GOPATH-$$HOME/go}/bin}/registrar main.go

test:
	$(eval GOPACKAGES = $(shell go list ./...  | grep -v /vendor/))
	@go test $(GOPACKAGES) -v -race -coverprofile=cover.out -covermode=atomic


.PHONY: install build lint coverage clean