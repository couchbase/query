#!/bin/bash
#
# To build the enterprise version, launch this  AS './build.sh -tags enterprise'
# To build the enterprise version with latest updates, launch this  AS './build.sh -u -tags enterprise'
# Add -s to fix standalone build issues. Keep indexer, eventing-ee generated files in ~/devbld


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
	   mkdir $GOPATH/lib
           cp -rp ~/devbld/build/goproj/src/github.com/couchbase/eventing-ee/evaluator/libjseval.* $GOPATH/lib
       fi
    # gocbcore v9 version point to master
       if [[ ! -h ../gocbcore/v9 ]]; then
           (cd ../gocbcore; ln -s . v9)
       fi
    # bleve version
       if [[ ! -d ../../blevesearch/bleve/v2 ]]; then
           (cd ../../blevesearch; git clone -b v2.0.2 http://github.com/blevesearch/bleve.git bleve/v2)
       fi
    # zapx versions
       if [[ ! -d ../../blevesearch/zapx/v11 ]]; then
           (cd ../../blevesearch; git clone -b v11.2.0 http://github.com/blevesearch/zapx.git zapx/v11)
       fi
       if [[ ! -d ../../blevesearch/zapx/v12 ]]; then
           (cd ../../blevesearch; git clone -b v12.2.0 http://github.com/blevesearch/zapx.git zapx/v12)
       fi
       if [[ ! -d ../../blevesearch/zapx/v13 ]]; then
           (cd ../../blevesearch; git clone -b v13.2.0 http://github.com/blevesearch/zapx.git zapx/v13)
       fi
       if [[ ! -d ../../blevesearch/zapx/v14 ]]; then
           (cd ../../blevesearch; git clone -b v14.2.0 http://github.com/blevesearch/zapx.git zapx/v14)
       fi
       if [[ ! -d ../../blevesearch/zapx/v15 ]]; then
           (cd ../../blevesearch; git clone -b v15.2.0 http://github.com/blevesearch/zapx.git zapx/v15)
       fi
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

echo cd shell/cbq.old
cd shell/cbq.old
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
