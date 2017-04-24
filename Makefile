.PHONY:

EXECUTABLE ?= travisTest
IMAGE ?= bin/$(EXECUTABLE)
PACKAGES = $(shell go list ./...)

all: build

test:
	go test -race -cover --ldflags '${EXTLDFLAGS}' $(PACKAGES)

container:
	CGO_ENABLED=0 go build --ldflags '${EXTLDFLAGS}' -o ${IMAGE} github.com/stephenrlouie/travisTest/cmd

build:
	mkdir -p bin
	GOBIN=$(GOPATH)/src/github.com/stephenrlouie/travisTest/bin go install --ldflags '${EXTLDFLAGS}' github.com/stephenrlouie/travisTest/cmd
	mv bin/cmd ${IMAGE}
