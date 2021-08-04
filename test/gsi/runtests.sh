#!/bin/bash


# Copyright 2016-Present Couchbase, Inc.
#
# Use of this software is governed by the Business Source License included in
# the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
# file, in accordance with the Business Source License, use of this software will
# be governed by the Apache License, Version 2.0, included in the file
# licenses/APL.txt.

verbose=$1
Site=http://127.0.0.1:8091/pools/nodes/
Auth=Administrator:password

export GO111MODULE=off
export CGO_CFLAGS="-I$GOPATH/src/github.com/couchbase/eventing-ee/evaluator/worker/include $CGO_FLAGS"
export CGO_LDFLAGS="-L$GOPATH/lib $CGO_LDLAGS"
export LD_LIBRARY_PATH=$GOPATH/lib:$LD_LIBRARY_PATH

go clean -testcache

cd $GOPATH/src/github.com/couchbase/query/test/gsi
for i in test_cases/*
do
    source ./exportval.sh $*
    cd $i
    go test $verbose -p 1 -tags enterprise ./...
    cd ../..
    source ./resetval.sh $*
done
