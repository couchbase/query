#!/bin/bash

cd $GOPATH/src/github.com/couchbase/query/test/gsi
source ./exportval.sh $*
go test ./...
source ./resetval.sh $*


