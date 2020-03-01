#!/bin/bash
export GOBIN=$PWD/bin
export PATH=$GOBIN:$PATH
go install github.com/rakyll/statik

cd cmd/frontend
echo "packing web ui using rakyll/statik"
go generate

cd ../..

echo "building linux version"
env GOOS=linux GOARCH=amd64 go build -o bcg-frontend-linux cmd/frontend/bcg-frontend.go
echo "building arm version"
env GOOS=linux GOARCH=arm go build -o bcg-frontend-arm cmd/frontend/bcg-frontend.go
echo "building osx version"
env GOOS=darwin GOARCH=amd64 go build -o bcg-frontend-osx cmd/frontend/bcg-frontend.go
