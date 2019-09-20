#!/bin/bash

# hack for now
go mod vendor

docker run --rm --privileged multiarch/qemu-user-static:register --reset

docker run --rm -v "$PWD":/usr/gopath/src/github.com/nstehr/bobcaygeon -w /usr/gopath/src/github.com/nstehr/bobcaygeon balenalib/raspberry-pi-golang ./build-arm.sh
