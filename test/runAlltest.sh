#!/bin/bash

# Copyright 2018-Present Couchbase, Inc.
#
# Use of this software is governed by the Business Source License included in
# the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
# file, in accordance with the Business Source License, use of this software will
# be governed by the Apache License, Version 2.0, included in the file
# licenses/APL.txt.

export GO111MODULE=off
export CGO_CFLAGS="-I$GOPATH/src/github.com/couchbase/eventing-ee/evaluator/worker/include $CGO_FLAGS"

go clean -testcache
verbose=$1

./bucket_delete.sh
./bucket_create.sh 100

cd ../
go test $verbose ./...

#Run gsi
cd test/gsi
./runtests.sh $verbose

cd ../
./bucket_delete.sh
