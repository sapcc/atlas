IMAGE   ?= hub.global.cloud.sap/monsoon/atlas
VERSION = $(shell git rev-parse --verify HEAD | head -c 8)

GOOS    ?= $(shell go env | grep GOOS | cut -d'"' -f2)
BINARY  := atlas

LDFLAGS := -X github.com/sapcc/atlas/pkg/atlas.VERSION=$(VERSION)
GOFLAGS := -mod vendor -ldflags "-s -w $(LDFLAGS)"

SRCDIRS  := cmd pkg internal
PACKAGES := $(shell find $(SRCDIRS) -type d)
GOFILES  := $(addsuffix /*.go,$(PACKAGES))
GOFILES  := $(wildcard $(GOFILES))


all: bin/$(GOOS)/$(BINARY)

bin/%/$(BINARY): $(GOFILES) Makefile
	GOOS=$* GOARCH=amd64 go build $(GOFLAGS) -o bin/$*/$(BINARY) ./cmd/atlas

build:
	docker build -t $(IMAGE):$(VERSION) .

push: build
	docker push $(IMAGE):$(VERSION)

clean:
	rm -rf bin/*

vendor:
	GO111MODULE=on go get -u ./... && go mod tidy && go mod vendor
