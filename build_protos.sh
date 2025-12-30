#!/bin/bash

export GOBIN=$PWD/bin
export PATH=$GOBIN:$PATH
go install github.com/golang/protobuf/protoc-gen-go

protoc -I api/ --go_out=api --go-grpc_out=api api/bobcaygeon.proto
protoc -I cmd/mgmt/api --go_out=cmd/mgmt/api --go-grpc_out=cmd/mgmt/api cmd/mgmt/api/management.proto

