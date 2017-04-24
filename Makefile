.PHONY:

EXECUTABLE ?= travis
IMAGE ?= bin/$(EXECUTABLE)
PACKAGES = $(shell go list ./...)

all: build

test:
	go test -race -cover --ldflags '${EXTLDFLAGS}' $(PACKAGES)

container:
	CGO_ENABLED=0 go build --ldflags '${EXTLDFLAGS}' -o ${IMAGE} github.com/stephenrlouie/travis/cmd

build:
	mkdir -p bin
	GOBIN=$(GOPATH)/src/github.com/stephenrlouie/travis/bin go install --ldflags '${EXTLDFLAGS}' github.com/stephenrlouie/travis/cmd
	mv bin/cmd ${IMAGE}
