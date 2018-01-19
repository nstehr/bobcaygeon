Bobcaygeon
=========
![gord downie](https://github.com/nstehr/bobcaygeon/blob/master/downie1a.jpg)

Multi room speaker application.

## Overview
Bobcaygeon is a multi-room speaker application.  Built on top of Apple airplay, the goal is an application that will run on a raspberry pi (or similar hardware) capable of playing streamed music on one or many hardware deployments.  With an initial goal of the same music on every speaker, and eventual goal of more fine grained control.

## Current Status
Full functional airplay server; no multi-room capability.
Currently tested on OSX, not on raspberry pi yet

## Build
I've followed the practice of committing vendor (https://github.com/golang/dep/blob/master/docs/FAQ.md#should-i-commit-my-vendor-directory)
1. `go build`

## Run
```
-name string
        The name for the service. (default "Bobcaygeon")
  -port int
        Set the port the service is listening to. (default 5000)
  -verbose
        Verbose logging; logs requests and responses
```
