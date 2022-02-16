#!/bin/bash

# Copyright 2014-Present Couchbase, Inc.
#
# Use of this software is governed by the Business Source License included in
# the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
# file, in accordance with the Business Source License, use of this software
# will be governed by the Apache License, Version 2.0, included in the file
# licenses/APL2.txt.

#
# To build the enterprise version, launch this  AS './build.sh -tags enterprise'
# To build the enterprise version with latest updates, launch this  AS './build.sh -u -tags enterprise'
# Add -s to fix standalone build issues. Keep indexer generated files in ~/devbld
# Note standalone build requires libraries from installed server, make sure installed server is
# compatible with source that is being built

PRODUCT_VERSION=${PRODUCT_VERSION:-"7.1.0-local_build"}
export PRODUCT_VERSION

args=""

enterprise=0
uflag=
sflag=0
fflag=1
while [ $# -gt 0 ]; do
  case $1 in
    -tags)
      shift
      [[ "$1" == "enterprise" ]] && enterprise=1
      args="$args -tags $1"
      ;;
    -u) uflag=-u ;;
    -s) sflag=1 ;;
    -nofmt) fflag=0 ;;
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
       if [[ ( ! -L $GOPATH/lib ) ]]; then
           if [[ -d $GOPATH/lib ]]
           then
             rm -rf $GOPATH/lib
           fi
           if [[ "Linux" = `uname` ]]
           then
             ln -s /opt/couchbase/lib $GOPATH/lib
           elif [[ "Darwin" = `uname` ]]
           then
             ln -s "/Applications/Couchbase Server.app/Contents/Resources/couchbase-core/lib" $GOPATH/lib
           fi
       fi
       if [[ ! -f ../eventing-ee/evaluator/impl/gen/parser/global_config_schema.go ]]; then
           (cd ../eventing-ee/evaluator/impl/gen/convertschema; go run generate.go  ../../parser/global_config_schema.json GlobalConfigSchema ../parser/global_config_schema.go)
       fi
    # gocbcore points to master; gocbcore/v9 points to 9.1.8
       if [[ -d ../gocbcore/v9 ]]
       then
           cd ../gocbcore/v9
           C=`git describe --tags|grep -c "v9.1.8"`
           cd -
       else
           C=0
       fi
       if [[ $C -eq 0 ]]
       then
           (cd ..; rm -rf gocbcore/v9; git clone -b v9.1.8 https://github.com/couchbase/gocbcore.git gocbcore/v9)
       fi
    # gocbcore points to master; gocbcore/v10 points to 10.0.8
       if [[ -d ../gocbcore/v10 ]]
       then
           cd ../gocbcore/v10
           C=`git describe --tags|grep -c "v10.0.8"`
           cd -
       else
           C=0
       fi
       if [[ $C -eq 0 ]]
       then
           (cd ..; rm -rf gocbcore/v10; git clone -b v10.0.8 https://github.com/couchbase/gocbcore.git gocbcore/v10)
       fi
    # bleve version
       if [[ ! -d ../../blevesearch/bleve/v2 ]]; then
           (cd ../../blevesearch; git clone -b v2.2.2 http://github.com/blevesearch/bleve.git bleve/v2)
       fi
    # zapx versions
       if [[ ! -d ../../blevesearch/zapx/v11 ]]; then
           (cd ../../blevesearch; git clone -b v11.3.1 http://github.com/blevesearch/zapx.git zapx/v11)
       fi
       if [[ ! -d ../../blevesearch/zapx/v12 ]]; then
           (cd ../../blevesearch; git clone -b v12.3.1 http://github.com/blevesearch/zapx.git zapx/v12)
       fi
       if [[ ! -d ../../blevesearch/zapx/v13 ]]; then
           (cd ../../blevesearch; git clone -b v13.3.1 http://github.com/blevesearch/zapx.git zapx/v13)
       fi
       if [[ ! -d ../../blevesearch/zapx/v14 ]]; then
           (cd ../../blevesearch; git clone -b v14.3.1 http://github.com/blevesearch/zapx.git zapx/v14)
       fi
       if [[ ! -d ../../blevesearch/zapx/v15 ]]; then
           (cd ../../blevesearch; git clone -b v15.3.1 http://github.com/blevesearch/zapx.git zapx/v15)
       fi
       (cd $GOPATH/src/golang.org/x/net; git checkout `go version |  awk -F'[. ]' '{print "release-branch." $3 "." $4}'`)
}

# turn off go module for non repo sync build or standalone build
if [[ ( ! -d ../../../../../cbft && "$GOPATH" != "") || ( $sflag == 1) ]]; then
     export GO111MODULE=off
     export CGO_CFLAGS="-I$GOPATH/src/github.com/couchbase/eventing-ee/evaluator/worker/include $CGO_FLAGS"
     export CGO_LDFLAGS="-L$GOPATH/lib $CGO_LDFLAGS"
     export LD_LIBRARY_PATH=$GOPATH/lib:${LD_LIBRARY_PATH}
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

if [[ ($fflag != 0) ]]
then
  echo go fmt ./...
  go fmt ./...
  if [[ $enterprise == 1 ]]; then
    (echo go fmt ../query-ee/...; cd ../query-ee; export GO111MODULE=off; go fmt ./...)
  fi
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
