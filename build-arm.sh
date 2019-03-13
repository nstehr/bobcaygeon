#!/bin/bash
sudo apt-get update
sudo apt-get install libasound2-dev
export GOPATH=/usr/gopath
go build -o bcg-arm cmd/bcg.go