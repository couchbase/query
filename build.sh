#!/bin/bash
#
# to build the enterprise version, with schema inferencing, launch this 
# as './build.sh -tags "enterprise"

echo go get $* -d -v ./...
go get $* -d -v ./...

echo cd parser/n1ql
cd parser/n1ql
./build.sh $*
cd ../..

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
