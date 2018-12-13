# Bobcaygeon

[![Build Status](https://travis-ci.org/nstehr/bobcaygeon.svg?branch=master)](https://travis-ci.org/nstehr/bobcaygeon) [![Coverage Status](https://coveralls.io/repos/github/nstehr/bobcaygeon/badge.svg?branch=master)](https://coveralls.io/github/nstehr/bobcaygeon?branch=master)

![gord downie](https://github.com/nstehr/bobcaygeon/blob/master/downie1a.jpg)

Multi room speaker application.

## Overview
Bobcaygeon is a multi-room speaker application.  Built on top of Apple airplay, the goal is an application that will run on a raspberry pi (or similar hardware) capable of playing streamed music on one or many hardware deployments.  With an initial goal of the same music on every speaker, and eventual goal of more fine grained control.

## Current Status
Full functional airplay server; Basic multi-room functionality.  Will stream to multiple clients.
Currently tested on OSX, not on raspberry pi yet

## Build
I've followed the practice of committing vendor (https://github.com/golang/dep/blob/master/docs/FAQ.md#should-i-commit-my-vendor-directory)
1. `go build cmd/bcg.go`

To regenerate the the grpc service:
`protoc -I api/ --go_out=plugins=grpc:api api/bobcaygeon.proto`
`protoc -I cmd/mgmt/api --go_out=plugins=grpc:cmd/mgmt/api cmd/mgmt/api/management.proto`
`protoc -I=cmd/mgmt/api cmd/mgmt/api/management.proto --js_out=import_style=commonjs:cmd/frontend/webui`
`protoc -I=cmd/mgmt/api cmd/mgmt/api/management.proto --grpc-web_out=import_style=commonjs,mode=grpcwebtext:cmd/frontend/webui`


## Run
```
-config string
        Config file to run the service, see `bcg.toml` and `bcg-mgmt.toml`
  -verbose
        Verbose logging; logs requests and response
```
