export GOBIN := $(PWD)/bin
export PATH := $(GOBIN):$(PATH)

GOLANGLINT_VERSION := 1.59.1
PROTOC_VERSION = 27.2

ifeq ($(shell uname), Darwin)
	OS_NAME=osx
endif
ARCH_NAME=$(shell uname -m)


.PHONY: default
default: all



./bin:
	mkdir -p ./bin

./bin/goimports: | ./bin
	go install -modfile tools/go.mod golang.org/x/tools/cmd/goimports

./bin/golangci-lint: | ./bin
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ./bin v$(GOLANGLINT_VERSION)

./bin/gowrap: | ./bin
	go install -modfile tools/go.mod github.com/hexdigest/gowrap/cmd/gowrap

./bin/minimock: | ./bin
	go install -modfile tools/go.mod github.com/gojuno/minimock/v3/cmd/minimock

./bin/protoc: | ./bin
	curl -sSfL https://github.com/protocolbuffers/protobuf/releases/download/v$(PROTOC_VERSION)/protoc-$(PROTOC_VERSION)-$(OS_NAME)-$(ARCH_NAME).zip -o ./bin/protoc-$(PROTOC_VERSION).zip
	unzip ./bin/protoc-$(PROTOC_VERSION).zip -d ./bin/protoc-$(PROTOC_VERSION)
	mv ./bin/protoc-$(PROTOC_VERSION)/bin/protoc ./bin/protoc

./bin/protoc-get-go: | ./bin
	go install -modfile tools/go.mod google.golang.org/protobuf/cmd/protoc-gen-go

./bin/protoc-gen-go-grpc: | ./bin
	go install -modfile tools/go.mod google.golang.org/grpc/cmd/protoc-gen-go-grpc

.PHONY: clean
clean:
	rm -rf ./bin



.PHONY: all
all: generate lint test build

.PHONY: genproto
genproto: ./bin/protoc ./bin/protoc-get-go ./bin/protoc-gen-go-grpc
	protoc \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		./pkg/api/users.proto

.PHONY: generate
generate: ./bin/gowrap ./bin/minimock ./bin/goimports
	go generate ./...
	goimports -w -local github.com/farawaygg .

.PHONY: lint
lint: ./bin/golangci-lint
	golangci-lint run -v ./...

.PHONY: test
test:
	GOGC=off go test -race $(GOFLAGS) -v ./... -count 1

.PHONY: build
build:
	GOGC=off go build -v -o ./bin/manager ./cmd/manager

.PHONY: tidy
tidy:
	go mod tidy
	cd tools && go mod tidy
