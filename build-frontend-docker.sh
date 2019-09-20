#!/bin/bash

echo "Building web frontend"
docker run --rm -v "$PWD":/usr/gopath/src/github.com/nstehr/bobcaygeon -w /usr/gopath/src/github.com/nstehr/bobcaygeon node:10 sh -c 'cd cmd/frontend/webui && npm install && npm run build'
echo "Building go binaries"
docker run --rm -v "$PWD":/home/nstehr/bobcaygeon -w /home/nstehr/bobcaygeon golang:1.12-stretch ./build-frontend-binary.sh
