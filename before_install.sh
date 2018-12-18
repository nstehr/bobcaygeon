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
go install ./vendor/github.com/golang/protobuf/protoc-gen-go/
