#!/bin/bash
# Copyright 2018-Present Couchbase, Inc.
#
# Use of this software is governed by the Business Source License included in
# the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
# file, in accordance with the Business Source License, use of this software
# will be governed by the Apache License, Version 2.0, included in the file
# licenses/APL2.txt.
args=""
verbose=
skip=
while [ $# -gt 0 ]; do
  case $1 in
    -v) verbose="$1" ;;
    -s) skip="$1" ;;
    *) args="$args $1" ;;
  esac
  shift
done

set -- $args

export CGO_CFLAGS="-I$GOPATH/src/github.com/couchbase/eventing-ee/evaluator/worker/include $CGO_FLAGS"

go clean -testcache

./bucket_delete.sh
./bucket_create.sh 100

cd ../
go test $verbose ./... $*

#Run gsi
cd test/gsi
./runtests.sh $verbose $skip $*

cd ../
./bucket_delete.sh
