#!/bin/bash

echo Delete Buckets

Site=http://127.0.0.1:8091/pools/default/buckets/
Auth=Administrator:password
bucket=(customer orders product purchase review)


echo POST /pools/default/buckets

for i in "${bucket[@]}"
do
echo curl -u $Auth $Site$i
curl -X DELETE -u $Auth $Site$i
done

echo rm -rf data/
rm -rf data/


