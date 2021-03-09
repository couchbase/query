#!/bin/bash
verbose=$1
Site=http://127.0.0.1:8091/pools/nodes/
Auth=Administrator:password

export GO111MODULE=off
export CGO_CFLAGS="-I$GOPATH/src/github.com/couchbase/eventing-ee/evaluator/worker/include $CGO_FLAGS"
export CGO_LDFLAGS="-L$GOPATH/lib $CGO_LDLAGS"
export LD_LIBRARY_PATH=$GOPATH/lib:$LD_LIBRARY_PATH

cd $GOPATH/src/github.com/couchbase/query/test/gsi
for i in test_cases/*
do
    source ./exportval.sh $*
    cd $i
    go test $verbose -p 1 -tags enterprise ./...
    cd ../..
    source ./resetval.sh $*
done
