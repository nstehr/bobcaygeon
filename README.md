# Bobcaygeon

[![Build Status](https://travis-ci.org/nstehr/bobcaygeon.svg?branch=master)](https://travis-ci.org/nstehr/bobcaygeon) [![Coverage Status](https://coveralls.io/repos/github/nstehr/bobcaygeon/badge.svg?branch=master)](https://coveralls.io/github/nstehr/bobcaygeon?branch=master) [![Go Report Card](https://goreportcard.com/badge/github.com/nstehr/bobcaygeon)](https://goreportcard.com/report/github.com/nstehr/bobcaygeon)

![gord downie](https://github.com/nstehr/bobcaygeon/blob/master/downie1a.jpg)

Multi room speaker application.

## Overview
Bobcaygeon is a multi-room speaker application.  Built on top of Apple airplay, the goal is an application that will run on a raspberry pi (or similar hardware) capable of playing streamed music on one or many hardware deployments.  With an initial goal of the same music on every speaker, and eventual goal of more fine grained control.

## Current Status
Full functional airplay server; Basic multi-room functionality.  Will stream to multiple clients.
Currently tested on OSX and on raspberry pi.

## Build
I've followed the practice of committing vendor (https://github.com/golang/dep/blob/master/docs/FAQ.md#should-i-commit-my-vendor-directory)
1. `go build cmd/bcg.go`

## Run
```
-name string
        The name for the service. (default "Bobcaygeon")
  -port int
        Set the port the service is listening to. (default 5000)
  -verbose
        Verbose logging; logs requests and response
  -clusterPort
        Port to listen for cluster events (default 7676)
```

## Raspberry Pi Notes
To get this to work on raspberry pi, you'll need to compile the code directly on the device.  This is
because [oto](https://github.com/hajimehoshi/oto) which is used for playing
music uses cgo, so it is tricky to cross-compile it for ARM. 

You will also need to enable ipv6 on your raspberry pi.  To do this, add `ipv6` to your `/etc/modules` and reboot
the pi.
