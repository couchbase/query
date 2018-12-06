#!/bin/bash
cd multistore/
./bucket_delete.sh
./bucket_create.sh 100

cd ../../
go test ./...

#Run gsi
cd test/gsi
./runtests.sh

cd ../multistore
./bucket_delete.sh