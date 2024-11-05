#!/bin/bash

# Copyright 2019-Present Couchbase, Inc.
#
# Use of this software is governed by the Business Source License included in
# the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
# file, in accordance with the Business Source License, use of this software
# will be governed by the Apache License, Version 2.0, included in the file
# licenses/APL2.txt.

Site=http://127.0.0.1:8091/pools/default/buckets/
Auth=Administrator:password
bucket=(customer orders product purchase review shellTest)

for i in "${bucket[@]}"
do
curl --silent -X DELETE -u $Auth $Site$i > /dev/null
done

cd filestore
rm -rf data/
cd ../

UserSite=http://localhost:8091/settings/rbac/users/local/
for i in "${bucket[@]}"
do
Id=${i}owner
curl --silent -X DELETE -u $Auth $UserSite$Id > /dev/null 
done

curl --silent -X DELETE -u $Auth $UserSite/testAdmin > /dev/null 

