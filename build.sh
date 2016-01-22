#!/bin/bash

echo go get -d -v ./...
go get -d -v ./...

echo cd parser/n1ql
cd parser/n1ql
./build.sh $1
cd ../..

echo cd server/cbq-engine
cd server/cbq-engine
./build.sh $1
cd ../..

echo cd shell/cbq
cd shell/cbq
./build.sh $1
cd ../..

echo cd shell/go_cbq
cd shell/go_cbq
./build.sh $1
cd ../..

echo cd tutorial
cd tutorial
./build.sh $1
cd ..

echo go install  -tags "enterprise" ./...
go install  -tags "enterprise" ./...
