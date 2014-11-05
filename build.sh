#!/bin/bash

echo go get
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

echo cd tutorial
cd tutorial
echo go build
go build
cd ..
