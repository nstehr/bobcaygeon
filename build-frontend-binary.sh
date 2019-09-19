#!/bin/bash

sudo apt-get update
sudo apt-get install libasound2-dev

export GOBIN=$PWD/bin
export PATH=$GOBIN:$PATH
go install github.com/gobuffalo/packr/v2/packr2

cd cmd/frontend
echo "packing web ui using packr2"
packr2

cd ../..

echo "building linux version"
env GOOS=linux GOARCH=amd64 go build -o bcg-frontend-linux cmd/frontend/bcg-frontend.go
