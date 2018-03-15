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
if [[ $TRAVIS_OS_NAME == 'linux' ]]
then
   $GOPATH/bin/goveralls -v -service=travis-ci
fi  

go build -o bcg-$TRAVIS_OS_NAME cmd/bcg.go