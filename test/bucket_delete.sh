#!/bin/bash

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

