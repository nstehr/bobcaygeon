#!/bin/bash

# executes the tests in each package, and collects their coverage
echo "mode: set" > acc.out
for Dir in $(go list ./...); 
do
    if [[ ${Dir} != *"/vendor/"* ]]
    then
        returnval=`go test -coverprofile=profile.out $Dir`
        echo ${returnval}
        if [[ ${returnval} != *FAIL* ]]
        then
            if [ -f profile.out ]
            then
                cat profile.out | grep -v "mode: set" >> acc.out 
            fi
        else
            exit 1
        fi
    else
        exit 1
    fi  

done
if [[ $TRAVIS_OS_NAME == 'osx' ]]
then
   $GOPATH/bin/goveralls -coverprofile=acc.out -service=travis-ci
fi  

protoc -I api/ --go_out=plugins=grpc:api api/bobcaygeon.proto
protoc -I cmd/mgmt/api --go_out=plugins=grpc:cmd/mgmt/api cmd/mgmt/api/management.proto
protoc -I=cmd/mgmt/api cmd/mgmt/api/management.proto --js_out=import_style=commonjs:cmd/frontend/webui
protoc -I=cmd/mgmt/api cmd/mgmt/api/management.proto --grpc-web_out=import_style=commonjs,mode=grpcwebtext:cmd/frontend/webui
go build -o bcg-$TRAVIS_OS_NAME cmd/bcg.go
go build -o bcg-mgmt-$TRAVIS_OS_NAME cmd/mgmt/bcg-mgmt.go
go build -o bcg-frontend-$TRAVIS_OS_NAME cmd/frontend/bcg-frontend.go