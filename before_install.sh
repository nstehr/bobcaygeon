#!/bin/bash

#TODO: refactor these, copy and pasting here galore!
if [[ $TRAVIS_OS_NAME == 'linux' ]]
then
   PROTOC_ZIP=protoc-3.6.1-linux-x86_64.zip
   curl -OL https://github.com/google/protobuf/releases/download/v3.6.1/$PROTOC_ZIP
   sudo unzip -o $PROTOC_ZIP -d /usr/local bin/protoc
   rm -f $PROTOC_ZIP
   # plugin for web proto generation
   WEB_PROTOC_ZIP=protoc-gen-grpc-web-1.0.3-linux-x86_64
   curl -OL https://github.com/grpc/grpc-web/releases/download/1.0.3/$WEB_PROTOC_ZIP
   sudo mv $WEB_PROTOC_ZIP /usr/local/bin/protoc-gen-grpc-web
   chmod +x /usr/local/bin/protoc-gen-grpc-web
fi

if [[ $TRAVIS_OS_NAME == 'osx' ]]
then
   PROTOC_ZIP=protoc-3.6.1-osx-x86_64.zip
   curl -OL https://github.com/google/protobuf/releases/download/v3.6.1/$PROTOC_ZIP
   sudo unzip -o $PROTOC_ZIP -d /usr/local bin/protoc
   rm -f $PROTOC_ZIP
   # plugin for web proto generation
   WEB_PROTOC_ZIP=protoc-gen-grpc-web-1.0.3-darwin-x86_64
   curl -OL https://github.com/grpc/grpc-web/releases/download/1.0.3/$WEB_PROTOC_ZIP
   sudo mv $WEB_PROTOC_ZIP /usr/local/bin/protoc-gen-grpc-web
   chmod +x /usr/local/bin/protoc-gen-grpc-web
fi

go get github.com/mattn/goveralls
export GOBIN=$PWD/bin
export PATH=$GOBIN:$PATH
go install github.com/golang/protobuf/protoc-gen-go

protoc -I api/ --go_out=plugins=grpc:api api/bobcaygeon.proto
protoc -I cmd/mgmt/api --go_out=plugins=grpc:cmd/mgmt/api cmd/mgmt/api/management.proto
protoc -I=cmd/mgmt/api cmd/mgmt/api/management.proto --js_out=import_style=commonjs:cmd/frontend/webui/src/api
protoc -I=cmd/mgmt/api cmd/mgmt/api/management.proto --grpc-web_out=import_style=commonjs,mode=grpcwebtext:cmd/frontend/webui/src/api