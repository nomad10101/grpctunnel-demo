.DEFAULT_GOAL := help

PROTOC_DIR := ./proto/hubspoke/v1/
PROTOC_SRC := $(wildcard $(PROTOC_DIR)/*)

protoc-go: $(PROTOC_SRC)  ## regenerate go proto/grpc stubs
	@echo  'Processing: "$(PROTOC_SRC)'
	protoc --proto_path=./proto/hubspoke/v1 --go_out=./gen_pb/ --go_opt=paths=source_relative --go-grpc_out=./gen_pb/ --go-grpc_opt=paths=source_relative endpoint_service.proto

build: protoc-go  ## build all go codebase
	go build network_client.go
	go build network_server.go
