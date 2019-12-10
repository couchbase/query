#!/bin/bash
#
# to build the enterprise version, launch this 
# as './build.sh -tags "enterprise"

args=$*

enterprise=0
while [ $# -gt 0 ]; do
  case $1 in
    -tags)
      shift
      [[ "$1" == "enterprise" ]] && enterprise=1
      ;;
  esac
  shift
done

set -- $args

echo go get $* -d -v ./...
go get $* -d -v ./...

echo cd parser/n1ql
cd parser/n1ql
./build.sh $*
cd ../..

echo go fmt ./...
go fmt ./...
if [[ $enterprise == 1 ]]; then
  echo go fmt ../query-ee/...
  cd ../query-ee
  go fmt ./...
  cd ../query
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
