#!/bin/bash

# Copyright 2022-Present Couchbase, Inc.
#
# Use of this software is governed by the Business Source License included in
# the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
# file, in accordance with the Business Source License, use of this software will
# be governed by the Apache License, Version 2.0, included in the file
# licenses/APL.txt.

if [ $# -ne 1 ]
then
  echo "USAGE: $0 test-name"
  echo "  Run a single GSI test."
  echo
  echo "   e.g. $0 string_functions"
  echo
  echo "This script allows you to speed up repeat testing when working on a specific issue."
  echo "You should have run [1;32;40mbucket_create.sh[0m before running this and once done with repeated runs, run [1;31;40mbucket_delete.sh[0m to clean up."
  echo
  exit 2
else
  TEST=$1
fi
Site=http://127.0.0.1:8091/pools/nodes/
Auth=Administrator:password

export CGO_CFLAGS="-I$GOPATH/src/github.com/couchbase/eventing-ee/evaluator/worker/include -I$GOPATH/src/github.com/couchbase/sigar/include $CGO_FLAGS"
export CGO_LDFLAGS="-L$GOPATH/lib -Wl,-no_warn_duplicate_libraries $CGO_LDLAGS"
export LD_LIBRARY_PATH=$GOPATH/lib:$LD_LIBRARY_PATH
if [[ `uname` == "Darwin" ]]
then
  export DYLD_LIBRARY_PATH=${LD_LIBRARY_PATH}
  export JSEVALUATOR_PATH="/Applications/Couchbase Server.app/Contents/Resources/couchbase-core/bin"
elif [[ "Linux" = `uname` ]]
then
  export JSEVALUATOR_PATH="/opt/couchbase/bin"
fi

go clean -cache -modcache -testcache -i -r

cd $GOPATH/src/github.com/couchbase/query/test/gsi
i=test_cases/$TEST
source ./exportval.sh
cd $i
# strip indexer client info-level messages to retain some utility in the output
(go test -v -p 1 -tags enterprise ./... 2>&1) | grep -v "\[Info\]" | grep -v "\[Warn\]" | grep -v "Index inst \: \[partitions\]"
cd ../..
source ./resetval.sh
