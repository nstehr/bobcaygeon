#!/bin/bash

sudo apt-get update
sudo apt-get install libasound2-dev

echo "Starting bcg build"
go build -o bcg-arm cmd/bcg.go
go build -o bcg-mgmt-arm cmd/mgmt/bcg-mgmt.go
go build -o bcg-frontend-arm cmd/frontend/bcg-frontend.go
