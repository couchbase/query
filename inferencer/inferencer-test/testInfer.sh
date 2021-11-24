#!/bin/bash

# Copyright 2019-Present Couchbase, Inc.
#
# Use of this software is governed by the Business Source License included in
# the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
# file, in accordance with the Business Source License, use of this software
# will be governed by the Apache License, Version 2.0, included in the file
# licenses/APL2.txt.

#
# create some sample data in a bucket and make sure that INFER works correctly on a variety of schemas
#
# the following three variables control whether or not each step of the process is performed. Leave
# a variable unset to skip that step
#
export host=http://127.0.0.1:9091
export user=Administrator
export pass=bluehorse
export bin=/Users/eben/src/vulcan/install/bin
#
# tests shauld have:
# - a .json file with the data
# - a .schema.json with a valid INFER
tests=("beer_sample" "gamesim_sample" "travel_sample" "simple_docs" "nested_arrays")
#tests=("beer_sample")
#
# create the bucket
#
echo "creating infer_test bucket";
echo $bin/couchbase-cli bucket-create -c $host -u $user -p $pass --bucket infer_test --bucket-type couchbase --bucket-ramsize 100 --bucket-replica 0 --enable-flush 1
$bin/couchbase-cli bucket-create -c $host -u $user -p $pass --bucket infer_test --bucket-type couchbase --bucket-ramsize 100 --bucket-replica 0 --enable-flush 1
curl --user $user:$pass --data "statement=create primary index on infer_test;" $host/_p/query/query/service &> /dev/null
#
# test 1 - beer-sample
#
for test in ${tests[@]}
do  
    echo Test: $test
    yes | $bin/couchbase-cli bucket-flush  -c $host -u $user -p $pass --bucket infer_test  &> /dev/null
    echo "waiting for flush..."
    sleep 5
    $bin/cbimport json -c $host --username $user --password $pass --bucket infer_test --d file://$test.json -f list -g key::#UUID#
    echo "wating for data..."
    sleep 5
    curl --user $user:$pass -H "ns-server-proxy-timeout: 60000" --data "statement=infer infer_test with {\"sample_size\": 50000,\"num_sample_values\":0};" $host/_p/query/query/service 2> /dev/null > $test.out
    cat $test.out | jq .results > $test.out.results
    cat $test.out | jq .errors > $test.out.errors
    export result=`diff -q $test.out.results $test.key`
    if [ -n  "$result" ]
    then
        echo "ERROR - inferred schema doesn't match: " $result
        opendiff $test.out.results $test.key
    fi
done
#
# remove the bucket
#
echo "  deleting the bucket..."
$bin/couchbase-cli bucket-delete -c $host -u $user -p $pass --bucket infer_test 

