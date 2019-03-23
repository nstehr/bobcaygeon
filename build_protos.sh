#!/bin/bash

export GOBIN=$PWD/bin
export PATH=$GOBIN:$PATH
go install github.com/golang/protobuf/protoc-gen-go

protoc -I api/ --go_out=plugins=grpc:api api/bobcaygeon.proto
protoc -I cmd/mgmt/api --go_out=plugins=grpc:cmd/mgmt/api cmd/mgmt/api/management.proto
protoc -I=cmd/mgmt/api cmd/mgmt/api/management.proto --js_out=import_style=commonjs:cmd/frontend/webui/src/api
protoc -I=cmd/mgmt/api cmd/mgmt/api/management.proto --grpc-web_out=import_style=commonjs,mode=grpcwebtext:cmd/frontend/webui/src/api