#!/bin/bash
$GOPATH/bin/goveralls -service=travis-ci
go build -o bcg-$TRAVIS_OS_NAME cmd/bcg.go