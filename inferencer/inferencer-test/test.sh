# Copyright 2019-Present Couchbase, Inc.
#
# Use of this software is governed by the Business Source License included in
# the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
# file, in accordance with the Business Source License, use of this software
# will be governed by the Apache License, Version 2.0, included in the file
# licenses/APL2.txt.

export test=beer_sample
    export result=`diff -q $test.out.json $test.out.json`
    echo "result is: *" $result "*"
    if [ -n  "$result" ]
    then
        echo "ERROR - inferred schema doesn't match: " $result
    fi
