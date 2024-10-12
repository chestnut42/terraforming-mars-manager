export GOBIN := $(PWD)/bin
export PATH := $(GOBIN):$(PATH)

GOLANGLINT_VERSION := 1.61.0
PROTOC_VERSION := 28.2
GOOGLE_API_COMMIT := 0fa9ce880be5ea7c3027015849cd4fbfb04812c5
GRPC_GATEWAY_VERSION := $(shell cat go.mod | grep github.com/grpc-ecosystem/grpc-gateway/v2 | cut -w -f 3 | cut -c 2-)

ifeq ($(shell uname), Darwin)
	OS_NAME=osx
endif
ifeq ($(shell uname -m), arm64)
	ARCH_NAME=aarch_64
else
	ARCH_NAME=$(shell uname -m)
endif


.PHONY: default
default: all



./bin:
	mkdir -p ./bin


# Tools
./bin/goimports: | ./bin
	go install -modfile tools/go.mod golang.org/x/tools/cmd/goimports

./bin/golangci-lint: | ./bin
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ./bin v$(GOLANGLINT_VERSION)

./bin/gowrap: | ./bin
	go install -modfile tools/go.mod github.com/hexdigest/gowrap/cmd/gowrap

./bin/minimock: | ./bin
	go install -modfile tools/go.mod github.com/gojuno/minimock/v3/cmd/minimock


# Proto Tools
./bin/protoc-$(PROTOC_VERSION): | ./bin
	curl -sSfL https://github.com/protocolbuffers/protobuf/releases/download/v$(PROTOC_VERSION)/protoc-$(PROTOC_VERSION)-$(OS_NAME)-$(ARCH_NAME).zip -o ./bin/protoc-$(PROTOC_VERSION).zip
	unzip -q ./bin/protoc-$(PROTOC_VERSION).zip -d ./bin/protoc-$(PROTOC_VERSION)

./bin/protoc: | ./bin/protoc-$(PROTOC_VERSION)
	mv ./bin/protoc-$(PROTOC_VERSION)/bin/protoc ./bin/protoc

./bin/protoc-gen-go: | ./bin
	go install -modfile tools/go.mod google.golang.org/protobuf/cmd/protoc-gen-go

./bin/protoc-gen-go-grpc: | ./bin
	go install -modfile tools/go.mod google.golang.org/grpc/cmd/protoc-gen-go-grpc

./bin/protoc-gen-grpc-gateway: | ./bin
	go install -modfile tools/go.mod github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway

./bin/protoc-gen-openapiv2: | ./bin
	go install -modfile tools/go.mod github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2


# Proto Dependencies
./bin/googleapis-$(GOOGLE_API_COMMIT): | ./bin
	curl -sSfL https://github.com/googleapis/googleapis/archive/$(GOOGLE_API_COMMIT).zip -o ./bin/googleapis-$(GOOGLE_API_COMMIT).zip
	unzip -q ./bin/googleapis-$(GOOGLE_API_COMMIT).zip -d ./bin

./bin/include/google/api: ./bin/googleapis-$(GOOGLE_API_COMMIT)
	mkdir -p ./bin/include/google
	cp -R ./bin/googleapis-$(GOOGLE_API_COMMIT)/google/api ./bin/include/google

./bin/include/google/protobuf: ./bin/protoc-$(PROTOC_VERSION) | ./bin
	cp -R ./bin/protoc-$(PROTOC_VERSION)/include ./bin

./bin/grpc-gateway-$(GRPC_GATEWAY_VERSION): | ./bin
	curl -sSfL https://github.com/grpc-ecosystem/grpc-gateway/archive/refs/tags/v$(GRPC_GATEWAY_VERSION).zip -o ./bin/grpc-gateway-$(GRPC_GATEWAY_VERSION).zip
	unzip -q ./bin/grpc-gateway-$(GRPC_GATEWAY_VERSION).zip -d ./bin

./bin/include/protoc-gen-openapiv2/options: ./bin/grpc-gateway-$(GRPC_GATEWAY_VERSION) | ./bin
	mkdir -p ./bin/include/protoc-gen-openapiv2
	cp -R ./bin/grpc-gateway-$(GRPC_GATEWAY_VERSION)/protoc-gen-openapiv2/options ./bin/include/protoc-gen-openapiv2



.PHONY: clean
clean:
	rm -rf ./bin



.PHONY: all
all: generate lint test build

.PHONY: proto
proto: ./bin/protoc ./bin/protoc-gen-go ./bin/protoc-gen-go-grpc ./bin/protoc-gen-grpc-gateway ./bin/protoc-gen-openapiv2
proto: ./bin/include/google/api ./bin/include/google/protobuf ./bin/include/protoc-gen-openapiv2/options
	protoc \
		-I. -I./bin/include \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=. --grpc-gateway_opt=paths=source_relative \
		--openapiv2_out=. \
		./pkg/api/*.proto
	goimports -w -local github.com/farawaygg .

.PHONY: generate
generate: ./bin/gowrap ./bin/minimock ./bin/goimports
	go generate ./...
	goimports -w -local github.com/farawaygg .

.PHONY: lint
lint: ./bin/golangci-lint
	golangci-lint run -v ./...

.PHONY: test-prepare
test-prepare:
	docker rm -f terraforming-mars-postgres-test || true
	docker run --name terraforming-mars-postgres-test -e POSTGRES_PASSWORD=postgres -d -p 5432:5432 postgres:16.3

.PHONY: test
test: test-prepare
	GOGC=off go test -race $(GOFLAGS) -v ./... -count 1

.PHONY: build
build:
	GOGC=off go build -v -o ./bin/manager ./cmd/manager

.PHONY: tidy
tidy:
	go mod tidy
	cd tools && go mod tidy
