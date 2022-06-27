#!/bin/bash
# Copyright 2016-Present Couchbase, Inc.
#
# Use of this software is governed by the Business Source License included in
# the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
# file, in accordance with the Business Source License, use of this software
# will be governed by the Apache License, Version 2.0, included in the file
# licenses/APL2.txt.

args=""
skiptests=""
verbose=
while [ $# -gt 0 ]; do
  case $1 in
    -v) verbose="$1" ;;
    -s) skiptests="test_cases/curl test_cases/fts" ;;
    *) args="$args $1" ;;
  esac
  shift
done

set -- $args

verbose=$1
Site=http://127.0.0.1:8091/pools/nodes/
Auth=Administrator:password

export GO111MODULE=off
export CGO_CFLAGS="-I$GOPATH/src/github.com/couchbase/eventing-ee/evaluator/worker/include -I$GOPATH/src/github.com/couchbase/sigar/include $CGO_FLAGS"
export CGO_LDFLAGS="-L$GOPATH/lib ${CGO_LDFLAGS}"
export LD_LIBRARY_PATH=$GOPATH/lib:${LD_LIBRARY_PATH}

go clean -testcache

cd $GOPATH/src/github.com/couchbase/query/test/gsi
for i in test_cases/*
do
    if [[ $skiptests =~ (^|[[:space:]])"$i"($|[[:space:]]) ]] ; then
        continue
    fi
    source ./exportval.sh $*
    cd $i
    if [[ `uname` == "Darwin" ]] ; then
        go test -exec "env LD_LIBRARY_PATH=${LD_LIBRARY_PATH} DYLD_LIBRARY_PATH=${LD_LIBRARY_PATH}" $verbose -p 1 -tags enterprise ./...
    else
        go test $verbose -p 1 -tags enterprise ./...
    fi
    cd ../..
    source ./resetval.sh $*
done
