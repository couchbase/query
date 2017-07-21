#!/bin/bash

cd $GOPATH/src/github.com/couchbase/query/test/gsi
for i in test_cases/*
do
    source ./exportval.sh $*
    cd $i
    go test ./...
    cd ../..
    source ./resetval.sh $*
done

