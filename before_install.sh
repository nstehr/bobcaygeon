#!/bin/bash

if [[ $TRAVIS_OS_NAME == 'linux' ]]
then
   PROTOC_ZIP=protoc-3.3.0-linux-x86_64.zip
   curl -OL https://github.com/google/protobuf/releases/download/v3.3.0/$PROTOC_ZIP
   sudo unzip -o $PROTOC_ZIP -d /usr/local bin/protoc
   rm -f $PROTOC_ZIP
fi

if [[ $TRAVIS_OS_NAME == 'osx' ]]
then
   PROTOC_ZIP=protoc-3.3.0-osx-x86_64.zip
   curl -OL https://github.com/google/protobuf/releases/download/v3.3.0/$PROTOC_ZIP
   sudo unzip -o $PROTOC_ZIP -d /usr/local bin/protoc
   rm -f $PROTOC_ZIP
fi

go get github.com/mattn/goveralls
go get -u github.com/golang/protobuf/protoc-gen-go
protoc -I api/ --go_out=plugins=grpc:api api/bobcaygeon.proto