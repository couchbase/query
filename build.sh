#!/bin/bash

echo go get
go get -d -v ./...

echo cd parser/n1ql
cd parser/n1ql
./build.sh $1
cd ../..

echo cd server/main
cd server/main
./build.sh $1
cd ../..

echo cd shell
cd shell
./build.sh $1
cd ..

echo cd tutorial
cd tutorial
echo go build
go build
cd ..
