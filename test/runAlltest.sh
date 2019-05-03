#!/bin/bash
go clean -testcache

./bucket_delete.sh
./bucket_create.sh 100

cd ../
go test ./...

#Run gsi
cd test/gsi
./runtests.sh

cd ../
./bucket_delete.sh
