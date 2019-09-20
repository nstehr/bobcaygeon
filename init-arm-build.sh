#!/bin/bash

# hack for now
go mod vendor

docker run --rm --privileged multiarch/qemu-user-static:register --reset

docker run --rm -v "$PWD":/home/nstehr/bobcaygeon -w /home/nstehr/bobcaygeon balenalib/raspberry-pi-golang ./build-arm.sh
