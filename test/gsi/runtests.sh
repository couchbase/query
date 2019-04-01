#!/bin/bash

Site=http://127.0.0.1:8091/pools/nodes/
Auth=Administrator:password
flag=`curl -s -u $Auth $Site | awk '{print match($0,"fts")}'`;
ftstests="test_cases/fts test_cases/ftsclient"

cd $GOPATH/src/github.com/couchbase/query/test/gsi
for i in test_cases/*
do
   
#    if [ $flag -le 0 ];then
        for ft in $ftstests
        do
            if [ $ft == $i ];then
                 continue 2
            fi
        done
#    fi
    source ./exportval.sh $*
    cd $i
    go test  -tags enterprise ./...
    cd ../..
    source ./resetval.sh $*
done

