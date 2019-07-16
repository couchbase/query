#!/bin/bash
verbose=$1
Site=http://127.0.0.1:8091/pools/nodes/
Auth=Administrator:password

cd $GOPATH/src/github.com/couchbase/query/test/gsi
for i in test_cases/*
do
    source ./exportval.sh $*
    cd $i
    go test $verbose -p 1 -tags enterprise ./...
    cd ../..
    source ./resetval.sh $*
done
