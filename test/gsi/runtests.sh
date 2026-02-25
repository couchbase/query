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
    -s) skiptests="test_cases/curl test_cases/fts test_cases/natural" ;;
    *) args="$args $1" ;;
  esac
  shift
done

set -- $args

verbose=$1
Site=http://127.0.0.1:8091/pools/nodes/
Auth=Administrator:password

# Source shared functions
source "$(dirname "$0")/../../build_util.sh"
OPENSSL_VERSION=$(get_openssl_version)

export CGO_CFLAGS="-I$GOPATH/src/github.com/couchbase/eventing-ee/evaluator/worker/include -I$GOPATH/src/github.com/couchbase/sigar/include -I$GOPATH/src/couchbasedeps/openssl/$OPENSSL_VERSION/include $CGO_FLAGS"
export CGO_LDFLAGS="-L$GOPATH/lib ${CGO_LDFLAGS}"
export LD_LIBRARY_PATH=$GOPATH/lib:${LD_LIBRARY_PATH}

go clean -testcache

cd $GOPATH/src/github.com/couchbase/query/test/gsi
for i in test_cases/*
do
    if [[ $skiptests =~ (^|[[:space:]])"$i"($|[[:space:]]) ]] ; then
        continue
    fi

    if [[ $i == "test_cases/natural" ]] ; then 
        read -sp "Enter natural_cred: " natural_cred;echo; export natural_cred;echo;
        read -sp "Enter natural_orgid: " natural_orgid;echo; export natural_orgid;echo;
    fi

    source ./exportval.sh $*
    cd $i
    dir=`pwd`
    if [[ `uname` == "Darwin" ]] ; then
        if [[ $i == "test_cases/vector_search" ]] ; then
            /Applications/Couchbase\ Server.app/Contents/Resources/couchbase-core/bin/cbimport json -c couchbase://127.0.0.1 -u Administrator -p password -b product -g %docKey% -d file://${dir}/product_export.json -f list --scope-collection-exp %my_scope%.%my_collection% > ${dir}/cbimport.out
        fi
        export JSEVALUATOR_PATH="/Applications/Couchbase Server.app/Contents/Resources/couchbase-core/bin"
        (go test -exec "env LD_LIBRARY_PATH=${LD_LIBRARY_PATH} DYLD_LIBRARY_PATH=${LD_LIBRARY_PATH}" $verbose -p 1 -tags enterprise ./... 2>&1) | grep -v "\[Info\]" | grep -v "\[Warn\]" | grep -v "Index inst \: \[partitions\]"
    else
        if [[ $i == "test_cases/vector_search" ]] ; then
            /opt/couchbase/bin/cbimport json -c couchbase://127.0.0.1 -u Administrator -p password -b product -g %docKey% -d file://${dir}/product_export.json -f list --scope-collection-exp %my_scope%.%my_collection% > ${dir}/cbimport.out
        fi
        export JSEVALUATOR_PATH="/opt/couchbase/bin"
        (go test $verbose -p 1 -tags enterprise ./... 2>&1) | grep -v "\[Info\]" | grep -v "\[Warn\]" | grep -v "Index inst \: \[partitions\]"
    fi
    cd ../..
    source ./resetval.sh $*
done
