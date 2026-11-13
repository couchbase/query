#!/bin/bash
# Copyright 2018-Present Couchbase, Inc.
#
# Use of this software is governed by the Business Source License included in
# the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
# file, in accordance with the Business Source License, use of this software
# will be governed by the Apache License, Version 2.0, included in the file
# licenses/APL2.txt.

CHILD=
TESTDIR=$(cd "$(dirname "$0")" && pwd)
function cleanup()
{
  # may be called from any directory (e.g. interrupt during the unit test step)
  cd "$TESTDIR"
  ./bucket_delete.sh
}
function interrupt()
{
  if [ "X$CHILD" != "X" ]
  then
    kill -n 2 $CHILD
  fi
  echo -e "\n\n\nInterrupted\n\n"
  cleanup
  exit 1
}

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

# the rest of the script runs relative to test/
cd "$TESTDIR"

export CGO_CFLAGS="-I$GOPATH/src/github.com/couchbase/eventing-ee/evaluator/worker/include  -I$GOPATH/src/github.com/couchbase/sigar/include $CGO_CFLAGS"
export CGO_LDFLAGS="-L$GOPATH/lib ${CGO_LDFLAGS}"
export LD_LIBRARY_PATH=$GOPATH/lib:${LD_LIBRARY_PATH}


go clean -testcache

./bucket_delete.sh
./bucket_create.sh

trap interrupt 2 15
cd ../

# test/gsi packages only run with GSI_TEST set, so leave them to runtests.sh below
pkgs=`go list -e ./... | grep -v /test/gsi/`
if [ -z "$pkgs" ] ; then
     echo "go list found no packages"
     cleanup
     exit 1
fi

if [[ `uname` == "Darwin" ]] ; then
     go test -exec "env LD_LIBRARY_PATH=${LD_LIBRARY_PATH} DYLD_LIBRARY_PATH=${LD_LIBRARY_PATH}" $verbose $pkgs $*
else
     go test $verbose $pkgs $*
fi
rc=$?

#Run gsi
cd test/gsi
./runtests.sh $verbose $skip $*
gsirc=$?

cleanup

if [ $rc -ne 0 -o $gsirc -ne 0 ] ; then
     exit 1
fi
