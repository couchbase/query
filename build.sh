#!/bin/bash

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
