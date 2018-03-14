#!/bin/bash
if [[ $TRAVIS_OS_NAME == 'osx' ]]; then
  $GOPATH/bin/goveralls -service=travis-ci
fi

go build -o bcg-$TRAVIS_OS_NAME cmd/bcg.go