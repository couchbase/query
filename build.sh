#!/bin/bash

# Copyright 2014-Present Couchbase, Inc.
#
# Use of this software is governed by the Business Source License included in
# the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
# file, in accordance with the Business Source License, use of this software will
# be governed by the Apache License, Version 2.0, included in the file
# licenses/APL.txt.

#
# To build the enterprise version, launch this  AS './build.sh -tags enterprise'
# To build the enterprise version with latest updates, launch this  AS './build.sh -u -tags enterprise'
# Add -s to fix standalone build issues. Keep indexer, eventing-ee generated files in ~/devbld

PRODUCT_VERSION=${PRODUCT_VERSION:-"7.1.0-local_build"}
export PRODUCT_VERSION

args=""

enterprise=0
uflag=
sflag=0
while [ $# -gt 0 ]; do
  case $1 in
    -tags)
      shift
      [[ "$1" == "enterprise" ]] && enterprise=1
      args="$args -tags $1"
      ;;
    -u) uflag=-u ;;
    -s) sflag=1 ;;
    *) args="$args $1" ;;
  esac
  shift
done

set -- $args

DevStandaloneSetup() {
    # curl fix match manifest
       (cd ../../couchbasedeps/go-curl; git checkout 20161221-couchbase)
    # indexer generated files
       if [[ (! -f ../indexing/secondary/protobuf/query/query.pb.go) && ( -f ~/devbld/query.pb.go ) ]]; then
           cp ~/devbld/query.pb.go ../indexing/secondary/protobuf/query/query.pb.go
       fi
    # eventing-ee generated files
       if [[ ( ! -d $GOPATH/lib ) ]]; then
           if [[ "Linux" = `uname` ]]
           then
             JSEVAL=~/devbld/build/goproj/src/github.com/couchbase/eventing-ee/evaluator/libjseval.so
             [[ ! -f $JSEVAL ]] && JSEVAL=/opt/couchbase/lib/libjseval.so
           else #macos
             JSEVAL=~/devbld/build/goproj/src/github.com/couchbase/eventing-ee/evaluator/libjseval.dylib
             [[ ! -f $JSEVAL ]] && JSEVAL="/Applications/Couchbase Server.app/Contents/Resources/couchbase-core/lib/libjseval.dylib"
           fi
           if [[ "X" != "X$JSEVAL"  &&  -f $JSEVAL ]]
           then
             mkdir $GOPATH/lib
             cp -rp $JSEVAL $GOPATH/lib
           fi
       fi
    # gocbcore points to master; gocbcore/v9 points to 9.1.6
       if [[ -d ../gocbcore/v9 ]]
       then
           cd ../gocbcore/v9
           C=`git describe --tags|grep -c "v9.1.6"`
           cd -
       else
           C=0
       fi
       if [[ $C -eq 0 ]]
       then
           (cd ..; rm -rf gocbcore/v9; git clone -b v9.1.6 https://github.com/couchbase/gocbcore.git gocbcore/v9)
       fi
    # bleve version
       if [[ ! -d ../../blevesearch/bleve/v2 ]]; then
           (cd ../../blevesearch; git clone -b v2.1.0 http://github.com/blevesearch/bleve.git bleve/v2)
       fi
    # zapx versions
       if [[ ! -d ../../blevesearch/zapx/v11 ]]; then
           (cd ../../blevesearch; git clone -b v11.2.2 http://github.com/blevesearch/zapx.git zapx/v11)
       fi
       if [[ ! -d ../../blevesearch/zapx/v12 ]]; then
           (cd ../../blevesearch; git clone -b v12.2.2 http://github.com/blevesearch/zapx.git zapx/v12)
       fi
       if [[ ! -d ../../blevesearch/zapx/v13 ]]; then
           (cd ../../blevesearch; git clone -b v13.2.2 http://github.com/blevesearch/zapx.git zapx/v13)
       fi
       if [[ ! -d ../../blevesearch/zapx/v14 ]]; then
           (cd ../../blevesearch; git clone -b v14.2.2 http://github.com/blevesearch/zapx.git zapx/v14)
       fi
       if [[ ! -d ../../blevesearch/zapx/v15 ]]; then
           (cd ../../blevesearch; git clone -b v15.2.2 http://github.com/blevesearch/zapx.git zapx/v15)
       fi
       (cd $GOPATH/src/golang.org/x/net; git checkout `go version |  awk -F'[. ]' '{print "release-branch." $3 "." $4}'`)
}

# turn off go module for non repo sync build or standalone build
if [[ ( ! -d ../../../../../cbft && "$GOPATH" != "") || ( $sflag == 1) ]]; then
     export GO111MODULE=off
     export CGO_CFLAGS="-I$GOPATH/src/github.com/couchbase/eventing-ee/evaluator/worker/include $CGO_FLAGS"
     export CGO_LDFLAGS="-L$GOPATH/lib $CGO_LDLAGS"
     echo go get $* $uflag -d -v ./...
     go get $* $uflag -d -v ./...
     if [[ $sflag == 1 ]]; then
         DevStandaloneSetup
     fi
fi


echo cd parser/n1ql
cd parser/n1ql
./build.sh $*
cd ../..

echo go fmt ./...
go fmt ./...
if [[ $enterprise == 1 ]]; then
  (echo go fmt ../query-ee/...; cd ../query-ee; export GO111MODULE=off; go fmt ./...)
fi

echo cd server/cbq-engine
cd server/cbq-engine
./build.sh $*
cd ../..

echo cd shell/cbq
cd shell/cbq
./build.sh $*
cd ../..

echo cd tutorial
cd tutorial
./build.sh $*
cd ..

echo go install  $* ./...
go install $* ./...
