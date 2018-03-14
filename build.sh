#!/bin/bash

if [[ $TRAVIS_OS_NAME == 'osx' ]]; then
    go build -o bcg-osx cmd/bcg.go
else
    go build -o bcg-linux cmd/bcg.go
fi