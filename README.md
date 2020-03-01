# Bobcaygeon

[![Build Status](https://travis-ci.org/nstehr/bobcaygeon.svg?branch=master)](https://travis-ci.org/nstehr/bobcaygeon) [![Coverage Status](https://coveralls.io/repos/github/nstehr/bobcaygeon/badge.svg?branch=master)](https://coveralls.io/github/nstehr/bobcaygeon?branch=master) [![Go Report Card](https://goreportcard.com/badge/github.com/nstehr/bobcaygeon)](https://goreportcard.com/report/github.com/nstehr/bobcaygeon)

![gord downie](https://github.com/nstehr/bobcaygeon/blob/master/downie1a.jpg)

Multi room speaker application.

## Overview
Bobcaygeon is a multi-room speaker application.  Built on top of Apple airplay, Bobcaygeon is an application (more specifically a set of applications) that will run on a raspberry pi (or similar hardware) capable of playing streamed music on one or many hardware deployments. 

## Current Status
Full functional airplay server; Basic multi-room functionality.  Will stream to multiple clients.  
Standalone frontend to provide a basic UI into the running cluster of speakers.  High level API to control speakers
and build zones.

Currently tested on OSX, raspberry pi and linux x86.


## Build

1. `go build cmd/bcg.go`
2. `go build cmd/mgmt/bcg-mgmt.go`

### Frontend build
Install rakyll/statik

```
export GOBIN=$PWD/bin
export PATH=$GOBIN:$PATH
go install github.com/rakyll/statik
```

```
1. cd cmd/frontend/webui
2. npm install && npm run build
3. cd ..
4. go generate
5. cd ../..
6. go build cmd/frontend/bcg-frontend.go
```

To regenerate the the grpc service:
`protoc -I api/ --go_out=plugins=grpc:api api/bobcaygeon.proto`
`protoc -I cmd/mgmt/api --go_out=plugins=grpc:cmd/mgmt/api cmd/mgmt/api/management.proto`
`protoc -I=cmd/mgmt/api cmd/mgmt/api/management.proto --js_out=import_style=commonjs:cmd/frontend/webui`
`protoc -I=cmd/mgmt/api cmd/mgmt/api/management.proto --grpc-web_out=import_style=commonjs,mode=grpcwebtext:cmd/frontend/webui`

Or use `build_protos.sh`


## Run
```
-config string
        Config file to run the service, see `bcg.toml` and `bcg-mgmt.toml`
  -verbose
        Verbose logging; logs requests and response
```

## Usage
There are a couple of ways you can run the bobcaygeon system.
1. Install one or more instances of the `bcg` application on your pi's/computers.  By default, the first instance of 
a `bcg` application in the cluster will act as the leader, and every subsequent instance will join in.  This is the simplest way to get multi-room streaming, put a pi in each room, load up `bcg` on each one, and then you can connect over airplay.
2. The slighly more advanced method of deploying atleast one `bcg-mgmt` and `bcg-frontend` instance.  This will give you both a management API and a simple frontend web UI.  If you want to use the web ui provided by `bcg-frontend` you'll also need to start an instance of the Envoy proxy.  You can use the `launch_envoy.sh` script for that.

## API
There are two layers of API to interact with, if you would like.  Both are built on grpc.
1. Each instance of `bcg` has a basic GRPC API: https://github.com/nstehr/bobcaygeon/blob/master/api/bobcaygeon.proto
2. `bcg-mgmt` has a richer management GRPC API: https://github.com/nstehr/bobcaygeon/blob/master/cmd/mgmt/api/management.proto

## Raspberry Pi Notes
You can grab the `bcg-arm` build and drop it on your raspberry pi.  You'll need to make sure you
have ALSA setup, with the development headers (libasound2-dev)

You need to enable ipv6 on your raspberry pi.  To do this, add `ipv6` to your `/etc/modules` and reboot
the pi.

## Dev Builds
Dev builds can be found in the google storage bucket here: [bcg_artifacts](https://storage.googleapis.com/bcg_artifacts).  This will be the most up to date builds, and they should be relatively stable
