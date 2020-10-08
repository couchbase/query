#!/bin/bash

export GO111MODULE=off

go clean -testcache
verbose=$1

./bucket_delete.sh
./bucket_create.sh 100

cd ../
go test $verbose ./...

#Run gsi
cd test/gsi
./runtests.sh $verbose

cd ../
./bucket_delete.sh
