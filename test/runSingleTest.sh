#!/bin/bash
# Copyright 2022-Present Couchbase, Inc.
#
# Use of this software is governed by the Business Source License included in
# the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
# file, in accordance with the Business Source License, use of this software
# will be governed by the Apache License, Version 2.0, included in the file
# licenses/APL2.txt.

function usage {
  echo -e "USAGE: $0 [-v] package [package...]"
  echo -e "\ne.g. \"$0 value\" to run the go tests for query/value"
  echo -e "     \"$0 query-ee/updstat\" to run the go tests for query-ee/updstat (\"query-ee/...\" for all of query-ee)\n"
  echo -e "Don't use this script to launch tests requiring buckets.\n"
  return 2
}

if [ $# -lt 1 ]
then
  usage
  exit $?
fi

verbose=
args=

while [ $# -gt 0 ]
do
  if [ "X$1" == "X-v" ]
  then
    verbose="-v"
  elif [[ "$1" == query-ee || "$1" == query-ee/* ]]
  then
    # query-ee is a separate module (go.mod replaces it with ../query-ee): use its import path
    args="$args github.com/couchbase/$1"
  else
    args="$args ./$1"
  fi
  shift
done

set -- $args
if [ $# -lt 1 ]
then
  usage
  exit $?
fi

# Source shared functions
source "$(dirname "$0")/../build_util.sh"
OPENSSL_VERSION=$(get_openssl_version)

export CGO_CFLAGS="-I$GOPATH/src/github.com/couchbase/eventing-ee/evaluator/worker/include  -I$GOPATH/src/github.com/couchbase/sigar/include -I$GOPATH/src/couchbasedeps/openssl/$OPENSSL_VERSION/include $CGO_CFLAGS"
export CGO_LDFLAGS="-L$GOPATH/lib ${CGO_LDFLAGS}"
export LD_LIBRARY_PATH=$GOPATH/lib:${LD_LIBRARY_PATH}

go clean -testcache

cd ../
if [[ `uname` == "Darwin" ]] ; then
     go test -tags enterprise -exec "env LD_LIBRARY_PATH=${LD_LIBRARY_PATH} DYLD_LIBRARY_PATH=${LD_LIBRARY_PATH}" $verbose $*
else
     go test -tags enterprise $verbose $*
fi
