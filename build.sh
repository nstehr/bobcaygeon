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

# build the UI code
cd cmd/frontend/webui
npm install
npm run build

# pack the UI using packr
cd ..
echo "Packing UI using packr"
packr2
cd ../..

go build -o bcg-$TRAVIS_OS_NAME cmd/bcg.go
go build -o bcg-mgmt-$TRAVIS_OS_NAME cmd/mgmt/bcg-mgmt.go
go build -o bcg-frontend-$TRAVIS_OS_NAME cmd/frontend/bcg-frontend.go

#TODO: refactor linux build overall
if [[ $TRAVIS_OS_NAME == 'linux' ]]
then
   echo "executing docker based ARM build"
   ./init-arm-build.sh
fi  

